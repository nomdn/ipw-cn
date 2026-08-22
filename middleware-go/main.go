package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

// ==================== 全局变量 ====================

// apiInfo 对应 setting.json 节点池中的 { label, id, url } 配置项
type apiInfo struct {
	Label string `json:"label"`
	ID    string `json:"id"`
	URL   string `json:"url"`
	WS    wsFlag `json:"ws"` // 是否经 WS 通道通信（缺省 false = HTTP）；支持 "ws": true 或 "ws": "true"
}

// wsFlag 兼容 JSON 布尔与字符串两种写法
type wsFlag bool

func (w *wsFlag) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	s = strings.Trim(s, `"`)
	switch s {
	case "true", "1", "yes":
		*w = true
	case "false", "0", "", "no":
		*w = false
	default:
		return fmt.Errorf("invalid ws flag %q", s)
	}
	return nil
}

// UseWS 判断该节点是否走 WS 通道
func (a *apiInfo) UseWS() bool { return a != nil && bool(a.WS) }

// stackConfig 对应节点池中的 { DualStack, IPv4, IPv6 } 三栈
type stackConfig struct {
	DualStack []apiInfo `json:"DualStack"`
	IPv4      []apiInfo `json:"IPv4"`
	IPv6      []apiInfo `json:"IPv6"`
}

// middlewareConfig 仅用于解析 setting.json（JSON 键名统一用连接线，如 http-timeout-seconds）
// 节点池结构：APIBaseURL 含 {IPv6,IPv4,DualStack} 三栈；IPLocationAPI 为纯数组（无栈区分）。
// 原 apiBaseUrls/whois·ssl·detail、NSLookup/dns·dnssec、TCPing、SpeedTest 统一并入 APIBaseURL，
// 原 IPLocationAPI/location·asn 并入 IPLocationAPI。
type middlewareConfig struct {
	Port            stringOrNumber    `json:"port"`
	HTTPTimeoutSeconds int            `json:"http-timeout-seconds"`
	RateLimit       *int              `json:"rate-limit"` // 单 IP 每分钟限流次数；缺省=默认120，0=不限流
	WSPort          stringOrNumber   `json:"ws-port"`    // WS 端口：兼容 "8092" 与 8092；缺省=8092，"0"/0=关闭 WS 通道
	RemoteConfigURL string            `json:"remote-config-url"`
	RemoteIngoreConfig []string       `json:"remote-ingore-config"` // 不被远端覆盖的配置项列表
	Cors            string            `json:"cors"`
	APIBaseURL      stackConfig       `json:"APIBaseURL"` // whois/ssl/detail/dns/dnssec/tcping/speed 上游节点池
	IPLocationAPI   []apiInfo         `json:"IPLocationAPI"` // location/asn 上游节点池（纯数组，无栈）
	APIKeys         map[string]string `json:"apiKeys"` // HTTP 转发鉴权：backendID → token（注入 Authorization: Bearer）
	WSKeys          map[string]string `json:"wsKeys"`  // WS 注册校验：nodeID → key（节点 register 的 key 须匹配）
}

// stringOrNumber 兼容 JSON 中的字符串与数字（如 "8080" 或 8080）
type stringOrNumber string

// 全局配置变量，由 readConfig 从 setting.json 填充
var (
	VERSION          string            // 版本号（构建时由 ldflags 注入）
	COMMIT           string            // 提交哈希（构建时由 ldflags 注入）
	BUILD_TIME       string            // 构建时间（构建时由 ldflags 注入）
	PORT             string            // 监听端口
	HTTP_TIMEOUT     int               // 上游请求超时（秒）
	CORS             string            // CORS 配置原始字符串（逗号分隔的允许域名，可空）
	ACCEPT_DOMAINS   []string          // 允许跨域的域名列表（由 CORS 拆分而来，空则允许所有）
	API_BASE_URLS    stackConfig       // whois/ssl/detail/dns/dnssec/tcping/speed 上游节点池（DualStack/IPv4/IPv6）
	IP_LOCATION_APIS []apiInfo         // location/asn 上游节点池（纯数组，无栈）
	API_KEYS         map[string]string // backendID → HTTP 转发 token（注入 Authorization: Bearer）
	WS_KEYS          map[string]string // backendID → WS 注册校验 key
	CONFIG_SOURCE    string            // 配置文件路径
	RATE_LIMIT       int               // 单 IP 每分钟限流次数（0 表示不限流），默认 120
	WS_PORT          int               // WS 服务端口（0 = 关闭），缺省 8092，由 readConfig 统一解析
	REMOTE_INGORE_CONFIG []string      // 不被远端覆盖的配置项列表（remote-ingore-config / REMOTE_INGORE_CONFIG）
)

