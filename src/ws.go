package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"lemon-ipw/ipdb"
	"lemon-ipw/webtest"
)

// ==================== WS 客户端（测速节点接入中间件 WS 通道） ====================
//
// 配置（readConfig 解析，环境变量优先，其次 setting.json，远端配置可覆盖）：
//
//	WS_URL    ws://<中间件>:8092/ws,ws://<备中间件>:8092/ws   （逗号分隔多个中间件，同时连接全部，多活；不配置 = 不启用）
//	NODE_ID   节点 id（与中间件 ws-keys 键 / 前端配置 id 一致）
//	NODE_KEY  注册 key（与中间件 ws-keys[节点id] 一致；中间件未配置该节点 key 时留空）
//
// 与 HTTP handler 共用同一批 webtest 探针函数和缓存：收到 probe 后按 apiType 取数（带缓存），
// 返回处理在本文件内完成，结果经 probe_result 上报。节点 HTTP 接口完全不变。
//
// 心跳：NODE 与 MIDDLEWARE 双向心跳。本端每 10s 发一次 ping，发送失败重试，连续失败 3 次
// 判定当前中间件不可用，断开该连接并按同一 URL 独立重连（不影响其他中间件连接）。

// wsProbe 学 HTTP handler：从函数拿数据 + 缓存，返回 (status, body)。
// raw 与 query 来自中间件 probe 消息（等价于路由参数）。
func wsProbe(apiType, raw string, query map[string]string) (int, any) {
	switch apiType {
	case "detail":
		testURL := normalizeURL(raw)
		if cached, ok := websiteCache.Load(testURL); ok {
			entry := cached.(websiteCacheEntry)
			if time.Since(entry.timestamp) < 5*time.Minute {
				return 200, entry.result
			}
			websiteCache.Delete(testURL)
		}
		ipv4, ipv6 := wsDual[webtest.WebsiteCheckDetail](
			func(v string) (*webtest.WebsiteCheckDetail, error) { return webtest.CheckWebsite(testURL, v) },
			func(err error) *webtest.WebsiteCheckDetail {
				return &webtest.WebsiteCheckDetail{HostRecord: "Error: " + err.Error(), IsReachable: false}
			},
			func(v string) *webtest.WebsiteCheckDetail {
				return &webtest.WebsiteCheckDetail{HostRecord: "Skipped due to SINGLE_STACK=" + v, IsReachable: false}
			},
		)
		result := &WebsiteCheckResult{IPv4: ipv4, IPv6: ipv6}
		websiteCache.Store(testURL, websiteCacheEntry{result: result, timestamp: time.Now()})
		return 200, result

	case "ssl":
		testURL := normalizeURL(raw)
		if cached, ok := sslCache.Load(testURL); ok {
			entry := cached.(sslCacheEntry)
			if time.Since(entry.timestamp) < 5*time.Minute {
				return 200, entry.result
			}
			sslCache.Delete(testURL)
		}
		ipv4, ipv6 := wsDual[webtest.SSLCheckDetail](
			func(v string) (*webtest.SSLCheckDetail, error) { return webtest.CheckSSL(testURL, v) },
			func(err error) *webtest.SSLCheckDetail {
				return &webtest.SSLCheckDetail{HostRecord: "Error: " + err.Error(), IsExpired: true, IsReachable: false}
			},
			func(v string) *webtest.SSLCheckDetail {
				return &webtest.SSLCheckDetail{HostRecord: "Skipped due to SINGLE_STACK=" + v, IsExpired: true, IsReachable: false}
			},
		)
		result := &SSLCheckResult{IPv4: ipv4, IPv6: ipv6}
		sslCache.Store(testURL, sslCacheEntry{result: result, timestamp: time.Now()})
		return 200, result

	case "speed":
		// raw 形如 "v4/example.com"：首段为版本，其余为 URL
		version, testURL := splitRaw(raw)
		if version == "" {
			version = "v4"
		}
		testURL = normalizeURL(testURL)
		cacheKey := fmt.Sprintf("%s:%s", testURL, version)
		if cached, ok := speedCache.Load(cacheKey); ok {
			entry := cached.(speedCacheEntry)
			if time.Since(entry.timestamp) < 5*time.Minute {
				return 200, entry.result
			}
			speedCache.Delete(cacheKey)
		}
		r, e := webtest.SpeedTest(testURL, version)
		if e != nil {
			errorResult := &webtest.WebsiteSpeedTestResult{HostRecord: "Error: " + e.Error()}
			speedCache.Store(cacheKey, speedCacheEntry{result: errorResult, timestamp: time.Now()})
			return 200, errorResult
		}
		speedCache.Store(cacheKey, speedCacheEntry{result: r, timestamp: time.Now()})
		return 200, r

	case "tcping":
		host := raw
		port := query["port"]
		if port == "" {
			port = "80"
		}
		count := query["count"]
		n := 4
		if count != "" {
			if c, err := parseInt(count); err == nil && c >= 1 && c <= 20 {
				n = c
			}
		}
		cacheKey := fmt.Sprintf("%s:%s:%d", host, port, n)
		if cached, ok := pingCache.Load(cacheKey); ok {
			entry := cached.(pingCacheEntry)
			if time.Since(entry.timestamp) < 5*time.Minute {
				return 200, entry.result
			}
			pingCache.Delete(cacheKey)
		}
		ipv4, ipv6 := wsDual[webtest.TCPingStats](
			func(v string) (*webtest.TCPingStats, error) {
				return webtest.TCPingRun(host, port, n, v, 10*time.Second, 100*time.Millisecond)
			},
			func(err error) *webtest.TCPingStats { return &webtest.TCPingStats{IP: "Error: " + err.Error()} },
			func(v string) *webtest.TCPingStats { return &webtest.TCPingStats{IP: "Skipped due to SINGLE_STACK=" + v} },
		)
		result := &TCPingResult{IPv4: ipv4, IPv6: ipv6}
		pingCache.Store(cacheKey, pingCacheEntry{result: result, timestamp: time.Now()})
		return 200, result

	case "dns":
		// raw 形如 "a/example.com"：首段为记录类型
		recodeType, domain := splitRaw(raw)
		parsedURL, err := parseURL(domain)
		if err != nil {
			return 400, map[string]string{"error": "Invalid domain"}
		}
		domain = parsedURL.Host
		var result any
		switch recodeType {
		case "a":
			result, err = webtest.ResolveARecord(domain)
		case "aaaa":
			result, err = webtest.ResolveAAAARecord(domain)
		case "cname":
			result, err = webtest.ResolveCNAMERecord(domain)
		case "mx":
			result, err = webtest.ResolveMXRecord(domain)
		case "ns":
			result, err = webtest.ResolveNSRecord(domain)
		case "txt":
			result, err = webtest.ResolveTXTRecord(domain)
		case "srv":
			result, err = webtest.ResolveSRVRecord(domain)
		case "ptr":
			result, err = webtest.ResolvePTRRecord(domain)
		case "caa":
			result, err = webtest.ResolveCAARecord(domain)
		default:
			return 400, map[string]string{"error": "Unsupported record type: " + recodeType}
		}
		if err != nil {
			return 500, map[string]string{"error": err.Error()}
		}
		return 200, result

	case "dnssec":
		result, err := webtest.ResolveDNSSEC(raw)
		if err != nil {
			return 500, map[string]string{"error": err.Error()}
		}
		return 200, result

	case "whois":
		if raw == "" {
			return 400, map[string]string{"error": "Domain parameter is required"}
		}
		if cached, ok := whoisCache.Load(raw); ok {
			entry := cached.(whoisCacheEntry)
			if time.Since(entry.timestamp) < 5*time.Minute {
				return 200, entry.result
			}
			whoisCache.Delete(raw)
		}
		result, err := webtest.QueryWhois(raw)
		if err != nil {
			return 500, map[string]string{"error": err.Error()}
		}
		whoisCache.Store(raw, whoisCacheEntry{result: result, timestamp: time.Now()})
		return 200, result

	case "location":
		return 200, ipdb.SearchIP(raw)

	case "asn":
		if raw == "" {
			return 400, map[string]string{"error": "IP parameter is required"}
		}
		result := ipdb.SearchIP(raw, "maxmind_asn", "dbip_asn", "ip2location_asn")
		asnResult := map[string]any{"ip": raw}
		if ip2locASN, ok := result["ip2location_asn"].(map[string]string); ok && ip2locASN["asn"] != "" {
			asnResult["ip2location_asn"] = map[string]string{"asn": ip2locASN["asn"], "as": ip2locASN["as"]}
		} else if errStr, ok := result["ip2location_asn"].(string); ok {
			asnResult["ip2location_asn"] = map[string]string{"error": errStr}
		}
		if maxmindASN, ok := result["maxmind_asn"].(*ipdb.MMDBASNResult); ok {
			asnResult["geolite2_asn"] = map[string]string{"asn": maxmindASN.ASN, "org": maxmindASN.Org}
		} else if errStr, ok := result["maxmind_asn"].(string); ok {
			asnResult["geolite2_asn"] = map[string]string{"error": errStr}
		}
		if dbipASN, ok := result["dbip_asn"].(*ipdb.MMDBASNResult); ok {
			asnResult["dbip_asn"] = map[string]string{"asn": dbipASN.ASN, "org": dbipASN.Org}
		} else if errStr, ok := result["dbip_asn"].(string); ok {
			asnResult["dbip_asn"] = map[string]string{"error": errStr}
		}
		if maxmindASN, ok := result["maxmind_asn"].(*ipdb.MMDBASNResult); ok {
			whoisData, err := webtest.QueryASNWhois(maxmindASN.ASN)
			if err == nil {
				asnResult["whois"] = whoisData
			}
		}
		return 200, asnResult

	default:
		return 400, map[string]string{"error": "Unsupported apiType: " + apiType}
	}
}

