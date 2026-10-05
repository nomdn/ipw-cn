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
	"os/signal"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/AdguardTeam/golibs/netutil/sysresolv"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
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
	// 响应体上限：detail / speed / ssl 只要一个字节数，没有理由把整个响应体读进内存
	applyResponseBodyLimit()
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
//
// 解码器选项必须收紧：klauspost/compress 的默认值是「窗口上限 512MB、解码上限 64GiB、并发 4」，
// 而窗口是按流里声明的值在解析帧头时分配的（早于任何输出，max-response-body 拦不住）。
// 详见 resources.go 的 zstdMaxWindow。
//
// 池与兜底分支共用 newZstdDecoder，避免两处选项漂移（兜底若用 zstd.NewReader 的默认选项，
// 就等于给恶意流留了一条绕过限制的路径）。
func newZstdDecoder() *zstd.Decoder {
	decoder, _ := zstd.NewReader(nil,
		zstd.WithDecoderConcurrency(1), // 流式场景够用，且不再为异步块解码额外分配
		zstd.WithDecoderMaxWindow(zstdMaxWindow),
	)
	return decoder
}

var zstdReaderPool = sync.Pool{
	New: func() interface{} {
		// 当池子空了，创建一个新的解码器
		return newZstdDecoder()
	},
}

func decompressZstd(r io.ReadCloser) (io.ReadCloser, error) {
	zr := zstdReaderPool.Get().(*zstd.Decoder)
	if err := zr.Reset(r); err != nil {
		zr.Close()
		// 池里的解码器可能已被关闭（Close 后不会再放回池），换一个新的重试一次
		zr = newZstdDecoder()
		if newErr := zr.Reset(r); newErr != nil {
			zr.Close()
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
	IPDB_ENABLED         bool // IP 数据库开关（ipdb / IPDB）：缺省开；关闭则不加载数据库、不注册 /v1/location 与 /v1/asn
	CORS                 string
	ACCEPT_DOMAINS       []string
	ACCESS_TOKEN         string
	REMOTE_CONFIG_URL    string
	REMOTE_IGNORE_CONFIG []string   // 不被远端配置覆盖的配置项列表（remote-ignore-config / REMOTE_IGNORE_CONFIG）
	WS_URL               string     // WS 客户端：中间件 WS 地址（ws://host:port/ws）
	WS_NODE_ID           string     // WS 客户端：节点 id（与中间件 ws-keys 键一致）
	WS_NODE_KEY          string     // WS 客户端：注册 key（与中间件 ws-keys[节点id] 一致，可空）
	REPORT_URL           string     // 数据上报：收集中心 HTTP 接口基址（report-url / REPORT_URL），WS 全部掉线时的兜底通道
	REPORT_TOKEN         string     // 数据上报：/report 鉴权 token（report-token / REPORT_TOKEN）
	REPORT_INTERVAL      int        // 数据上报间隔秒（report-interval-seconds / REPORT_INTERVAL_SECONDS），缺省 15
	TRUSTED_PROXIES      string     // 可信代理列表（trusted-proxies / TRUSTED_PROXIES），逗号分隔 IP/CIDR，供 c.IP() 判定
	NODE_OTA             bool       // OTA 升级开关（node-ota / NODE_OTA）：缺省允许收集中心下发；只读容器等不可自更新部署显式设 false，节点拒绝指令并回传原因
	MAX_RESPONSE_BODY    int64      // 单次拨测可读的响应体上限（字节，解压后；max-response-body / MAX_RESPONSE_BODY），<=0 不限制，详见 resources.go
	MEMORY_LIMIT         int64      // Go 运行时软内存上限（字节；memory-limit / MEMORY_LIMIT），<=0 表示自动探测 cgroup，详见 resources.go
	ACCESS_LOG           bool       // 访问日志开关（access-log / ACCESS_LOG）：缺省开；显式 false 后每个请求不再写一条 slog（panic 兜底与其余日志不受影响），可热更新
	fiberApp             *fiber.App // 显式持有的 Fiber 应用（收到退出信号时优雅停机需要先 Shutdown）
	VERSION              string
	COMMIT               string
	BUILD_TIME           string
)

// 缓存存活时长（所有 sync.Map 缓存共用）：
//   - 成功：30s —— 只用来吸收同一目标的并发重访与秒级重复点击，不追求长时间复用；
//   - 失败：5s —— 让业务方重试时能尽快重新探测，而不是把一次网络抖动固化住。
//
// 失败条目不再靠"后台 goroutine 睡 30s 再 Delete"提前失效：那等于给每次失败都留一个
// 待唤起的协程，而且失效前读到的仍是失败结果。改成写进条目里，读时按 failed 判 TTL。
const (
	cacheTTLSuccess = 30 * time.Second
	cacheTTLFailure = 5 * time.Second
)

// cacheTTL 按结果是否失败取对应 TTL
func cacheTTL(failed bool) time.Duration {
	if failed {
		return cacheTTLFailure
	}
	return cacheTTLSuccess
}

// tcpingFailed 判定单侧 tcping 结果是否失败。
// 两种失败形态：
//   - 解析 / SSRF 阶段出错：IP 字段被写成 "Error: ..."（webtest.TCPingRun 只在这种情况下返回 error）
//   - 拨号全部失败：IP 字段是正常 IP，但 Success 为 0
//
// 只认前者会让"端口被拒/超时"这类最常见的失败漏判成成功，照 30s 缓存。
func tcpingFailed(s *webtest.TCPingStats) bool {
	if s == nil {
		return true
	}
	return strings.HasPrefix(s.IP, "Error:") || s.Success == 0
}

// dualAnyFailed 双栈结果的整体失败判定：单栈模式下被跳过的另一侧必然不可达，
// 只能看实际探测的那一侧，否则成功结果也会被判失败；双栈模式任一侧失败即算失败。
func dualAnyFailed[T any](ipv4, ipv6 *T, failed func(*T) bool) bool {
	switch SINGLE_STACK {
	case "ipv4":
		return failed(ipv4)
	case "ipv6":
		return failed(ipv6)
	default:
		return failed(ipv4) || failed(ipv6)
	}
}

type websiteCacheEntry struct {
	result    *WebsiteCheckResult
	timestamp time.Time
	failed    bool
}

type sslCacheEntry struct {
	result    *SSLCheckResult
	timestamp time.Time
	failed    bool
}

type pingCacheEntry struct {
	result    *TCPingResult
	timestamp time.Time
	failed    bool
}

type speedCacheEntry struct {
	result    *webtest.WebsiteSpeedTestResult
	timestamp time.Time
	failed    bool
}

type whoisCacheEntry struct {
	result    *webtest.WhoisResult
	timestamp time.Time
	failed    bool
}

type asnWhoisCacheEntry struct {
	result    *webtest.ASNWhoisResult
	timestamp time.Time
	failed    bool
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

func checkWebsiteHandler(c fiber.Ctx) error {
	testUrl := c.Params("*")
	if testUrl == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "URL parameter is required",
		})
	}

	testUrl = normalizeURL(testUrl)

	parsedURL, err := url.Parse(testUrl)
	if err == nil && ssrf.HasLocalOrPrivateIP(parsedURL.Hostname()) {
		return c.Status(200).JSON(&WebsiteCheckResult{
			IPv4: fakePerfectWebsiteResult(testUrl),
			IPv6: fakePerfectWebsiteResult(testUrl),
		})
	}

	if cached, ok := websiteCache.Load(testUrl); ok {
		entry := cached.(websiteCacheEntry)
		if time.Since(entry.timestamp) < cacheTTL(entry.failed) {
			return c.Status(200).JSON(entry.result)
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

		failed := dualAnyFailed(result.IPv4, result.IPv6, func(d *webtest.WebsiteCheckDetail) bool {
			return d != nil && !d.IsReachable
		})
		websiteCache.Store(testUrl, websiteCacheEntry{result: result, timestamp: time.Now(), failed: failed})

		return result, nil
	})

	return c.Status(200).JSON(rawResult.(*WebsiteCheckResult))
}
func websiteSpeedTestHandler(c fiber.Ctx) error {
	testUrl := c.Params("*")
	version := c.Params("version")
	if testUrl == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "URL parameter is required",
		})
	}
	url := normalizeURL(testUrl)

	// 检查请求版本是否与 SINGLE_STACK 配置匹配
	switch SINGLE_STACK {
	case "ipv4":
		if version != "v4" {
			return c.Status(http.StatusBadRequest).JSON(&webtest.WebsiteSpeedTestResult{
				Version:    "v4",
				HostRecord: "Skipped due to SINGLE_STACK=ipv4",
			})
		}
	case "ipv6":
		if version != "v6" {
			return c.Status(http.StatusBadRequest).JSON(&webtest.WebsiteSpeedTestResult{
				Version:    "v6",
				HostRecord: "Skipped due to SINGLE_STACK=ipv6",
			})
		}
	}

	// 缓存键：URL + 版本
	cacheKey := fmt.Sprintf("%s:%s", url, version)

	// 检查缓存
	if cached, ok := speedCache.Load(cacheKey); ok {
		entry := cached.(speedCacheEntry)
		if time.Since(entry.timestamp) < cacheTTL(entry.failed) {
			return c.Status(200).JSON(entry.result)
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
				speedCache.Store(cacheKey, speedCacheEntry{result: errorResult, timestamp: time.Now(), failed: true})
				return errorResult, nil
			}
			speedCache.Store(cacheKey, speedCacheEntry{result: r, timestamp: time.Now()})
			return r, nil
		})
		result = rawResult.(*webtest.WebsiteSpeedTestResult)
	default:
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid version",
		})
	}

	return c.Status(200).JSON(result)
}

