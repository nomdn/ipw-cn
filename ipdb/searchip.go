package ipdb

import (
	"bufio"

	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	ipdbgo "github.com/ipipdotnet/ipdb-go"
	"github.com/lionsoul2014/ip2region/binding/golang/service"
	maxminddb "github.com/oschwald/maxminddb-golang/v2"
	"resty.dev/v3"
)

var (
	ip2region     *service.Ip2Region
	ip2regionMu   sync.RWMutex
	qqwryDB       *ipdbgo.City
	qqwryMu       sync.RWMutex
	mmdbDBs       map[string]*maxminddb.Reader
	mmdbMu        sync.RWMutex
	divisionFull  map[int]string
	divisionShort map[int]string
	bilibiliCache sync.Map
)

type bilibiliCacheEntry struct {
	result    *BilibiliResult
	timestamp time.Time
}

type MMDBCityResult struct {
	Country     string `json:"country"`
	CountryCode string `json:"country_code"`
	Region      string `json:"administrative_area"`
	City        string `json:"city"`
}

type MMDBASNResult struct {
	ASN string `json:"asn"`
	Org string `json:"org"`
}

type GeoCNResult struct {
	DivisionCode string `json:"division_code"`
	ISP          string `json:"isp"`
	Region       string `json:"administrative_area,omitempty"`
	City         string `json:"city,omitempty"`
	District     string `json:"district,omitempty"`
}

type BilibiliIPQueryResponse struct {
	Code    int                 `json:"code"`
	Msg     string              `json:"msg"`
	Message string              `json:"message"`
	Data    BilibiliIPQueryData `json:"data"`
}

type BilibiliIPQueryData struct {
	Addr      string  `json:"addr"`
	Country   string  `json:"country"`
	Province  string  `json:"province"`
	City      string  `json:"city"`
	ISP       string  `json:"isp"`
	Latitude  float64 `json:"latitude,string"`
	Longitude float64 `json:"longitude,string"`
}

