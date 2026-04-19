package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"lemon-ipw/ipdb"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/lionsoul2014/ip2region/binding/golang/service"
	"resty.dev/v3"
)

type Setting struct {
	Port    string `json:"port"`
	GHProxy string `json:"gh-proxy"`
}

var (
	ip2region   *service.Ip2Region
	ip2regionMu sync.RWMutex
	PORTS       string
	GH_PROXY    string
	LOG_LEVEL   string
)

type WebsiteCheckResult struct {
	IPv4 *WebsiteCheckDetail `json:"ipv4"` // IPv4 检测结果
	IPv6 *WebsiteCheckDetail `json:"ipv6"` // IPv6 检测结果
}

type WebsiteCheckDetail struct {
	HostRecord       string  `json:"host_record"`       // 主机记录
	HTTPStatusCode   int     `json:"http_status_code"`  // HTTP 访问返回码
	HTTPSSStatusCode int     `json:"https_status_code"` // HTTPS 访问返回码
	DNSLookupTime    float64 `json:"dns_lookup_time"`   // 域名解析耗时 (ms)
	TCPConnectTime   float64 `json:"tcp_connect_time"`  // TCP 连接耗时 (ms)
	HTTPConnectTime  float64 `json:"http_connect_time"` // HTTP 连接耗时 (ms)
	FirstByteTime    float64 `json:"first_byte_time"`   // 首字节传输耗时 (ms)
	TotalTime        float64 `json:"total_time"`        // 总耗时 (ms)
	PageSize         int64   `json:"page_size"`         // 网页大小 (bytes)
	DownloadSpeed    float64 `json:"download_speed"`    // 下载速度 (KB/s)
}

type SSLCheckDetail struct {
	CertValidityDays   int       `json:"cert_validity_days"` // 证书有效期 (天)
	CertStartTime      time.Time `json:"cert_start_time"`    // 证书开始时间
	CertEndTime        time.Time `json:"cert_end_time"`      // 证书结束时间
	HTTPVersion        string    `json:"http_version"`       // HTTP 版本
	HostRecord         string    `json:"host_record"`        // 主机记录
	HTTPSSStatusCode   int       `json:"https_status_code"`  // HTTPS 访问返回码
	TotalTime          float64   `json:"total_time"`         // 总耗时 (ms)
	DownloadSpeed      float64   `json:"download_speed"`     // 下载速度 (KB/s)
	Domain             string    `json:"domain"`             // 下载速度 (KB/s)
	IssuerOrganization []string  `json:"issuer_organization"`
	IssuerCommonName   string    `json:"issuer_common_name"`
	SubjectCommonName  string    `json:"subject_common_name"`
	IsExpired          bool      `json:"is_expired"`
}

type SSLCheckResult struct {
	IPv4 *SSLCheckDetail `json:"ipv4"` // IPv4 SSL 检测结果
	IPv6 *SSLCheckDetail `json:"ipv6"` // IPv6 SSL 检测结果
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

func syncDatabase() error {
	for {
		err, errv6 := ipdb.PullDatabase(GH_PROXY)
		if err != nil || errv6 != nil {
			slog.Error("Error pulling database", "error", err, "error_v6", errv6)
			if err != nil {
				return err
			}
			return errv6
		}

		if err := loadIp2Region(); err != nil {
			slog.Error("Error loading ip2region", "error", err)
		}

		slog.Info("Database sync completed, waiting for next sync...")
		time.Sleep(time.Hour * 24)
	}
}

func checkWebsite(url string, version string) (*WebsiteCheckDetail, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}

	var network string
	if version == "v6" {
		network = "tcp6"
	} else {
		network = "tcp4"
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, net, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, addr)
		},
	}

	client := resty.New().SetTransport(transport).SetTimeout(10 * time.Second)

	startTime := time.Now()
	resp, err := client.R().EnableTrace().Get(url)

	if err != nil {
		return nil, err
	}
	endTime := time.Now()

	body := resp.Bytes()
	trace := resp.Request.TraceInfo()

	// 清理主机记录中的端口和 IPv6 的 []
	hostRecord := cleanHostRecord(trace.RemoteAddr)

	totalTime := float64(endTime.Sub(startTime).Milliseconds())
	downloadSpeed := float64(len(body)) / 1024.0 / (totalTime / 1000.0)

	result := &WebsiteCheckDetail{
		HostRecord:       hostRecord,
		HTTPStatusCode:   resp.StatusCode(),
		HTTPSSStatusCode: resp.StatusCode(),
		DNSLookupTime:    float64(trace.DNSLookup.Milliseconds()),
		TCPConnectTime:   float64(trace.TCPConnTime.Milliseconds()),
		HTTPConnectTime:  float64(trace.ConnTime.Milliseconds()),
		FirstByteTime:    float64(trace.ServerTime.Milliseconds()),
		TotalTime:        totalTime,
		PageSize:         int64(len(body)),
		DownloadSpeed:    downloadSpeed,
	}

	return result, nil
}

