package main

// ==================== 运行时配置接口（HTTP + WS） ====================
//
// 三个动作：
//
//	get     查看当前生效配置（凭据类键遮蔽为 ***）
//	patch   修改运行时配置（可选写回 setting.json）
//	refresh 重新从 REMOTE_CONFIG_URL 拉取远端配置并应用
//
// 通道：
//
//	HTTP  /v1/config (GET/PATCH) · /v1/config/refresh (POST)
//	      鉴权与业务接口一致（tokenCheck）。刻意不挂 v1 组的 nodeReportMiddleware：
//	      配置管理不是拨测请求，不该被计入拨测统计。
//	WS    type=config，data.action = get|patch|refresh，应答 type=config_result。
//	      节点是 WS 客户端，消息来自已注册的中间件连接，鉴权由注册时的 node-key 保证。
//
// 备注：远端下发（applyRemoteConfig）与本地 patch 共用 applyConfigMap，避免两处键映射漂移。
// 远端下发的忽略名单统一由 remoteIgnoreList() 给出（受保护凭据 + remote-ignore-config），
// 凭据保护见 configRemoteProtectedKeys。
//
// 需重启的键（configRestartKeys）：patch / refresh 里出现**值真的变了**的这类键时，
// 只在应答的 restartRequired 里如实报出，**节点不会自行重启**——重启会中断在途服务，
// 时机由运维掌握（容器编排 / supervisor 也可能接管重启）。端口、CORS 中间件、WS 连接等
// 启动期固定的东西，要等进程真正重启后才换过来。
//
// 通道鉴权差异（重要）：
//
//	HTTP  access-token 未配置时整个管理面关闭——此时节点业务接口也无鉴权，
//	      暴露配置读写等于公网可改配置。
//	WS    常开。节点是 WS 客户端，指令只来自已通过中间件 ws-keys 校验的注册连接，
//	      鉴权在中间件侧完成，与节点是否配 access-token 无关。

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"lemon-ipw/ssrf"
	"lemon-ipw/webtest"
)

// wsClientReconcileDelay ws-url 热更新后延后多久收敛 WS 连接（见 ws.go reconcileWSClient）。
// 改 ws-url 会取消"承载这次指令的那条旧连接"，同步收敛会把应答一起掐断：控制台只能看到超时，
// 未配 access-token 的节点连 HTTP 回退都没有。延后一点让应答先写回去。
const wsClientReconcileDelay = 300 * time.Millisecond

// errNoRemoteURL 未配置 remote-config-url 却要求刷新远端配置
var errNoRemoteURL = errors.New("remote-config-url 未配置")

// configSecretKeys 凭据类键：GET 快照里遮蔽，PATCH 仍可写（本地管理员的显式行为）
var configSecretKeys = map[string]bool{
	"access-token": true,
	"node-key":     true,
	"report-token": true,
}

// configRemoteProtectedKeys 远端下发（remote-config-url，含收集中心的"托管配置"）**永不覆盖**的凭据键。
//
// 为什么硬编码、而不交给 remote-ignore-config：
//
//	这两把凭据被远端改错的故障形态都是"远端一错、本地失联、且无法再远程修回"——
//	report-token 被改 → 节点上报被收集中心 /report 拒收，控制台上节点仍显示在线，数据却静默断了；
//	access-token 被改 → 控制台再也连不上该节点的管理面，连 PATCH 回去的通道都没了。
//	所以这层解耦必须由节点自己兜底，不能依赖"配置源别写错"。
//
// node-key 有意不在此列：托管部署下常由远端统一下发节点身份；若要一并锁死，
// 把它写进 remote-ignore-config 即可（本名单只增不减，且对本地 PATCH 无影响）。
var configRemoteProtectedKeys = []string{"access-token", "report-token"}

// remoteIgnoreList 远端下发的忽略名单 = 受保护凭据 + 操作员自选的 remote-ignore-config。
// 本地 PATCH（HTTP /v1/config、WS patch）传 nil，不受本名单限制——那是管理员的显式操作。
func remoteIgnoreList() []string {
	out := make([]string, 0, len(configRemoteProtectedKeys)+len(REMOTE_IGNORE_CONFIG))
	out = append(out, configRemoteProtectedKeys...)
	out = append(out, REMOTE_IGNORE_CONFIG...)
	return out
}

