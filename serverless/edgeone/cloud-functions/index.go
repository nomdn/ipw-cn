package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"edgeone-cloud-functions/ssrf"
	"edgeone-cloud-functions/webtest"
	"encoding/json"
	"fmt"
	"io"
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
	"resty.dev/v3"
)

type Setting struct {
	Port         any    `json:"port"`
	GHProxy      string `json:"gh-proxy"`
	SINGLE_STACK string `json:"single-stack"`
	CORS         string `json:"cors"`
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
	SINGLE_STACK string
	sslCache     sync.Map
	pingCache    sync.Map
	speedCache   sync.Map
	whoisCache   sync.Map
	DNS_SERVER   string
	DNSSEC_DNS_SERVER string // DNSSEC 专用 DNS 服务器（dnssec-server / DNSSEC_DNS_SERVER），留空沿用 DNS_SERVER
	defaultPort  = fmt.Sprintf("%d", 5<<4)
	V6Client     *resty.Client
	V4Client     *resty.Client
	CORS         string
	ACCEPT_DOMAINS []string
)

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

type websiteCacheEntry struct {
	result    *WebsiteCheckResult
	timestamp time.Time
}

type sslCacheEntry struct {
	result    *SSLCheckResult
	timestamp time.Time
}

type pingCacheEntry struct {
	result    *TCPingDualResult
	timestamp time.Time
}

type speedCacheEntry struct {
	result    *WebsiteSpeedTestResult
	timestamp time.Time
}

type whoisCacheEntry struct {
	result    *webtest.WhoisResult
	timestamp time.Time
}

type TCPingDualResult struct {
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

// initHttpClient initializes the pooled HTTP clients (V4/V6) with SSRF protection.
// initHttpClient 初始化带 SSRF 防护的 V4/V6 连接池 HTTP 客户端。
// initHttpClient initializes the pooled HTTP clients (V4/V6) with SSRF protection.
// initHttpClient 初始化带 SSRF 防护的 V4/V6 连接池 HTTP 客户端。
func initHttpClient() {
	setTransport := func(network string) *http.Transport {
		dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
		return &http.Transport{
			DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
				if ssrf.Enabled() {
					host, port, err := net.SplitHostPort(addr)
					if err != nil {
						return nil, err
					}
					if ip := net.ParseIP(host); ip != nil {
						if ssrf.IsPrivateIP(ip) {
							slog.Warn("Blocked connection to private IP", "host", host)
							return nil, fmt.Errorf("request to private/internal address is not allowed")
						}
					} else {
						var dnsResult webtest.DNSResult
						if network == "tcp4" {
							dnsResult, err = webtest.ResolveARecord(host)
						} else {
							dnsResult, err = webtest.ResolveAAAARecord(host)
						}
						if err != nil {
							return nil, err
						}
						for _, ipStr := range dnsResult.Record {
							ip := net.ParseIP(ipStr)
							if ip != nil && ssrf.IsPrivateIP(ip) {
								slog.Warn("Blocked connection to private IP", "host", host, "ip", ip)
								return nil, fmt.Errorf("request to private/internal address is not allowed")
							}
						}
						if len(dnsResult.Record) > 0 {
							addr = net.JoinHostPort(dnsResult.Record[0], port)
						}
					}
				}
				return dialer.DialContext(ctx, network, addr)
			},
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
			DisableKeepAlives:  true,
		}
	}
	V6Client = resty.New()
	V4Client = resty.New()
	V6Client.SetTransport(setTransport("tcp6"))
	V4Client.SetTransport(setTransport("tcp4"))
	V6Client.SetTimeout(10 * time.Second)
	V4Client.SetTimeout(10 * time.Second)
	// 响应体上限：探针会读取任意用户提供的 URL，无上限时 10s 窗口内可灌入数 GB 导致 OOM
	V6Client.SetResponseBodyLimit(10 * 1024 * 1024)
	V4Client.SetResponseBodyLimit(10 * 1024 * 1024)
	V6Client.SetRedirectPolicy(resty.RedirectPolicyFunc(ssrf.SecureCheckRedirect))
	V4Client.SetRedirectPolicy(resty.RedirectPolicyFunc(ssrf.SecureCheckRedirect))
	V6Client.AddContentDecompresser("zstd", decompressZstd)
	V4Client.AddContentDecompresser("zstd", decompressZstd)
}