func sslCheckHandler(c fiber.Ctx) error {
	testUrl := c.Params("*")
	if testUrl == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "URL parameter is required",
		})
	}

	testUrl = normalizeURL(testUrl)

	parsedURL, err := url.Parse(testUrl)
	if err == nil && ssrf.HasLocalOrPrivateIP(parsedURL.Hostname()) {
		return c.Status(200).JSON(&SSLCheckResult{
			IPv4: fakeInvalidSSLResult(parsedURL.Hostname()),
			IPv6: fakeInvalidSSLResult(parsedURL.Hostname()),
		})
	}

	if cached, ok := sslCache.Load(testUrl); ok {
		entry := cached.(sslCacheEntry)
		if time.Since(entry.timestamp) < cacheTTL(entry.failed) {
			return c.Status(200).JSON(entry.result)
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

		failed := dualAnyFailed(result.IPv4, result.IPv6, func(d *webtest.SSLCheckDetail) bool {
			return d != nil && !d.IsReachable
		})
		sslCache.Store(testUrl, sslCacheEntry{result: result, timestamp: time.Now(), failed: failed})

		return result, nil
	})

	return c.Status(200).JSON(rawResult.(*SSLCheckResult))
}

func locateIP(c fiber.Ctx) error {
	ip := c.Params("ip")
	slog.Debug("Locating IP", "ip", ip)
	return c.Status(http.StatusOK).JSON(ipdb.SearchIP(ip))
}
func locateUserIP(c fiber.Ctx) error {
	ip := c.IP()
	// 可能会有误报，因为某些环境下 IP() 可能返回代理服务器的 IP 地址，而不是用户的真实 IP 地址
	slog.Debug("Locating user IP", "ip", ip)
	return c.Status(http.StatusOK).JSON(ipdb.SearchIP(ip))
}