// protectedKeysIn 取 cfg 中命中远端保护名单的键（升序）。
// 远端下发里出现这些键会被跳过，如实回报 / 记日志，避免"托管配置里写了却不生效"的困惑。
func protectedKeysIn(cfg map[string]any) []string {
	var out []string
	for _, k := range configRemoteProtectedKeys {
		if configValue(cfg, k) != "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// configRestartKeys 改了内存变量但需重启进程才完全生效的键：
//
//   - port / trusted-proxies / cors：监听地址与 gin 中间件在启动时构建并固定
//   - ipdb：ipdb.Init 与 location / asn 路由注册都按启动时的值判定
//   - report-interval-seconds：上报 ticker 在启动时按该值创建
//   - node-id / node-key：WS 注册身份在启动时确立；运行中改动只改内存变量，
//     已建立的连接不会换身份，必须重启进程才生效
//   - access-token：HTTP 管理面路由是否注册、v1 组是否挂 tokenCheck 都在启动时判定
//
// **不在本表但也不是"每次消费重读"的特例：`ws-url` 是热生效的**——由 WS 客户端控制器
// （ws.go 的 reconcileWSClient）按新地址多退少补，连接集合随配置变化，无需重启。
//
// 不在其中的键即为热生效：`dns-server` / `dnssec-server` / `block-private-ips` /
// `single-stack` / `node-ota` 每次消费时重新读变量。
//
// 判定口径：只有**值真的变了**才算需重启（见 applyConfigMap）——远端/托管配置每次都带全部键，
// 若按"键出现过"判定，每次 refresh 都会把节点误报成"需重启"。
//
// 本表只用于**如实回报** restartRequired，节点不据此自动重启（重启时机交给运维）。
var configRestartKeys = map[string]bool{
	"port":                    true,
	"cors":                    true,
	"ipdb":                    true,
	"report-interval-seconds": true,
	"trusted-proxies":         true,
	"node-id":                 true,
	"node-key":                true,
	"access-token":            true,
}

// applyConfigMap 把配置 map 逐键应用到运行时变量（远端下发与本地 patch 共用）。
// ignore 里的键直接跳过（远端用它实现 remote-ignore-config，本地 patch 传 nil 表示不限制）。
//
// 返回：
//
//	applied  实际生效的键（升序）
//	unknown  不认识的键（不静默丢弃，交调用方回给请求方，便于发现拼错的键）
//	restart  已改内存、需重启才完全生效、且**值确实变了**的键（升序）
func applyConfigMap(cfg map[string]any, ignore []string) (applied, unknown, restart []string) {
	skip := func(key string) bool {
		for _, k := range ignore {
			if k == key {
				return true
			}
		}
		return false
	}
	mark := func(key string) {
		applied = append(applied, key)
	}

	// 需重启类键的"改动前"取值：用于判断值是否真的变了。
	// applied 的口径是"请求方给了这个键"，不是"值变了"——直接拿它当重启依据的话，
	// 每次都带全量键的托管配置会让节点每 refresh 一次就误报一次"需重启"。
	before := make(map[string]string, len(configRestartKeys))
	for k := range configRestartKeys {
		before[k] = restartKeyValue(k)
	}

	// ws-url 改动后要收敛 WS 连接集合：这里只置标志位，循环结束后统一收敛一次（见下）。
	// 与需重启键同口径——**值真的变了**才算，否则每次带全量键的托管配置 refresh 都会白跑一次收敛。
	beforeWSURL := WS_URL
	wsClientDirty := false

	for _, key := range sortedKeys(cfg) {
		v := configValue(cfg, key) // 空值视为"不覆盖"，与远端配置一致
		if v == "" || skip(key) {
			continue
		}
		switch key {
		case "port":
			PORTS = v
		case "gh-proxy":
			GH_PROXY = v
		case "single-stack":
			SINGLE_STACK = strings.ToLower(v)
		case "dns-server":
			DNS_SERVER = v
			webtest.SetDNSServer(v)
		case "dnssec-server":
			DNSSEC_DNS_SERVER = v
			webtest.SetDNSSecServer(v)
		case "ipdb":
			// 仅改变量不够：ipdb.Init 与 location/asn 路由注册都在启动时按该值判定，运行中不重建
			IPDB = v
		case "cors":
			CORS = v
			ACCEPT_DOMAINS = splitAndTrim(CORS, ",")
		case "block-private-ips":
			ssrf.SetEnabled(v != "false" && v != "0")
		case "trusted-proxies":
			TRUSTED_PROXIES = v
		case "remote-config-url":
			REMOTE_CONFIG_URL = v
		case "remote-ignore-config":
			if arr, ok := cfg[key].([]any); ok { // setting.json 里是数组，也可给逗号分隔串
				out := make([]string, 0, len(arr))
				for _, item := range arr {
					if s, ok := item.(string); ok {
						if s = strings.TrimSpace(s); s != "" {
							out = append(out, s)
						}
					}
				}
				REMOTE_IGNORE_CONFIG = out
			} else {
				REMOTE_IGNORE_CONFIG = splitAndTrim(v, ",")
			}
		case "access-token":
			ACCESS_TOKEN = v
		case "ws-url":
			WS_URL = v
			wsClientDirty = WS_URL != beforeWSURL
		case "node-id":
			WS_NODE_ID = v
		case "node-key":
			WS_NODE_KEY = v
		case "report-url":
			REPORT_URL = v
		case "report-token":
			REPORT_TOKEN = v
		case "report-interval-seconds":
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 {
				unknown = append(unknown, key) // 值非法：不覆盖原值，明确报给调用方
				continue
			}
			REPORT_INTERVAL = n
		case "node-ota":
			// 热生效：下发 OTA 指令时才读 NODE_OTA（见 ota.go），故不在 configRestartKeys
			enabled, known := parseOTASwitch(v)
			if !known {
				unknown = append(unknown, key) // 值非法：不覆盖原值，明确报给调用方
				continue
			}
			NODE_OTA = enabled
		default:
			unknown = append(unknown, key)
			continue
		}
		mark(key)
	}

	// 需重启的键：值真的变了才报（见 before 注释）。仅作为信息回报给调用方，
	// 节点不据此重启——重启时机会中断在途服务，由运维决定。
	for _, key := range applied {
		if configRestartKeys[key] && restartKeyValue(key) != before[key] {
			restart = append(restart, key)
		}
	}

	// ws-url 热更新：把在跑的连接收敛到新地址（多退少补）。
	// 放在循环外是为了避免一条指令里 ws-url 重复出现时收敛多次；
	// 异步延后则是为了先让本次请求的应答写回去——收敛会取消承载它自己的那条旧连接。
	if wsClientDirty {
		go func() {
			time.Sleep(wsClientReconcileDelay)
			reconcileWSClient()
		}()
	}

	sort.Strings(applied)
	sort.Strings(unknown)
	sort.Strings(restart)
	return applied, unknown, restart
}

// restartKeyValue 取需重启类键的当前值（字符串化），只用于比对改动前后是否真的变了。
// 不认识的键返回 ""。
func restartKeyValue(key string) string {
	switch key {
	case "port":
		return PORTS
	case "cors":
		return CORS
	case "ipdb":
		return IPDB
	case "trusted-proxies":
		return TRUSTED_PROXIES
	case "ws-url":
		return WS_URL
	case "node-id":
		return WS_NODE_ID
	case "node-key":
		return WS_NODE_KEY
	case "access-token":
		return ACCESS_TOKEN
	case "report-interval-seconds":
		return strconv.Itoa(REPORT_INTERVAL)
	}
	return ""
}

// configSnapshot 当前生效配置的快照（凭据类键只回显 ***，不泄露明文）
func configSnapshot() map[string]any {
	out := map[string]any{
		"port":                    PORTS,
		"gh-proxy":                GH_PROXY,
		"single-stack":            SINGLE_STACK,
		"dns-server":              DNS_SERVER,
		"dnssec-server":           DNSSEC_DNS_SERVER,
		"ipdb":                    IPDB,
		"cors":                    CORS,
		"block-private-ips":       ssrf.Enabled(),
		"trusted-proxies":         TRUSTED_PROXIES,
		"remote-config-url":       REMOTE_CONFIG_URL,
		"remote-ignore-config":    REMOTE_IGNORE_CONFIG,
		"ws-url":                  WS_URL,
		"node-id":                 WS_NODE_ID,
		"node-ota":                NODE_OTA,
		"report-url":              REPORT_URL,
		"report-interval-seconds": REPORT_INTERVAL,
	}
	if ACCESS_TOKEN != "" {
		out["access-token"] = "***"
	} else {
		out["access-token"] = ""
	}
	if WS_NODE_KEY != "" {
		out["node-key"] = "***"
	} else {
		out["node-key"] = ""
	}
	if REPORT_TOKEN != "" {
		out["report-token"] = "***"
	} else {
		out["report-token"] = ""
	}
	return out
}

// persistConfig 把已应用的键写回 setting.json（viper 保留文件中其它未改动的键）。
// 无 setting.json（纯环境变量部署）时新建一份。
func persistConfig(body map[string]any, applied []string) error {
	for _, key := range applied {
		viper.Set(key, body[key])
	}
	if err := viper.WriteConfig(); err == nil {
		return nil
	}
	return viper.SafeWriteConfigAs("setting.json")
}

// sortedKeys 配置键升序，保证 applied/unknown 输出顺序稳定（便于日志 diff 与测试）
func sortedKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// refreshRemoteConfig 重新拉取远端配置并应用；受保护凭据（access-token / report-token）
// 永不随远端下发，保持"ENV / setting.json 本地管理"的口径（见 configRemoteProtectedKeys）。
func refreshRemoteConfig() (map[string]any, error) {
	url := strings.TrimSpace(REMOTE_CONFIG_URL)
	if url == "" {
		return nil, errNoRemoteURL
	}
	cfg, err := fetchRemoteConfig(url)
	if err != nil {
		return nil, err
	}
	applied, unknown, restart := applyConfigMap(cfg, remoteIgnoreList())
	protected := protectedKeysIn(cfg)
	if len(protected) > 0 {
		slog.Warn("远端配置里的凭据键已被忽略（凭据由节点本地管理）", "keys", protected, "url", url)
	}
	// 远端配置本身就是持久来源（重启后会重新拉取），因此不需要补写 setting.json；
	// restart 里的键只如实回报，重启时机交给运维
	return map[string]any{
		"source":              url,
		"applied":             applied,
		"unknown":             unknown,
		"restartRequired":     restart,
		"config":              configSnapshot(),
		"protectedIgnored":    protected,
		"remoteProtectedKeys": configRemoteProtectedKeys,
	}, nil
}

// ==================== HTTP ====================

// registerConfigRoutes 注册运行时配置接口（/v1/config 组：与业务接口同鉴权，但不计入拨测统计）。
//
// access-token 未配置时**关闭整个 HTTP 管理面**：此时节点业务接口本身也无鉴权，
// 再暴露配置读写等于公网任何人可改节点配置。WS 通道不受影响——它由节点主动外连中间件，
// 注册需通过中间件的 ws-keys 校验，指令只发给已注册连接，天然有鉴权。
func registerConfigRoutes(r *gin.Engine) {
	g := r.Group("/v1/config")
	if ACCESS_TOKEN == "" {
		slog.Warn("config HTTP API disabled: access-token not set, use WS channel instead")
		disabled := func(c *gin.Context) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "配置管理接口已关闭：节点未配置 access-token，请通过 WS 通道操作",
			})
		}
		g.GET("", disabled)
		g.PATCH("", disabled)
		g.POST("/refresh", disabled)
		return
	}
	g.Use(tokenCheck())
	g.GET("", configGetHandler)
	g.PATCH("", configPatchHandler)
	g.POST("/refresh", configRefreshHandler)
}

func configGetHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"config":              configSnapshot(),
		"secretKeys":          sortedKeysBool(configSecretKeys),
		"restartRequiredKeys": sortedKeysBool(configRestartKeys),
		"remoteProtectedKeys": configRemoteProtectedKeys,
	})
}

func configPatchHandler(c *gin.Context) {
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体须为 JSON 对象：" + err.Error()})
		return
	}
	if len(body) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体为空"})
		return
	}
	// 本地显式 PATCH 不受 remote-ignore-config 限制：那是"防远端覆盖"的白名单
	applied, unknown, restart := applyConfigMap(body, nil)

	persisted := false
	if p := c.Query("persist"); p == "1" || strings.EqualFold(p, "true") {
		if err := persistConfig(body, applied); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "写回 setting.json 失败：" + err.Error(),
				"applied": applied,
			})
			return
		}
		persisted = true
	}
	// 未要求持久化时只改内存：需重启的键不落盘，进程重启后会回到 setting.json / 远端配置的值。
	// 这是刻意的语义——PATCH 是"当前进程内的即时改动"，要不要固化由调用方用 persist 明确决定。
	c.JSON(http.StatusOK, gin.H{
		"applied":         applied,
		"unknown":         unknown,
		"restartRequired": restart,
		"persisted":       persisted,
		"config":          configSnapshot(),
	})
}