// HTTP_CLIENT 用于转发上游请求（等价于 TS 中的 $fetch），超时由 HTTP_TIMEOUT 决定
// HTTP_CLIENT forwards requests to upstream backends (equivalent to $fetch in the TS version).
var HTTP_CLIENT = &http.Client{}

// wsSrv 为全局 WS 服务端（WS_PORT>0 时启动；nil = 未启用）。见 ws.go
var wsSrv *wsServer

// ==================== 工具函数 ====================

// envJSON 从环境变量读取 JSON 字符串并反序列化到 target（用于数组/对象等映射表类配置项）。
// 环境变量为空时不做任何事；解析失败返回错误，避免静默使用空配置。
func envJSON(key string, target any) error {
	raw := os.Getenv(key)
	if raw == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		return fmt.Errorf("parse env %s: %w", key, err)
	}
	return nil
}

func (s *stringOrNumber) UnmarshalJSON(data []byte) error {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch t := v.(type) {
	case string:
		*s = stringOrNumber(t)
	case float64:
		*s = stringOrNumber(strconv.FormatFloat(t, 'f', -1, 64))
	default:
		return fmt.Errorf("cannot unmarshal %T as string-or-number", v)
	}
	return nil
}

// badRequest 返回 400 错误，响应格式与 h3 createError 一致
func badRequest(c fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"statusCode":    fiber.StatusBadRequest,
		"statusMessage": msg,
	})
}

// nonEmpty 等价于 TS 中的 .filter(Boolean)
func nonEmpty(parts []string) []string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// findURL 等价于 TS 中的 new Map(apiBaseUrls.map(...)).get(backendID)
func findURL(list []apiInfo, id string) string {
	for _, api := range list {
		if api.ID == id {
			return api.URL
		}
	}
	return ""
}

// findNode 按 id 在节点列表中查找（返回指针；找不到返回 nil）
func findNode(list []apiInfo, id string) *apiInfo {
	for i := range list {
		if list[i].ID == id {
			return &list[i]
		}
	}
	return nil
}

// flattenStack 将节点池三栈展开为一个节点列表（DualStack + IPv4 + IPv6）
func flattenStack(s stackConfig) []apiInfo {
	out := make([]apiInfo, 0, len(s.DualStack)+len(s.IPv4)+len(s.IPv6))
	out = append(out, s.DualStack...)
	out = append(out, s.IPv4...)
	out = append(out, s.IPv6...)
	return out
}

// stackNonEmpty 判断节点池三栈是否至少有一项
func stackNonEmpty(s stackConfig) bool {
	return len(s.DualStack) > 0 || len(s.IPv4) > 0 || len(s.IPv6) > 0
}

func keysOf(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// fetchURL 发起上游 GET 请求，返回上游原始状态码与响应 body。
// 仅网络层错误（连接失败/超时/读失败）返回 error；上游任何 HTTP 状态码都原样透传，不包装成 500。
func fetchURL(target string, headers map[string]string, query url.Values) (int, []byte, error) {
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return 0, nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := HTTP_CLIENT.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, body, nil
}

// forwardUpstream 转发上游结果：网络层错误返回 502，上游任意状态码与 body 原样透传
func forwardUpstream(c fiber.Ctx, apiBaseUrl string, target string, headers map[string]string, query url.Values) error {
	status, data, err := fetchURL(target, headers, query)
	if err != nil {
		log.Printf("Error fetching from %s: %v", apiBaseUrl, err)
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"statusCode":    fiber.StatusBadGateway,
			"statusMessage": "Backend unreachable",
		})
	}
	return c.Status(status).Type("json").Send(data)
}