// normalizeASN 把各数据源的 ASN 字符串统一成 "AS"+数字。
// 数据源格式不一：maxmind/dbip 的 MMDBASNResult.ASN 自带 "AS" 前缀（searchMMDBASN 拼的），
// ip2location 的 asn 是裸数字。含非数字字符时返回空串（视为无效）。
func normalizeASN(s string) string {
	s = strings.TrimSpace(strings.ToUpper(s))
	s = strings.TrimSpace(strings.TrimPrefix(s, "AS"))
	if s == "" {
		return ""
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return "AS" + s
}

// resolveASN 从 SearchIP 结果中按优先级提取可用 ASN（maxmind → dbip → ip2location），返回 "ASxxxx"。
//
// 之所以要兜底：此前 whois 段只认 maxmind_asn，某个 IP 一旦未命中该库，整段 ASN WHOIS 就凭空消失，
// 哪怕 dbip_asn / ip2location_asn 明明有数据。
func resolveASN(result map[string]interface{}) string {
	for _, key := range []string{"maxmind_asn", "dbip_asn"} {
		if r, ok := result[key].(*ipdb.MMDBASNResult); ok {
			if asn := normalizeASN(r.ASN); asn != "" {
				return asn
			}
		}
	}
	if m, ok := result["ip2location_asn"].(map[string]string); ok {
		if asn := normalizeASN(m["asn"]); asn != "" {
			return asn
		}
	}
	return ""
}

// asnWhoisFailed 判定 ASN WHOIS 结果是否算失败，决定缓存 TTL（失败 30s / 成功 5min）。
// 注意 likexian/whois 出错时也返回 (result, nil)，错误塞在 result.Error 里，不能只看 err 参数。
func asnWhoisFailed(r *webtest.ASNWhoisResult) bool {
	return r == nil || r.Error != "" || (r.ASName == "" && r.OrgName == "")
}

// lookupASNWhois 带缓存的 ASN WHOIS 查询，handler 与 ws 两条路径共用。
// 缓存键取归一化后的 ASN（"AS13335"）：历史上是 "AS"+已带前缀的 ASN，实际拼成了 "ASAS13335"。
func lookupASNWhois(asn string) (*webtest.ASNWhoisResult, bool) {
	asn = normalizeASN(asn)
	if asn == "" {
		return nil, false
	}

	if cached, ok := asnWhoisCache.Load(asn); ok {
		if entry, ok := cached.(asnWhoisCacheEntry); ok && entry.result != nil {
			if time.Since(entry.timestamp) < cacheTTL(entry.failed) {
				return entry.result, true
			}
		}
		asnWhoisCache.Delete(asn)
	}

	whoisData, err := webtest.QueryASNWhois(asn)
	if err != nil {
		return nil, false
	}
	asnWhoisCache.Store(asn, asnWhoisCacheEntry{
		result:    whoisData,
		timestamp: time.Now(),
		failed:    asnWhoisFailed(whoisData),
	})
	return whoisData, true
}

func asnLookupHandler(c fiber.Ctx) error {
	ip := c.Params("ip")
	if ip == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "IP parameter is required",
		})
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

	// 使用 WHOIS 对 ASN 进行进一步解析（maxmind 未命中时回落到 dbip / ip2location）
	if asn := resolveASN(result); asn != "" {
		if whoisData, ok := lookupASNWhois(asn); ok {
			asnResult["whois"] = whoisData
		}
	}

	return c.Status(http.StatusOK).JSON(asnResult)
}

func whoisHandler(c fiber.Ctx) error {
	domain := c.Params("domain")
	if domain == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Domain parameter is required",
		})
	}

	if cached, ok := whoisCache.Load(domain); ok {
		entry := cached.(whoisCacheEntry)
		if time.Since(entry.timestamp) < cacheTTL(entry.failed) {
			return c.Status(200).JSON(entry.result)
		}
		whoisCache.Delete(domain)
	}

	result, err := webtest.QueryWhois(domain)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	whoisCache.Store(domain, whoisCacheEntry{result: result, timestamp: time.Now()})
	return c.Status(http.StatusOK).JSON(result)
}

func dnssecHandler(c fiber.Ctx) error {
	domain := c.Params("domain")
	if domain == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Domain parameter is required",
		})
	}

	result, err := webtest.ResolveDNSSEC(domain)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(result)
}

func dnsQueryHandler(c fiber.Ctx) error {

	domain := c.Params("*")
	parsedURL, err := parseURL(domain)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid domain",
		})
	}
	domain = parsedURL.Host
	recodeType := c.Params("type")
	switch recodeType {
	case "a":
		result, err := webtest.ResolveARecord(domain)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(http.StatusOK).JSON(result)
	case "aaaa":
		result, err := webtest.ResolveAAAARecord(domain)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(http.StatusOK).JSON(result)
	case "cname":
		result, err := webtest.ResolveCNAMERecord(domain)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(http.StatusOK).JSON(result)
	case "mx":
		result, err := webtest.ResolveMXRecord(domain)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(http.StatusOK).JSON(result)
	case "ns":
		result, err := webtest.ResolveNSRecord(domain)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(http.StatusOK).JSON(result)
	case "ptr":
		result, err := webtest.ResolvePTRRecord(domain)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(http.StatusOK).JSON(result)
	case "srv":
		result, err := webtest.ResolveSRVRecord(domain)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(http.StatusOK).JSON(result)
	case "txt":
		result, err := webtest.ResolveTXTRecord(domain)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(http.StatusOK).JSON(result)
	case "caa":
		result, err := webtest.ResolveCAARecord(domain)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(http.StatusOK).JSON(result)
	default:
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid record type",
		})
	}
}
func pingHandler(c fiber.Ctx) error {
	host := c.Params("ip")
	port := c.Query("port")
	if host == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "IP or hostname parameter is required",
		})
	}
	if port == "" {
		port = "80"
	}
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid port number",
		})
	}
	// 拨号用归一化后的端口：Atoi 会放行 "+80"/"0080"，原样拼接会导致拨号必败
	port = strconv.Itoa(portNum)

	count := 4
	if countStr := c.Query("count"); countStr != "" {
		n, err := strconv.Atoi(countStr)
		if err != nil || n < 1 || n > 20 {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "count must be an integer between 1 and 20",
			})
		}
		count = n
	}

	cacheKey := fmt.Sprintf("%s:%s:%d", host, port, count)
	if cached, ok := pingCache.Load(cacheKey); ok {
		entry := cached.(pingCacheEntry)
		if time.Since(entry.timestamp) < cacheTTL(entry.failed) {
			return c.Status(200).JSON(entry.result)
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

		// tcping 的口径与 detail/ssl 不同：只有双栈都失败才算整体失败，
		// 单侧失败仍返回了可用的延迟数据，按成功计 TTL
		ipv4Failed := tcpingFailed(result.IPv4)
		ipv6Failed := tcpingFailed(result.IPv6)
		pingCache.Store(cacheKey, pingCacheEntry{result: result, timestamp: time.Now(), failed: ipv4Failed && ipv6Failed})

		return result, nil
	})

	return c.Status(200).JSON(rawResult.(*TCPingResult))
}

// healchCheck 健康检查（GET /）。只回答"进程是否活着"，不回报版本与能力
// —— 版本号与能力清单走 GET /info（见 nodeInfoHandler）。
//
// 分开的理由：健康检查是免鉴权的对外端点，被负载均衡、OTA 就绪探测、运维 curl 打，
// 混进版本号等于向公网公开"这个节点跑的是哪个版本"，也让一个只该回答存活的端点
// 承担了与本意无关的职责。
func healchCheck(c fiber.Ctx) error {
	return c.Status(http.StatusOK).JSON(fiber.Map{"status": "ok"})
}

