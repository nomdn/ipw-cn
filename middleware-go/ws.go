package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// ==================== WS 通信（middleware 作服务端） ====================
//
// 后端节点作为 WS 客户端连接 middleware 的 WS 服务端（默认端口 8092，路径 /ws，环境变量 WS_PORT 覆盖，0 关闭）。
// 节点配置项 `"ws": true`（或 "true"）时，该节点的拨测请求改走 WS 通道转发；缺省 false 保持原有 HTTP 转发。
// 现有 HTTP 接口（/v1/* /middleware/*）完全不变，WS 是独立端口上的新增数据面。
//
// 消息信封（JSON 文本帧）：{ "type": "...", "nodeId": "...", "ts": <unix秒>, "data": {...} }
//   - 节点 → middleware：register / probe_result / pong / command
//   - middleware → 节点：register_ok / register_error / probe / ping / status
//
// probe 消息 data：{ "requestId": "...", "apiType": "tcping", "raw": "qq.com", "query": {"port":"443"} }
// probe_result data：{ "requestId": "...", "status": 200, "body": <JSON 值> }（body 为 JSON 字符串时按原文透传）

type wsMessage struct {
	Type   string          `json:"type"`
	NodeID string          `json:"nodeId,omitempty"`
	TS     int64           `json:"ts"`
	Data   json.RawMessage `json:"data,omitempty"`
}

type wsProbeRequest struct {
	RequestID string            `json:"requestId"`
	APIType   string            `json:"apiType"`
	Raw       string            `json:"raw"`
	Query     map[string]string `json:"query,omitempty"`
}

type wsProbeResult struct {
	RequestID string          `json:"requestId"`
	Status    int             `json:"status"`
	Body      json.RawMessage `json:"body"`
	Error     string          `json:"error,omitempty"`
}

