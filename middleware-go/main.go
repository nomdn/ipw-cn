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

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

// ==================== 全局变量 ====================

// apiInfo 对应 setting.json apiBaseUrls 等数组中的 { label, id, url } 配置项
type apiInfo struct {
	Label string `json:"label"`
	ID    string `json:"id"`
	URL   string `json:"url"`
}

// stackConfig 对应 setting.json TCPing / SpeedTest
type stackConfig struct {
	DualStack []apiInfo `json:"DualStack"`
	IPv4      []apiInfo `json:"IPv4"`
	IPv6      []apiInfo `json:"IPv6"`
}

// middlewareConfig 仅用于解析 setting.json（键名与前端 config/index.ts 保持一致）
type middlewareConfig struct {
	Port               stringOrNumber    `json:"port"`
	HTTPTimeoutSeconds int               `json:"httpTimeoutSeconds"`
	Cors               string            `json:"cors"`
	APIBaseURLs        []apiInfo         `json:"apiBaseUrls"`
	IPLocationAPIs     []apiInfo         `json:"IPLocationAPIs"`
	TCPing             stackConfig       `json:"TCPing"`
	SpeedTest          stackConfig       `json:"SpeedTest"`
	NSLookup           []apiInfo         `json:"NSLookup"`
	APIKeys            map[string]string `json:"apiKeys"`
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
	API_BASE_URLS    []apiInfo         // whois/ssl/detail 上游列表
	IP_LOCATION_APIS []apiInfo         // location/asn 上游列表
	TCPING           stackConfig       // tcping 上游列表（DualStack/IPv4/IPv6）
	SPEED_TEST       stackConfig       // speed 上游列表（DualStack/IPv4/IPv6）
	NS_LOOKUP        []apiInfo         // dns/dnssec 上游列表
	API_KEYS         map[string]string // backendID → token
	CONFIG_SOURCE    string            // 配置文件路径
)

// HTTP_CLIENT 用于转发上游请求（等价于 TS 中的 $fetch），超时由 HTTP_TIMEOUT 决定
// HTTP_CLIENT forwards requests to upstream backends (equivalent to $fetch in the TS version).
var HTTP_CLIENT = &http.Client{}

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

	// 构造转发给上游的请求头：所有接口统一透传客户端请求的 Origin，没有 Origin 头则不传
	authHeaders := map[string]string{}
	if origin := c.Get("Origin"); origin != "" {
		authHeaders["Origin"] = origin
	}
	if apiKey != "" {
		authHeaders["Authorization"] = "Bearer " + apiKey
	}

	switch apiType {
	case "whois", "dns", "location", "ssl", "asn", "dnssec", "detail":
		var apiBaseUrls []apiInfo
		switch apiType {
		case "whois", "ssl", "detail":
			apiBaseUrls = API_BASE_URLS
		case "dns", "dnssec":
			apiBaseUrls = NS_LOOKUP
		case "location", "asn":
			apiBaseUrls = IP_LOCATION_APIS
		}

		apiBaseUrl := findURL(apiBaseUrls, backendID)
		if apiBaseUrl == "" {
			return badRequest(c, "Invalid backend ID")
		}
		if !strings.HasSuffix(apiBaseUrl, "/") {
			apiBaseUrl += "/"
		}

		// 上游错误原样透传（状态码 + body），网络错误返回 502
		return forwardUpstream(c, apiBaseUrl, apiBaseUrl+"v1/"+apiType+"/"+raw, authHeaders, nil)

	case "tcping", "udping", "speed":
		var apiTypeConfig stackConfig
		switch apiType {
		case "tcping":
			apiTypeConfig = TCPING
		case "speed":
			apiTypeConfig = SPEED_TEST
		default:
			return badRequest(c, "Invalid API type")
		}

		apiBaseUrls := make([]apiInfo, 0, len(apiTypeConfig.DualStack)+len(apiTypeConfig.IPv4)+len(apiTypeConfig.IPv6))
		apiBaseUrls = append(apiBaseUrls, apiTypeConfig.DualStack...)
		apiBaseUrls = append(apiBaseUrls, apiTypeConfig.IPv4...)
		apiBaseUrls = append(apiBaseUrls, apiTypeConfig.IPv6...)

		apiBaseUrl := findURL(apiBaseUrls, backendID)
		if apiBaseUrl == "" {
			return badRequest(c, "Invalid backend ID")
		}
		if !strings.HasSuffix(apiBaseUrl, "/") {
			apiBaseUrl += "/"
		}

		// 转发 query 参数（过滤空值，等价于 TS 中 URLSearchParams 过滤 undefined）
		queryString := url.Values{}
		for k, v := range query {
			if v != "" {
				queryString.Set(k, v)
			}
		}

		// 上游错误原样透传（状态码 + body），网络错误返回 502
		return forwardUpstream(c, apiBaseUrl, apiBaseUrl+"v1/"+apiType+"/"+raw, authHeaders, queryString)

	default:
		return badRequest(c, "Invalid API type")
	}
}