// nodeInfoHandler 节点信息（GET /info）：节点版本号 + 能力清单。
//
// 注册在 /v1 组之外，所以与健康检查一样免鉴权，也不计入拨测统计
// （收集中心对纯 HTTP 版节点探活后取回，见 ipw-boce nodeHealth.go）。
// 用途：节点状态页展示版本；下发 config 等管理指令前判断节点能否理解对应指令。
// 能力清单与 WS register 报文同源（见 ws.go nodeCapabilities）。
func nodeInfoHandler(c fiber.Ctx) error {
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"version":      VERSION,
		"capabilities": nodeCapabilities(),
	})
}

// ==================== 中间件（鉴权 / 上报 / 访问日志 / panic 兜底） ====================

// tokenCheck /v1 业务接口的 Bearer 鉴权。access-token 未配置时不由路由挂载
// （见 bizGet 与 registerConfigRoutes），等于整个管理面关闭。
func tokenCheck() fiber.Handler {
	return func(c fiber.Ctx) error {
		token := c.Get("Authorization")
		if token != "Bearer "+ACCESS_TOKEN {
			// 直接 return 即终止链，后续中间件与 handler 都不再执行（等价 gin 的 Abort）
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}
		return c.Next()
	}
}

// nodeBusinessAPITypes 计入拨测统计的业务接口（与下方路由表一致）。
//
// 用途是兜底：fiber 的 Group/Use 按**路径前缀**匹配，中间件一旦挂到大范围的 router 上，
// /v1/config、/v1/ota 这类管理请求也会被算进来。这里再按 apiType 白名单过一道，
// 保证统计口径只覆盖真正的拨测业务（口径与收集中心 store.go 的 isProbeType 白名单对齐）。
var nodeBusinessAPITypes = map[string]bool{
	"detail":   true,
	"ssl":      true,
	"tcping":   true,
	"dns":      true,
	"dnssec":   true,
	"whois":    true,
	"speed":    true,
	"location": true,
	"asn":      true,
}

// nodeReportMiddleware 数据上报统计中间件：归属规则过滤 → 计数 → 拨测类补明细。
// 只挂在业务路由上（逐条挂载，见 bizGet），管理面路由不经过它。
func nodeReportMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		// 归属判定所需的信息必须在 c.Next() **之前**取完：
		// fiber 的 c.Params / c.Get / c.Request().URI() 都是对请求缓冲的零拷贝引用，
		// handler 执行期间这份缓冲会被复用，事后再读会拿到错乱内容（statistics 里的 raw 会串）。
		// Query 经 string() 转换天然拷贝，Params 侧由 nodeAPIFromPath 内部 Clone。
		apiType, raw := nodeAPIFromPath(c)
		query := string(c.Request().URI().QueryString())
		// 收集中心(ipw-boce)主动调度下发的拨测带 X-Scheduler-Probe 头：
		// 该次执行由收集中心侧本地落库(source=sched/biz)，本节点跳过计数与明细上报，避免双算。
		// 真实业务请求（用户/中间件转发）不带此头，仍按"节点是唯一记录者"正常上报。
		schedulerProbe := c.Get("X-Scheduler-Probe") != ""

		err := c.Next()

		if schedulerProbe || apiType == "" || !nodeBusinessAPITypes[apiType] {
			return err
		}
		latency := time.Since(start)
		status := c.Response().StatusCode()
		nodeRecordAPI(apiType, status, latency)
		if nodeIsProbeType(apiType) {
			nodeRecordProbe(nodeProbeRec{
				NodeID: nodeReportNodeID(), APIType: apiType, Raw: raw,
				Query: query, Status: status, LatencyMs: latency.Milliseconds(),
				Source: "http", CreatedAt: time.Now().Unix(),
			})
		}
		return err
	}
}

// accessLogMiddleware 访问日志中间件：每个请求结束后输出一条 slog 记录（method/path/status/延迟/客户端 IP/错误）。
//
// 对应 gin.Default() 里那个 Logger。这里用 fiber 官方的 logger 中间件承接，但把输出改投 slog：
// 节点其余日志都走 slog（默认落 stderr），访问日志并进同一路输出，journalctl / 日志采集侧才是统一的；
// fiber 内置格式还会按终端能力加 ANSI 颜色，重定向到文件时是转义乱码。
//
// 开关（access-log / ACCESS_LOG，缺省开）：**在请求路径上实时判定**，而不是在启动时决定挂不挂，
// 于是 PATCH /v1/config 或远端下发改完即生效、无需重启。关掉后连 logger 的计时与字段拼接都不做
// （直接 c.Next() 放行）——访问日志是每请求一次同步写，正是压测里把节点卡住的头号嫌疑，
// 关掉才能拿到真实的业务吞吐。
//
// 注意：它只管访问日志。panic 兜底（recoverMiddleware）与业务/启动日志都不受影响，
// 关掉访问日志不会让节点变成"出事无迹可查"。
func accessLogMiddleware() fiber.Handler {
	logHandler := logger.New(logger.Config{
		// 保留默认模板：它同时决定中间件是否开启 ${latency} 计时（fiber logger.New 里按模板里
		// 有没有 ${latency} 判断）。真正的输出由下面的 LoggerFunc 接管，不走模板渲染。
		Format: logger.DefaultFormat,
		LoggerFunc: func(c fiber.Ctx, data *logger.Data, _ *logger.Config) error {
			latency := data.Stop.Sub(data.Start)
			// 计时未开启（Start/Stop 为零值）时兜底成 0，避免打印出一个天文数字的耗时。
			if data.Start.IsZero() || data.Stop.IsZero() || latency < 0 {
				latency = 0
			}
			attrs := []any{
				"method", c.Method(),
				"path", c.OriginalURL(), // 带查询串，与 gin Logger 的口径一致
				"status", c.Response().StatusCode(),
				"latency_ms", latency.Milliseconds(),
				"ip", c.IP(), // 受 trusted-proxies 影响，见 buildFiberConfig
			}
			// 业务 handler 返回的错误（fiber 会在链走完后交由 ErrorHandler 出 500），补进日志便于定位。
			if data.ChainErr != nil {
				attrs = append(attrs, "error", data.ChainErr.Error())
			}
			slog.Info("http", attrs...)
			return nil
		},
	})

	// 关掉时整条中间件短路，不进 logger（见上方注释）。
	return func(c fiber.Ctx) error {
		if !ACCESS_LOG {
			return c.Next()
		}
		return logHandler(c)
	}
}

