package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"lemon-ipw/ipdb"
	"lemon-ipw/ssrf"
	"lemon-ipw/webtest"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/klauspost/compress/zstd"
	"github.com/spf13/viper"
	"golang.org/x/sync/singleflight"
	"resty.dev/v3"
)

func initHTTPClients() {
	setTransport := func(network string) *http.Transport {
		dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
		return &http.Transport{
			// DialContext performs SSRF validation before establishing connections.
			// 1. Reuse validated IPs from context (cached by ValidateOutboundTarget).
			// 2. Resolve hostname via DNS if no cached IPs exist.
			// 3. Block connections to private/internal IPs.
			// 4. Pin the connection to the first resolved IP to prevent DNS rebinding.
			DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
				if ssrf.Enabled() {
					host, port, err := net.SplitHostPort(addr)
					if err != nil {
						return nil, err
					}
					var ips []net.IP
					if v, ok := ctx.Value(ssrf.ValidatedIPsKey()).([]net.IP); ok {
						ips = v
					} else {
						ips, err = net.LookupIP(host)
						if err != nil {
							return nil, err
						}
					}
					for _, ip := range ips {
						if ssrf.IsPrivateIP(ip) {
							slog.Warn("Blocked connection to private IP", "host", host, "ip", ip)
							return nil, fmt.Errorf("request to private/internal address is not allowed")
						}
					}
					if len(ips) > 0 {
						addr = net.JoinHostPort(ips[0].String(), port)
					}
				}
				return dialer.DialContext(ctx, network, addr)
			},
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	V6Client = resty.New()
	V4Client = resty.New()
	V6Client.SetTransport(setTransport("tcp6"))
	V4Client.SetTransport(setTransport("tcp4"))
	V6Client.SetTimeout(10 * time.Second)
	V4Client.SetTimeout(10 * time.Second)
	V6Client.SetRedirectPolicy(resty.RedirectPolicyFunc(ssrf.SecureCheckRedirect))
	V4Client.SetRedirectPolicy(resty.RedirectPolicyFunc(ssrf.SecureCheckRedirect))
	V6Client.AddContentDecompresser("zstd", decompressZstd)
	V4Client.AddContentDecompresser("zstd", decompressZstd)

}

func fakePerfectWebsiteResult(host string) *WebsiteCheckDetail {
	cleanHost := strings.TrimPrefix(host, "https://")
	cleanHost = strings.TrimPrefix(cleanHost, "http://")
	return &WebsiteCheckDetail{
		HostRecord:       cleanHost,
		HTTPStatusCode:   200,
		HTTPSSStatusCode: 200,
		DNSLookupTime:    0.5,
		TCPConnectTime:   1.0,
		HTTPConnectTime:  1.5,
		FirstByteTime:    2.0,
		TotalTime:        100,
		PageSize:         52428,
		DownloadSpeed:    512.0,
		IsReachable:      true,
	}
}

func fakeInvalidSSLResult(host string) *SSLCheckDetail {
	return &SSLCheckDetail{
		CertValidityDays:   0,
		IsExpired:          true,
		CertStartTime:      time.Time{},
		CertEndTime:        time.Time{},
		HTTPVersion:        "",
		HostRecord:         host,
		HTTPSSStatusCode:   0,
		TotalTime:          0,
		DownloadSpeed:      0,
		Domain:             host,
		IssuerOrganization: []string{},
		IssuerCommonName:   "Invalid Certificate",
		SubjectCommonName:  host,
		IsReachable:        false,
	}
}

// Create Zstandard decompress logic
// 创建 Zstandard 解压缩逻辑
var zstdReaderPool = sync.Pool{
	New: func() interface{} {
		// 当池子空了，创建一个新的解码器
		decoder, _ := zstd.NewReader(nil)
		return decoder
	},
}

func decompressZstd(r io.ReadCloser) (io.ReadCloser, error) {
	zr := zstdReaderPool.Get().(*zstd.Decoder)

	err := zr.Reset(r)
	if err != nil {
		zstdReaderPool.Put(zr)
		zr, _ = zstd.NewReader(r)
	}
	defer zstdReaderPool.Put(zr)
	z := &zstdReader{s: r, r: zr}
	return z, nil
}

type zstdReader struct {
	s io.ReadCloser
	r *zstd.Decoder
}

func (b *zstdReader) Read(p []byte) (n int, err error) {
	return b.r.Read(p)
}

func (b *zstdReader) Close() error {
	b.r.Close()
	return b.s.Close()
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

// normalizeURL normalizes the input URL by ensuring it has a scheme (http or https).
// normalizeURL 通过确保输入 URL 具有方案（http 或 https）来规范化输入 URL。
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

// parseURL parses the input string into a URL object after normalizing it.
// parseURL 在规范化输入字符串后，将其解析为 URL 对象。

func parseURL(input string) (*url.URL, error) {
	input = normalizeURL(input)
	return url.Parse(input)
}

// Setting struct represents the configuration settings for the application, including port, GitHub proxy, and single-stack mode.
// Setting 结构体表示应用程序的配置设置，包括端口、GitHub 代理和单栈模式。
type Setting struct {
	Port         any    `json:"port"`
	GHProxy      string `json:"gh-proxy"`
	SINGLE_STACK string `json:"single-stack"`
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

// Global variables and structs
// 全局变量与结构体
var (
	PORTS        string
	GH_PROXY     string
	LOG_LEVEL    string
	websiteCache sync.Map
	SINGLE_STACK string
	DNS_SERVER   string
	sslCache     sync.Map
	pingCache    sync.Map
	speedCache   sync.Map
	sfGroup      singleflight.Group
	V6Client     *resty.Client
	V4Client     *resty.Client
)

type websiteCacheEntry struct {
	result    *WebsiteCheckResult
	timestamp time.Time
}

type sslCacheEntry struct {
	result    *SSLCheckResult
	timestamp time.Time
}

type pingCacheEntry struct {
	result    *TCPingResult
	timestamp time.Time
}

type speedCacheEntry struct {
	result    *WebsiteSpeedTestResult
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
type TCPingResult struct {
	IPv4 *webtest.TCPingStats `json:"ipv4"`
	IPv6 *webtest.TCPingStats `json:"ipv6"`
}
type WebsiteSpeedTestResult struct {
	Version          string  `json:"version"`
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
	Message          string  `json:"message"`
	Headers          string  `json:"headers"`
	IsReachable      bool    `json:"is_reachable"`
}

// Business Endpoints
// 业务端点

func checkWebsite(url string, version string) (*WebsiteCheckDetail, error) {
	ctx := context.Background()
	var err error
	ctx, err = ssrf.ValidateOutboundTarget(ctx, url)
	if err != nil {
		return nil, err
	}

	client := V4Client
	if version == "v6" {
		client = V6Client
	}

	startTime := time.Now()
	resp, err := client.R().EnableTrace().SetContext(ctx).Get(url)

	// HTTPS 请求失败时 fallback 到 HTTP
	fallbackToHTTP := false
	if err != nil && strings.HasPrefix(url, "https://") {
		httpURL := strings.Replace(url, "https://", "http://", 1)
		startTime = time.Now()
		resp, err = client.R().EnableTrace().SetContext(ctx).Get(httpURL)
		fallbackToHTTP = true
	}

	if err != nil {
		return nil, err
	}
	endTime := time.Now()

	body := resp.Bytes()
	trace := resp.Request.TraceInfo()

	hostRecord := cleanHostRecord(trace.RemoteAddr)

	totalTime := float64(endTime.Sub(startTime).Milliseconds())
	var downloadSpeed float64
	if totalTime > 0 {
		downloadSpeed = float64(len(body)) / 1024.0 / (totalTime / 1000.0)
	}

	httpStatus := resp.StatusCode()
	httpsStatus := resp.StatusCode()
	if fallbackToHTTP {
		httpsStatus = 0
	}

	result := &WebsiteCheckDetail{
		HostRecord:       hostRecord,
		HTTPStatusCode:   httpStatus,
		HTTPSSStatusCode: httpsStatus,
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

func websiteSpeed(url string, version string) (*WebsiteSpeedTestResult, error) {
	ctx := context.Background()
	var err error
	ctx, err = ssrf.ValidateOutboundTarget(ctx, url)
	if err != nil {
		return nil, err
	}

	client := V4Client
	if version == "v6" {
		client = V6Client
	}

	startTime := time.Now()
	resp, err := client.R().EnableTrace().SetContext(ctx).Get(url)

	fallbackToHTTP := false
	if err != nil && strings.HasPrefix(url, "https://") {
		httpURL := strings.Replace(url, "https://", "http://", 1)
		startTime = time.Now()
		resp, err = client.R().EnableTrace().SetContext(ctx).Get(httpURL)
		fallbackToHTTP = true
	}

	if err != nil {
		return nil, err
	}
	endTime := time.Now()

	body := resp.Bytes()
	trace := resp.Request.TraceInfo()

	hostRecord := cleanHostRecord(trace.RemoteAddr)

	totalTime := float64(endTime.Sub(startTime).Milliseconds())
	var downloadSpeed float64
	if totalTime > 0 {
		downloadSpeed = float64(len(body)) / 1024.0 / (totalTime / 1000.0)
	}
	dumpBytes, _ := httputil.DumpResponse(resp.RawResponse, false)
	httpStatus := resp.StatusCode()
	httpsStatus := resp.StatusCode()
	if fallbackToHTTP {
		httpsStatus = 0
	}
	result := &WebsiteSpeedTestResult{
		Version:          version,
		Headers:          string(dumpBytes),
		HostRecord:       hostRecord,
		HTTPStatusCode:   httpStatus,
		HTTPSSStatusCode: httpsStatus,
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

	ctx := context.Background()
	ctx, err = ssrf.ValidateOutboundTarget(ctx, url)
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

	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	if ssrf.Enabled() {
		var ipStr string
		if v, ok := ctx.Value(ssrf.ValidatedIPsKey()).([]net.IP); ok && len(v) > 0 {
			ipStr = v[0].String()
		} else {
			ips, err := net.LookupIP(host)
			if err != nil {
				return nil, err
			}
			ipStr = ips[0].String()
		}
		parsedIP := net.ParseIP(ipStr)
		if parsedIP != nil && ssrf.IsPrivateIP(parsedIP) {
			slog.Warn("Blocked SSL connection to private IP", "host", host, "ip", ipStr)
			return nil, fmt.Errorf("request to private/internal address is not allowed")
		}
		addr = net.JoinHostPort(ipStr, port)
	}
	conn, err := dialer.DialContext(ctx, network, addr)
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
			IsReachable:        false,
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

	client := V4Client
	if version == "v6" {
		client = V6Client
	}

	startTime := time.Now()
	resp, err := client.R().EnableTrace().Get(url)
	if err != nil {
		return nil, err
	}
	endTime := time.Now()

	body := resp.Bytes()
	totalTime := float64(endTime.Sub(startTime).Milliseconds())
	var downloadSpeed float64
	if totalTime > 0 {
		downloadSpeed = float64(len(body)) / 1024.0 / (totalTime / 1000.0)
	}

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

func checkWebsiteHandler(c *gin.Context) {
	testUrl := c.Param("url")
	if testUrl == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URL parameter is required",
		})
		return
	}

	testUrl = normalizeURL(testUrl)

	parsedURL, _ := url.Parse(testUrl)
	if ssrf.HasLocalOrPrivateIP(parsedURL.Hostname()) {
		c.JSON(200, &WebsiteCheckResult{
			IPv4: fakePerfectWebsiteResult(testUrl),
			IPv6: fakePerfectWebsiteResult(testUrl),
		})
		return
	}

	if cached, ok := websiteCache.Load(testUrl); ok {
		entry := cached.(websiteCacheEntry)
		if time.Since(entry.timestamp) < 5*time.Minute {
			c.JSON(200, entry.result)
			return
		}
		websiteCache.Delete(testUrl)
	}

	rawResult, _, _ := sfGroup.Do(testUrl, func() (interface{}, error) {
		result := &WebsiteCheckResult{}
		switch SINGLE_STACK {
		case "ipv4":
			ipv4, errV4 := checkWebsite(testUrl, "v4")
			if errV4 != nil {
				ipv4 = &WebsiteCheckDetail{
					HostRecord:  "Error: " + errV4.Error(),
					IsReachable: false,
				}
			}
			result.IPv4 = ipv4
			result.IPv6 = &WebsiteCheckDetail{
				HostRecord:  "Skipped due to SINGLE_STACK=ipv4",
				IsReachable: false,
			}
		case "ipv6":
			ipv6, errV6 := checkWebsite(testUrl, "v6")
			if errV6 != nil {
				ipv6 = &WebsiteCheckDetail{
					HostRecord:  "Error: " + errV6.Error(),
					IsReachable: false,
				}
			}
			result.IPv6 = ipv6
			result.IPv4 = &WebsiteCheckDetail{
				HostRecord:  "Skipped due to SINGLE_STACK=ipv6",
				IsReachable: false,
			}
		default:
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
		}

		websiteCache.Store(testUrl, websiteCacheEntry{result: result, timestamp: time.Now()})

		if (result.IPv4 != nil && !result.IPv4.IsReachable) || (result.IPv6 != nil && !result.IPv6.IsReachable) {
			go func() {
				time.Sleep(30 * time.Second)
				websiteCache.Delete(testUrl)
			}()
		}

		return result, nil
	})

	c.JSON(200, rawResult.(*WebsiteCheckResult))
}
func websiteSpeedTestHandler(c *gin.Context) {
	testUrl := c.Param("url")
	version := c.Param("version")
	if testUrl == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URL parameter is required",
		})
		return
	}
	url := normalizeURL(testUrl)

	// 检查请求版本是否与 SINGLE_STACK 配置匹配
	switch SINGLE_STACK {
	case "ipv4":
		if version != "v4" {
			c.JSON(http.StatusBadRequest, &WebsiteSpeedTestResult{
				Version:    "v4",
				HostRecord: "Skipped due to SINGLE_STACK=ipv4",
			})
			return
		}
	case "ipv6":
		if version != "v6" {
			c.JSON(http.StatusBadRequest, &WebsiteSpeedTestResult{
				Version:    "v6",
				HostRecord: "Skipped due to SINGLE_STACK=ipv6",
			})
			return
		}
	}

	// 缓存键：URL + 版本
	cacheKey := fmt.Sprintf("%s:%s", url, version)

	// 检查缓存
	if cached, ok := speedCache.Load(cacheKey); ok {
		entry := cached.(speedCacheEntry)
		if time.Since(entry.timestamp) < 1*time.Minute {
			c.JSON(200, entry.result)
			return
		}
		speedCache.Delete(cacheKey)
	}

	var result *WebsiteSpeedTestResult

	switch version {
	case "v6", "v4":
		rawResult, _, _ := sfGroup.Do(cacheKey, func() (interface{}, error) {
			r, e := websiteSpeed(url, version)
			if e != nil {
				errorResult := &WebsiteSpeedTestResult{
					HostRecord: "Error: " + e.Error(),
				}
				speedCache.Store(cacheKey, speedCacheEntry{result: errorResult, timestamp: time.Now()})
				go func() {
					time.Sleep(30 * time.Second)
					speedCache.Delete(cacheKey)
				}()
				return errorResult, nil
			}
			speedCache.Store(cacheKey, speedCacheEntry{result: r, timestamp: time.Now()})
			return r, nil
		})
		result = rawResult.(*WebsiteSpeedTestResult)
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid version",
		})
		return
	}

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

	parsedURL, _ := url.Parse(testUrl)
	if ssrf.HasLocalOrPrivateIP(parsedURL.Hostname()) {
		c.JSON(200, &SSLCheckResult{
			IPv4: fakeInvalidSSLResult(parsedURL.Hostname()),
			IPv6: fakeInvalidSSLResult(parsedURL.Hostname()),
		})
		return
	}

	if cached, ok := sslCache.Load(testUrl); ok {
		entry := cached.(sslCacheEntry)
		if time.Since(entry.timestamp) < 5*time.Minute {
			c.JSON(200, entry.result)
			return
		}
		sslCache.Delete(testUrl)
	}

	rawResult, _, _ := sfGroup.Do(testUrl, func() (interface{}, error) {
		result := &SSLCheckResult{}
		switch SINGLE_STACK {
		case "ipv4":
			ipv4, errV4 := checkSSL(testUrl, "v4")
			if errV4 != nil {
				ipv4 = &SSLCheckDetail{
					HostRecord:  "Error: " + errV4.Error(),
					IsExpired:   true,
					IsReachable: false,
				}
			}
			result.IPv4 = ipv4
			result.IPv6 = &SSLCheckDetail{
				HostRecord:  "Skipped due to SINGLE_STACK=ipv4",
				IsExpired:   true,
				IsReachable: false,
			}
		case "ipv6":
			ipv6, errV6 := checkSSL(testUrl, "v6")
			if errV6 != nil {
				ipv6 = &SSLCheckDetail{
					HostRecord:  "Error: " + errV6.Error(),
					IsExpired:   true,
					IsReachable: false,
				}
			}
			result.IPv6 = ipv6
			result.IPv4 = &SSLCheckDetail{
				HostRecord:  "Skipped due to SINGLE_STACK=ipv6",
				IsExpired:   true,
				IsReachable: false,
			}
		default:
			var wg sync.WaitGroup
			wg.Add(2)

			go func() {
				defer wg.Done()
				ipv6, errV6 := checkSSL(testUrl, "v6")
				if errV6 != nil {
					ipv6 = &SSLCheckDetail{
						HostRecord:  "Error: " + errV6.Error(),
						IsExpired:   true,
						IsReachable: false,
					}
				}
				result.IPv6 = ipv6
			}()

			go func() {
				defer wg.Done()
				ipv4, errV4 := checkSSL(testUrl, "v4")
				if errV4 != nil {
					ipv4 = &SSLCheckDetail{
						HostRecord:  "Error: " + errV4.Error(),
						IsExpired:   true,
						IsReachable: false,
					}
				}
				result.IPv4 = ipv4
			}()

			wg.Wait()
		}

		sslCache.Store(testUrl, sslCacheEntry{result: result, timestamp: time.Now()})

		if (result.IPv4 != nil && !result.IPv4.IsReachable) || (result.IPv6 != nil && !result.IPv6.IsReachable) {
			go func() {
				time.Sleep(30 * time.Second)
				sslCache.Delete(testUrl)
			}()
		}

		return result, nil
	})

	c.JSON(200, rawResult.(*SSLCheckResult))
}