type BilibiliResult struct {
	Country   string  `json:"country"`
	Region    string  `json:"administrative_area"`
	City      string  `json:"city"`
	ISP       string  `json:"isp"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func loadDivisionCode(path string, sep string) (map[int]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	m := make(map[int]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, sep, 2)
		if len(parts) != 2 {
			continue
		}
		code, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}
		m[code] = strings.TrimSpace(parts[1])
	}
	return m, scanner.Err()
}

func resolveDivision(code int) (province, city, district string) {
	provinceCode := (code / 10000) * 10000
	cityCode := (code / 100) * 100

	if v, ok := divisionFull[provinceCode]; ok {
		province = v
	}
	if cityCode != provinceCode {
		if v, ok := divisionFull[cityCode]; ok {
			city = v
		}
	}
	if code != cityCode && code != provinceCode {
		if v, ok := divisionFull[code]; ok {
			district = v
		}
	}
	return
}

func loadDivisionData() error {
	var err error
	divisionFull, err = loadDivisionCode("full.txt", "\t")
	if err != nil {
		slog.Error("failed to load full.txt", "error", err)
		return err
	}
	divisionShort, err = loadDivisionCode("short.txt", "  ")
	if err != nil {
		slog.Error("failed to load short.txt", "error", err)
		return err
	}
	slog.Info("division data loaded", "full", len(divisionFull), "short", len(divisionShort))
	return nil
}

func loadIp2Region() error {
	ip2regionMu.Lock()
	defer ip2regionMu.Unlock()

	if ip2region != nil {
		ip2region.Close()
		slog.Info("Previous ip2region closed")
	}

	v4Config, err := service.NewV4Config(service.VIndexCache, "ip2region_v4.xdb", 20)
	if err != nil {
		slog.Error("failed to create v4 config", "error", err)
		return err
	}

	v6Config, err := service.NewV6Config(service.VIndexCache, "ip2region_v6.xdb", 20)
	if err != nil {
		slog.Error("failed to create v6 config", "error", err)
		return err
	}

	ip2region, err = service.NewIp2Region(v4Config, v6Config)
	if err != nil {
		slog.Error("failed to create ip2region service", "error", err)
		return err
	}

	slog.Info("ip2region loaded successfully")
	return nil
}

func loadQQWry() error {
	qqwryMu.Lock()
	defer qqwryMu.Unlock()
	if qqwryDB != nil {
		qqwryDB = nil
		slog.Info("Previous qqwry.ipdb closed")
	}
	var err error
	qqwryDB, err = ipdbgo.NewCity("qqwry.ipdb")
	if err != nil {
		slog.Error("failed to load qqwry.ipdb", "error", err)
		return err
	}
	slog.Info("qqwry.ipdb loaded successfully")
	return nil
}

func reloadQQWry() error {
	// 1. 先创建新实例
	newDB, err := ipdbgo.NewCity("qqwry.ipdb")
	if err != nil {
		return fmt.Errorf("reload qqwry failed: %w", err)
	}

	// 2. 加锁替换（旧实例由 GC 回收，ipdb-go 底层 mmap 会自动释放）
	qqwryMu.Lock()
	qqwryDB = newDB
	qqwryMu.Unlock()

	slog.Info("qqwry.ipdb reloaded successfully")
	return nil
}

func loadMMDB() error {
	mmdbMu.Lock()
	defer mmdbMu.Unlock()

	for name, db := range mmdbDBs {
		db.Close()
		slog.Info("Previous mmdb closed", "name", name)
	}

	mmdbFiles := map[string]string{
		"geolite2_city": "GeoLite2-City.mmdb",
		"geolite2_asn":  "GeoLite2-ASN.mmdb",
		"geocn":         "GeoCN.mmdb",
		"dbip_city":     "dbip-city-lite.mmdb",
	}

	mmdbDBs = make(map[string]*maxminddb.Reader)
	for name, path := range mmdbFiles {
		db, err := maxminddb.Open(path)
		if err != nil {
			slog.Error("failed to load mmdb", "name", name, "error", err)
			continue
		}
		mmdbDBs[name] = db
		slog.Info("mmdb loaded", "name", name)
	}
	if len(mmdbDBs) < len(mmdbFiles) {
		return fmt.Errorf("some mmdb files missing (%d/%d loaded)", len(mmdbDBs), len(mmdbFiles))
	}
	return nil
}

func reloadMMDB() error {
	mmdbMu.Lock()
	defer mmdbMu.Unlock()
	for name, db := range mmdbDBs {
		db.Close()
		slog.Info("Previous mmdb closed", "name", name)
	}

	mmdbFiles := map[string]string{
		"geolite2_city": "GeoLite2-City.mmdb",
		"geolite2_asn":  "GeoLite2-ASN.mmdb",
		"geocn":         "GeoCN.mmdb",
		"dbip_city":     "dbip-city-lite.mmdb",
	}

	mmdbDBs = make(map[string]*maxminddb.Reader)
	for name, path := range mmdbFiles {
		db, err := maxminddb.Open(path)
		if err != nil {
			slog.Error("failed to reload mmdb", "name", name, "error", err)
			continue
		}
		mmdbDBs[name] = db
		slog.Info("mmdb reloaded", "name", name)
	}
	return nil
}

func searchMMDBCity(ip string) (*MMDBCityResult, error) {
	mmdbMu.RLock()
	defer mmdbMu.RUnlock()
	db, ok := mmdbDBs["geolite2_city"]
	if !ok {
		return nil, fmt.Errorf("geolite2_city not loaded")
	}

	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return nil, err
	}

	var record struct {
		Country struct {
			ISOCode string            `maxminddb:"iso_code"`
			Names   map[string]string `maxminddb:"names"`
		} `maxminddb:"country"`
		Subdivisions []struct {
			Names map[string]string `maxminddb:"names"`
		} `maxminddb:"subdivisions"`
		City struct {
			Names map[string]string `maxminddb:"names"`
		} `maxminddb:"city"`
	}
	if err := db.Lookup(addr).Decode(&record); err != nil {
		return nil, err
	}

	result := &MMDBCityResult{
		CountryCode: record.Country.ISOCode,
	}
	if name, ok := record.Country.Names["zh-CN"]; ok {
		result.Country = name
	} else if name, ok := record.Country.Names["en"]; ok {
		result.Country = name
	}
	if len(record.Subdivisions) > 0 {
		if name, ok := record.Subdivisions[0].Names["zh-CN"]; ok {
			result.Region = name
		} else if name, ok := record.Subdivisions[0].Names["en"]; ok {
			result.Region = name
		}
	}
	if name, ok := record.City.Names["zh-CN"]; ok {
		result.City = name
	} else if name, ok := record.City.Names["en"]; ok {
		result.City = name
	}
	return result, nil
}

func searchDBIPCity(ip string) (*MMDBCityResult, error) {
	mmdbMu.RLock()
	defer mmdbMu.RUnlock()
	db, ok := mmdbDBs["dbip_city"]
	if !ok {
		return nil, fmt.Errorf("dbip_city not loaded")
	}

	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return nil, err
	}

	var record struct {
		Country struct {
			ISOCode string            `maxminddb:"iso_code"`
			Names   map[string]string `maxminddb:"names"`
		} `maxminddb:"country"`
		Subdivisions []struct {
			Names map[string]string `maxminddb:"names"`
		} `maxminddb:"subdivisions"`
		City struct {
			Names map[string]string `maxminddb:"names"`
		} `maxminddb:"city"`
	}
	if err := db.Lookup(addr).Decode(&record); err != nil {
		return nil, err
	}

	result := &MMDBCityResult{
		CountryCode: record.Country.ISOCode,
	}
	if name, ok := record.Country.Names["zh-CN"]; ok {
		result.Country = name
	} else if name, ok := record.Country.Names["en"]; ok {
		result.Country = name
	}
	if len(record.Subdivisions) > 0 {
		if name, ok := record.Subdivisions[0].Names["zh-CN"]; ok {
			result.Region = name
		} else if name, ok := record.Subdivisions[0].Names["en"]; ok {
			result.Region = name
		}
	}
	if name, ok := record.City.Names["zh-CN"]; ok {
		result.City = name
	} else if name, ok := record.City.Names["en"]; ok {
		result.City = name
	}
	return result, nil
}

func searchMMDBASN(ip string) (*MMDBASNResult, error) {
	mmdbMu.RLock()
	defer mmdbMu.RUnlock()
	db, ok := mmdbDBs["geolite2_asn"]
	if !ok {
		return nil, fmt.Errorf("geolite2_asn not loaded")
	}

	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return nil, err
	}

	var record struct {
		AutonomousSystemNumber       uint32 `maxminddb:"autonomous_system_number"`
		AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization"`
	}
	if err := db.Lookup(addr).Decode(&record); err != nil {
		return nil, err
	}

	return &MMDBASNResult{
		ASN: fmt.Sprintf("AS%d", record.AutonomousSystemNumber),
		Org: record.AutonomousSystemOrganization,
	}, nil
}