var zstdReaderPool = sync.Pool{
	New: func() interface{} {
		decoder, _ := zstd.NewReader(nil)
		return decoder
	},
}

func decompressZstd(r io.ReadCloser) (io.ReadCloser, error) {
	zr := zstdReaderPool.Get().(*zstd.Decoder)
	if err := zr.Reset(r); err != nil {
		zr.Close()
		var newErr error
		zr, newErr = zstd.NewReader(r)
		if newErr != nil {
			return nil, newErr
		}
	}
	return &zstdReader{s: r, r: zr}, nil
}

type zstdReader struct {
	s         io.ReadCloser
	r         *zstd.Decoder
	closeOnce sync.Once
	closeErr  error
}

func (b *zstdReader) Read(p []byte) (n int, err error) {
	return b.r.Read(p)
}

func (b *zstdReader) Close() error {
	b.closeOnce.Do(func() {
		b.closeErr = b.s.Close()
		if err := b.r.Reset(nil); err != nil {
			b.r.Close()
			if b.closeErr == nil {
				b.closeErr = err
			}
		} else {
			zstdReaderPool.Put(b.r)
		}
	})
	return b.closeErr
}

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

	dnsLookupTime := trace.DNSLookup.Seconds() * 1000
	if dnsLookupTime == 0 {
		dnsLookupTime = measureDNSTime(url, version)
	}
	tcpConnectTime := trace.TCPConnTime.Seconds() * 1000
	httpConnectTime := trace.ConnTime.Seconds() * 1000
	firstByteTime := trace.ServerTime.Seconds() * 1000

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
		DNSLookupTime:    dnsLookupTime,
		TCPConnectTime:   tcpConnectTime,
		HTTPConnectTime:  httpConnectTime,
		FirstByteTime:    firstByteTime,
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

	dnsLookupTime := trace.DNSLookup.Seconds() * 1000
	if dnsLookupTime == 0 {
		dnsLookupTime = measureDNSTime(url, version)
	}
	tcpConnectTime := trace.TCPConnTime.Seconds() * 1000
	httpConnectTime := trace.ConnTime.Seconds() * 1000
	firstByteTime := trace.ServerTime.Seconds() * 1000

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
		DNSLookupTime:    dnsLookupTime,
		TCPConnectTime:   tcpConnectTime,
		HTTPConnectTime:  httpConnectTime,
		FirstByteTime:    firstByteTime,
		TotalTime:        totalTime,
		PageSize:         int64(len(body)),
		DownloadSpeed:    downloadSpeed,
		IsReachable:      true,
	}

	return result, nil
}

func measureDNSTime(urlStr string, version string) float64 {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return 0
	}
	host := parsed.Hostname()
	if host == "" {
		return 0
	}
	start := time.Now()
	var dnsErr error
	if version == "v6" {
		_, dnsErr = webtest.ResolveAAAARecord(host)
	} else {
		_, dnsErr = webtest.ResolveARecord(host)
	}
	if dnsErr != nil {
		return 0
	}
	return time.Since(start).Seconds() * 1000
}