func checkSSL(url string, version string) (*SSLCheckDetail, error) {
	parsedURL, err := parseURL(url)
	if err != nil {
		return nil, err
	}

	var network string
	if version == "v6" {
		network = "tcp6"
	} else {
		network = "tcp4"
	}

	host := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		port = "443"
	}

	addr := fmt.Sprintf("%s:%s", host, port)

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(context.Background(), network, addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	tlsConn := tls.Client(conn, &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true,
	})
	defer tlsConn.Close()

	if err := tlsConn.Handshake(); err != nil {
		// TLS 握手失败，返回证书无效
		return &SSLCheckDetail{
			CertValidityDays:   0,
			IsExpired:          true,
			CertStartTime:      time.Time{},
			CertEndTime:        time.Time{},
			HTTPVersion:        "",
			HostRecord:         addr,
			HTTPSSStatusCode:   0,
			TotalTime:          0,
			DownloadSpeed:      0,
			Domain:             host,
			IssuerOrganization: []string{},
			IssuerCommonName:   "TLS Handshake Failed",
			SubjectCommonName:  host,
		}, nil
	}

	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return nil, fmt.Errorf("no SSL certificate found")
	}

	cert := state.PeerCertificates[0]
	now := time.Now()
	remainingDays := int(cert.NotAfter.Sub(now).Hours() / 24)
	isExpired := now.After(cert.NotAfter) || now.Before(cert.NotBefore)

	// 使用 HTTP 客户端获取性能数据
	dialer2 := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, net, addr string) (net.Conn, error) {
			return dialer2.DialContext(ctx, network, addr)
		},
	}
	client := resty.New().SetTransport(transport).SetTimeout(10 * time.Second)

	startTime := time.Now()
	resp, err := client.R().EnableTrace().Get(url)
	if err != nil {
		return nil, err
	}
	endTime := time.Now()

	body := resp.Bytes()
	totalTime := float64(endTime.Sub(startTime).Milliseconds())
	downloadSpeed := float64(len(body)) / 1024.0 / (totalTime / 1000.0)

	hostRecord := resp.Request.TraceInfo().RemoteAddr
	hostRecord = cleanHostRecord(hostRecord)
	domain := cleanHostRecord(state.ServerName)

	result := &SSLCheckDetail{
		CertValidityDays:   remainingDays,
		IsExpired:          isExpired,
		CertStartTime:      cert.NotBefore,
		CertEndTime:        cert.NotAfter,
		HTTPVersion:        resp.Proto(),
		HostRecord:         hostRecord,
		HTTPSSStatusCode:   resp.StatusCode(),
		TotalTime:          totalTime,
		DownloadSpeed:      downloadSpeed,
		Domain:             domain,
		IssuerOrganization: cert.Issuer.Organization, // []string
		IssuerCommonName:   cert.Issuer.CommonName,   // string（备选）
		SubjectCommonName:  cert.Subject.CommonName,
	}

	return result, nil
}

func cleanHostRecord(addr string) string {
	// 处理带方括号的IPv6地址格式 [IPv6]:port
	if strings.HasPrefix(addr, "[") {
		// 找到右方括号的位置
		rightBracket := strings.Index(addr, "]")
		if rightBracket != -1 {
			// 提取IPv6地址部分（去掉方括号）
			ipv6Addr := addr[1:rightBracket]
			return ipv6Addr
		}
		// 如果没有找到右方括号，继续后续处理
	}

	// 处理不带方括号的情况（IPv4或纯IPv6地址）
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		// 检查是否是 IPv6 地址 (有多个:)
		colonCount := strings.Count(addr, ":")
		if colonCount > 1 {
			// 纯IPv6地址（无端口）或IPv6+端口，保留到最后一个:之前的部分 (去掉端口)
			return addr[:idx]
		}
		// IPv4 地址带端口，去掉端口
		if colonCount == 1 {
			return addr[:idx]
		}
	}

	// 如果没有端口，直接返回（可能已经是纯IP地址）
	return addr
}

func normalizeURL(input string) string {
	input = strings.TrimSpace(input)
	input = strings.TrimPrefix(input, "/")
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return input
	}
	if strings.HasPrefix(input, "//") {
		return "https:" + input
	}
	return "https://" + input
}