type wsCommand struct {
	Command string          `json:"command"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type wsPeer struct {
	conn *websocket.Conn
	id   string
	last time.Time
}

type wsServer struct {
	mu    sync.Mutex
	peers map[string]*wsPeer

	probeMu   sync.Mutex
	probeWait map[string]chan wsProbeResult

	// 统计
	statMu    sync.Mutex
	totalReqs int64
	errReqs   int64
	startedAt time.Time
}

func newWSServer() *wsServer {
	return &wsServer{
		peers:     make(map[string]*wsPeer),
		probeWait: make(map[string]chan wsProbeResult),
		startedAt: time.Now(),
	}
}

func (s *wsServer) Handler(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		log.Printf("[ws] accept failed: %v", err)
		return
	}
	defer c.Close(websocket.StatusInternalError, "closed")

	ctx := r.Context()
	peer := &wsPeer{conn: c}
	registered := false

	// 读循环
	for {
		_, data, err := c.Read(ctx)
		if err != nil {
			if !registered {
				log.Printf("[ws] peer closed before register")
			} else {
				log.Printf("[ws] node %s disconnected: %v", peer.id, err)
			}
			break
		}

		var msg wsMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("[ws] invalid message from %s: %v", peer.id, err)
			continue
		}
		peer.last = time.Now()

		// ===== 数据阶段 =====
		switch msg.Type {
		case "register":
			var reg struct {
				NodeID string `json:"nodeId"`
				Key    string `json:"key"` // 注册凭证：与 setting.json apiKeys 配置比对
			}
			_ = json.Unmarshal(msg.Data, &reg)
			if reg.NodeID == "" {
				s.sendJSON(c, wsMessage{Type: "register_error", Data: mustRaw(wsCommand{Command: "empty nodeId"})})
				continue
			}
			if registered {
				s.sendJSON(c, wsMessage{Type: "register_error", Data: mustRaw(wsCommand{Command: "already registered"})})
				continue
			}
			// 注册 key 校验：节点配置了 apiKeys[节点id] 就必须传对 key，否则 401 并断开；
			// 未配置 key 的节点（开放）无需传 key
			if expected := lookupAPIKey(reg.NodeID); expected != "" && reg.Key != expected {
				log.Printf("[ws] AUTH FAILED for node %s: invalid key", reg.NodeID)
				s.sendJSON(c, wsMessage{Type: "register_error", Data: mustRaw(struct {
					Code    int    `json:"code"`
					Command string `json:"command"`
				}{Code: 401, Command: "invalid key"})})
				c.Close(websocket.StatusPolicyViolation, "invalid key")
				return
			}
			registered = true
			peer.id = reg.NodeID
			s.mu.Lock()
			// 同 id 新连接替换旧连接（旧连接由读循环自然退出）
			if old, ok := s.peers[reg.NodeID]; ok && old.conn != c {
				old.conn.Close(websocket.StatusPolicyViolation, "replaced by new connection")
			}
			s.peers[reg.NodeID] = peer
			s.mu.Unlock()
			log.Printf("[ws] node registered: %s", reg.NodeID)
			s.sendJSON(c, wsMessage{Type: "register_ok", NodeID: reg.NodeID, TS: time.Now().Unix(), Data: mustRaw(struct {
				Heartbeat int `json:"heartbeatSeconds"`
			}{Heartbeat: 20})})

		case "probe_result":
			if !registered {
				continue
			}
			var res wsProbeResult
			if err := json.Unmarshal(msg.Data, &res); err != nil {
				log.Printf("[ws] bad probe_result from %s: %v", peer.id, err)
				continue
			}
			s.probeMu.Lock()
			ch, ok := s.probeWait[res.RequestID]
			delete(s.probeWait, res.RequestID)
			s.probeMu.Unlock()
			if ok {
				ch <- res
			}

		case "pong":
			// 心跳应答，仅刷新 last

		case "ping":
			// 节点主动心跳：回 pong
			if registered {
				s.sendJSON(c, wsMessage{Type: "pong", NodeID: peer.id, TS: time.Now().Unix()})
			}

		default:
			log.Printf("[ws] unknown message type from %s: %s", peer.id, msg.Type)
		}
	}

	// 断开清理
	if registered {
		s.mu.Lock()
		if s.peers[peer.id] == peer {
			delete(s.peers, peer.id)
		}
		s.mu.Unlock()
	}
}

func (s *wsServer) sendJSON(c *websocket.Conn, msg wsMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = c.Write(ctx, websocket.MessageText, mustRaw(msg))
}

// RequestProbe 通过 WS 通道向指定节点发送拨测请求并等待结果（超时返回错误）。
func (s *wsServer) RequestProbe(nodeID, apiType, raw string, query map[string]string, timeout time.Duration) (int, []byte, error) {
	s.mu.Lock()
	peer, ok := s.peers[nodeID]
	s.mu.Unlock()
	if !ok || peer.conn == nil {
		return 0, nil, fmt.Errorf("ws node %s not connected", nodeID)
	}

	reqID := fmt.Sprintf("m%d-%d", time.Now().UnixNano(), genSeq())
	ch := make(chan wsProbeResult, 1)
	s.probeMu.Lock()
	s.probeWait[reqID] = ch
	s.probeMu.Unlock()
	defer func() {
		s.probeMu.Lock()
		delete(s.probeWait, reqID)
		s.probeMu.Unlock()
	}()

	payload := wsProbeRequest{RequestID: reqID, APIType: apiType, Raw: raw, Query: query}
	s.sendJSON(peer.conn, wsMessage{Type: "probe", NodeID: nodeID, TS: time.Now().Unix(), Data: mustRaw(payload)})

	s.statMu.Lock()
	s.totalReqs++
	s.statMu.Unlock()

	select {
	case res := <-ch:
		if res.Error != "" {
			s.statMu.Lock()
			s.errReqs++
			s.statMu.Unlock()
			return 0, nil, fmt.Errorf("ws probe error: %s", res.Error)
		}
		body := res.Body
		// body 为 JSON 字符串时按原文透传（兼容文本响应）
		if len(body) > 0 && body[0] == '"' {
			var str string
			if json.Unmarshal(body, &str) == nil {
				body = []byte(str)
			}
		}
		return res.Status, body, nil
	case <-time.After(timeout):
		s.statMu.Lock()
		s.errReqs++
		s.statMu.Unlock()
		return 0, nil, fmt.Errorf("ws probe timeout for node %s", nodeID)
	}
}

// Start 启动 WS 服务端（阻塞）。
func (s *wsServer) Start(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.Handler)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("[ws] server listening on %s/ws", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Printf("[ws] server stopped: %v", err)
	}
}

// maintenanceLoop 心跳 + 状态上报（middleware → 节点）。
func (s *wsServer) maintenanceLoop() {
	for {
		time.Sleep(20 * time.Second)
		s.mu.Lock()
		peers := make([]*wsPeer, 0, len(s.peers))
		for _, p := range s.peers {
			peers = append(peers, p)
		}
		s.mu.Unlock()

		s.statMu.Lock()
		stats := struct {
			UptimeSeconds int64 `json:"uptimeSeconds"`
			TotalRequests int64 `json:"totalRequests"`
			ErrorRequests int64 `json:"errorRequests"`
			ConnectedAt   int64 `json:"connectedAt"`
		}{
			UptimeSeconds: int64(time.Since(s.startedAt).Seconds()),
			TotalRequests: s.totalReqs,
			ErrorRequests: s.errReqs,
		}
		s.statMu.Unlock()

		for _, p := range peers {
			// 心跳
			s.sendJSON(p.conn, wsMessage{Type: "ping", NodeID: p.id, TS: time.Now().Unix()})
			// 状态/统计上报
			s.sendJSON(p.conn, wsMessage{Type: "status", NodeID: p.id, TS: time.Now().Unix(), Data: mustRaw(stats)})
		}
	}
}

// genSeq 自增序号（进程内）。
var seqMu sync.Mutex
var seq int64

func genSeq() int64 {
	seqMu.Lock()
	defer seqMu.Unlock()
	seq++
	return seq
}

func mustRaw(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("{}")
	}
	return b
}

// ==================== 指令处理与超时 ====================

// wsProbeTimeout 拨测超时：复用 HTTP_TIMEOUT（默认 30s）
func wsProbeTimeout() time.Duration {
	t := HTTP_TIMEOUT
	if t <= 0 {
		t = 30
	}
	return time.Duration(t) * time.Second
}