func locateIP(c *gin.Context) {
	ip := c.Param("ip")
	slog.Debug("Locating IP", "ip", ip)
	c.JSON(http.StatusOK, ipdb.SearchIP(ip))
}
func locateUserIP(c *gin.Context) {
	ip := c.ClientIP()
	// 可能会有误报，因为某些环境下 ClientIP() 可能返回代理服务器的 IP 地址，而不是用户的真实 IP 地址
	slog.Debug("Locating user IP", "ip", ip)
	c.JSON(http.StatusOK, ipdb.SearchIP(ip))
}
func dnsQueryHandler(c *gin.Context) {

	domain := c.Param("domain")
	parsedURL, err := parseURL(domain)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid domain",
		})
		return
	}
	domain = parsedURL.Host
	recodeType := c.Param("type")
	switch recodeType {
	case "a":
		result, err := webtest.QueryA(domain)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, result)
	case "aaaa":
		result, err := webtest.ResolveAAAARecord(domain)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, result)
	case "cname":
		result, err := webtest.ResolveCNAMERecord(domain)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, result)
	case "mx":
		result, err := webtest.ResolveMXRecord(domain)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, result)
	case "ns":
		result, err := webtest.ResolveNSRecord(domain)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, result)
	case "ptr":
		result, err := webtest.ResolvePTRRecord(domain)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, result)
	case "srv":
		result, err := webtest.ResolveSRVRecord(domain)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, result)
	case "txt":
		result, err := webtest.ResolveTXTRecord(domain)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, result)
	case "caa":
		result, err := webtest.ResolveCAARecord(domain)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, result)
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid record type",
		})
		return
	}
}
func pingHandler(c *gin.Context) {
	host := c.Param("ip")
	port := c.Query("port")
	if host == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "IP or hostname parameter is required",
		})
		return
	}
	if port == "" {
		port = "80"
	}
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid port number",
		})
		return
	}

	count := 4
	if countStr := c.Query("count"); countStr != "" {
		n, err := strconv.Atoi(countStr)
		if err != nil || n < 1 || n > 20 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "count must be an integer between 1 and 20",
			})
			return
		}
		count = n
	}

	cacheKey := fmt.Sprintf("%s:%s:%d", host, port, count)
	if cached, ok := pingCache.Load(cacheKey); ok {
		entry := cached.(pingCacheEntry)
		if time.Since(entry.timestamp) < 1*time.Minute {
			c.JSON(200, entry.result)
			return
		}
		pingCache.Delete(cacheKey)
	}

	rawResult, _, _ := sfGroup.Do(cacheKey, func() (interface{}, error) {
		result := &TCPingResult{}

		switch SINGLE_STACK {
		case "ipv4":
			ipv4, errV4 := webtest.TCPingRun(host, port, count, "v4", 10*time.Second, 100*time.Millisecond)
			if errV4 != nil {
				ipv4 = &webtest.TCPingStats{
					IP: "Error: " + errV4.Error(),
				}
			}
			result.IPv4 = ipv4
			result.IPv6 = &webtest.TCPingStats{
				IP: "Skipped due to SINGLE_STACK=ipv4",
			}
		case "ipv6":
			ipv6, errV6 := webtest.TCPingRun(host, port, count, "v6", 10*time.Second, 100*time.Millisecond)
			if errV6 != nil {
				ipv6 = &webtest.TCPingStats{
					IP: "Error: " + errV6.Error(),
				}
			}
			result.IPv6 = ipv6
			result.IPv4 = &webtest.TCPingStats{
				IP: "Skipped due to SINGLE_STACK=ipv6",
			}
		default:
			var wg sync.WaitGroup
			wg.Add(2)

			go func() {
				defer wg.Done()
				ipv6, errV6 := webtest.TCPingRun(host, port, count, "v6", 10*time.Second, 100*time.Millisecond)
				if errV6 != nil {
					ipv6 = &webtest.TCPingStats{
						IP: "Error: " + errV6.Error(),
					}
				}
				result.IPv6 = ipv6
			}()

			go func() {
				defer wg.Done()
				ipv4, errV4 := webtest.TCPingRun(host, port, count, "v4", 10*time.Second, 100*time.Millisecond)
				if errV4 != nil {
					ipv4 = &webtest.TCPingStats{
						IP: "Error: " + errV4.Error(),
					}
				}
				result.IPv4 = ipv4
			}()

			wg.Wait()
		}

		pingCache.Store(cacheKey, pingCacheEntry{result: result, timestamp: time.Now()})

		ipv4Failed := result.IPv4 != nil && strings.HasPrefix(result.IPv4.IP, "Error:")
		ipv6Failed := result.IPv6 != nil && strings.HasPrefix(result.IPv6.IP, "Error:")
		if ipv4Failed && ipv6Failed {
			go func() {
				time.Sleep(30 * time.Second)
				pingCache.Delete(cacheKey)
			}()
		}

		return result, nil
	})

	c.JSON(200, rawResult.(*TCPingResult))
}

func healchCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
func readConfig() {
	PORTS = os.Getenv("PORTS")
	GH_PROXY = os.Getenv("GH_PROXY")
	// SINGLE_STACK can be "ipv4", "ipv6", or empty for both.
	// Empty string is a valid value meaning dual-stack, not "unconfigured".
	// 如果当前测速节点机器是单栈网络，建议设置 SINGLE_STACK 环境变量来跳过另一个协议的测试，以避免不必要的错误日志和延迟
	SINGLE_STACK = strings.ToLower(strings.TrimSpace(os.Getenv("SINGLE_STACK")))
	DNS_SERVER = os.Getenv("DNS_SERVER")
	ssrf.SetEnabled(os.Getenv("BLOCK_PRIVATE_IPS") != "false" && os.Getenv("BLOCK_PRIVATE_IPS") != "0")

	// SINGLE_STACK is intentionally excluded: empty string is a valid value (dual-stack).
	needConfigFile := PORTS == "" || GH_PROXY == "" || DNS_SERVER == ""
	if needConfigFile {
		viper.SetConfigName("setting")
		viper.SetConfigType("json")
		viper.AddConfigPath(".")
		if err := viper.ReadInConfig(); err != nil {
			slog.Warn("Failed to read config file, using defaults", "error", err)
		}
		if PORTS == "" {
			PORTS = viper.GetString("port")
		}
		if GH_PROXY == "" {
			GH_PROXY = viper.GetString("gh-proxy")
		}
		if SINGLE_STACK == "" {
			SINGLE_STACK = strings.ToLower(strings.TrimSpace(viper.GetString("single-stack")))
		}
		if DNS_SERVER == "" {
			DNS_SERVER = viper.GetString("dns-server")
		}
	}

	if PORTS == "" {
		PORTS = "8080"
	}
	slog.Info("SSRF protection initialized", "blockPrivateIPs", ssrf.Enabled())
}

func main() {
	readConfig()
	webtest.SetDNSServer(DNS_SERVER)
	initHTTPClients()
	slog.Info("Starting server", "port", PORTS, "gh_proxy", GH_PROXY, "single_stack", SINGLE_STACK, "dns_server", DNS_SERVER)
	ipdb.Init(GH_PROXY)

	r := gin.Default()
	r.Use(cors.Default())

	r.GET("/v1/detail/*url", checkWebsiteHandler)
	r.GET("/v1/ssl/*url", sslCheckHandler)
	r.GET("/v1/location/:ip", locateIP)
	r.GET("/v1/location", locateUserIP)
	r.GET("/v1/tcping/:ip", pingHandler)
	r.GET("/v1/dns/:type/*domain", dnsQueryHandler)
	r.GET("/v1/speed/:version/*url", websiteSpeedTestHandler)

	r.GET("/", healchCheck)

	if err := r.Run(":" + PORTS); err != nil {
		slog.Error("Server failed to start", "error", err)
	}
}