func searchGeoCN(ip string) (*GeoCNResult, error) {
	mmdbMu.RLock()
	defer mmdbMu.RUnlock()
	db, ok := mmdbDBs["geocn"]
	if !ok {
		return nil, fmt.Errorf("geocn not loaded")
	}

	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return nil, err
	}

	var record struct {
		DivisionCode uint32 `maxminddb:"division_code"`
		ISP          string `maxminddb:"isp"`
	}
	if err := db.Lookup(addr).Decode(&record); err != nil {
		return nil, err
	}

	result := &GeoCNResult{
		DivisionCode: fmt.Sprintf("%d", record.DivisionCode),
		ISP:          record.ISP,
	}

	code := int(record.DivisionCode)
	if code > 0 {
		province, city, district := resolveDivision(code)
		result.Region = province
		result.City = city
		result.District = district
	}

	return result, nil
}

func searchBilibili(ip string) (*BilibiliResult, error) {
	if cached, ok := bilibiliCache.Load(ip); ok {
		entry := cached.(bilibiliCacheEntry)
		if time.Since(entry.timestamp) < 5*24*time.Hour {
			return entry.result, nil
		}
		bilibiliCache.Delete(ip)
	}

	client := resty.New()
	defer client.Close()
	resp, err := client.R().SetQueryParam("ip", ip).SetResult(&BilibiliIPQueryResponse{}).Get("https://api.live.bilibili.com/ip_service/v1/ip_service/get_ip_addr?ip=" + ip)
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode())
	}
	result := &BilibiliResult{
		Country:   resp.Result().(*BilibiliIPQueryResponse).Data.Country,
		Region:    resp.Result().(*BilibiliIPQueryResponse).Data.Province,
		City:      resp.Result().(*BilibiliIPQueryResponse).Data.City,
		ISP:       resp.Result().(*BilibiliIPQueryResponse).Data.ISP,
		Latitude:  resp.Result().(*BilibiliIPQueryResponse).Data.Latitude,
		Longitude: resp.Result().(*BilibiliIPQueryResponse).Data.Longitude,
	}

	bilibiliCache.Store(ip, bilibiliCacheEntry{result: result, timestamp: time.Now()})
	return result, nil
}

func loadAll() {
	if err := loadIp2Region(); err != nil {
		slog.Error("Error loading ip2region", "error", err)
	}
	if err := loadQQWry(); err != nil {
		slog.Error("Error loading qqwry.ipdb", "error", err)
	}
	if err := loadMMDB(); err != nil {
		slog.Error("Error loading mmdb", "error", err)
	}
	if err := loadDivisionData(); err != nil {
		slog.Error("Error loading division data", "error", err)
	}
}