// recoverMiddleware 兜住 handler 里的 panic：记一条带栈的日志，把响应收敛成 500，而不是让整个进程死掉。
//
// 必要性来自 fasthttp：它明确不做 panic 恢复（server.go 注释原话"任何 panic 会掀翻整个服务器"），
// fiber.New() 也不带恢复。gin.Default() 是自带 Recovery 的，丢了它就等于把"单请求 panic"
// 升级成"节点进程被杀"——对 1c1g 的小机器尤其致命。
// 栈与 panic 值都走 slog（对齐 gin.Recovery 会打印栈的行为），响应体只给固定的 500 文案，不外泄内部信息。
func recoverMiddleware() fiber.Handler {
	return recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c fiber.Ctx, e any) {
			slog.Error("panic recovered",
				"panic", fmt.Sprint(e),
				"method", c.Method(),
				"path", c.OriginalURL(),
				"stack", string(debug.Stack()),
			)
		},
		PanicHandler: func(fiber.Ctx, any) error {
			// 只回固定文案：默认实现会把 panic 详情塞进响应体（内部实现细节不该给到调用方）。
			return fiber.ErrInternalServerError
		},
	})
}

// bizGet 注册一条业务路由，处理链为：tokenCheck（配了 access-token 时）+ 上报统计 + 业务 handler。
//
// 逐条挂载而不是挂在 /v1 组上：fiber 的 Group 中间件按路径前缀匹配，挂在 /v1 上会一并
// 作用到 /v1/config、/v1/ota —— 配置/OTA 是管理请求，不该计入拨测统计
// （原 gin 的 v1.Use 只对组内注册的路由生效，这里保持同样的作用域）。
//
// fiber v3 的路由签名是 Get(path string, handler any, handlers ...any)：变参展开只能占一个位置，
// 所以这里显式取首元素、其余展开。
func bizGet(router fiber.Router, path string, h fiber.Handler) {
	handlers := make([]any, 0, 3)
	if ACCESS_TOKEN != "" {
		handlers = append(handlers, tokenCheck())
	}
	handlers = append(handlers, nodeReportMiddleware(), h)
	router.Get(path, handlers[0], handlers[1:]...)
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

// literalString 把配置源里的**任意字面量**归一成字符串 —— 两条读取路径唯一的转换表：
//
//	启动：setting.json（经 viper）/ 环境变量   → configString / configBool / configInt / configBytes
//	运行：远端下发 / PATCH 的 map[string]any  → configValue → applyConfigMap
//
// 两条路径共用本函数，所以同一个值无论写成 JSON 布尔 `false`、数字 `0` 还是字符串 `"false"`，
// 读出来**完全一致**。旧实现里启动路径靠 viper 的隐式 cast、运行路径靠自己的 type switch，
// 两套规则只是"碰巧"一致 —— 只要有一边不认布尔，`ipdb: false` 就会被静默当成"开"。
//
//	nil      → ""（键不存在或显式 null，一律视为"未配置"）
//	string   → 去首尾空白
//	bool     → "true" / "false"
//	float64  → 十进制整数形式（JSON 数字统一解析成 float64，443.0 要还原成 "443"）
//	其它      → fmt.Sprint 兜底（数组 / 对象只可能出现在非字符串类键上）
func literalString(v any) string {
	switch s := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(s)
	case bool:
		return strconv.FormatBool(s)
	case float64:
		return fmt.Sprintf("%.0f", s)
	case int:
		return strconv.Itoa(s)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

// configValue 返回配置 map 中指定 key 的非空字符串值；
// key 不存在或值为空时返回 ""（表示不覆盖本地配置）。
func configValue(CONFIG map[string]any, key string) string {
	return literalString(CONFIG[key])
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
	// 键映射统一在 applyConfigMap（与本地 PATCH / WS 指令共用，见 config_api.go）；
	// 忽略名单由 remoteIgnoreList() 给出：受保护凭据（access-token / report-token）
	// + 操作员自选的 remote-ignore-config —— 凭据始终保持"ENV > setting.json"的本地优先级
	applied, unknown, _ := applyConfigMap(CONFIG, remoteIgnoreList())
	if CORS != "" {
		ACCEPT_DOMAINS = splitAndTrim(CORS, ",")
	}
	slog.Info("Remote config applied", "url", url, "applied", applied, "unknown", unknown)
	if protected := protectedKeysIn(CONFIG); len(protected) > 0 {
		slog.Warn("远端配置里的凭据键已被忽略（凭据由节点本地管理，见 configRemoteProtectedKeys）",
			"keys", protected, "url", url)
	}
}

// ==================== 启动配置读取（唯一入口） ====================
//
// 全项目「从配置文件 / 环境变量读取启动配置」的动作都收敛在本函数里，取值口径统一为
// 「环境变量 > setting.json > 默认值」。**其它文件不得直接调 os.Getenv / viper 读配置**：
// resources.go 只负责"拿到值之后怎么用"（解析字节后缀、应用到运行时）。
//
// 与运行时热更新别混淆（两件事、两套键映射，刻意不合并）：
//   - 本函数 + 下面几个 config* 取值器 = 进程启动时"读配置文件"，只在 main 里跑一次；
//   - applyConfigMap（config_api.go） = 运行中"改内存变量"，来源是 PATCH / 远端托管配置，
//     它不是读配置文件；需重启键、热生效键都在那边定义（见 configRestartKeys）。

// configString 读取字符串配置：环境变量 > setting.json > def。返回值两侧空白已去除。
// setting.json 一侧经 literalString 归一，因此布尔 / 数字字面量（`false` / `0` / `443`）
// 与字符串写法完全等价（见 literalString）。
func configString(envKey, cfgKey, def string) string {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v
	}
	if v := literalString(viper.Get(cfgKey)); v != "" {
		return v
	}
	return def
}

// configInt 读取整数配置：环境变量 > setting.json > def。解析失败时告警并用 def。
func configInt(envKey, cfgKey string, def int) int {
	raw := configString(envKey, cfgKey, "")
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		slog.Warn("Invalid integer config, falling back to default", "key", cfgKey, "value", raw, "default", def)
		return def
	}
	return n
}

// configBytes 读取"字节数"类配置：环境变量 > setting.json > def。
// 支持裸数字与 K/M/G 后缀（见 resources.go 的 parseByteSize）。解析失败时告警并用 def
// —— 不静默把非法值当成 0：0 在 max-response-body 里意味着"不限制"，正是要防的那种误配。
func configBytes(envKey, cfgKey string, def int64) int64 {
	raw := configString(envKey, cfgKey, "")
	if raw == "" {
		return def
	}
	n, ok := parseByteSize(raw)
	if !ok {
		slog.Warn("Invalid byte size config, falling back to default", "key", cfgKey, "value", raw, "default", def)
		return def
	}
	return n
}