func parseURL(input string) (*url.URL, error) {
	input = normalizeURL(input)
	return url.Parse(input)
}

func testWebsite(c *gin.Context) {
	testUrl := c.Param("url")
	if testUrl == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URL parameter is required",
		})
		return
	}
	testUrl = normalizeURL(testUrl)
	v6 := testWebTools(testUrl, "v6")
	v4 := testWebTools(testUrl, "v4")
	c.JSON(200, gin.H{
		"v6": v6,
		"v4": v4,
	})
}

func testWebTools(url, versions string) string {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if versions == "v6" {
				return dialer.DialContext(ctx, "tcp6", addr)
			}
			return dialer.DialContext(ctx, "tcp4", addr)
		},
	}
	client := resty.New().SetTransport(transport)
	resp, err := client.R().EnableTrace().Get(url)
	if err != nil {
		return "Error: " + err.Error()
	}
	record := resp.Request.TraceInfo()
	return record.RemoteAddr
}

func checkWebsiteHandler(c *gin.Context) {
	testUrl := c.Param("url")
	if testUrl == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URL parameter is required",
		})
		return
	}

	testUrl = normalizeURL(testUrl)

	result := &WebsiteCheckResult{}

	// IPv6 检测
	ipv6, errV6 := checkWebsite(testUrl, "v6")
	if errV6 != nil {
		ipv6 = &WebsiteCheckDetail{
			HostRecord: "Error: " + errV6.Error(),
		}
	}

	// IPv4 检测
	ipv4, errV4 := checkWebsite(testUrl, "v4")
	if errV4 != nil {
		ipv4 = &WebsiteCheckDetail{
			HostRecord: "Error: " + errV4.Error(),
		}
	}

	result.IPv4 = ipv4
	result.IPv6 = ipv6

	c.JSON(200, result)
}

func sslCheckHandler(c *gin.Context) {
	testUrl := c.Param("url")
	if testUrl == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URL parameter is required",
		})
		return
	}

	testUrl = normalizeURL(testUrl)

	result := &SSLCheckResult{}

	// IPv6 检测
	ipv6, errV6 := checkSSL(testUrl, "v6")
	if errV6 != nil {
		ipv6 = &SSLCheckDetail{
			HostRecord: "Error: " + errV6.Error(),
		}
	}

	// IPv4 检测
	ipv4, errV4 := checkSSL(testUrl, "v4")
	if errV4 != nil {
		ipv4 = &SSLCheckDetail{
			HostRecord: "Error: " + errV4.Error(),
		}
	}

	result.IPv4 = ipv4
	result.IPv6 = ipv6

	c.JSON(200, result)
}
func locateIP(c *gin.Context) {
	ip := c.Param("ip")
	slog.Debug("Locating IP", "ip", ip)

	ip2regionMu.RLock()
	defer ip2regionMu.RUnlock()

	if ip2region == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "ip2region not initialized",
		})
		return
	}

	region, err := ip2region.Search(ip)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ip":     ip,
		"region": region,
	})
}
func readConfig() {
	// 1. 优先从环境变量读取
	PORTS = os.Getenv("PORTS")
	GH_PROXY = os.Getenv("GH_PROXY")

	// 2. 如果环境变量未设置，尝试从 setting.json 读取
	if PORTS == "" || GH_PROXY == "" {
		settingFile := "setting.json"
		if data, err := os.ReadFile(settingFile); err == nil {
			var setting Setting
			if err := json.Unmarshal(data, &setting); err == nil {
				// 环境变量未设置时才使用配置文件中的值
				if PORTS == "" && setting.Port != "" {
					PORTS = setting.Port
				}
				if GH_PROXY == "" && setting.GHProxy != "" {
					GH_PROXY = setting.GHProxy
				}
			}
		}
	}

	// 3. 最后使用默认值
	if PORTS == "" {
		PORTS = "8080"
	}
}
func main() {
	go syncDatabase()
	readConfig()

	r := gin.Default()
	r.Use(cors.Default())
	r.GET("/v1/detail/*url", checkWebsiteHandler)
	r.GET("/v1/ssl/*url", sslCheckHandler)
	r.GET("/v1/location/:ip", locateIP)

	if err := r.Run(":" + PORTS); err != nil {
		slog.Error("Server failed to start", "error", err)
	}

	ip2regionMu.Lock()
	defer ip2regionMu.Unlock()
	if ip2region != nil {
		ip2region.Close()
		slog.Info("ip2region closed successfully")
	}
}
