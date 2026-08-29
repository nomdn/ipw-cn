package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

// ==================== 数据上报（节点视角 → 收集中心中间件） ====================
//
// 与中间件（ipw-boce）的上报协议对接（协议定义见 ipw-boce report.go）：
//   - WS 客户端启用（WS_URL 配置且至少一条连接在线）→ 经 WS 发 {"type":"report","data":...}
//   - WS 未启用或全部连接掉线 → HTTP POST 到收集中心 /report（REPORT_URL）
//
// 归属规则（防双算）：每条请求只由一个入口上报。
//   - 转发方（Go 中间件）转发时带 X-Boce-Reporter 标记头并自行上报 → 本节点跳过这些请求
//   - WS 下发的拨测（probe 消息带 requestId）→ 下发方中间件已记录上报 → 本节点跳过
//   - 其余流量（直连 / 前端内置 TS 中间件 / 边缘函数）→ 本节点上报（唯一记录者）
//     注意：TS 中间件/边缘函数若也做"调一次上报一次"，其转发请求必须同样携带
//     X-Boce-Reporter 头，否则会与本节点的上报双算。

const (
	reportMarkerHeader = "X-Boce-Reporter" // 与 ipw-boce 转发标记头一致
	reportMaxProbes    = 500
	reportMaxStats     = 500
)

var (
	nodeStatsMu    sync.Mutex
	nodeStats      = map[string]*nodeStatVal{} // apiType → 计数
	nodeProbeCh    chan nodeProbeRec
	nodeProbeDropN int64
	nodeDropMu     sync.Mutex
)

type nodeStatVal struct {
	total, errs, latSum, latMax int64
}

// nodeProbeRec 一条拨测明细（结构与 ipw-boce reportProbe 对齐）
type nodeProbeRec struct {
	RequestID string `json:"requestId,omitempty"`
	NodeID    string `json:"nodeId"`
	APIType   string `json:"apiType"`
	Raw       string `json:"raw"`
	Query     string `json:"query,omitempty"`
	Status    int    `json:"status"`
	LatencyMs int64  `json:"latencyMs"`
	Error     string `json:"error,omitempty"`
	Source    string `json:"source,omitempty"`
	Body      string `json:"body,omitempty"`
	CreatedAt int64  `json:"createdAt,omitempty"`
}

// nodeIsProbeType 拨测类接口（结果需明细上报）
func nodeIsProbeType(apiType string) bool {
	switch apiType {
	case "tcping", "speed", "udping":
		return true
	}
	return false
}

// startNodeReporter 启动上报循环（WS 启用或配置 REPORT_URL 时；main() 调用）
func startNodeReporter() {
	wsOn := WS_URL != ""
	if !wsOn && REPORT_URL == "" {
		return
	}
	if REPORT_INTERVAL <= 0 {
		REPORT_INTERVAL = 15
	}
	nodeProbeCh = make(chan nodeProbeRec, 2048)
	go nodeReportLoop()
	slog.Info("node reporter enabled", "ws", wsOn, "httpFallback", REPORT_URL, "interval", REPORT_INTERVAL)
}

// nodeReportMiddleware /v1 组统计中间件：归属规则过滤 → 计数 → 拨测类补明细
func nodeReportMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 中间件已带标记：它自己会计数上报，本节点跳过（防双算）
		if c.Request.Header.Get(reportMarkerHeader) != "" {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()

		apiType, raw := nodeAPIFromPath(c)
		if apiType == "" {
			return
		}
		latency := time.Since(start)
		status := c.Writer.Status()
		nodeRecordAPI(apiType, status, latency)
		if nodeIsProbeType(apiType) {
			nodeRecordProbe(nodeProbeRec{
				NodeID: nodeReportNodeID(), APIType: apiType, Raw: raw,
				Query: c.Request.URL.RawQuery, Status: status, LatencyMs: latency.Milliseconds(),
				Source: "http", CreatedAt: time.Now().Unix(),
			})
		}
	}
}

// nodeRecordWSProbe WS 拨测出口（ws.go wsHandleProbe 调用）。
// requestId 非空 = 中间件下发的请求，中间件侧已计数上报，本节点跳过（防双算）
func nodeRecordWSProbe(requestID, apiType, raw string, query map[string]string, status int, latency time.Duration) {
	if requestID != "" {
		return
	}
	if apiType == "" {
		return
	}
	nodeRecordAPI(apiType, status, latency)
	if nodeIsProbeType(apiType) {
		q := ""
		for k, v := range query {
			if q != "" {
				q += "&"
			}
			q += k + "=" + v
		}
		nodeRecordProbe(nodeProbeRec{
			NodeID: nodeReportNodeID(), APIType: apiType, Raw: raw, Query: q,
			Status: status, LatencyMs: latency.Milliseconds(), Source: "ws",
			CreatedAt: time.Now().Unix(),
		})
	}
}

func nodeRecordAPI(apiType string, status int, latency time.Duration) {
	ms := latency.Milliseconds()
	nodeStatsMu.Lock()
	v, ok := nodeStats[apiType]
	if !ok {
		v = &nodeStatVal{}
		nodeStats[apiType] = v
	}
	v.total++
	if status == 0 || status >= 500 {
		v.errs++
	}
	v.latSum += ms
	if ms > v.latMax {
		v.latMax = ms
	}
	nodeStatsMu.Unlock()
}