func configRefreshHandler(c *gin.Context) {
	res, err := refreshRemoteConfig()
	if err == errNoRemoteURL {
		c.JSON(http.StatusBadRequest, gin.H{"error": "remote-config-url 未配置，无法刷新"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "拉取远端配置失败：" + err.Error(), "source": REMOTE_CONFIG_URL})
		return
	}
	c.JSON(http.StatusOK, res)
}

// sortedKeysBool 取 map 键的升序列表（用于向调用方说明哪些键是凭据/需重启）
func sortedKeysBool(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ==================== WS ====================

// wsConfigResult 配置指令的 WS 应答（type=config_result）
type wsConfigResult struct {
	RequestID       string         `json:"requestId,omitempty"`
	Action          string         `json:"action"`
	OK              bool           `json:"ok"`
	Error           string         `json:"error,omitempty"`
	Source          string         `json:"source,omitempty"`
	Applied         []string       `json:"applied,omitempty"`
	Unknown         []string       `json:"unknown,omitempty"`
	RestartRequired []string       `json:"restartRequired,omitempty"`
	Persisted       bool           `json:"persisted,omitempty"`
	Config          map[string]any `json:"config,omitempty"`
	// ProtectedIgnored 本次 refresh 中因凭据保护被跳过的键：远端配置里写了也不生效（见 configRemoteProtectedKeys）
	ProtectedIgnored []string `json:"protectedIgnored,omitempty"`
	// 元信息：与 HTTP GET 一致，供控制台标注「凭据键不下发」「远端不可覆盖」「该键改动需重启」
	SecretKeys          []string `json:"secretKeys,omitempty"`
	RestartKeys         []string `json:"restartRequiredKeys,omitempty"`
	RemoteProtectedKeys []string `json:"remoteProtectedKeys,omitempty"`
}

// wsHandleConfig 处理中间件下发的配置指令：get / patch / refresh
func wsHandleConfig(c *websocket.Conn, msg wsMsg) {
	var req struct {
		RequestID string         `json:"requestId"`
		Action    string         `json:"action"`
		Config    map[string]any `json:"config"`
		Persist   bool           `json:"persist"`
	}
	_ = json.Unmarshal(msg.Data, &req)

	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		action = "get"
	}
	res := wsConfigResult{RequestID: req.RequestID, Action: action, OK: true}

	switch action {
	case "get":
		res.Config = configSnapshot()

	case "patch":
		if len(req.Config) == 0 {
			res.OK, res.Error = false, "config 为空"
			break
		}
		applied, unknown, restart := applyConfigMap(req.Config, nil)
		// 未要求持久化时只改内存（同 HTTP PATCH）：需重启的键不落盘，重启后会回到原值
		if req.Persist {
			if err := persistConfig(req.Config, applied); err != nil {
				res.OK, res.Error = false, "写回 setting.json 失败："+err.Error()
				break
			}
			res.Persisted = true
		}
		res.Applied, res.Unknown, res.RestartRequired = applied, unknown, restart
		res.Config = configSnapshot()

	case "refresh":
		out, err := refreshRemoteConfig()
		if err != nil {
			res.OK, res.Error = false, err.Error()
			break
		}
		res.Source, _ = out["source"].(string)
		res.Applied, _ = out["applied"].([]string)
		res.Unknown, _ = out["unknown"].([]string)
		res.RestartRequired, _ = out["restartRequired"].([]string)
		res.ProtectedIgnored, _ = out["protectedIgnored"].([]string)
		res.Config, _ = out["config"].(map[string]any)

	default:
		res.OK, res.Error = false, "未知 action："+action
	}

	// 元信息随每次应答带上，控制台据此标注凭据键、远端不可覆盖键与需重启键（与 HTTP GET 保持一致）
	res.SecretKeys = sortedKeysBool(configSecretKeys)
	res.RestartKeys = sortedKeysBool(configRestartKeys)
	res.RemoteProtectedKeys = configRemoteProtectedKeys
	wsSend(c, wsMsg{Type: "config_result", Data: wsRaw(res)})
}