func checkSSL(url string, version string) (*SSLCheckDetail, error) {
	ssrfCtx := context.Background()
	var err error
	ssrfCtx, err = ssrf.ValidateOutboundTarget(ssrfCtx, url)
	if err != nil {
		return nil, err
	}

	client := V4Client
	if version == "v6" {
		client = V6Client
	}

	startTime := time.Now()
	resp, err := client.R().EnableTrace().SetContext(ssrfCtx).Get(url)
	if err != nil {
		return nil, err
	}
	endTime := time.Now()

	trace := resp.Request.TraceInfo()
	hostRecord := cleanHostRecord(trace.RemoteAddr)

	totalTime := float64(endTime.Sub(startTime).Milliseconds())
	body := resp.Bytes()
	var downloadSpeed float64
	if totalTime > 0 {
		downloadSpeed = float64(len(body)) / 1024.0 / (totalTime / 1000.0)
	}

	rawResp := resp.RawResponse
	var cert *x509.Certificate
	var remainingDays int
	var isExpired bool
	var certStartTime, certEndTime time.Time
	var issuerOrganization []string
	var issuerCommonName, subjectCommonName, domain string

	if rawResp.TLS != nil && len(rawResp.TLS.PeerCertificates) > 0 {
		cert = rawResp.TLS.PeerCertificates[0]
		now := time.Now()
		remainingDays = int(cert.NotAfter.Sub(now).Hours() / 24)
		isExpired = now.After(cert.NotAfter) || now.Before(cert.NotBefore)
		certStartTime = cert.NotBefore
		certEndTime = cert.NotAfter
		issuerOrganization = cert.Issuer.Organization
		issuerCommonName = cert.Issuer.CommonName
		subjectCommonName = cert.Subject.CommonName
		domain = cleanHostRecord(cert.Subject.CommonName)
	} else {
		return nil, fmt.Errorf("no SSL certificate found")
	}

	result := &SSLCheckDetail{
		CertValidityDays:   remainingDays,
		IsExpired:          isExpired,
		CertStartTime:      certStartTime,
		CertEndTime:        certEndTime,
		HTTPVersion:        rawResp.Proto,
		HostRecord:         hostRecord,
		HTTPSSStatusCode:   resp.StatusCode(),
		TotalTime:          totalTime,
		DownloadSpeed:      downloadSpeed,
		Domain:             domain,
		IssuerOrganization: issuerOrganization,
		IssuerCommonName:   issuerCommonName,
		SubjectCommonName:  subjectCommonName,
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
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return input
	}
	// 协议相对路径必须先于剥前导斜杠判断，否则 "//example.com" 被剥成 "/example.com"，
	// 该分支永远不会命中，最终拼出空 host 的 "https:///example.com"
	if strings.HasPrefix(input, "//") {
		return "https:" + input
	}
	input = strings.TrimPrefix(input, "/")
	return "https://" + input
}

func parseURL(input string) (*url.URL, error) {
	input = normalizeURL(input)
	return url.Parse(input)
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

	parsedURL, err := url.Parse(testUrl)
	if err == nil && ssrf.HasLocalOrPrivateIP(parsedURL.Hostname()) {
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

	// If both IPv4 and IPv6 fail, only cache for 30 seconds
	// 如果 IPv4 和 IPv6 都失败，只缓存30秒
	if (result.IPv4 != nil && !result.IPv4.IsReachable) && (result.IPv6 != nil && !result.IPv6.IsReachable) {
		go func() {
			time.Sleep(30 * time.Second)
			websiteCache.Delete(testUrl)
		}()
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

	parsedURL, err := url.Parse(testUrl)
	if err == nil && ssrf.HasLocalOrPrivateIP(parsedURL.Hostname()) {
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

	result := &SSLCheckResult{}
	switch SINGLE_STACK {
	case "ipv4":
		ipv4, errV4 := checkSSL(testUrl, "v4")
		if errV4 != nil {
			ipv4 = &SSLCheckDetail{
				HostRecord: "Error: " + errV4.Error(),
				IsExpired:  true,
			}
		}
		result.IPv4 = ipv4
		result.IPv6 = &SSLCheckDetail{
			HostRecord: "Skipped due to SINGLE_STACK=ipv4",
			IsExpired:  true,
		}
	case "ipv6":
		ipv6, errV6 := checkSSL(testUrl, "v6")
		if errV6 != nil {
			ipv6 = &SSLCheckDetail{
				HostRecord: "Error: " + errV6.Error(),
				IsExpired:  true,
			}
		}
		result.IPv6 = ipv6
		result.IPv4 = &SSLCheckDetail{
			HostRecord: "Skipped due to SINGLE_STACK=ipv6",
			IsExpired:  true,
		}
	default:
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
	}

	sslCache.Store(testUrl, sslCacheEntry{result: result, timestamp: time.Now()})

	// 如果 IPv4 和 IPv6 都失败，只缓存30秒
	if (result.IPv4 != nil && !result.IPv4.IsReachable) && (result.IPv6 != nil && !result.IPv6.IsReachable) {
		go func() {
			time.Sleep(30 * time.Second)
			sslCache.Delete(testUrl)
		}()
	}

	c.JSON(200, result)
}

func websiteSpeedTestHandler(c *gin.Context) {
	testUrl := c.Param("url")
	version := c.Param("version")
	var result *WebsiteSpeedTestResult
	var err error
	if testUrl == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URL parameter is required",
		})
		return
	}
	url := normalizeURL(testUrl)

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

	cacheKey := fmt.Sprintf("%s:%s", url, version)

	if cached, ok := speedCache.Load(cacheKey); ok {
		entry := cached.(speedCacheEntry)
		if time.Since(entry.timestamp) < 5*time.Minute {
			c.JSON(200, entry.result)
			return
		}
		speedCache.Delete(cacheKey)
	}

	switch version {
	case "v6":
		result, err = websiteSpeed(url, "v6")
	case "v4":
		result, err = websiteSpeed(url, "v4")
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid version",
		})
		return
	}

	if err != nil {
		errorResult := &WebsiteSpeedTestResult{
			HostRecord: "Error: " + err.Error(),
		}
		// 错误结果只缓存30秒
		speedCache.Store(cacheKey, speedCacheEntry{result: errorResult, timestamp: time.Now()})
		go func() {
			time.Sleep(30 * time.Second)
			speedCache.Delete(cacheKey)
		}()
		c.JSON(http.StatusInternalServerError, errorResult)
		return
	}

	speedCache.Store(cacheKey, speedCacheEntry{result: result, timestamp: time.Now()})
	c.JSON(200, result)
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
		result, err := webtest.ResolveARecord(domain)
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
		port = defaultPort
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
		if time.Since(entry.timestamp) < 5*time.Minute {
			c.JSON(200, entry.result)
			return
		}
		pingCache.Delete(cacheKey)
	}

	result := &TCPingDualResult{}

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

	// 如果 IPv4 和 IPv6 都失败，只缓存30秒
	ipv4Failed := result.IPv4 != nil && strings.HasPrefix(result.IPv4.IP, "Error:")
	ipv6Failed := result.IPv6 != nil && strings.HasPrefix(result.IPv6.IP, "Error:")
	if ipv4Failed && ipv6Failed {
		go func() {
			time.Sleep(30 * time.Second)
			pingCache.Delete(cacheKey)
		}()
	}

	c.JSON(200, result)
}

