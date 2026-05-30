package main

import (
	"context"
	"crypto/tls"
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
	"github.com/spf13/viper"
	"resty.dev/v3"
)

type Setting struct {
	Port    any    `json:"port"`
	GHProxy string `json:"gh-proxy"`
}

func (s *Setting) PortString() string {
	switch v := s.Port.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	default:
		return ""
	}
}

var (
	PORTS        string
	GH_PROXY     string
	LOG_LEVEL    string
	websiteCache sync.Map
	sslCache     sync.Map
)

type websiteCacheEntry struct {
	result    *WebsiteCheckResult
	timestamp time.Time
}

type sslCacheEntry struct {
	result    *SSLCheckResult
	timestamp time.Time
}

type WebsiteCheckResult struct {
	IPv4 *WebsiteCheckDetail `json:"ipv4"`
	IPv6 *WebsiteCheckDetail `json:"ipv6"`
}

type WebsiteCheckDetail struct {
	HostRecord       string  `json:"host_record"`
	HTTPStatusCode   int     `json:"http_status_code"`
	HTTPSSStatusCode int     `json:"https_status_code"`
	DNSLookupTime    float64 `json:"dns_lookup_time"`
	TCPConnectTime   float64 `json:"tcp_connect_time"`
	HTTPConnectTime  float64 `json:"http_connect_time"`
	FirstByteTime    float64 `json:"first_byte_time"`
	TotalTime        float64 `json:"total_time"`
	PageSize         int64   `json:"page_size"`
	DownloadSpeed    float64 `json:"download_speed"`
	IsReachable      bool    `json:"is_reachable"`
}

type SSLCheckDetail struct {
	CertValidityDays   int       `json:"cert_validity_days"`
	CertStartTime      time.Time `json:"cert_start_time"`
	CertEndTime        time.Time `json:"cert_end_time"`
	HTTPVersion        string    `json:"http_version"`
	HostRecord         string    `json:"host_record"`
	HTTPSSStatusCode   int       `json:"https_status_code"`
	TotalTime          float64   `json:"total_time"`
	DownloadSpeed      float64   `json:"download_speed"`
	Domain             string    `json:"domain"`
	IssuerOrganization []string  `json:"issuer_organization"`
	IssuerCommonName   string    `json:"issuer_common_name"`
	SubjectCommonName  string    `json:"subject_common_name"`
	IsExpired          bool      `json:"is_expired"`
	IsReachable        bool      `json:"is_reachable"`
}

type SSLCheckResult struct {
	IPv4 *SSLCheckDetail `json:"ipv4"`
	IPv6 *SSLCheckDetail `json:"ipv6"`
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
		IsReachable:      true,
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
			IsReachable:        true,
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
		IssuerOrganization: cert.Issuer.Organization,
		IssuerCommonName:   cert.Issuer.CommonName,
		SubjectCommonName:  cert.Subject.CommonName,
		IsReachable:        true,
	}

	return result, nil
}

func cleanHostRecord(addr string) string {
	if strings.HasPrefix(addr, "[") {
		rightBracket := strings.Index(addr, "]")
		if rightBracket != -1 {
			return addr[1:rightBracket]
		}
	}

	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		colonCount := strings.Count(addr, ":")
		if colonCount > 1 {
			return addr[:idx]
		}
		if colonCount == 1 {
			return addr[:idx]
		}
	}

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

	if cached, ok := websiteCache.Load(testUrl); ok {
		entry := cached.(websiteCacheEntry)
		if time.Since(entry.timestamp) < 5*time.Minute {
			c.JSON(200, entry.result)
			return
		}
		websiteCache.Delete(testUrl)
	}

	result := &WebsiteCheckResult{}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		ipv6, errV6 := checkWebsite(testUrl, "v6")
		if errV6 != nil {
			ipv6 = &WebsiteCheckDetail{
				HostRecord:  "Error: " + errV6.Error(),
				IsReachable: false,
			}
		}
		result.IPv6 = ipv6
	}()

	go func() {
		defer wg.Done()
		ipv4, errV4 := checkWebsite(testUrl, "v4")
		if errV4 != nil {
			ipv4 = &WebsiteCheckDetail{
				HostRecord:  "Error: " + errV4.Error(),
				IsReachable: false,
			}
		}
		result.IPv4 = ipv4
	}()

	wg.Wait()

	websiteCache.Store(testUrl, websiteCacheEntry{result: result, timestamp: time.Now()})
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

	if cached, ok := sslCache.Load(testUrl); ok {
		entry := cached.(sslCacheEntry)
		if time.Since(entry.timestamp) < 5*time.Minute {
			c.JSON(200, entry.result)
			return
		}
		sslCache.Delete(testUrl)
	}

	result := &SSLCheckResult{}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		ipv6, errV6 := checkSSL(testUrl, "v6")
		if errV6 != nil {
			ipv6 = &SSLCheckDetail{
				HostRecord: "Error: " + errV6.Error(),
				IsExpired:  true,
			}
		}
		result.IPv6 = ipv6
	}()

	go func() {
		defer wg.Done()
		ipv4, errV4 := checkSSL(testUrl, "v4")
		if errV4 != nil {
			ipv4 = &SSLCheckDetail{
				HostRecord: "Error: " + errV4.Error(),
				IsExpired:  true,
			}
		}
		result.IPv4 = ipv4
	}()

	wg.Wait()

	sslCache.Store(testUrl, sslCacheEntry{result: result, timestamp: time.Now()})
	c.JSON(200, result)
}

func locateIP(c *gin.Context) {
	ip := c.Param("ip")
	slog.Debug("Locating IP", "ip", ip)
	c.JSON(http.StatusOK, ipdb.SearchIP(ip))
}

func readConfig() {
	PORTS = os.Getenv("PORTS")
	GH_PROXY = os.Getenv("GH_PROXY")

	if PORTS == "" || GH_PROXY == "" {
		viper.SetConfigName("setting")
		viper.SetConfigType("json")
		viper.AddConfigPath(".")
		if err := viper.ReadInConfig(); err != nil {
			slog.Warn("Failed to read config file, using defaults", "error", err)
		}
		PORTS = viper.GetString("port")
		GH_PROXY = viper.GetString("gh-proxy")

	}

	if PORTS == "" {
		PORTS = "8080"
	}
}

func main() {
	readConfig()
	slog.Info("Starting server", "port", PORTS, "gh_proxy", GH_PROXY)
	ipdb.Init(GH_PROXY)

	r := gin.Default()
	r.Use(cors.Default())

	r.GET("/v1/detail/*url", checkWebsiteHandler)
	r.GET("/v1/ssl/*url", sslCheckHandler)
	r.GET("/v1/location/:ip", locateIP)

	if err := r.Run(":" + PORTS); err != nil {
		slog.Error("Server failed to start", "error", err)
	}
}