// lookupAPIKey 查找 backendID 对应的 token。
// 优先级: setting.json apiKeys > 环境变量 APIKEYS (JSON 字符串)
func lookupAPIKey(backendID string) string {
	var apiKeys map[string]string
	if len(API_KEYS) > 0 {
		apiKeys = API_KEYS
	} else if raw := os.Getenv("APIKEYS"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &apiKeys); err != nil {
			return ""
		}
	} else {
		return ""
	}

	apiKey := apiKeys[backendID]

	return apiKey
}

// lookupWSKey 查找 nodeID 对应的 WS 注册校验 key。
// 优先级: setting.json wsKeys > 环境变量 WSKEYS (JSON 字符串)
// 与 lookupAPIKey 相互独立：apiKeys 用于 HTTP 转发鉴权，wsKeys 用于 WS 注册校验。
func lookupWSKey(nodeID string) string {
	var wsKeys map[string]string
	if len(WS_KEYS) > 0 {
		wsKeys = WS_KEYS
	} else if raw := os.Getenv("WSKEYS"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &wsKeys); err != nil {
			return ""
		}
	} else {
		return ""
	}

	wsKey := wsKeys[nodeID]

	return wsKey
}

// ==================== 业务接口 ====================

// middlewareHandler 对应前端 server/routes/middleware/[...slug].get.ts
// 路径格式: /{prefix}/{backendID}/{apiType}/{raw...}
func middlewareHandler(c fiber.Ctx) error {
	// fiber 默认不解码路径参数，而 h3 的 params 是解码后的，这里手动对齐
	slugString := c.Params("*")
	if decoded, err := url.PathUnescape(slugString); err == nil {
		slugString = decoded
	}
	if slugString == "" {
		return badRequest(c, "Missing slug parameter")
	}

	// 将路径参数转为数组
	slug := strings.Split(slugString, "/")

	// 验证 slug 数组的长度是否在允许范围内
	if len(slug) == 0 {
		return badRequest(c, "Invalid slug")
	}

	// 如果分段超过4个，可能是 raw 部分包含协议（如 https://example.com），
	// 按 / 分割后会产生额外分段，需要重新拼回
	if len(slug) > 4 {
		backendID := slug[0]
		apiType := slug[1]
		protocol := slug[2]
		rest := strings.Join(nonEmpty(slug[3:]), "/")
		if protocol == "https:" || protocol == "http:" {
			slug = []string{backendID, apiType, protocol + "//" + rest}
		} else {
			return badRequest(c, "Invalid slug")
		}
	}

	// 分割参数
	backendID := slug[0]
	apiType := slug[1]
	raw := strings.Join(slug[2:], "/")
	if backendID == "" || apiType == "" || raw == "" {
		return badRequest(c, "Missing parameters in slug")
	}

	query := c.Queries()

	// 从 API_KEYS 查找 token，未配置时回退到环境变量 APIKEYS (JSON 字符串)
	apiKey := lookupAPIKey(backendID)

	// 构造转发给上游的请求头：不透传客户端 Origin，避免上游 CORS 误判
	authHeaders := map[string]string{}
	if apiKey != "" {
		authHeaders["Authorization"] = "Bearer " + apiKey
	}

	switch apiType {
	case "whois", "dns", "location", "ssl", "asn", "dnssec", "detail":
		// 节点池结构：location/asn 走 IPLocationAPI（纯数组，无栈），其余走 APIBaseURL（三栈平铺）
		var apiBaseUrls []apiInfo
		switch apiType {
		case "location", "asn":
			apiBaseUrls = IP_LOCATION_APIS
		default:
			apiBaseUrls = flattenStack(API_BASE_URLS)
		}

		node := findNode(apiBaseUrls, backendID)
		if node == nil || (!node.UseWS() && node.URL == "") {
			return badRequest(c, "Invalid backend ID")
		}
		// ws:true 节点：拨测请求经 WS 通道转发（数据上传走 WS），失败返回 502
		if node.UseWS() && wsSrv != nil {
			status, body, err := wsSrv.RequestProbe(backendID, apiType, raw, nil, wsProbeTimeout())
			if err != nil {
				return c.Status(fiber.StatusBadGateway).SendString("WS probe failed: " + err.Error())
			}
			// 节点上报的 body 为 JSON，显式指定类型（fiber 默认 text/plain）
			return c.Status(status).Type("json").Send(body)
		}
		apiBaseUrl := node.URL
		if !strings.HasSuffix(apiBaseUrl, "/") {
			apiBaseUrl += "/"
		}

		// 上游错误原样透传（状态码 + body），网络错误返回 502
		return forwardUpstream(c, apiBaseUrl, apiBaseUrl+"v1/"+apiType+"/"+raw, authHeaders, nil)

	case "tcping", "udping", "speed":
		// tcping/udping/speed 统一走 APIBaseURL 节点池
		apiBaseUrls := flattenStack(API_BASE_URLS)

		// 转发 query 参数（过滤空值，等价于 TS 中 URLSearchParams 过滤 undefined）
		queryString := url.Values{}
		queryMap := make(map[string]string)
		for k, v := range query {
			if v != "" {
				queryString.Set(k, v)
				queryMap[k] = v
			}
		}

		node := findNode(apiBaseUrls, backendID)
		if node == nil || (!node.UseWS() && node.URL == "") {
			return badRequest(c, "Invalid backend ID")
		}
		// ws:true 节点：拨测请求经 WS 通道转发
		if node.UseWS() && wsSrv != nil {
			status, body, err := wsSrv.RequestProbe(backendID, apiType, raw, queryMap, wsProbeTimeout())
			if err != nil {
				return c.Status(fiber.StatusBadGateway).SendString("WS probe failed: " + err.Error())
			}
			// 节点上报的 body 为 JSON，显式指定类型（fiber 默认 text/plain）
			return c.Status(status).Type("json").Send(body)
		}
		apiBaseUrl := node.URL
		if !strings.HasSuffix(apiBaseUrl, "/") {
			apiBaseUrl += "/"
		}

		// 上游错误原样透传（状态码 + body），网络错误返回 502
		return forwardUpstream(c, apiBaseUrl, apiBaseUrl+"v1/"+apiType+"/"+raw, authHeaders, queryString)

	default:
		return badRequest(c, "Invalid API type")
	}
}

