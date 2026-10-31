package ipdb

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ip2location/ip2location-go/v9"
	ipdbgo "github.com/ipipdotnet/ipdb-go"
	"github.com/lionsoul2014/ip2region/binding/golang/service"
	maxminddb "github.com/oschwald/maxminddb-golang/v2"
	"golang.org/x/sync/singleflight"
	"resty.dev/v3"
)

// loadDivisionCoords 读区划代码→政府驻地坐标表（GB/T 2260 六位 → [lng, lat]，
// GCJ-02）。数据来自 nomdn/division-coords 仓的 release（源：AreaCity 的
// ok_geo.csv，tag 跟随其数据版本），随节点下载任务落到工作目录，与 full.txt 同批重载。
func loadDivisionCoords(path string) (map[string][2]float64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := make(map[string][2]float64)
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

var (
	ip2region      *service.Ip2Region
	ip2regionMu    sync.RWMutex
	ip2locDB       *ip2location.DB
	ip2locMu       sync.RWMutex
	ip2locASNDB    *ip2location.DB
	ip2locASNMu    sync.RWMutex
	qqwryDB        *ipdbgo.City
	qqwryMu        sync.RWMutex
	mmdbDBs        map[string]*maxminddb.Reader
	mmdbMu         sync.RWMutex
	divisionFull   map[int]string
	divisionShort  map[int]string
	divisionCoords map[string][2]float64
	divisionMu     sync.RWMutex
	bilibiliCache  sync.Map
	// bilibiliGroup 同一 IP 的并发未命中合并为一次上游调用（SearchIP 之外
	// 定时任务/多 handler 并发触达同一 IP 时，避免重复打 B 站 API）
	bilibiliGroup singleflight.Group
)

// bilibiliClient 共享的 B 站归属地客户端：连接池跨查询复用。
// 原实现每次 searchBilibili 都 resty.New()，每个未缓存 IP 都付一次完整 TCP+TLS 握手
// （实测首查 ~440ms，其中 TLS 209ms；复用后热查询 ~80ms）。进程生命周期内不 Close。
var bilibiliClient = resty.New().SetTimeout(10 * time.Second)

type bilibiliCacheEntry struct {
	result    *BilibiliResult
	errMsg    string // 非空 = 负缓存（上游失败），TTL 远短于正缓存，避免对非法/不可达 IP 反复打上游
	timestamp time.Time
}

// ttl：正结果 24h（归属地几乎不变），负结果 1 分钟
func (e bilibiliCacheEntry) ttl() time.Duration {
	if e.errMsg != "" {
		return time.Minute
	}
	return 24 * time.Hour
}

func (e bilibiliCacheEntry) err() error {
	if e.errMsg == "" {
		return nil
	}
	return fmt.Errorf("%s", e.errMsg)
}

type MMDBCityResult struct {
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Region      string  `json:"administrative_area"`
	City        string  `json:"city"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

type MMDBASNResult struct {
	ASN string `json:"asn"`
	Org string `json:"org"`
}

type GeoCNResult struct {
	DivisionCode string `json:"division_code"`
	ISP          string `json:"isp"`
	Type         string `json:"type"`
	Region       string `json:"administrative_area,omitempty"`
	City         string `json:"city,omitempty"`
	District     string `json:"district,omitempty"`
	// 政府驻地坐标（GCJ-02，按 division_code 查内置对照表）；
	// 缺表/新设区划查不到时不输出坐标字段
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
}

type BilibiliIPQueryResponse struct {
	Code    int                 `json:"code"`
	Msg     string              `json:"msg"`
	Message string              `json:"message"`
	Data    BilibiliIPQueryData `json:"data"`
}

type BilibiliIPQueryData struct {
	Addr      string `json:"addr"`
	Country   string `json:"country"`
	Province  string `json:"province"`
	City      string `json:"city"`
	ISP       string `json:"isp"`
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
}

type BilibiliResult struct {
	Country   string `json:"country"`
	Region    string `json:"administrative_area"`
	City      string `json:"city"`
	ISP       string `json:"isp"`
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
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

	divisionMu.RLock()
	defer divisionMu.RUnlock()

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
	newFull, err := loadDivisionCode("full.txt", "\t")
	if err != nil {
		slog.Error("failed to load full.txt", "error", err)
		return err
	}
	newShort, err := loadDivisionCode("short.txt", "  ")
	if err != nil {
		slog.Error("failed to load short.txt", "error", err)
		return err
	}

	divisionMu.Lock()
	divisionFull = newFull
	divisionShort = newShort
	divisionMu.Unlock()

	// 坐标表是 GeoCN 打点的增强数据：缺文件/坏 JSON 只少坐标，不影响区划名解析，
	// 故不随 full/short 一起 return err
	newCoords, err := loadDivisionCoords("division-coords.json")
	if err != nil {
		slog.Warn("failed to load division-coords.json", "error", err)
	} else {
		divisionMu.Lock()
		divisionCoords = newCoords
		divisionMu.Unlock()
	}

	slog.Info("division data loaded", "full", len(newFull), "short", len(newShort), "coords", len(newCoords))
	return nil
}

// —— 加载器的统一模式：先在锁外构建新实例，再持锁原子换指针，旧实例放锁外 Close ——
// 原实现是「持锁 → 关旧 → 开新」，重载期间该库有几百 ms 到几秒的查询真空窗
// （查询拿到 "not loaded"）；换成换指针模式后窗口降到 μs 级。
// 榜样本来就是 reloadQQWry 的写法，这里推广到全部加载器。

func loadIp2Region() error {
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

	next, err := service.NewIp2Region(v4Config, v6Config)
	if err != nil {
		slog.Error("failed to create ip2region service", "error", err)
		return err
	}

	ip2regionMu.Lock()
	old := ip2region
	ip2region = next
	ip2regionMu.Unlock()
	if old != nil {
		old.Close()
		slog.Info("Previous ip2region closed")
	}

	slog.Info("ip2region loaded successfully")
	return nil
}

// loadIP2Location 加载 IP2Location LITE DB11 数据库（.BIN 格式，官方 ip2location-go 库读取）
func loadIP2Location() error {
	db, err := ip2location.OpenDB("./IP2LOCATION-LITE-DB11.IPV6.BIN")
	if err != nil {
		slog.Error("failed to open ip2location db", "error", err)
		return err
	}

	ip2locMu.Lock()
	old := ip2locDB
	ip2locDB = db
	ip2locMu.Unlock()
	if old != nil {
		old.Close()
		slog.Info("Previous ip2location closed")
	}

	slog.Info("ip2location loaded successfully")
	return nil
}

// loadIP2LocationASN 加载 IP2Location LITE ASN 数据库（仅 asn / as 字段）
func loadIP2LocationASN() error {
	db, err := ip2location.OpenDB("./IP2LOCATION-LITE-ASN.IPV6.BIN")
	if err != nil {
		slog.Error("failed to open ip2location asn db", "error", err)
		return err
	}

	ip2locASNMu.Lock()
	old := ip2locASNDB
	ip2locASNDB = db
	ip2locASNMu.Unlock()
	if old != nil {
		old.Close()
		slog.Info("Previous ip2location asn closed")
	}

	slog.Info("ip2location asn loaded successfully")
	return nil
}

func loadQQWry() error {
	// 先建新实例，再持锁替换（旧实例由 GC 回收，ipdb-go 底层 mmap 会自动释放）
	newDB, err := ipdbgo.NewCity("qqwry.ipdb")
	if err != nil {
		slog.Error("failed to load qqwry.ipdb", "error", err)
		return fmt.Errorf("load qqwry failed: %w", err)
	}

	qqwryMu.Lock()
	qqwryDB = newDB
	qqwryMu.Unlock()

	slog.Info("qqwry.ipdb loaded successfully")
	return nil
}

var mmdbFiles = map[string]string{
	"geolite2_city": "GeoLite2-City.mmdb",
	"geolite2_asn":  "GeoLite2-ASN.mmdb",
	"geocn":         "GeoCN.mmdb",
	"dbip_city":     "dbip-city-lite.mmdb",
	"dbip_asn":      "dbip-asn-lite.mmdb",
}

// loadMMDB 重建全部 mmdb 并原子换指针。个别文件加载失败时沿用旧库（数据旧一点
// 好过整库消失），只有首次加载（无旧库）才真正缺库。
func loadMMDB() error {
	next := make(map[string]*maxminddb.Reader, len(mmdbFiles))
	for name, path := range mmdbFiles {
		db, err := maxminddb.Open(path)
		if err != nil {
			slog.Error("failed to load mmdb", "name", name, "error", err)
			continue
		}
		next[name] = db
		slog.Info("mmdb loaded", "name", name)
	}

	mmdbMu.Lock()
	old := mmdbDBs
	merged := make(map[string]*maxminddb.Reader, len(mmdbFiles))
	for name := range mmdbFiles {
		if db, ok := next[name]; ok {
			merged[name] = db
		} else if db, ok := old[name]; ok {
			// 新文件损坏：沿用旧库，不跟着一起下线
			merged[name] = db
			delete(next, name)
		}
	}
	mmdbDBs = merged
	mmdbMu.Unlock()

	// 换指针后再关被替换的旧库（合并时沿用的旧库保留，不关）
	for name, db := range old {
		if _, keep := merged[name]; !keep {
			db.Close()
			slog.Info("Previous mmdb closed", "name", name)
		}
	}

	if len(merged) < len(mmdbFiles) {
		return fmt.Errorf("some mmdb files missing (%d/%d loaded)", len(merged), len(mmdbFiles))
	}
	return nil
}

// searchMMDBCityIn 在指定库上查城市（searchMMDBCity / searchDBIPCity 的公共实现）
func searchMMDBCityIn(db *maxminddb.Reader, ip string) (*MMDBCityResult, error) {
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
		Location struct {
			Latitude  float64 `maxminddb:"latitude"`
			Longitude float64 `maxminddb:"longitude"`
		} `maxminddb:"location"`
	}
	if err := db.Lookup(addr).Decode(&record); err != nil {
		return nil, err
	}

	result := &MMDBCityResult{
		CountryCode: record.Country.ISOCode,
		Latitude:    record.Location.Latitude,
		Longitude:   record.Location.Longitude,
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

// searchMMDBASNIn 在指定库上查 ASN（searchMMDBASN / searchDBIPASN 的公共实现）
func searchMMDBASNIn(db *maxminddb.Reader, ip string) (*MMDBASNResult, error) {
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

func searchMMDBCity(ip string) (*MMDBCityResult, error) {
	mmdbMu.RLock()
	db, ok := mmdbDBs["geolite2_city"]
	mmdbMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("geolite2_city not loaded")
	}
	return searchMMDBCityIn(db, ip)
}

func searchDBIPCity(ip string) (*MMDBCityResult, error) {
	mmdbMu.RLock()
	db, ok := mmdbDBs["dbip_city"]
	mmdbMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("dbip_city not loaded")
	}
	return searchMMDBCityIn(db, ip)
}

func searchMMDBASN(ip string) (*MMDBASNResult, error) {
	mmdbMu.RLock()
	db, ok := mmdbDBs["geolite2_asn"]
	mmdbMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("geolite2_asn not loaded")
	}
	return searchMMDBASNIn(db, ip)
}

func searchDBIPASN(ip string) (*MMDBASNResult, error) {
	mmdbMu.RLock()
	db, ok := mmdbDBs["dbip_asn"]
	mmdbMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("dbip_asn not loaded")
	}
	return searchMMDBASNIn(db, ip)
}

func searchGeoCN(ip string) (*GeoCNResult, error) {
	mmdbMu.RLock()
	db, ok := mmdbDBs["geocn"]
	mmdbMu.RUnlock()
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
		Type         string `maxminddb:"type"`
	}
	if err := db.Lookup(addr).Decode(&record); err != nil {
		return nil, err
	}

	result := &GeoCNResult{
		DivisionCode: fmt.Sprintf("%d", record.DivisionCode),
		ISP:          record.ISP,
		Type:         record.Type,
	}

	code := int(record.DivisionCode)
	if code > 0 {
		province, city, district := resolveDivision(code)
		result.Region = province
		result.City = city
		result.District = district
		// 政府驻地打点：按区划代码查坐标表（GCJ-02），缺表/新设区划查不到时省略坐标
		divisionMu.RLock()
		c, ok := divisionCoords[strconv.Itoa(code)]
		divisionMu.RUnlock()
		if ok {
			result.Longitude = c[0]
			result.Latitude = c[1]
		}
	}

	return result, nil
}

// loadBilibiliEntry 查缓存（按正/负各自的 TTL 判新鲜），过期顺手删除
func loadBilibiliEntry(ip string) (bilibiliCacheEntry, bool) {
	if cached, ok := bilibiliCache.Load(ip); ok {
		e := cached.(bilibiliCacheEntry)
		if time.Since(e.timestamp) < e.ttl() {
			return e, true
		}
		bilibiliCache.Delete(ip)
	}
	return bilibiliCacheEntry{}, false
}

func fetchBilibili(ip string) bilibiliCacheEntry {
	resp, err := bilibiliClient.R().
		SetQueryParam("ip", ip).
		SetResult(&BilibiliIPQueryResponse{}).
		Get("https://api.live.bilibili.com/ip_service/v1/ip_service/get_ip_addr")
	if err != nil {
		return bilibiliCacheEntry{errMsg: err.Error(), timestamp: time.Now()}
	}
	if resp.IsError() {
		return bilibiliCacheEntry{errMsg: fmt.Sprintf("HTTP %d", resp.StatusCode()), timestamp: time.Now()}
	}
	r := resp.Result().(*BilibiliIPQueryResponse)
	return bilibiliCacheEntry{
		result: &BilibiliResult{
			Country:   r.Data.Country,
			Region:    r.Data.Province,
			City:      r.Data.City,
			ISP:       r.Data.ISP,
			Latitude:  r.Data.Latitude,
			Longitude: r.Data.Longitude,
		},
		timestamp: time.Now(),
	}
}

func searchBilibili(ip string) (*BilibiliResult, error) {
	if net.ParseIP(ip) == nil {
		// 非法 IP 是纯内存判断，不进缓存也不打上游
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}

	if e, ok := loadBilibiliEntry(ip); ok {
		return e.result, e.err()
	}

	v, _, _ := bilibiliGroup.Do(ip, func() (any, error) {
		// singleflight 等待期间可能已被同 IP 的前一个请求填进缓存
		if e, ok := loadBilibiliEntry(ip); ok {
			return e, nil
		}
		e := fetchBilibili(ip)
		bilibiliCache.Store(ip, e)
		return e, nil
	})
	e := v.(bilibiliCacheEntry)
	return e.result, e.err()
}

// SweepBilibiliCache 清理过期的 Bilibili 归属地缓存（由外部定时调用，
// 防止公网端点被唯一 key 洪打时缓存内存无限增长）。
// 负缓存的 TTL 只有 1 分钟，但条目若按 24h 清扫会滞留内存——按各自 TTL 清。
func SweepBilibiliCache() {
	now := time.Now()
	bilibiliCache.Range(func(k, v any) bool {
		if e, ok := v.(bilibiliCacheEntry); ok && now.Sub(e.timestamp) >= e.ttl() {
			bilibiliCache.Delete(k)
		}
		return true
	})
}

// reloadAll 全量重载（各加载器自身即「先建新再换指针」，重载期间查询不中断）
func reloadAll() {
	if err := loadIp2Region(); err != nil {
		slog.Error("Error loading ip2region", "error", err)
	}
	if err := loadIP2Location(); err != nil {
		slog.Error("Error loading ip2location", "error", err)
	}
	if err := loadIP2LocationASN(); err != nil {
		slog.Error("Error loading ip2location asn", "error", err)
	}
	if err := loadQQWry(); err != nil {
		slog.Error("Error loading qqwry.ipdb", "error", err)
	}
	if err := loadMMDB(); err != nil {
		slog.Error("Error loading mmdb", "error", err)
	}
	if err := loadDivisionData(); err != nil {
		slog.Error("Error reloading division data", "error", err)
	}
}

func Init(ghproxy string) {
	// 先用本地文件全量加载；有缺失不再同步阻塞下载（原实现会把节点启动拖住几分钟，
	// 下载 450MB 全程不对外服务）——SearchIP 的结果结构天然支持 "not loaded" 降级，
	// 立即起服务，缺的库后台补齐后换指针上线。
	localOK := true
	if err := loadIp2Region(); err != nil {
		slog.Warn("Local ip2region not available", "error", err)
		localOK = false
	}
	if err := loadIP2Location(); err != nil {
		slog.Warn("Local ip2location not available", "error", err)
		localOK = false
	}
	if err := loadIP2LocationASN(); err != nil {
		slog.Warn("Local ip2location asn not available", "error", err)
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
		slog.Info("Some databases missing, downloading in background (serving degraded results meanwhile)")
		go func() {
			if err := PullDatabase(ghproxy); err != nil {
				slog.Error("Background database download failed", "error", err)
			}
			reloadAll()
			slog.Info("Background database initialization completed")
		}()
	}

	// 每日同步：PullDatabase 内部带条件请求（ETag/Last-Modified），文件没变的
	// 节点只花 12 个小请求，不再全量重拉 450MB。
	go func() {
		for {
			jitter := time.Duration(rand.Int63n(int64(time.Hour)))
			sleepDuration := 24*time.Hour + jitter
			time.Sleep(sleepDuration)
			slog.Info("Starting database sync...")
			if err := PullDatabase(ghproxy); err != nil {
				slog.Error("Database sync failed", "error", err)
				continue
			}
			reloadAll()
			slog.Info("Database sync completed, waiting for next sync...")
		}
	}()
}

var allDatabases = []string{"ip2region", "ip2location", "ip2location_asn", "qqwry", "maxmind_city", "maxmind_asn", "dbip_asn", "geocn", "dbip_city", "bilibili"}

func SearchIP(ip string, databases ...string) map[string]interface{} {
	if len(databases) == 0 {
		databases = allDatabases
	}

	result := map[string]interface{}{"ip": ip}

	// bilibili 是 10 个源里唯一的远程调用（实测 80~440ms，超时 10s），其余 9 个本地
	// 库合计 <5ms：把它并行出去，不让整个接口陪它等；本地库保持串行
	// （微秒级操作，并行反而净增调度开销）。结果写进独立变量，Wait 后合并——
	// Go 的 map 并发写（哪怕 key 不同）是不安全的。
	var wg sync.WaitGroup
	var bili any

	for _, name := range databases {
		switch name {
		case "bilibili":
			wg.Add(1)
			go func() {
				defer wg.Done()
				res, err := searchBilibili(ip)
				if err != nil {
					bili = "error: " + err.Error()
				} else {
					bili = res
				}
			}()

		case "ip2region":
			ip2regionMu.RLock()
			if ip2region != nil {
				region, err := ip2region.Search(ip)
				if err != nil {
					result["ip2region"] = "error: " + err.Error()
				} else {
					// 预分割为结构化 JSON，前端无需再按 "|" 拆字符串
					result["ip2region"] = parseIP2Region(region)
				}
			} else {
				result["ip2region"] = "not loaded"
			}
			ip2regionMu.RUnlock()

		case "ip2location":
			ip2locMu.RLock()
			if ip2locDB != nil {
				rec, err := ip2locDB.Get_all(ip)
				if err != nil {
					result["ip2location"] = "error: " + err.Error()
				} else {
					// LITE 免费版不提供 asn/isp 等字段，库会返回占位提示串，统一过滤为空
					clean := func(s string) string {
						if s == "" || strings.HasPrefix(s, "This parameter is unavailable") {
							return ""
						}
						return s
					}
					loc := map[string]string{
						"country":             clean(rec.Country_long),
						"country_code":        clean(rec.Country_short),
						"administrative_area": clean(rec.Region),
						"city":                clean(rec.City),
						"isp":                 clean(rec.Isp),
						"asn":                 clean(rec.Asn),
						"district":            clean(rec.District),
						"zipcode":             clean(rec.Zipcode),
						"timezone":            clean(rec.Timezone),
						"usagetype":           clean(rec.Usagetype),
					}
					if rec.Latitude != 0 || rec.Longitude != 0 {
						loc["latitude"] = strconv.FormatFloat(float64(rec.Latitude), 'f', -1, 32)
						loc["longitude"] = strconv.FormatFloat(float64(rec.Longitude), 'f', -1, 32)
					}
					result["ip2location"] = loc
				}
			} else {
				result["ip2location"] = "not loaded"
			}
			ip2locMu.RUnlock()

		case "ip2location_asn":
			ip2locASNMu.RLock()
			if ip2locASNDB != nil {
				rec, err := ip2locASNDB.Get_all(ip)
				if err != nil {
					result["ip2location_asn"] = "error: " + err.Error()
				} else {
					clean := func(s string) string {
						if s == "" || strings.HasPrefix(s, "This parameter is unavailable") {
							return ""
						}
						return s
					}
					result["ip2location_asn"] = map[string]string{
						"asn": clean(rec.Asn),
						"as":  clean(rec.As),
					}
				}
			} else {
				result["ip2location_asn"] = "not loaded"
			}
			ip2locASNMu.RUnlock()

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

		case "dbip_asn":
			asn, err := searchDBIPASN(ip)
			if err != nil {
				result["dbip_asn"] = "error: " + err.Error()
			} else {
				result["dbip_asn"] = asn
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

		default:
			result[name] = "unknown database"
		}
	}

	wg.Wait()
	if bili != nil {
		result["bilibili"] = bili
	}

	return result
}

// parseIP2Region 将 ip2region xdb 的管道分隔串预分割为结构化 JSON。
// 实测 v4/v6 xdb 格式：国家|省份|城市|ISP(或"0")|国家代码（5 段，v4 的 ISP 段多为 "0"）；
// 兼容老 4 段格式：国家|省份|城市|ISP。
// 输出字段与 bilibili / geocn / maxmind_city 对齐：country / administrative_area / city / isp。
func parseIP2Region(s string) map[string]string {
	parts := strings.Split(s, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	get := func(i int) string {
		if i < len(parts) {
			return parts[i]
		}
		return ""
	}
	res := map[string]string{"country": get(0)}
	// 5 段：国家|省份|城市|ISP(可能为"0")|国家代码；4 段及以下：国家|省份|城市|ISP
	res["administrative_area"] = get(1)
	res["city"] = get(2)
	if isp := get(3); isp != "" && isp != "0" {
		res["isp"] = isp
	}
	return res
}