// configStringSlice 读取字符串列表配置：环境变量（JSON 数组字符串）> setting.json（数组）。
// env 写了但非法时返回 nil 并告警，**不回退**到 setting.json —— 两个来源语义打架时静默改用
// 另一个，比直接用空更难排查。
func configStringSlice(envKey, cfgKey string) []string {
	raw := strings.TrimSpace(os.Getenv(envKey))
	if raw == "" {
		return viper.GetStringSlice(cfgKey)
	}
	var list []string
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		slog.Warn("Invalid string-list config, ignored", "key", envKey, "value", raw, "error", err)
		return nil
	}
	return list
}

// parseBoolSwitch 开关类字面值的**唯一**解析表：四个开关（ipdb / block-private-ips /
// node-ota / access-log）在启动读取、远端下发、运行时 PATCH 三条路径上都走这里。
//
// 统一之前的四种口径（同一个 "no" 在 node-ota 上是关、在另外三个上静默当开，运维侧推不出来）：
//
//	ipdb              IPDB != "false"        —— 大小写敏感，连 "False" 都算开
//	block-private-ips 仅 "false"/"0" 算关     —— "no"/"off" 静默当开
//	access-log        仅 "false"/"0" 算关     —— 同上
//	node-ota          true/1/yes/on + false/0/no/off —— 唯一认 yes/no 的
//
// 现在四条共用一张表（忽略大小写与首尾空白）：
//
//	真：true / 1 / yes / on / enable / enabled
//	假：false / 0 / no / off / disable / disabled
//
// 入参是**已归一的字符串**：启动路径由 configString 给出、运行路径由 configValue 给出，
// 两者都先过 literalString —— 所以 setting.json 里写布尔 `false` 到这里就是 "false"，
// 与手写字符串 "false" 走完全相同的判定，不需要本函数再判类型。
//
// 空串（未配置）与不认识的串一律返回 known=false —— 本函数只做"字面值 → 布尔"的翻译，
// 不替调用方决定默认值。由调用方处置：启动阶段告警并落默认值，PATCH / 远端下发计入 unknown
// 并保留原值 —— 关键点是**绝不静默把拼错的值当成"开"**，那正是旧口径最难查的地方。
func parseBoolSwitch(raw string) (val, known bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes", "on", "enable", "enabled":
		return true, true
	case "false", "0", "no", "off", "disable", "disabled":
		return false, true
	}
	return false, false
}

// configBool 读取开关类配置（取值口径 env > setting.json > def），raw 经 configString 归一
// （因此 JSON 布尔 / 数字字面量同样识别），字面值交 parseBoolSwitch。
// 值不认识的告警并落 def —— 启动阶段没有"回报给请求方"的通道，只能告警。
// 运行中的 PATCH 不走这里，走 applyConfigMap 里的 parseBoolSwitch 以便计入 unknown。
func configBool(envKey, cfgKey string, def bool) bool {
	raw := configString(envKey, cfgKey, "")
	if strings.TrimSpace(raw) == "" {
		return def
	}
	val, known := parseBoolSwitch(raw)
	if !known {
		slog.Warn("Invalid boolean config, falling back to default",
			"key", cfgKey, "value", raw, "default", def)
		return def
	}
	return val
}