// ==================== readConfig ====================

// viperValue 从 viper 读取配置值并反序列化到 target（数组/对象经 JSON 中转，支持自定义类型如 wsFlag）
func viperValue(key string, target any) error {
	v := viper.Get(key)
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}

// fetchRemoteConfig 从远端 URL 拉取配置内容（middleware setting.json 格式）
func fetchRemoteConfig(url string) ([]byte, error) {	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote config returned status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// applyRemoteConfig 应用远端配置，优先级：远端 > 环境变量 > setting.json。
// 远端地址来源：环境变量 REMOTE_CONFIG_URL 优先，其次 setting.json 的 remote-config-url。
// 远端字段为空时不覆盖本地值；拉取或解析失败时打印警告并回退本地配置。
func applyRemoteConfig(mw *middlewareConfig) error {
	url := os.Getenv("REMOTE_CONFIG_URL")
	if url == "" {
		url = mw.RemoteConfigURL
	}
	if url == "" {
		return nil
	}
	body, err := fetchRemoteConfig(url)
	if err != nil {
		log.Printf("[middleware] WARN failed to fetch remote config, fallback to local: %v", err)
		return nil
	}
	var remote middlewareConfig
	if err := json.Unmarshal(body, &remote); err != nil {
		log.Printf("[middleware] WARN invalid remote config JSON, fallback to local: %v", err)
		return nil
	}
	// ignore 列表：数组中的配置项不被远端覆盖（逐键判断跳过）
	ignored := func(key string) bool {
		for _, k := range mw.RemoteIngoreConfig {
			if k == key {
				return true
			}
		}
		return false
	}
	if !ignored("port") && string(remote.Port) != "" {
		mw.Port = remote.Port
	}
	if !ignored("http-timeout-seconds") && remote.HTTPTimeoutSeconds != 0 {
		mw.HTTPTimeoutSeconds = remote.HTTPTimeoutSeconds
	}
	if !ignored("rate-limit") && remote.RateLimit != nil {
		mw.RateLimit = remote.RateLimit
	}
	if !ignored("ws-port") && remote.WSPort != "" {
		mw.WSPort = remote.WSPort
	}
	if !ignored("remote-config-url") && remote.RemoteConfigURL != "" {
		mw.RemoteConfigURL = remote.RemoteConfigURL
	}
	if !ignored("cors") && remote.Cors != "" {
		mw.Cors = remote.Cors
	}
	// 节点池结构：APIBaseURL 与 IPLocationAPI（远端非空栈时覆盖本地）
	if !ignored("APIBaseURL") && stackNonEmpty(remote.APIBaseURL) {
		mw.APIBaseURL = remote.APIBaseURL
	}
	if !ignored("IPLocationAPI") && len(remote.IPLocationAPI) > 0 {
		mw.IPLocationAPI = remote.IPLocationAPI
	}
	// apiKeys / wsKeys 强制忽略：密钥凭据不随远端配置覆盖（与后端 access-token 一致），只从本地 setting.json / env 读取
	log.Printf("[middleware] remote config applied from %s", url)
	return nil
}

// readConfig 从 setting.json 读取配置并写入全局配置变量。
// 查找顺序: 环境变量 SETTING_FILE > ./setting.json > ../setting.json
func readConfig() error {
	// 统一用 viper 读取 setting.json（SETTING_FILE 指定路径，否则查找 ./ 与 ../）
	path := os.Getenv("SETTING_FILE")
	viper.SetConfigType("json")
	if path != "" {
		viper.SetConfigFile(path)
	} else {
		viper.SetConfigName("setting")
		viper.AddConfigPath(".")
		viper.AddConfigPath("../")
	}
	if err := viper.ReadInConfig(); err != nil {
		if path == "" {
			return fmt.Errorf("setting.json not found (set SETTING_FILE to specify its path)")
		}
		return err
	}

	var mw middlewareConfig

	// 统一读取模式：先 Getenv，再 viper.GetString（env 优先 > setting.json > 默认值）
	if v := os.Getenv("PORT"); v != "" {
		mw.Port = stringOrNumber(v)
	} else {
		mw.Port = stringOrNumber(viper.GetString("port"))
	}
	if v := os.Getenv("HTTP_TIMEOUT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("parse env HTTP_TIMEOUT: %w", err)
		}
		mw.HTTPTimeoutSeconds = n
	} else {
		mw.HTTPTimeoutSeconds = viper.GetInt("http-timeout-seconds")
	}
	if v := os.Getenv("CORS"); v != "" {
		mw.Cors = v
	} else {
		mw.Cors = viper.GetString("cors")
	}
	if v := os.Getenv("REMOTE_CONFIG_URL"); v != "" {
		mw.RemoteConfigURL = v
	} else {
		mw.RemoteConfigURL = viper.GetString("remote-config-url")
	}
	// rateLimit（*int：env > setting(IsSet) > 默认120；0 = 不限流）
	if v := os.Getenv("RATE_LIMIT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("parse env RATE_LIMIT: %w", err)
		}
		mw.RateLimit = &n
	} else if viper.IsSet("rate-limit") {
		n := viper.GetInt("rate-limit")
		mw.RateLimit = &n
	}
	// ws-port（stringOrNumber：env > setting(IsSet) > 默认8092；"0"/0 = 关闭 WS 通道）
	if v := os.Getenv("WS_PORT"); v != "" {
		mw.WSPort = stringOrNumber(v)
	} else if viper.IsSet("ws-port") {
		mw.WSPort = stringOrNumber(viper.GetString("ws-port"))
	}
	// remote-ingore-config：不被远端覆盖的配置项列表（env JSON 数组 > setting.json 数组）
	if raw := os.Getenv("REMOTE_INGORE_CONFIG"); raw != "" {
		var list []string
		if err := json.Unmarshal([]byte(raw), &list); err != nil {
			return fmt.Errorf("parse env REMOTE_INGORE_CONFIG: %w", err)
		}
		mw.RemoteIngoreConfig = list
	} else {
		mw.RemoteIngoreConfig = viper.GetStringSlice("remote-ingore-config")
	}

	// 节点池结构：APIBaseURL 与 IPLocationAPI（env JSON > setting.json）
	if err := envJSON("API_BASE_URLS", &mw.APIBaseURL); err != nil {
		return err
	}
	if !stackNonEmpty(mw.APIBaseURL) && viper.Get("APIBaseURL") != nil {
		if err := viperValue("APIBaseURL", &mw.APIBaseURL); err != nil {
			return fmt.Errorf("parse APIBaseURL: %w", err)
		}
	}
	if err := envJSON("IP_LOCATION_APIS", &mw.IPLocationAPI); err != nil {
		return err
	}
	if len(mw.IPLocationAPI) == 0 && viper.Get("IPLocationAPI") != nil {
		if err := viperValue("IPLocationAPI", &mw.IPLocationAPI); err != nil {
			return fmt.Errorf("parse IPLocationAPI: %w", err)
		}
	}
	if err := envJSON("APIKEYS", &mw.APIKeys); err != nil {
		return err
	}
	if len(mw.APIKeys) == 0 {
		if err := viperValue("apiKeys", &mw.APIKeys); err != nil {
			return fmt.Errorf("parse apiKeys: %w", err)
		}
	}
	// wsKeys：WS 注册校验表（nodeID → key），优先级 env WSKEYS > setting.json wsKeys，与 apiKeys 相互独立
	if err := envJSON("WSKEYS", &mw.WSKeys); err != nil {
		return err
	}
	if len(mw.WSKeys) == 0 {
		if err := viperValue("wsKeys", &mw.WSKeys); err != nil {
			return fmt.Errorf("parse wsKeys: %w", err)
		}
	}

	// 远端配置（最高优先级）：REMOTE_CONFIG_URL 拉取后覆盖（优先级：远端 > 环境变量 > setting.json）
	if err := applyRemoteConfig(&mw); err != nil {
		return err
	}

	// 校验配置存在（防止在错误的文件上静默空跑）
	if !stackNonEmpty(mw.APIBaseURL) && len(mw.IPLocationAPI) == 0 {
		return fmt.Errorf("invalid config in %s: missing endpoint pools (expected APIBaseURL / IPLocationAPI node pools)", path)
	}

	// 将配置内容写入全局变量
	PORT = string(mw.Port)
	HTTP_TIMEOUT = mw.HTTPTimeoutSeconds
	// rateLimit：缺省默认 120 次/分钟；显式 0 表示不限流
	if mw.RateLimit == nil {
		RATE_LIMIT = 120
	} else {
		RATE_LIMIT = *mw.RateLimit
	}
	// ws-port：缺省默认 8092；显式 "0"/0 表示关闭 WS 通道（兼容字符串与数字写法）
	if mw.WSPort == "" {
		WS_PORT = 8092
	} else {
		n, err := strconv.Atoi(string(mw.WSPort))
		if err != nil {
			return fmt.Errorf("parse ws-port %q: %w", mw.WSPort, err)
		}
		WS_PORT = n
	}
	REMOTE_INGORE_CONFIG = mw.RemoteIngoreConfig
	CORS = mw.Cors
	// 逗号分隔的 CORS 域名列表（对齐根 main.go 的 ACCEPT_DOMAINS 用法）
	if CORS != "" {
		ACCEPT_DOMAINS = strings.Split(CORS, ",")
	}
	API_BASE_URLS = mw.APIBaseURL
	IP_LOCATION_APIS = mw.IPLocationAPI
	API_KEYS = mw.APIKeys
	WS_KEYS = mw.WSKeys
	CONFIG_SOURCE = viper.ConfigFileUsed()
	return nil
}

