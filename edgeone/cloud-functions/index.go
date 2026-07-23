package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"edgeone-cloud-functions/webtest"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptrace"
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
	DNS_SERVER   string
	defaultPort  = fmt.Sprintf("%d", 5<<4)
	V6Client     *resty.Client
	V4Client     *resty.Client
)

var blockPrivateIPs = true

type contextKey string

const ssrfIPsKey contextKey = "ssrf_validated_ips"

func init() {
	if v := os.Getenv("BLOCK_PRIVATE_IPS"); v != "" {
		blockPrivateIPs = v != "false" && v != "0"
	}
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsPrivate() {
		return true
	}
	if ip.IsLoopback() {
		return true
	}
	if ip.IsLinkLocalUnicast() {
		return true
	}
	if ip.IsUnspecified() {
		return true
	}
	return false
}

func validateOutboundTarget(ctx context.Context, targetURL string) (context.Context, error) {
	if !blockPrivateIPs {
		return ctx, nil
	}
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return ctx, err
	}
	host := parsed.Hostname()
	if host == "" {
		return ctx, fmt.Errorf("empty host")
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return ctx, err
	}
	for _, ip := range ips {
		if isPrivateIP(ip) {
			slog.Warn("Blocked request to private IP", "host", host, "ip", ip)
			return ctx, fmt.Errorf("request to private/internal address is not allowed")
		}
	}
	return context.WithValue(ctx, ssrfIPsKey, ips), nil
}

func secureCheckRedirect(req *http.Request, via []*http.Request) error {
	if !blockPrivateIPs {
		return nil
	}
	for _, r := range via {
		redirectURL := r.URL
		host := redirectURL.Hostname()
		if host == "" {
			continue
		}
		ips, err := net.LookupIP(host)
		if err != nil {
			return err
		}
		for _, ip := range ips {
			if isPrivateIP(ip) {
				slog.Warn("Blocked redirect to private IP", "host", host, "ip", ip)
				return fmt.Errorf("redirect to private/internal address is not allowed")
			}
		}
	}
	return nil
}

func isLocalOrPrivateIP(ip net.IP) bool {
	if ip.IsPrivate() {
		return true
	}
	if ip.IsLoopback() {
		return true
	}
	if ip.IsLinkLocalUnicast() {
		return true
	}
	if ip.IsUnspecified() {
		return true
	}
	return false
}

func hasLocalOrPrivateIP(host string) bool {
	ips, err := net.LookupIP(host)
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if isLocalOrPrivateIP(ip) {
			return true
		}
	}
	return false
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

func initHttpClient() {
	setTransport := func(network string) *http.Transport {
		dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
		return &http.Transport{
			DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
				if blockPrivateIPs {
					host, _, err := net.SplitHostPort(addr)
					if err != nil {
						return nil, err
					}
					var ips []net.IP
					if v, ok := ctx.Value(ssrfIPsKey).([]net.IP); ok {
						ips = v
					} else {
						ips, err = net.LookupIP(host)
						if err != nil {
							return nil, err
						}
					}
					for _, ip := range ips {
						if isPrivateIP(ip) {
							slog.Warn("Blocked connection to private IP", "host", host, "ip", ip)
							return nil, fmt.Errorf("request to private/internal address is not allowed")
						}
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
	V6Client.SetRedirectPolicy(resty.RedirectPolicyFunc(secureCheckRedirect))
	V4Client.SetRedirectPolicy(resty.RedirectPolicyFunc(secureCheckRedirect))
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

func checkWebsite(url string, version string) (*WebsiteCheckDetail, error) {
	ctx := context.Background()
	var err error
	ctx, err = validateOutboundTarget(ctx, url)
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
	ctx, err = validateOutboundTarget(ctx, url)
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
	ssrfCtx := context.Background()
	var err error
	ssrfCtx, err = validateOutboundTarget(ssrfCtx, url)
	if err != nil {
		return nil, err
	}

	var network string
	if version == "v6" {
		network = "tcp6"
	} else {
		network = "tcp4"
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		DialContext: func(dialCtx context.Context, _, addr string) (net.Conn, error) {
			host, port, _ := net.SplitHostPort(addr)
			var ip string
			if v, ok := ssrfCtx.Value(ssrfIPsKey).([]net.IP); ok && len(v) > 0 {
				ip = v[0].String()
			} else {
				var resolveErr error
				ip, resolveErr = webtest.ResolveIP(host, version)
				if resolveErr != nil {
					return nil, resolveErr
				}
			}
			if blockPrivateIPs {
				parsedIP := net.ParseIP(ip)
				if parsedIP != nil && isPrivateIP(parsedIP) {
					slog.Warn("Blocked SSL connection to private IP", "host", host, "ip", ip)
					return nil, fmt.Errorf("request to private/internal address is not allowed")
				}
			}
			return dialer.DialContext(dialCtx, network, net.JoinHostPort(ip, port))
		},
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	hostRecord := ""
	trace := &httptrace.ClientTrace{
		GotConn: func(connInfo httptrace.GotConnInfo) {
			hostRecord = connInfo.Conn.RemoteAddr().String()
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
		CheckRedirect: secureCheckRedirect,
	}

	ctx := httptrace.WithClientTrace(ssrfCtx, trace)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	endTime := time.Now()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	totalTime := float64(endTime.Sub(startTime).Milliseconds())
	var downloadSpeed float64
	if totalTime > 0 {
		downloadSpeed = float64(len(body)) / 1024.0 / (totalTime / 1000.0)
	}

	var cert *x509.Certificate
	var remainingDays int
	var isExpired bool
	var certStartTime, certEndTime time.Time
	var issuerOrganization []string
	var issuerCommonName, subjectCommonName, domain string

	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		cert = resp.TLS.PeerCertificates[0]
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
		HTTPVersion:        resp.Proto,
		HostRecord:         hostRecord,
		HTTPSSStatusCode:   resp.StatusCode,
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
	if hasLocalOrPrivateIP(parsedURL.Hostname()) {
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

	parsedURL, _ := url.Parse(testUrl)
	if hasLocalOrPrivateIP(parsedURL.Hostname()) {
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
		if time.Since(entry.timestamp) < 1*time.Minute {
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
		if time.Since(entry.timestamp) < 1*time.Minute {
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

func healchCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
func readConfig() {
	PORTS = os.Getenv("PORTS")
	SINGLE_STACK = os.Getenv("SINGLE_STACK")
	DNS_SERVER = os.Getenv("DNS_SERVER")
	if PORTS == "" {
		PORTS = "8080"
	}
}

func main() {
	readConfig()
	webtest.SetDNSServer(DNS_SERVER)
	slog.Info("Starting server", "port", PORTS, "single_stack", SINGLE_STACK)
	r := gin.Default()
	r.Use(cors.Default())

	r.GET("/v1/detail/*url", checkWebsiteHandler)
	r.GET("/v1/ssl/*url", sslCheckHandler)
	r.GET("/v1/tcping/:ip", pingHandler)
	r.GET("/v1/dns/:type/*domain", dnsQueryHandler)
	r.GET("/v1/speed/:version/*url", websiteSpeedTestHandler)
	r.GET("/", healchCheck)

	if err := r.Run(":" + PORTS); err != nil {
		slog.Error("Server failed to start", "error", err)
	}
}