// readConfig 启动配置的唯一读取入口（取值口径与职责边界见上方注释块）。
func readConfig() {
	viper.SetConfigName("setting")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		slog.Warn("Failed to read config file, using defaults", "error", err)
	}

	// —— 基础项 ——
	PORTS = configString("PORTS", "port", "8080")
	GH_PROXY = configString("GH_PROXY", "gh-proxy", "")
	CORS = configString("CORS", "cors", "")
	// IP 数据库开关：缺省开启（首次启动自动下载约 450MB）。
	// 与另外三个开关同一口径（configBool），不再用 `IPDB != "false"` 那种大小写敏感的字符串比较
	// —— 旧写法下 `ipdb: "False"` / `"no"` 都会被当成"开"，同一个值跟 node-ota 的判定还相反。
	IPDB_ENABLED = configBool("IPDB", "ipdb", true)
	TRUSTED_PROXIES = configString("TRUSTED_PROXIES", "trusted-proxies", "")
	ACCESS_TOKEN = configString("ACCESS_TOKEN", "access-token", "")

	// SINGLE_STACK 取 "ipv4" / "ipv6" / 空。**空是合法值**（双栈），不能被默认值顶掉。
	// 若节点机器是单栈网络，设成对应协议可跳过另一协议的测试，少一堆无谓的错误日志与延迟。
	SINGLE_STACK = strings.ToLower(configString("SINGLE_STACK", "single-stack", ""))

	// DNS：留空则由本函数末尾用系统解析器兜底（见下方 SYSTEM_DNS）
	DNS_SERVER = configString("DNS_SERVER", "dns-server", "")
	DNSSEC_DNS_SERVER = configString("DNSSEC_DNS_SERVER", "dnssec-server", "")

	// SSRF 防护开关：缺省开启（见 ssrf 包）。
	// 注意必须在 viper.ReadInConfig 之后读：早期版本只读 ENV，setting.json 里的同名键在启动阶段
	// 被静默忽略（只有运行中 patch / 远端配置才生效），与其余键的"env > setting.json"口径不一致。
	ssrf.SetEnabled(configBool("BLOCK_PRIVATE_IPS", "block-private-ips", true))
	// OTA 升级开关：缺省允许收集中心下发 OTA（见 ota.go）
	NODE_OTA = configBool("NODE_OTA", "node-ota", true)

	// 访问日志开关：缺省开启（见 accessLogMiddleware）。
	// 判定在请求路径上做，故运行中热改即时生效。
	ACCESS_LOG = configBool("ACCESS_LOG", "access-log", true)

	// —— 远端配置源 ——
	REMOTE_CONFIG_URL = configString("REMOTE_CONFIG_URL", "remote-config-url", "")
	// 不被远端覆盖的配置项列表（env 用 JSON 数组字符串，setting.json 里是数组）
	REMOTE_IGNORE_CONFIG = configStringSlice("REMOTE_IGNORE_CONFIG", "remote-ignore-config")

	// —— WS 客户端（接入中间件 WS 通道）——
	WS_URL = configString("WS_URL", "ws-url", "")
	WS_NODE_ID = configString("NODE_ID", "node-id", "")
	WS_NODE_KEY = configString("NODE_KEY", "node-key", "")

	// —— 数据上报（report-url / report-token / report-interval-seconds，详见 report.go）——
	REPORT_URL = configString("REPORT_URL", "report-url", "")
	REPORT_TOKEN = configString("REPORT_TOKEN", "report-token", "")
	REPORT_INTERVAL = configInt("REPORT_INTERVAL_SECONDS", "report-interval-seconds", 15)
	if REPORT_INTERVAL <= 0 {
		REPORT_INTERVAL = 15 // 零/负间隔无意义，clamp 回默认
	}

	// —— 资源上限（防单点打爆小内存机器，详见 resources.go）——
	MAX_RESPONSE_BODY = configBytes("MAX_RESPONSE_BODY", "max-response-body", defaultMaxResponseBody)
	MEMORY_LIMIT = configBytes("MEMORY_LIMIT", "memory-limit", 0)

	if CORS != "" {
		ACCEPT_DOMAINS = splitAndTrim(CORS, ",")
	}

	applyRemoteConfig()
	// 软内存上限要在远端配置之后应用（远端可能下发 memory-limit）。
	// 响应体上限不在这里应用 —— 出站客户端还没创建，交 initHTTPClients（main 里紧随其后）。
	applyMemoryLimit()
	var SYSTEM_DNS string
	if DNS_SERVER == "" || DNSSEC_DNS_SERVER == "" {
		sysDNS, err := sysresolv.NewSystemResolvers(nil, 53)
		if err != nil {
			// 失败时 sysDNS 为 nil，绝不能再调用 Refresh()，否则启动即 panic；
			// SYSTEM_DNS 留空，交给下游的内置默认 DNS 兜底
			slog.Warn("Cannot setup system resolvers, DNS_SERVER will fall back to webtest defaults", "error", err)
		} else {
			sysDNS.Refresh()
			SYSTEM_DNS = webtest.AddrPortsToCSV(sysDNS.Addrs())
		}
	}
	if DNSSEC_DNS_SERVER == "" {
		DNSSEC_DNS_SERVER = SYSTEM_DNS
	}
	if DNS_SERVER == "" {
		DNS_SERVER = SYSTEM_DNS
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

// sweepExpiredCaches 按条目时间戳清扫各 sync.Map 缓存。
// 条目最长 TTL 为 30s（见 cacheTTLSuccess），清扫窗口取 2 分钟已足够宽裕：
// 只要保证"过期条目不会因为没人再访问同一 key 而永久留在内存里"即可。
func sweepExpiredCaches() {
	cutoff := time.Now().Add(-2 * time.Minute)
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
			case asnWhoisCacheEntry:
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
	sweep(&asnWhoisCache)
	ipdb.SweepBilibiliCache()
}

// buildFiberConfig 组装 Fiber 应用配置。
//
// 代理信任策略：**未配置 trusted-proxies / TRUSTED_PROXIES 时等同信任所有来源**（塞
// 0.0.0.0/0 + ::/0），即 X-Forwarded-For 一律生效 —— 口径对齐 gin 的
// SetTrustedProxies(["0.0.0.0/0"])，也让节点放在 CDN/反代后开箱即可拿到真实客户端 IP。
// 配置了可信段则只在来源落在该段内时读代理头，收窄信任面。
//
// 注意这意味着 XFF 的解释权在调用方：直连节点（不经 CDN）的请求可以伪造 XFF 改写
// /v1/location 的归属地与访问日志里的 ip。要防这点就显式配置 trusted-proxies。
func buildFiberConfig() fiber.Config {
	cfg := fiber.Config{
		// EnableIPValidation 必须开。它只在"信任代理、需要读 ProxyHeader"时生效：
		// fiber 不开它时 c.IP() 会原样返回 ProxyHeader 的值（见 req.go extractIPFromHeader 首个分支），
		// 于是多跳 CDN 拿到整条链 "客户端, 代理1, 代理2" 而不是单个 IP，非法值也原样透传。
		// 开了才走"自右向左跳过可信代理、取第一个非可信 IP"的解析，取不到时退回对端地址
		// —— 正是 gin c.ClientIP() 的口径，/v1/location 与访问日志的 ip 都依赖它。
		EnableIPValidation: true,
	}

	// TRUSTED_PROXIES / trusted-proxies：逗号分隔的可信代理（IP/CIDR），如 "10.0.0.0/8,172.16.0.1"。
	// 非法条目告警后丢弃（fiber 自身是静默忽略，这里补一条可排查的日志），保留合法项。
	trustedProxies := make([]string, 0, 4)
	for _, p := range splitAndTrim(TRUSTED_PROXIES, ",") {
		if net.ParseIP(p) != nil {
			trustedProxies = append(trustedProxies, p)
			continue
		}
		if _, _, err := net.ParseCIDR(p); err != nil {
			slog.Warn("Invalid TRUSTED_PROXIES entry, ignored", "entry", p)
			continue
		}
		trustedProxies = append(trustedProxies, p)
	}

	// 一个有效条目都不剩（未配置，或全被丢弃）：退回"信任所有来源"。
	if len(trustedProxies) == 0 {
		trustedProxies = []string{"0.0.0.0/0", "::/0"}
	}

	// 三个字段要一起设：IsProxyTrusted 先看 TrustProxy，再看对端地址是否落在可信段内，
	// 两者都满足才会去读 ProxyHeader（见 req.go IsProxyTrusted / IP）。
	// 配了可信段时，来自非可信地址的请求仍会被忽略 XFF、退回对端地址。
	cfg.ProxyHeader = "X-Forwarded-For"
	cfg.TrustProxy = true
	cfg.TrustProxyConfig = fiber.TrustProxyConfig{Proxies: trustedProxies}
	return cfg
}

// newFiberApp 组装 Fiber 应用：全局中间件（访问日志 / panic 兜底 / CORS）+ 全部路由。
//
// 整个 HTTP 层的"接线"集中在这里，与 buildFiberConfig 相邻；main 只负责调用，
// 不与配置读取、拨测依赖、后台任务那些启动步骤混写。
func newFiberApp() *fiber.App {
	app := fiber.New(buildFiberConfig())

	// 访问日志 + panic 兜底：对应 gin.Default() 的 Logger / Recovery（fiber.New() 都不带）。
	// 挂载顺序与 gin.Default() 一致（在最外层），后续所有请求都经过它们。
	// logger 在外、recover 在内：panic 被 recover 转成错误后仍会流经 logger，
	// 于是"崩掉的那次请求"也会留下一条带 error 字段的访问日志。
	app.Use(accessLogMiddleware())
	app.Use(recoverMiddleware())

	// CORS：全局挂载（等价原来的 r.Use(cors.New(...))），业务路由与 /v1/config、/v1/ota 一并生效。
	// ACCEPT_DOMAINS 为空则允许所有来源（等价 gin 的 AllowAllOrigins）。
	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           12 * 3600,
	}
	if len(ACCEPT_DOMAINS) > 0 {
		corsConfig.AllowOrigins = ACCEPT_DOMAINS
	} else {
		corsConfig.AllowOrigins = []string{"*"}
	}
	app.Use(cors.New(corsConfig))

	// 业务路由：鉴权与上报中间件逐条挂载（见 bizGet 的说明），
	// 这样 /v1/config、/v1/ota 不会被 v1 组的中间件带上。
	// 路径写法对应 gin：`*url` / `*domain` 在 fiber 里统一是无名通配符 `*`，取值用 c.Params("*")。
	v1 := app.Group("/v1")
	{
		bizGet(v1, "/detail/*", checkWebsiteHandler)
		bizGet(v1, "/ssl/*", sslCheckHandler)
		bizGet(v1, "/tcping/:ip", pingHandler)
		bizGet(v1, "/dns/:type/*", dnsQueryHandler)
		bizGet(v1, "/dnssec/:domain", dnssecHandler)
		bizGet(v1, "/whois/:domain", whoisHandler)
		bizGet(v1, "/speed/:version/*", websiteSpeedTestHandler)

		if IPDB_ENABLED {
			bizGet(v1, "/location/:ip", locateIP)
			bizGet(v1, "/location", locateUserIP)
			bizGet(v1, "/asn/:ip", asnLookupHandler)
		}
	}

	// 运行时配置接口（/v1/config）：鉴权同业务接口，但不计入拨测统计
	registerConfigRoutes(app)

	// OTA 升级接口（POST /v1/ota）：收集中心 HTTP 回退通道，鉴权语义同 config（见 ota.go）
	registerOTARoute(app)

	app.Get("/", healchCheck)
	// 节点信息（版本 + 能力清单）：与健康检查分开，免鉴权、不计入统计（见 nodeInfoHandler）
	app.Get("/info", nodeInfoHandler)

	return app
}