// ==================== main ====================

func main() {
	// -v / --version / version 参数打印版本信息（对齐根 main.go 的行为）
	for _, arg := range os.Args {
		if arg == "-v" || arg == "--version" || arg == "version" {
			fmt.Println("MIDDLEWARE-GO VERSION " + VERSION)
			fmt.Println("COMMIT " + COMMIT)
			fmt.Println("BUILD_TIME " + BUILD_TIME)
			return
		}
	}
	log.Printf("[middleware] version=%s commit=%s build_time=%s", VERSION, COMMIT, BUILD_TIME)

	if err := readConfig(); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	log.Printf("[middleware] config loaded from %s", CONFIG_SOURCE)

	timeout := HTTP_TIMEOUT
	if timeout <= 0 {
		timeout = 30
	}
	HTTP_CLIENT.Timeout = time.Duration(timeout) * time.Second

	// WS 服务端（端口由配置 wsPort / 环境变量 WS_PORT 决定，缺省 8092，0 = 关闭；统一走 readConfig）。
	// 后端节点作为 WS 客户端连入；节点配置 "ws": true 时拨测请求改走 WS 通道。现有 HTTP 路由不受影响。
	if WS_PORT > 0 {
		wsSrv = newWSServer()
		go wsSrv.Start(fmt.Sprintf(":%d", WS_PORT))
		go wsSrv.maintenanceLoop()
		log.Printf("[ws] ws channel enabled on port %d (node flag \"ws\": true to use)", WS_PORT)
	} else {
		log.Printf("[ws] ws channel disabled (wsPort=0)")
	}

	app := fiber.New()

	// 单 IP 限流（fiber 内置中间件，默认按客户端 IP 计数）：次数由配置 rateLimit / 环境变量 RATE_LIMIT 决定（默认 120 次/分钟），
	// 0 表示不限流。必须放在所有路由注册之前，fiber 中间件只对注册在其后的路由生效。
	if RATE_LIMIT > 0 {
		app.Use(limiter.New(limiter.Config{
			Max:        RATE_LIMIT,
			Expiration: time.Minute,
		}))
	}

	app.Get("/", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "ok",
		})
	})

	// 中间件路由，对应前端 server/routes/middleware/[...slug].get.ts
	// 同时兼容 /v1/... 与 /middleware/... 两种前缀
	// CORS: 配置了 cors 域名列表则仅允许这些来源，否则允许所有（对齐根 main.go 的 ACCEPT_DOMAINS 逻辑）
	corsConfig := cors.Config{
		AllowMethods: []string{"GET", "POST", "HEAD", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		MaxAge:       86400,
	}
	if len(ACCEPT_DOMAINS) > 0 {
		corsConfig.AllowOrigins = ACCEPT_DOMAINS
	} else {
		corsConfig.AllowOrigins = []string{"*"}
	}
	app.Use(cors.New(corsConfig))

	app.Get("/v1/*", middlewareHandler)
	app.Get("/middleware/*", middlewareHandler)

	// 监听地址，优先级: 环境变量 PORT > setting.json port > 默认 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = PORT
	}
	if port == "" {
		port = "8080"
	}
	listenAddr := port
	if !strings.Contains(listenAddr, ":") {
		listenAddr = ":" + listenAddr
	}
	if err := app.Listen(listenAddr); err != nil {
		log.Fatal(err)
	}
}
