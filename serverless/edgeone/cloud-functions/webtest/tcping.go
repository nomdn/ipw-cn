package webtest

import (
	"fmt"
	"math"
	"net"
	"strings"
	"time"

	"edgeone-cloud-functions/ssrf"
)

type TCPingResult struct {
	IP        string    `json:"ip"`
	Port      string    `json:"port"`
	Success   bool      `json:"success"`
	RTT       float64   `json:"rtt"`
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}

type TCPingStats struct {
	IP       string         `json:"ip"`
	Port     string         `json:"port"`
	Sent     int            `json:"sent"`
	Success  int            `json:"success"`
	LossRate float64        `json:"loss_rate"`
	MaxRTT   float64        `json:"max_rtt"`
	MinRTT   float64        `json:"min_rtt"`
	AvgRTT   float64        `json:"avg_rtt"`
	Results  []TCPingResult `json:"results"`
}

// ResolveHost 解析 host 为指定版本（v4/v6）的 IP 地址字符串。
// 字面 IP 不查 DNS，但仍要过私网校验（与 DNS 解析结果同口径）。
func ResolveHost(host string, version string) (string, error) {
	return ResolveHostWith(host, version, ReqResolve{})
}

// ResolveHostWith 同 ResolveHost，但带请求级解析配置（req 零值 = 用配置 DNS）。
// 强制解析（resolve=）命中就把目标当作新的 host——IP 交给下面的字面 IP 分支，域名继续走 DNS。
func ResolveHostWith(host string, version string, req ReqResolve) (string, error) {
	// 强制解析：命中就把目标当作新的 host——IP 交给下面的字面 IP 分支，域名继续走 DNS。
	if target := req.Override(host, version == "v6"); target != "" {
		host = strings.Trim(target, "[]")
	}
	cleanHost := strings.Trim(host, "[]")
	if ip := net.ParseIP(cleanHost); ip != nil {
		// SSRF：字面 IP 与 DNS 解析结果同口径，私网地址一律拒绝（tcping 是裸 TCP，
		// 不经过 DialContext，这里不做就完全没人拦）。放在族校验之前：安全判定
		// 优先于参数语义，也与 DialContext 里「先查私网、再拨号」的顺序一致。
		if ssrf.Enabled() && ssrf.IsPrivateIP(ip) {
			return "", fmt.Errorf("connection to private/internal address %s is not allowed", cleanHost)
		}
		switch version {
		case "v4":
			if ip.To4() != nil {
				return ip.String(), nil
			}
		case "v6":
			if ip.To4() == nil {
				return ip.String(), nil
			}
		}
		return "", fmt.Errorf("host %s is not a %s address", host, version)
	}

	var result DNSResult
	var err error
	if version == "v6" {
		result, err = ResolveAAAARecord(host, req.Servers("ipv6"))
	} else {
		result, err = ResolveARecord(host, req.Servers("ipv4"))
	}
	if err != nil {
		return "", fmt.Errorf("DNS lookup failed: %w", err)
	}
	if len(result.Record) == 0 {
		return "", fmt.Errorf("no %s address found for %s", version, host)
	}
	ipStr := result.Record[0]

	if ssrf.Enabled() {
		if ip := net.ParseIP(ipStr); ip != nil && ssrf.IsPrivateIP(ip) {
			return "", fmt.Errorf("connection to private/internal address %s is not allowed", ipStr)
		}
	}
	return ipStr, nil
}

// TCPing 执行单次 TCP 连接测试
// 参数：
//   - host: 目标主机名或 IP 地址
//   - port: 目标端口号
//   - version: 协议版本，"v4" 或 "v6"
//   - timeout: 连接超时时间
//
// 返回 TCPingResult 包含连接结果和响应时间
func TCPing(host string, port string, version string, timeout time.Duration) (*TCPingResult, error) {
	ip, err := ResolveHost(host, version)
	if err != nil {
		return nil, err
	}
	return tcpingDial(ip, port, version, timeout), nil
}

// tcpingDial 对已解析好的 IP 做一次 TCP 拨号，不再做 DNS 解析与 SSRF 校验。
// 拆出来是为了让 TCPingRun 的多次探测复用同一个解析结果：原实现每轮都走 TCPing→
// ResolveHost，既拖慢，又会因多 A 记录轮询导致 results 里每次连到不同 IP（与顶层 ip 不符）。
func tcpingDial(ip string, port string, version string, timeout time.Duration) *TCPingResult {
	addr := ""
	switch version {
	case "v6":
		addr = fmt.Sprintf("[%s]:%s", ip, port)
	default:
		addr = fmt.Sprintf("%s:%s", ip, port)
	}

	start := time.Now()

	conn, err := net.DialTimeout("tcp", addr, timeout)
	rtt := time.Since(start)

	result := &TCPingResult{
		IP:        ip,
		Port:      port,
		Timestamp: start,
	}

	if err != nil {
		result.Success = false
		result.RTT = -1
		result.Error = err.Error()
	} else {
		conn.Close()
		result.Success = true
		result.RTT = float64(rtt.Microseconds()) / 1000.0
	}

	return result
}

// TCPingRun 执行多次 TCP 连接测试并返回统计结果
// 参数：
//   - host: 目标主机名或 IP 地址
//   - port: 目标端口号
//   - count: 测试次数
//   - version: 协议版本，"v4" 或 "v6"
//   - timeout: 每次连接的超时时间
//   - interval: 两次测试之间的间隔时间
//
// 返回 TCPingStats 包含统计信息
func TCPingRun(host string, port string, count int, version string, timeout time.Duration, interval time.Duration) (*TCPingStats, error) {
	return TCPingRunWith(host, port, count, version, timeout, interval, ReqResolve{})
}

// TCPingRunWith 同 TCPingRun，但带请求级解析配置（req 零值 = 用配置 DNS）。
func TCPingRunWith(host string, port string, count int, version string, timeout time.Duration, interval time.Duration, req ReqResolve) (*TCPingStats, error) {
	ip, err := ResolveHostWith(host, version, req)
	if err != nil {
		return &TCPingStats{
			IP:       "Error: " + err.Error(),
			Port:     port,
			Sent:     count,
			Success:  0,
			Results:  nil,
			MinRTT:   -1,
			MaxRTT:   -1,
			AvgRTT:   -1,
			LossRate: 100,
		}, nil
	}

	stats := &TCPingStats{
		IP:      ip,
		Port:    port,
		Sent:    count,
		MinRTT:  math.MaxFloat64,
		Results: make([]TCPingResult, 0, count),
	}

	var totalRTT float64
	successCount := 0

	for i := 0; i < count; i++ {
		// 复用开头解析出的 ip，不再每轮重新 ResolveHost
		result := tcpingDial(ip, port, version, timeout)

		stats.Results = append(stats.Results, *result)

		if result.Success {
			successCount++
			totalRTT += result.RTT
			if result.RTT > stats.MaxRTT {
				stats.MaxRTT = result.RTT
			}
			if result.RTT < stats.MinRTT {
				stats.MinRTT = result.RTT
			}
		}

		if i < count-1 && interval > 0 {
			time.Sleep(interval)
		}
	}

	stats.Success = successCount
	stats.LossRate = math.Round(float64(count-successCount)*10000/float64(count)) / 100

	if successCount > 0 {
		stats.AvgRTT = math.Round(totalRTT*100/float64(successCount)) / 100
	} else {
		stats.MinRTT = -1
		stats.MaxRTT = -1
		stats.AvgRTT = -1
	}

	return stats, nil
}