// splitRaw 按首个 "/" 拆分为 (type/version, 剩余)
func splitRaw(raw string) (string, string) {
	i := strings.Index(raw, "/")
	if i == -1 {
		return raw, ""
	}
	return raw[:i], raw[i+1:]
}

// wsDual 双栈编排（ws.go 内部，学 main.go handler 的 SINGLE_STACK 处理）
func wsDual[T any](run func(version string) (*T, error), mkErr func(err error) *T, mkSkip func(version string) *T) (ipv4, ipv6 *T) {
	switch SINGLE_STACK {
	case "ipv4":
		v4, err := run("v4")
		if err != nil {
			v4 = mkErr(err)
		}
		return v4, mkSkip("ipv4")
	case "ipv6":
		v6, err := run("v6")
		if err != nil {
			v6 = mkErr(err)
		}
		return mkSkip("ipv6"), v6
	default:
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			v4, err := run("v4")
			if err != nil {
				v4 = mkErr(err)
			}
			ipv4 = v4
		}()
		go func() {
			defer wg.Done()
			v6, err := run("v6")
			if err != nil {
				v6 = mkErr(err)
			}
			ipv6 = v6
		}()
		wg.Wait()
		return
	}
}

// parseInt 简易整数解析
func parseInt(s string) (int, error) {
	n := 0
	neg := false
	for i, ch := range s {
		if i == 0 && ch == '-' {
			neg = true
			continue
		}
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid number %q", s)
		}
		n = n*10 + int(ch-'0')
	}
	if neg {
		n = -n
	}
	return n, nil
}

