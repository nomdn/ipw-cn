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

	"github.com/gin-gonic/gin"
)

// ==================== 数据上报（EO 节点 → 收集中心 ipw-boce） ====================
//
// 自主线 ipw-cn/src/report.go 提取，按 EO 部署形态裁剪：EO 版不连中间件 WS 通道
// （无 WebSocket 客户端），因此上报只走一条路——HTTP POST 收集中心 /report。
// 报文协议与 ipw-boce/report.go 完全一致：
//
//	POST /report  body = {"instance":"上报方标识","stats":[...],"probes":[...]}
//
// 节点是唯一记录者：统计与拨测明细都由本节点上报，收集中心不在转发路径上计数。
// 覆盖的请求来源只有一条：HTTP 接口收到的 /v1/*（nodeReportMiddleware 计数）。
//
// 进程模型前提：本实现假定云函数运行时是**长驻进程**（与 index.go 里缓存清扫 goroutine
// 同一假设）。若部署为按需冷启动，内存中的计数与明细缓冲会随实例回收丢失——上报只在
// 实例存活期内累积，间隔到期后即发出。
//
// 配置（env 优先，远端配置 REMOTE_CONFIG_URL 最高，见 index.go 的 readConfig/applyRemoteConfig）：
//
//	NODE_ID         上报身份（node-id），留空回退 hostname
//	REPORT_URL      收集中心 HTTP 基址（report-url），留空则不上报
//	REPORT_TOKEN    /report 鉴权 token（report-token）
//	REPORT_INTERVAL 上报间隔秒（report-interval-seconds），缺省 15

const (
	reportMaxProbes = 500
	reportMaxStats  = 500
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

// nodeIsProbeType 拨测类接口（结果需明细上报）。
// 明细覆盖：连通性/证书类 detail、ssl，DNS 查询类 dns，直连类 tcping/speed。
// dnssec 属功能性校验（结果是否签名/可信），不逐条进明细，只进统计聚合。
func nodeIsProbeType(apiType string) bool {
	switch apiType {
	case "tcping", "speed", "detail", "ssl", "dns":
		return true
	}
	return false
}

// startNodeReporter 启动上报循环（main() 调用，须在 readConfig 之后）。
// 循环常驻：即使当前没有配置 REPORT_URL 也照常起——留空只是"本轮不上报"，
// 便于将来接入热更新/远端配置后无需重启即可开启上报。
func startNodeReporter() {
	if REPORT_INTERVAL <= 0 {
		REPORT_INTERVAL = 15
	}
	nodeProbeCh = make(chan nodeProbeRec, 2048)
	go nodeReportLoop()
	slog.Info("node reporter enabled", "httpReport", REPORT_URL, "interval", REPORT_INTERVAL, "nodeId", nodeReportNodeID())
}

// nodeReportMiddleware /v1 组统计中间件：归属规则过滤 → 计数 → 拨测类补明细
func nodeReportMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		// 收集中心(ipw-boce)主动调度下发的拨测带 X-Scheduler-Probe 头：
		// 该次执行由收集中心侧本地落库(source=sched/biz)，本节点跳过计数与明细上报，避免双算。
		// 真实业务请求（用户/中间件转发）不带此头，仍按"节点是唯一记录者"正常上报。
		if c.Request.Header.Get("X-Scheduler-Probe") != "" {
			return
		}

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

// nodeReportNodeID 上报身份：优先 node-id（NODE_ID），否则 hostname
func nodeReportNodeID() string {
	if NODE_ID != "" {
		return NODE_ID
	}
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "unknown"
}

// nodeAPIFromPath 从 gin 路由模板提取 apiType 与拨测目标（对齐主线 WS probe 的 raw 格式）
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
		raw = c.Param("version") + c.Param("url") // v4/example.com
	case "dns":
		raw = c.Param("type") + c.Param("domain") // a/example.com
	case "detail", "ssl":
		raw = strings.TrimPrefix(c.Param("url"), "/")
	case "location", "asn":
		raw = c.Param("ip")
	}
	return apiType, raw
}

// nodeReportLoop 周期上报：取走计数与明细缓冲 → HTTP POST 收集中心 /report
func nodeReportLoop() {
	ticker := time.NewTicker(time.Duration(REPORT_INTERVAL) * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if REPORT_URL == "" {
			continue // 未配置收集中心地址：本轮不上报（配置就绪后自动开始）
		}
		payload := nodeBuildPayload()
		if payload == nil {
			continue
		}
		if err := nodeHTTPReport(payload); err != nil {
			slog.Warn("node http report failed", "error", err)
		}
	}
}

// nodeBuildPayload 取走当前计数与拨测缓冲，组装上报报文；无数据返回 nil。
// minute 传节点自己的分钟桶（收集端另有 minute=0 → 按自身时钟入桶的兜底口径）
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

// nodeHTTPReport 上报到收集中心 /report。
// at-most-once：失败只记日志不重试——统计是累加语义，重试会导致重复计算。
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