func closeAllDBs() {
	ip2regionMu.Lock()
	if ip2region != nil {
		ip2region.Close()
		ip2region = nil
	}
	ip2regionMu.Unlock()

	qqwryMu.Lock()
	qqwryDB = nil
	qqwryMu.Unlock()

	mmdbMu.Lock()
	for name, db := range mmdbDBs {
		db.Close()
		slog.Info("Closed mmdb", "name", name)
	}
	mmdbDBs = nil
	mmdbMu.Unlock()
}

func reloadAll() {
	if err := loadIp2Region(); err != nil {
		slog.Error("Error loading ip2region", "error", err)
	}
	if err := reloadQQWry(); err != nil {
		slog.Error("Error reloading qqwry.ipdb", "error", err)
	}
	if err := reloadMMDB(); err != nil {
		slog.Error("Error reloading mmdb", "error", err)
	}
	if err := loadDivisionData(); err != nil {
		slog.Error("Error reloading division data", "error", err)
	}
}

func Init(ghproxy string) {
	localOK := true
	if err := loadIp2Region(); err != nil {
		slog.Warn("Local ip2region not available", "error", err)
		localOK = false
	}
	if err := loadQQWry(); err != nil {
		slog.Warn("Local qqwry.ipdb not available", "error", err)
		localOK = false
	}
	if err := loadMMDB(); err != nil {
		slog.Warn("Local mmdb not available", "error", err)
		localOK = false
	}
	if err := loadDivisionData(); err != nil {
		slog.Warn("Local division data not available", "error", err)
		localOK = false
	}

	if localOK {
		slog.Info("All databases loaded from local files")
	} else {
		slog.Info("Downloading databases...")
		closeAllDBs()
		PullDatabase(ghproxy)
		loadAll()
	}

	go func() {
		for {
			time.Sleep(time.Hour * 24)
			closeAllDBs()
			PullDatabase(ghproxy)
			reloadAll()
			slog.Info("Database sync completed, waiting for next sync...")
		}
	}()
}

var allDatabases = []string{"ip2region", "qqwry", "maxmind_city", "maxmind_asn", "geocn", "dbip_city", "bilibili"}

func SearchIP(ip string, databases ...string) map[string]interface{} {
	if len(databases) == 0 {
		databases = allDatabases
	}

	result := map[string]interface{}{"ip": ip}

	for _, name := range databases {
		switch name {
		case "ip2region":
			ip2regionMu.RLock()
			if ip2region != nil {
				region, err := ip2region.Search(ip)
				if err != nil {
					result["ip2region"] = "error: " + err.Error()
				} else {
					result["ip2region"] = region
				}
			} else {
				result["ip2region"] = "not loaded"
			}
			ip2regionMu.RUnlock()

		case "qqwry":
			qqwryMu.RLock()
			if qqwryDB != nil {
				info, err := qqwryDB.FindInfo(ip, "CN")
				if err != nil {
					result["qqwry"] = "error: " + err.Error()
				} else {
					result["qqwry"] = map[string]string{
						"country":             info.CountryName,
						"administrative_area": info.RegionName,
						"city":                info.CityName,
						"isp":                 info.IspDomain,
						"country_code":        info.CountryCode,
					}
				}
			} else {
				result["qqwry"] = "not loaded"
			}
			qqwryMu.RUnlock()

		case "maxmind_city":
			city, err := searchMMDBCity(ip)
			if err != nil {
				result["maxmind_city"] = "error: " + err.Error()
			} else {
				result["maxmind_city"] = city
			}

		case "maxmind_asn":
			asn, err := searchMMDBASN(ip)
			if err != nil {
				result["maxmind_asn"] = "error: " + err.Error()
			} else {
				result["maxmind_asn"] = asn
			}

		case "geocn":
			cn, err := searchGeoCN(ip)
			if err != nil {
				result["geocn"] = "error: " + err.Error()
			} else {
				result["geocn"] = cn
			}

		case "dbip_city":
			dbip, err := searchDBIPCity(ip)
			if err != nil {
				result["dbip_city"] = "error: " + err.Error()
			} else {
				result["dbip_city"] = dbip
			}

		case "bilibili":
			bilibili, err := searchBilibili(ip)
			if err != nil {
				result["bilibili"] = "error: " + err.Error()
			} else {
				result["bilibili"] = bilibili
			}

		default:
			result[name] = "unknown database"
		}
	}

	return result
}