// startHTTPServer 启动 HTTP 服务：构建 Fiber 应用（见 newFiberApp）后在独立 goroutine 中监听端口。
//
// 显式持有 net.Listener：收到退出信号时先 Shutdown 释放端口，避免默认 Listen 的隐式关闭不可控。
// fiber 的 Shutdown 会让 Listener 正常返回（err == nil），故监听 goroutine 只需判非 nil 错误。
func startHTTPServer() error {
	fiberApp = newFiberApp()
	ln, err := net.Listen("tcp", ":"+PORTS)
	if err != nil {
		return err
	}
	go func() {
		if err := fiberApp.Listener(ln); err != nil {
			slog.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()
	return nil
}

func main() {
	// ---- 版本查询：-v / --version / version 之一，打印后直接退出，不启动服务 ----
	for _, value := range os.Args {
		if value == "-v" || value == "--version" || value == "version" {
			fmt.Println("LEMON IPW TEST NODE GOLANG VERSION" + VERSION)
			fmt.Println("COMMIT" + COMMIT)
			fmt.Println("BUILD_TIME" + BUILD_TIME)
			return
		}
	}
	slog.Info("LEMON IPW TEST NODE GOLANG VERSION", "version", VERSION, "commit", COMMIT, "build_time", BUILD_TIME)

	// ---- 配置与拨测依赖 ----
	readConfig()
	webtest.SetDNSServer(DNS_SERVER)
	webtest.SetDNSSecServer(DNSSEC_DNS_SERVER)
	initHTTPClients()
	// 注入出站 HTTP 客户端到 webtest（探针函数内部按版本取用）
	webtest.SetHTTPClient(V4Client, V6Client)
	if IPDB_ENABLED {
		ipdb.Init(GH_PROXY)
	}

	slog.Info("Starting server", "port", PORTS, "gh_proxy", GH_PROXY, "single_stack", SINGLE_STACK, "dns_server", DNS_SERVER, "CORS_ACCEPT", ACCEPT_DOMAINS)

	// ---- 常驻后台任务：WS 通道 / 数据上报 / 缓存清扫 ----

	// WS 客户端：接入中间件 WS 通道（HTTP 接口不变）。
	// 控制器常驻——即使当前 ws-url 为空也启动，否则运行中把 ws-url 热更新进来时没有协程去接管；
	// 改 ws-url 由 reconcileWSClient 多退少补（无需重启），改 node-id / node-key 需重启进程。
	startWSClientController()

	// 数据上报：WS 在线走 WS 广播，否则 HTTP POST 收集中心 /report（详见 report.go）
	startNodeReporter()

	// 缓存清扫：各 sync.Map 只在同 key 重访时惰性淘汰过期条目，
	// 公网端点被唯一 key 洪打时内存会无限增长，这里定期清扫。
	// 间隔取 1 分钟 —— 条目 TTL 已缩到 30s 量级，清扫跟着收紧，
	// 过期条目最多多留一个 tick 就被清掉。
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			sweepExpiredCaches()
		}
	}()

	// ---- HTTP 服务（Fiber）：中间件与路由都收在 newFiberApp 里，这里只负责启动 ----
	if err := startHTTPServer(); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}

	// ---- 阻塞至退出信号，随后优雅停机 ----
	// main 必须阻塞，否则 Server goroutine 随 main 返回而消亡，进程启动后立即退出。
	// 阻塞在退出信号上，收到后走优雅停机。
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	gracefulShutdown()
}

// gracefulShutdown 优雅停止 HTTP 服务：停止接收新请求，等待在途请求完成（上限 30s）。
// 供收到退出信号时调用，确保监听端口先释放再退出进程。
func gracefulShutdown() {
	if fiberApp == nil {
		return
	}
	slog.Info("Graceful shutdown started, waiting for in-flight requests")
	if err := fiberApp.ShutdownWithTimeout(30 * time.Second); err != nil {
		slog.Warn("Graceful shutdown timed out, forcing close", "error", err)
	}
}
