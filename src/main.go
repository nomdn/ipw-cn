package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"lemon-ipw/ipdb"
	"lemon-ipw/ssrf"
	"lemon-ipw/webtest"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/netutil/sysresolv"
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
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
			DisableKeepAlives: true,
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

func fakePerfectWebsiteResult(host string) *webtest.WebsiteCheckDetail {
	cleanHost := strings.TrimPrefix(host, "https://")
	cleanHost = strings.TrimPrefix(cleanHost, "http://")
	return &webtest.WebsiteCheckDetail{
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

func fakeInvalidSSLResult(host string) *webtest.SSLCheckDetail {
	return &webtest.SSLCheckDetail{
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

// Global variables and structs
// 全局变量与结构体
var (
	PORTS                string
	GH_PROXY             string
	LOG_LEVEL            string
	websiteCache         sync.Map
	SINGLE_STACK         string
	DNS_SERVER           string
	DNSSEC_DNS_SERVER    string // DNSSEC 专用 DNS 服务器（dnssec-server / DNSSEC_DNS_SERVER），留空沿用 DNS_SERVER
	sslCache             sync.Map
	pingCache            sync.Map
	speedCache           sync.Map
	whoisCache           sync.Map
	asnWhoisCache        sync.Map
	sfGroup              singleflight.Group
	V6Client             *resty.Client
	V4Client             *resty.Client
	IPDB                 string
	CORS                 string
	ACCEPT_DOMAINS       []string
	ACCESS_TOKEN         string
	REMOTE_CONFIG_URL    string
	REMOTE_IGNORE_CONFIG []string // 不被远端配置覆盖的配置项列表（remote-ignore-config / REMOTE_IGNORE_CONFIG）
	WS_URL               string   // WS 客户端：中间件 WS 地址（ws://host:port/ws）
	WS_NODE_ID           string   // WS 客户端：节点 id（与中间件 ws-keys 键一致）
	WS_NODE_KEY          string   // WS 客户端：注册 key（与中间件 ws-keys[节点id] 一致，可空）
	VERSION              string
	COMMIT               string
	BUILD_TIME           string
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
	result    *webtest.WebsiteSpeedTestResult
	timestamp time.Time
}

type whoisCacheEntry struct {
	result    *webtest.WhoisResult
	timestamp time.Time
}

type asnWhoisCacheEntry struct {
	result    *webtest.ASNWhoisResult
	timestamp time.Time
}

type WebsiteCheckResult struct {
	IPv4 *webtest.WebsiteCheckDetail `json:"ipv4"`
	IPv6 *webtest.WebsiteCheckDetail `json:"ipv6"`
}

type SSLCheckResult struct {
	IPv4 *webtest.SSLCheckDetail `json:"ipv4"`
	IPv6 *webtest.SSLCheckDetail `json:"ipv6"`
}
type TCPingResult struct {
	IPv4 *webtest.TCPingStats `json:"ipv4"`
	IPv6 *webtest.TCPingStats `json:"ipv6"`
}

// Business Endpoints
// 业务端点

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
			ipv4, errV4 := webtest.CheckWebsite(testUrl, "v4")
			if errV4 != nil {
				ipv4 = &webtest.WebsiteCheckDetail{
					HostRecord:  "Error: " + errV4.Error(),
					IsReachable: false,
				}
			}
			result.IPv4 = ipv4
			result.IPv6 = &webtest.WebsiteCheckDetail{
				HostRecord:  "Skipped due to SINGLE_STACK=ipv4",
				IsReachable: false,
			}
		case "ipv6":
			ipv6, errV6 := webtest.CheckWebsite(testUrl, "v6")
			if errV6 != nil {
				ipv6 = &webtest.WebsiteCheckDetail{
					HostRecord:  "Error: " + errV6.Error(),
					IsReachable: false,
				}
			}
			result.IPv6 = ipv6
			result.IPv4 = &webtest.WebsiteCheckDetail{
				HostRecord:  "Skipped due to SINGLE_STACK=ipv6",
				IsReachable: false,
			}
		default:
			var wg sync.WaitGroup
			wg.Add(2)

			go func() {
				defer wg.Done()
				ipv6, errV6 := webtest.CheckWebsite(testUrl, "v6")
				if errV6 != nil {
					ipv6 = &webtest.WebsiteCheckDetail{
						HostRecord:  "Error: " + errV6.Error(),
						IsReachable: false,
					}
				}
				result.IPv6 = ipv6
			}()

			go func() {
				defer wg.Done()
				ipv4, errV4 := webtest.CheckWebsite(testUrl, "v4")
				if errV4 != nil {
					ipv4 = &webtest.WebsiteCheckDetail{
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
			c.JSON(http.StatusBadRequest, &webtest.WebsiteSpeedTestResult{
				Version:    "v4",
				HostRecord: "Skipped due to SINGLE_STACK=ipv4",
			})
			return
		}
	case "ipv6":
		if version != "v6" {
			c.JSON(http.StatusBadRequest, &webtest.WebsiteSpeedTestResult{
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
		if time.Since(entry.timestamp) < 5*time.Minute {
			c.JSON(200, entry.result)
			return
		}
		speedCache.Delete(cacheKey)
	}

	var result *webtest.WebsiteSpeedTestResult

	switch version {
	case "v6", "v4":
		rawResult, _, _ := sfGroup.Do(cacheKey, func() (interface{}, error) {
			r, e := webtest.SpeedTest(url, version)
			if e != nil {
				errorResult := &webtest.WebsiteSpeedTestResult{
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
		result = rawResult.(*webtest.WebsiteSpeedTestResult)
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
			ipv4, errV4 := webtest.CheckSSL(testUrl, "v4")
			if errV4 != nil {
				ipv4 = &webtest.SSLCheckDetail{
					HostRecord:  "Error: " + errV4.Error(),
					IsExpired:   true,
					IsReachable: false,
				}
			}
			result.IPv4 = ipv4
			result.IPv6 = &webtest.SSLCheckDetail{
				HostRecord:  "Skipped due to SINGLE_STACK=ipv4",
				IsExpired:   true,
				IsReachable: false,
			}
		case "ipv6":
			ipv6, errV6 := webtest.CheckSSL(testUrl, "v6")
			if errV6 != nil {
				ipv6 = &webtest.SSLCheckDetail{
					HostRecord:  "Error: " + errV6.Error(),
					IsExpired:   true,
					IsReachable: false,
				}
			}
			result.IPv6 = ipv6
			result.IPv4 = &webtest.SSLCheckDetail{
				HostRecord:  "Skipped due to SINGLE_STACK=ipv6",
				IsExpired:   true,
				IsReachable: false,
			}
		default:
			var wg sync.WaitGroup
			wg.Add(2)

			go func() {
				defer wg.Done()
				ipv6, errV6 := webtest.CheckSSL(testUrl, "v6")
				if errV6 != nil {
					ipv6 = &webtest.SSLCheckDetail{
						HostRecord:  "Error: " + errV6.Error(),
						IsExpired:   true,
						IsReachable: false,
					}
				}
				result.IPv6 = ipv6
			}()

			go func() {
				defer wg.Done()
				ipv4, errV4 := webtest.CheckSSL(testUrl, "v4")
				if errV4 != nil {
					ipv4 = &webtest.SSLCheckDetail{
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

func asnLookupHandler(c *gin.Context) {
	ip := c.Param("ip")
	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "IP parameter is required",
		})
		return
	}

	result := ipdb.SearchIP(ip, "maxmind_asn", "dbip_asn", "ip2location_asn")

	asnResult := map[string]interface{}{
		"ip": ip,
	}

	if ip2locASN, ok := result["ip2location_asn"].(map[string]string); ok && ip2locASN["asn"] != "" {
		asnResult["ip2location_asn"] = map[string]string{
			"asn": ip2locASN["asn"],
			"as":  ip2locASN["as"],
		}
	} else if errStr, ok := result["ip2location_asn"].(string); ok {
		asnResult["ip2location_asn"] = map[string]string{
			"error": errStr,
		}
	}

	if maxmindASN, ok := result["maxmind_asn"].(*ipdb.MMDBASNResult); ok {
		asnResult["geolite2_asn"] = map[string]string{
			"asn": maxmindASN.ASN,
			"org": maxmindASN.Org,
		}
	} else if errStr, ok := result["maxmind_asn"].(string); ok {
		asnResult["geolite2_asn"] = map[string]string{
			"error": errStr,
		}
	}

	if dbipASN, ok := result["dbip_asn"].(*ipdb.MMDBASNResult); ok {
		asnResult["dbip_asn"] = map[string]string{
			"asn": dbipASN.ASN,
			"org": dbipASN.Org,
		}
	} else if errStr, ok := result["dbip_asn"].(string); ok {
		asnResult["dbip_asn"] = map[string]string{
			"error": errStr,
		}
	}

	// 使用 WHOIS 对 ASN 进行进一步解析
	if maxmindASN, ok := result["maxmind_asn"].(*ipdb.MMDBASNResult); ok {
		asnKey := "AS" + maxmindASN.ASN
		if cached, ok := asnWhoisCache.Load(asnKey); ok {
			entry := cached.(asnWhoisCacheEntry)
			if time.Since(entry.timestamp) < 5*time.Minute {
				asnResult["whois"] = entry.result
			} else {
				asnWhoisCache.Delete(asnKey)
			}
		} else {
			whoisData, err := webtest.QueryASNWhois(maxmindASN.ASN)
			if err == nil {
				asnResult["whois"] = whoisData
				asnWhoisCache.Store(asnKey, asnWhoisCacheEntry{result: whoisData, timestamp: time.Now()})
			}
		}
	}

	c.JSON(http.StatusOK, asnResult)
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
		if time.Since(entry.timestamp) < 5*time.Minute {
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
func tokenCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token != "Bearer "+ACCESS_TOKEN {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		} else {
			c.Next()
		}
	}
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

// applyRemoteConfig 从远端配置 URL（REMOTE_CONFIG_URL，可由环境变量或 setting.json
// 的 remote-config-url 提供）拉取配置并覆盖本地配置。
// 优先级：远端配置 > 环境变量 > setting.json。
// access-token 例外：保持原有优先级（环境变量 > setting.json），不随远端配置覆盖。
func applyRemoteConfig() {
	url := REMOTE_CONFIG_URL
	if url == "" {
		return
	}
	CONFIG, err := fetchRemoteConfig(url)
	if err != nil {
		slog.Warn("Failed to fetch remote config, falling back to local config", "url", url, "error", err)
		return
	}
	// ignore 列表：数组中的配置项不被远端覆盖（逐键判断跳过）
	ignored := func(key string) bool {
		for _, k := range REMOTE_IGNORE_CONFIG {
			if k == key {
				return true
			}
		}
		return false
	}
	if v := configValue(CONFIG, "port"); v != "" && !ignored("port") {
		PORTS = v
	}
	if v := configValue(CONFIG, "gh-proxy"); v != "" && !ignored("gh-proxy") {
		GH_PROXY = v
	}
	if v := configValue(CONFIG, "single-stack"); v != "" && !ignored("single-stack") {
		SINGLE_STACK = strings.ToLower(v)
	}
	if v := configValue(CONFIG, "dns-server"); v != "" && !ignored("dns-server") {
		DNS_SERVER = v
	}
	if v := configValue(CONFIG, "dnssec-server"); v != "" && !ignored("dnssec-server") {
		DNSSEC_DNS_SERVER = v
	}
	if v := configValue(CONFIG, "ipdb"); v != "" && !ignored("ipdb") {
		IPDB = v
	}
	if v := configValue(CONFIG, "cors"); v != "" && !ignored("cors") {
		CORS = v
	}
	// block-private-ips 与 setting.json 格式一致，允许远端覆盖
	if v := configValue(CONFIG, "block-private-ips"); v != "" && !ignored("block-private-ips") {
		ssrf.SetEnabled(v != "false" && v != "0")
	}
	// WS 客户端配置（远端可覆盖，除非在 ignore 列表中）
	if v := configValue(CONFIG, "ws-url"); v != "" && !ignored("ws-url") {
		WS_URL = v
	}
	if v := configValue(CONFIG, "node-id"); v != "" && !ignored("node-id") {
		WS_NODE_ID = v
	}
	if v := configValue(CONFIG, "node-key"); v != "" && !ignored("node-key") {
		WS_NODE_KEY = v
	}
	// access-token 不在此覆盖：保持原有优先级（环境变量 > setting.json）
	if CORS != "" {
		ACCEPT_DOMAINS = strings.Split(CORS, ",")
	}
	slog.Info("Remote config applied", "url", url)
}

func readConfig() {
	PORTS = os.Getenv("PORTS")
	GH_PROXY = os.Getenv("GH_PROXY")
	// SINGLE_STACK can be "ipv4", "ipv6", or empty for both.
	// Empty string is a valid value meaning dual-stack, not "unconfigured".
	// 如果当前测速节点机器是单栈网络，建议设置 SINGLE_STACK 环境变量来跳过另一个协议的测试，以避免不必要的错误日志和延迟
	SINGLE_STACK = strings.ToLower(strings.TrimSpace(os.Getenv("SINGLE_STACK")))
	DNS_SERVER = os.Getenv("DNS_SERVER")
	DNSSEC_DNS_SERVER = os.Getenv("DNSSEC_DNS_SERVER")
	IPDB = os.Getenv("IPDB")
	CORS = os.Getenv("CORS")
	ACCESS_TOKEN = os.Getenv("ACCESS_TOKEN")
	ssrf.SetEnabled(os.Getenv("BLOCK_PRIVATE_IPS") != "false" && os.Getenv("BLOCK_PRIVATE_IPS") != "0")

	// SINGLE_STACK is intentionally excluded: empty string is a valid value (dual-stack).

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
	if DNSSEC_DNS_SERVER == "" {
		DNSSEC_DNS_SERVER = viper.GetString("dnssec-server")
	}
	if IPDB == "" {
		IPDB = viper.GetString("ipdb")
	}
	if CORS == "" {
		CORS = viper.GetString("cors")
	}
	if PORTS == "" {
		PORTS = "8080"
	}
	if CORS != "" {
		ACCEPT_DOMAINS = strings.Split(CORS, ",")
	}
	if ACCESS_TOKEN == "" {
		ACCESS_TOKEN = viper.GetString("access-token")
	}
	// REMOTE_CONFIG_URL 优先级：环境变量 > setting.json 的 remote-config-url
	REMOTE_CONFIG_URL = os.Getenv("REMOTE_CONFIG_URL")
	if REMOTE_CONFIG_URL == "" {
		REMOTE_CONFIG_URL = viper.GetString("remote-config-url")
	}
	// WS 客户端（接入中间件 WS 通道）：WS_URL / NODE_ID / NODE_KEY，env 优先
	WS_URL = os.Getenv("WS_URL")
	if WS_URL == "" {
		WS_URL = viper.GetString("ws-url")
	}
	WS_NODE_ID = os.Getenv("NODE_ID")
	if WS_NODE_ID == "" {
		WS_NODE_ID = viper.GetString("node-id")
	}
	WS_NODE_KEY = os.Getenv("NODE_KEY")
	if WS_NODE_KEY == "" {
		WS_NODE_KEY = viper.GetString("node-key")
	}
	// REMOTE_IGNORE_CONFIG：不被远端覆盖的配置项列表（JSON 数组字符串，env 优先）
	if raw := os.Getenv("REMOTE_IGNORE_CONFIG"); raw != "" {
		var list []string
		if err := json.Unmarshal([]byte(raw), &list); err == nil {
			REMOTE_IGNORE_CONFIG = list
		} else {
			slog.Warn("invalid REMOTE_IGNORE_CONFIG, ignored", "error", err)
		}
	} else {
		REMOTE_IGNORE_CONFIG = viper.GetStringSlice("remote-ignore-config")
	}
	applyRemoteConfig()
	var SYSTEM_DNS string
	if DNS_SERVER == "" || DNSSEC_DNS_SERVER == "" {
		sysDNS, err := sysresolv.NewSystemResolvers(nil, 53)
		if err != nil {
			slog.Warn("Cannot setup reslovers")

		}
		sysDNS.Refresh()
		SYSTEM_DNS = webtest.AddrPortsToCSV(sysDNS.Addrs())
	}
	if DNSSEC_DNS_SERVER == "" {
		DNSSEC_DNS_SERVER = SYSTEM_DNS
	}
	if DNS_SERVER == "" {
		DNS_SERVER = SYSTEM_DNS
	}
	slog.Info("SSRF protection initialized", "blockPrivateIPs", ssrf.Enabled())
}

func main() {
	for _, vaule := range os.Args {
		if vaule == "-v" || vaule == "--version" || vaule == "version" {
			fmt.Println("LEMON IPW TEST NODE GOLANG VERSION" + VERSION)
			fmt.Println("COMMIT" + COMMIT)
			fmt.Println("BUILD_TIME" + BUILD_TIME)
			return
		}

	}
	slog.Info("LEMON IPW TEST NODE GOLANG VERSION", "version", VERSION, "commit", COMMIT, "build_time", BUILD_TIME)
	readConfig()
	webtest.SetDNSServer(DNS_SERVER)
	webtest.SetDNSSecServer(DNSSEC_DNS_SERVER)
	initHTTPClients()
	// 注入出站 HTTP 客户端到 webtest（探针函数内部按版本取用）
	webtest.SetHTTPClient(V4Client, V6Client)
	if IPDB != "false" {
		ipdb.Init(GH_PROXY)
	}

	slog.Info("Starting server", "port", PORTS, "gh_proxy", GH_PROXY, "single_stack", SINGLE_STACK, "dns_server", DNS_SERVER, "CORS_ACCEPT", ACCEPT_DOMAINS)

	// WS 客户端：接入中间件 WS 通道（配置 WS_URL 时启用，HTTP 接口不变）
	if WS_URL != "" {
		go wsClientLoop()
		slog.Info("WS client enabled", "url", WS_URL, "nodeId", WS_NODE_ID)
	}

	r := gin.Default()
	corsConfig := cors.DefaultConfig()

	if len(ACCEPT_DOMAINS) > 0 {
		corsConfig.AllowOrigins = ACCEPT_DOMAINS
	} else {
		corsConfig.AllowAllOrigins = true
	}
	r.Use(cors.New(corsConfig))
	v1 := r.Group("/v1")
	v1.Use(cors.New(corsConfig)) // Apply CORS middleware to the v1 group
	if ACCESS_TOKEN != "" {
		v1.Use(tokenCheck()) // Apply token check middleware to the v1 group
	}
	{
		v1.GET("/detail/*url", checkWebsiteHandler)
		v1.GET("/ssl/*url", sslCheckHandler)
		v1.GET("/tcping/:ip", pingHandler)
		v1.GET("/dns/:type/*domain", dnsQueryHandler)
		v1.GET("/dnssec/:domain", dnssecHandler)
		v1.GET("/whois/:domain", whoisHandler)
		v1.GET("/speed/:version/*url", websiteSpeedTestHandler)

		if IPDB != "false" {
			v1.GET("/location/:ip", locateIP)
			v1.GET("/location", locateUserIP)
			v1.GET("/asn/:ip", asnLookupHandler)
		}
	}

	r.GET("/", healchCheck)

	if err := r.Run(":" + PORTS); err != nil {
		slog.Error("Server failed to start", "error", err)
	}
}
