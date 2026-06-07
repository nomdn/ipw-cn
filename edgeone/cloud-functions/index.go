package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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
	var network string
	if version == "v6" {
		network = "tcp6"
	} else {
		network = "tcp4"
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, net, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, addr)
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
	}

	ctx := httptrace.WithClientTrace(context.Background(), trace)

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
	downloadSpeed := float64(len(body)) / 1024.0 / (totalTime / 1000.0)

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
	if PORTS == "" {
		PORTS = "8080"
	}
}

func main() {
	readConfig()
	slog.Info("Starting server", "port", PORTS, "single_stack", SINGLE_STACK)
	r := gin.Default()
	r.Use(cors.Default())

	r.GET("/v1/detail/*url", checkWebsiteHandler)
	r.GET("/v1/ssl/*url", sslCheckHandler)
	r.GET("/", healchCheck)

	if err := r.Run(":" + PORTS); err != nil {
		slog.Error("Server failed to start", "error", err)
	}
}