func whoisHandler(c *gin.Context) {
	domain := c.Param("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Domain parameter is required",
		})
		return
	}

	if cached, ok := whoisCache.Load(domain); ok {
		entry := cached.(whoisCacheEntry)
		if time.Since(entry.timestamp) < 5*time.Minute {
			c.JSON(200, entry.result)
			return
		}
		whoisCache.Delete(domain)
	}

	result, err := webtest.QueryWhois(domain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	whoisCache.Store(domain, whoisCacheEntry{result: result, timestamp: time.Now()})
	c.JSON(http.StatusOK, result)
}

func dnssecHandler(c *gin.Context) {
	domain := c.Param("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Domain parameter is required",
		})
		return
	}

	result, err := webtest.ResolveDNSSEC(domain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

func healchCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
// fetchRemoteConfig 从远端 URL 拉取配置文件（遵守 setting.json 的 JSON 格式）。
// 拉取或解析失败时返回错误，调用方应回退到本地配置。
func fetchRemoteConfig(url string) (map[string]any, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote config returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var CONFIG map[string]any
	if err := json.Unmarshal(body, &CONFIG); err != nil {
		return nil, fmt.Errorf("invalid remote config JSON: %w", err)
	}
	return CONFIG, nil
}

// configValue 返回配置 map 中指定 key 的非空字符串值；
// key 不存在或值为空时返回 ""（表示不覆盖本地配置）。
func configValue(CONFIG map[string]any, key string) string {
	v, ok := CONFIG[key]
	if !ok || v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return strings.TrimSpace(s)
	case float64:
		return fmt.Sprintf("%.0f", s)
	case bool:
		return strconv.FormatBool(s)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

// applyRemoteConfig 从环境变量 REMOTE_CONFIG_URL 拉取远端配置并覆盖本地配置。
// 优先级：远端配置 > 环境变量 > setting.json。
func applyRemoteConfig() {
	url := os.Getenv("REMOTE_CONFIG_URL")
	if url == "" {
		return
	}
	CONFIG, err := fetchRemoteConfig(url)
	if err != nil {
		slog.Warn("Failed to fetch remote config, falling back to local config", "url", url, "error", err)
		return
	}
	if v := configValue(CONFIG, "port"); v != "" {
		PORTS = v
	}
	if v := configValue(CONFIG, "single-stack"); v != "" {
		SINGLE_STACK = strings.ToLower(v)
	}
	if v := configValue(CONFIG, "dns-server"); v != "" {
		DNS_SERVER = v
	}
	if v := configValue(CONFIG, "dnssec-server"); v != "" {
		DNSSEC_DNS_SERVER = v
	}
	if v := configValue(CONFIG, "cors"); v != "" {
		CORS = v
	}
	if v := configValue(CONFIG, "block-private-ips"); v != "" {
		ssrf.SetEnabled(v != "false" && v != "0")
	}
	if CORS != "" {
		ACCEPT_DOMAINS = splitAndTrim(CORS, ",")
	}
	slog.Info("Remote config applied", "url", url)
}

// readConfig 加载配置：环境变量为本地来源，远端配置（REMOTE_CONFIG_URL）优先级最高。
func readConfig() {
	// 1) 环境变量
	if v := os.Getenv("PORTS"); v != "" {
		PORTS = v
	}
	if PORTS == "" {
		PORTS = os.Getenv("PORT")
	}
	if v := os.Getenv("SINGLE_STACK"); v != "" {
		SINGLE_STACK = strings.ToLower(strings.TrimSpace(v))
	}
	if v := os.Getenv("DNS_SERVER"); v != "" {
		DNS_SERVER = v
	}
	if v := os.Getenv("DNSSEC_DNS_SERVER"); v != "" {
		DNSSEC_DNS_SERVER = v
	}
	if v := os.Getenv("CORS"); v != "" {
		CORS = v
	}
	if v := os.Getenv("BLOCK_PRIVATE_IPS"); v != "" {
		ssrf.SetEnabled(v != "false" && v != "0")
	}

	// 2) 远端配置（最高优先级，覆盖环境变量）
	applyRemoteConfig()

	if PORTS == "" {
		PORTS = "8080"
	}
	if CORS != "" {
		ACCEPT_DOMAINS = splitAndTrim(CORS, ",")
	}
	slog.Info("SSRF protection initialized", "blockPrivateIPs", ssrf.Enabled())
}

// splitAndTrim 按逗号切分并去掉每段首尾空白：CORS 配置 "a.com, b.com" 带空格时，
// 直接 Split 会产生带前导空格的 origin，永远匹配不上请求头
func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// sweepExpiredCaches 按条目时间戳清扫各 sync.Map 缓存（TTL 均为 5 分钟，清扫窗口取 10 分钟），
// 缓存条目只在同 key 重访时惰性淘汰，公网端点被唯一 key 洪打时内存会无限增长
func sweepExpiredCaches() {
	cutoff := time.Now().Add(-10 * time.Minute)
	sweep := func(m *sync.Map) {
		m.Range(func(k, v any) bool {
			var ts time.Time
			switch e := v.(type) {
			case websiteCacheEntry:
				ts = e.timestamp
			case sslCacheEntry:
				ts = e.timestamp
			case pingCacheEntry:
				ts = e.timestamp
			case speedCacheEntry:
				ts = e.timestamp
			case whoisCacheEntry:
				ts = e.timestamp
			default:
				return true
			}
			if ts.Before(cutoff) {
				m.Delete(k)
			}
			return true
		})
	}
	sweep(&websiteCache)
	sweep(&sslCache)
	sweep(&pingCache)
	sweep(&speedCache)
	sweep(&whoisCache)
}

func main() {
	readConfig()
	initHttpClient()
	webtest.SetDNSServer(DNS_SERVER)
	webtest.SetDNSSecServer(DNSSEC_DNS_SERVER)
	slog.Info("Starting server", "port", PORTS, "single_stack", SINGLE_STACK)

	// 缓存清扫：定期淘汰过期条目
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			sweepExpiredCaches()
		}
	}()

	r := gin.Default()
	corsConfig := cors.DefaultConfig()
	if len(ACCEPT_DOMAINS) > 0{
		corsConfig.AllowOrigins = ACCEPT_DOMAINS
	}else{
		corsConfig.AllowAllOrigins = true
	}
	r.Use(cors.New(corsConfig))

	r.GET("/v1/detail/*url", checkWebsiteHandler)
	r.GET("/v1/ssl/*url", sslCheckHandler)
	r.GET("/v1/tcping/:ip", pingHandler)
	r.GET("/v1/dns/:type/*domain", dnsQueryHandler)
	r.GET("/v1/whois/:domain", whoisHandler)
	r.GET("/v1/dnssec/:domain", dnssecHandler)
	r.GET("/v1/speed/:version/*url", websiteSpeedTestHandler)
	r.GET("/", healchCheck)

	if err := r.Run(":" + PORTS); err != nil {
		slog.Error("Server failed to start", "error", err)
	}
}