// ---- 连接与消息 ----

type wsMsg struct {
	Type   string          `json:"type"`
	NodeID string          `json:"nodeId,omitempty"`
	TS     int64           `json:"ts"`
	Data   json.RawMessage `json:"data,omitempty"`
}

func wsSend(c *websocket.Conn, m wsMsg) {
	_ = wsSendErr(c, m)
}

// wsSendErr 发送消息并返回错误（心跳用）
func wsSendErr(c *websocket.Conn, m wsMsg) error {
	b, _ := json.Marshal(m)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.Write(ctx, websocket.MessageText, b)
}

func wsRaw(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("{}")
	}
	return b
}

// wsHandleProbe 处理中间件下发的拨测请求（并发执行）：取数 → probe_result
func wsHandleProbe(c *websocket.Conn, data json.RawMessage) {
	var req struct {
		RequestID string            `json:"requestId"`
		APIType   string            `json:"apiType"`
		Raw       string            `json:"raw"`
		Query     map[string]string `json:"query"`
	}
	if err := json.Unmarshal(data, &req); err != nil || req.RequestID == "" || req.APIType == "" {
		slog.Warn("ws client bad probe message")
		return
	}
	status, body := wsProbe(req.APIType, req.Raw, req.Query)
	payload := struct {
		RequestID string          `json:"requestId"`
		Status    int             `json:"status"`
		Body      json.RawMessage `json:"body"`
	}{req.RequestID, status, nil}
	if json.Valid(mustMarshal(body)) {
		payload.Body = mustMarshal(body)
	} else {
		payload.Body, _ = json.Marshal(string(mustMarshal(body)))
	}
	wsSend(c, wsMsg{Type: "probe_result", Data: wsRaw(payload)})
}

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// 心跳配置：每 10s 发一次 ping；发送失败重试，连续失败 3 次判定中间件不可用
const (
	wsHeartbeatInterval = 10 * time.Second
	wsHeartbeatRetry    = 3
	wsHeartbeatTimeout  = 5 * time.Second
)