func nodeRecordProbe(rec nodeProbeRec) {
	if nodeProbeCh == nil {
		return
	}
	select {
	case nodeProbeCh <- rec:
	default:
		nodeDropMu.Lock()
		nodeProbeDropN++
		n := nodeProbeDropN
		nodeDropMu.Unlock()
		if n%100 == 1 {
			slog.Warn("node report probe queue full", "dropped", n)
		}
	}
}

// nodeReportNodeID 上报身份：优先 node-id（WS_NODE_ID），否则 hostname
func nodeReportNodeID() string {
	if WS_NODE_ID != "" {
		return WS_NODE_ID
	}
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "unknown"
}

// nodeAPIFromPath 从 gin 路由模板提取 apiType 与拨测目标（对齐 WS probe 的 raw 格式）
func nodeAPIFromPath(c *gin.Context) (apiType, raw string) {
	path := c.FullPath() // 形如 /v1/tcping/:ip
	if !strings.HasPrefix(path, "/v1/") {
		return "", ""
	}
	seg := strings.SplitN(strings.TrimPrefix(path, "/v1/"), "/", 2)
	if len(seg) == 0 || seg[0] == "" {
		return "", ""
	}
	apiType = seg[0]
	switch apiType {
	case "tcping", "dnssec", "whois":
		raw = c.Param("ip")
		if raw == "" {
			raw = c.Param("domain")
		}
	case "speed":
		raw = c.Param("version") + c.Param("url") // v4/example.com（对齐 WS probe raw 格式）
	case "dns":
		raw = c.Param("type") + c.Param("domain") // a/example.com
	case "detail", "ssl":
		raw = strings.TrimPrefix(c.Param("url"), "/")
	case "location", "asn":
		raw = c.Param("ip")
	}
	return apiType, raw
}

// nodeReportLoop 周期上报：WS 在线走 WS 广播（所有已注册中间件），否则 HTTP POST 收集中心
func nodeReportLoop() {
	ticker := time.NewTicker(time.Duration(REPORT_INTERVAL) * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		payload := nodeBuildPayload()
		if payload == nil {
			continue
		}
		sent := wsBroadcastReport(payload)
		if !sent && REPORT_URL != "" {
			if err := nodeHTTPReport(payload); err != nil {
				slog.Warn("node http report failed", "error", err)
			}
		}
	}
}

// nodeBuildPayload 取走当前计数与拨测缓冲，组装上报报文；无数据返回 nil。
// minute 传 0 = 收集器按自己时钟入桶（节点时钟不可信时不影响聚合口径）
func nodeBuildPayload() map[string]any {
	nodeStatsMu.Lock()
	snap := nodeStats
	nodeStats = make(map[string]*nodeStatVal, len(snap))
	nodeStatsMu.Unlock()

	probes := make([]nodeProbeRec, 0, reportMaxProbes)
	if nodeProbeCh != nil {
	drain:
		for len(probes) < reportMaxProbes {
			select {
			case r := <-nodeProbeCh:
				probes = append(probes, r)
			default:
				break drain // 队列已空，本轮结束
			}
		}
	}
	if len(snap) == 0 && len(probes) == 0 {
		return nil
	}
	minute := time.Now().UTC().Unix() / 60
	stats := make([]map[string]any, 0, len(snap))
	for apiType, v := range snap {
		stats = append(stats, map[string]any{
			"minute": minute, "nodeId": nodeReportNodeID(), "apiType": apiType,
			"total": v.total, "errors": v.errs, "latencySumMs": v.latSum, "latencyMaxMs": v.latMax,
		})
	}
	payload := map[string]any{
		"instance": nodeReportNodeID(),
		"stats":    stats,
	}
	if len(probes) > 0 {
		payload["probes"] = probes
	}
	return payload
}

// wsBroadcastReport 经 WS 通道向所有在线中间件广播 report 消息；无在线连接返回 false（触发 HTTP 兜底）
func wsBroadcastReport(payload map[string]any) bool {
	sent := false
	data, _ := json.Marshal(payload)
	wsActiveConns.Range(func(_, v any) bool {
		if c, ok := v.(*websocket.Conn); ok && c != nil {
			if err := wsSendErr(c, wsMsg{Type: "report", NodeID: WS_NODE_ID, Data: data}); err != nil {
				slog.Warn("node ws report send failed", "error", err)
			} else {
				sent = true
			}
		}
		return true
	})
	return sent
}

// nodeHTTPReport HTTP 兜底上报（WS 未启用/全部掉线时）
func nodeHTTPReport(payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := strings.TrimSuffix(REPORT_URL, "/") + "/report"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if REPORT_TOKEN != "" {
		req.Header.Set("Authorization", "Bearer "+REPORT_TOKEN)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &httpStatusError{status: resp.StatusCode}
	}
	return nil
}

type httpStatusError struct{ status int }

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("collector returned status %d", e.status)
}