// ==================== readConfig ====================

// readConfig 从 setting.json 读取配置并写入全局配置变量。
// 查找顺序: 环境变量 SETTING_FILE > ./setting.json > ../setting.json
func readConfig() error {
	path := os.Getenv("SETTING_FILE")
	if path == "" {
		for _, candidate := range []string{"setting.json", "../setting.json"} {
			if _, err := os.Stat(candidate); err == nil {
				path = candidate
				break
			}
		}
	}
	if path == "" {
		return fmt.Errorf("setting.json not found (set SETTING_FILE to specify its path)")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var mw middlewareConfig
	if err := json.Unmarshal(data, &mw); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	// 环境变量覆盖配置（优先级: 环境变量 > setting.json > 默认值）：
	// 标量直接读；映射表（数组/对象）从环境变量读 JSON 字符串解析。
	// 部署时无需改动配置文件，用环境变量即可覆盖任意节点/密钥/端口等。
	if v := os.Getenv("PORT"); v != "" {
		mw.Port = stringOrNumber(v)
	}
	if v := os.Getenv("HTTP_TIMEOUT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("parse env HTTP_TIMEOUT: %w", err)
		}
		mw.HTTPTimeoutSeconds = n
	}
	if v := os.Getenv("CORS"); v != "" {
		mw.Cors = v
	}
	if err := envJSON("API_BASE_URLS", &mw.APIBaseURLs); err != nil {
		return err
	}
	if err := envJSON("IP_LOCATION_APIS", &mw.IPLocationAPIs); err != nil {
		return err
	}
	if err := envJSON("TCPING", &mw.TCPing); err != nil {
		return err
	}
	if err := envJSON("SPEED_TEST", &mw.SpeedTest); err != nil {
		return err
	}
	if err := envJSON("NS_LOOKUP", &mw.NSLookup); err != nil {
		return err
	}
	if err := envJSON("APIKEYS", &mw.APIKeys); err != nil {
		return err
	}

	// 校验配置存在（防止在错误的文件上静默空跑）
	hasEndpoints := len(mw.APIBaseURLs) > 0 || len(mw.NSLookup) > 0 || len(mw.IPLocationAPIs) > 0 ||
		len(mw.TCPing.DualStack) > 0 || len(mw.TCPing.IPv4) > 0 || len(mw.TCPing.IPv6) > 0 ||
		len(mw.SpeedTest.DualStack) > 0 || len(mw.SpeedTest.IPv4) > 0 || len(mw.SpeedTest.IPv6) > 0
	if !hasEndpoints {
		return fmt.Errorf("invalid config in %s: missing endpoint lists (expected flat middleware config)", path)
	}

	// 将配置内容写入全局变量
	PORT = string(mw.Port)
	HTTP_TIMEOUT = mw.HTTPTimeoutSeconds
	CORS = mw.Cors
	// 逗号分隔的 CORS 域名列表（对齐根 main.go 的 ACCEPT_DOMAINS 用法）
	if CORS != "" {
		ACCEPT_DOMAINS = strings.Split(CORS, ",")
	}
	API_BASE_URLS = mw.APIBaseURLs
	IP_LOCATION_APIS = mw.IPLocationAPIs
	TCPING = mw.TCPing
	SPEED_TEST = mw.SpeedTest
	NS_LOOKUP = mw.NSLookup
	API_KEYS = mw.APIKeys
	CONFIG_SOURCE = path
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

	app := fiber.New()

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