// wsClientOnce 连接并服务单个中间件 URL，返回下次重试前的等待时长（由 per-URL 连接循环调用）：
//   - 连接/读循环断开 → 3s（重试同一 URL）
//   - 注册被拒（register_error）→ 30s（多为 key 配置错误，加大间隔避免空转刷日志）
//   - 心跳连续失败 wsHeartbeatRetry 次 → 主动断开，3s
//   - 返回 0 = 配置缺失（不再重试）
func wsClientOnce(url string) time.Duration {
	ctx := context.Background()
	c, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		slog.Warn("ws client dial failed", "url", url, "error", err)
		return 3 * time.Second
	}
	defer c.CloseNow()

	reg := map[string]string{"nodeId": WS_NODE_ID}
	if WS_NODE_KEY != "" {
		reg["key"] = WS_NODE_KEY
	}
	wsSend(c, wsMsg{Type: "register", Data: wsRaw(reg)})

	_, data, err := c.Read(ctx)
	if err != nil {
		slog.Warn("ws client register read failed", "url", url, "error", err)
		return 3 * time.Second
	}
	var m wsMsg
	if json.Unmarshal(data, &m) != nil {
		return 3 * time.Second
	}
	if m.Type == "register_error" {
		slog.Warn("ws client register rejected", "nodeId", WS_NODE_ID, "url", url, "data", string(m.Data))
		return 30 * time.Second
	}
	if m.Type != "register_ok" {
		slog.Warn("ws client unexpected message", "type", m.Type)
		return 3 * time.Second
	}
	slog.Info("ws client registered", "nodeId", WS_NODE_ID, "url", url)

	// 心跳 goroutine：每 10s 发 ping，失败重试，连续 3 次失败主动断开（per-URL 循环重连同一中间件）
	stopHeartbeat := make(chan struct{})
	go func() {
		fail := 0
		for {
			select {
			case <-stopHeartbeat:
				return
			case <-time.After(wsHeartbeatInterval):
			}
			if err := wsSendErr(c, wsMsg{Type: "ping"}); err != nil {
				fail++
				slog.Warn("ws heartbeat send failed", "url", url, "fail", fail, "error", err)
				if fail >= wsHeartbeatRetry {
					slog.Warn("ws heartbeat failed too many times, closing connection (will retry same url)", "url", url)
					c.Close(websocket.StatusPolicyViolation, "heartbeat timeout")
					return
				}
			} else {
				fail = 0
			}
		}
	}()
	defer close(stopHeartbeat)

	for {
		_, data, err := c.Read(ctx)
		if err != nil {
			slog.Warn("ws client disconnected", "url", url, "error", err)
			return 3 * time.Second
		}
		var msg wsMsg
		if json.Unmarshal(data, &msg) != nil {
			continue
		}
		switch msg.Type {
		case "probe":
			go wsHandleProbe(c, msg.Data)
		case "ping":
			wsSend(c, wsMsg{Type: "pong"})
		case "status":
			slog.Debug("ws client status", "data", string(msg.Data))
		}
	}
}

// wsClientLoop 同时连接所有配置的中间件 URL（逗号分隔，多活）：
// 每个 URL 由独立 goroutine 负责，各自注册并保持连接，任一断开只重连自己，不影响其他中间件。
// 注意：多个 URL 指向同一中间件实例时，中间件按 nodeId 单连接，后注册的连接会顶掉先前的。
func wsClientLoop() {
	urls := splitWSURLs(WS_URL)
	if len(urls) == 0 {
		slog.Info("ws client disabled (WS_URL not set)")
		return
	}
	var wg sync.WaitGroup
	for _, url := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			wsClientOnceLoop(u)
		}(url)
	}
	wg.Wait()
}

// wsClientOnceLoop 单个中间件 URL 的持续连接循环：连接 → 注册 → 心跳/读循环 → 断开重试
func wsClientOnceLoop(url string) {
	for {
		retryDelay := wsClientOnce(url)
		if retryDelay == 0 {
			slog.Info("ws client stopped", "url", url)
			return
		}
		time.Sleep(retryDelay)
	}
}

// splitWSURLs 逗号分隔解析多个中间件 URL
func splitWSURLs(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
