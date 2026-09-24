package webtest

// EO（EdgeOne）特供版的 DNS 模块：与主线 ipw-cn/src/webtest/dns.go 同源，
// 但保留 EO 部署形态的两点差异：
//  1. DoH / UDP 双通道互备（单地址配置下主通道失败自动回退另一通道）；
//  2. 查询耗时沿调用链透出（float64 毫秒），ResolveIP 供 tcping / speed 复用。
// 主线对 dns.go 的行为修正（Rcode 透出、服务器地址格式校验）已同步至此。

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// 腾讯云标准 DoH 端点
var dohEndpoint string = "https://doh.pub/dns-query"

// UDP DNS 服务器（DoH 不可用时的回退；dns-server 配置为 ip:port 时作为主通道）
var dnsServer = "119.28.28.28:53"

// dnsMode 主通道：dns-server 配置为 URL 时为 "doh"，为 ip:port 时为 "udp"
var dnsMode = "doh"

// dnsServers 普通 DNS 查询服务器列表（主从：第一个为主，主失败自动切换后续从服务器）
var dnsServers []string

// dnssecServers DNSSEC 专用服务器列表（未配置时沿用 dnsServers）
var dnssecServers []string

// defaultUDPServer 兜底 UDP 服务器
const defaultUDPServer = "119.28.28.28:53"

// DNSResult 统一的 DNS 查询结果格式。
//
// Rcode / RcodeText 与主线及各节点保持一致：解析器明确应答 NXDOMAIN 时，属于「域名不存在」
// 这个业务结论，而不是节点故障 —— 前端据此区分两者，别把负响应误显示成拨测失败。
type DNSResult struct {
	Domain    string   `json:"domain"`
	Duration  float64  `json:"duration"`
	Record    []string `json:"record"`
	TTL       uint32   `json:"ttl"`
	Rcode     int      `json:"rcode"`      // 解析器应答的 Rcode（0=Success, 3=NXDOMAIN）
	RcodeText string   `json:"rcode_text"` // Rcode 可读名称，如 NXDOMAIN / SERVFAIL
}

// SetDNSServer 设置 DNS 服务器：支持逗号分隔多地址主从 failover（第一个为主，主失败自动切换后续从服务器）。
// 每项支持 "ip:port"（UDP）或 DoH URL http(s)://...；单地址时保持原有语义。
func SetDNSServer(server string) {
	if server == "" {
		return
	}
	// dns-server 既能由本地管理面 PATCH 改，也能由远端托管配置下发 —— 都是外部可控输入。
	// 在这里先挡掉畸形值，总比留到查询时才以"莫名其妙的超时/报错"暴露好排查；
	// 全部非法时保留原有配置（与"空串不覆盖"的既有语义一致）。
	list := validServers(splitServers(server))
	if len(list) == 0 {
		slog.Warn("all DNS server addresses invalid, keeping previous config", "server", server)
		return
	}
	dnsServers = list
	// 兼容单地址的旧语义：URL → DoH 主通道；ip:port → UDP 主通道
	if len(list) == 1 {
		if strings.HasPrefix(list[0], "http://") || strings.HasPrefix(list[0], "https://") {
			dohEndpoint = list[0]
			dnsMode = "doh"
		} else {
			dnsServer = list[0]
			dnsMode = "udp"
		}
	}
}

// SetDNSSecServer 设置 DNSSEC 专用 DNS 服务器（逗号分隔主从；留空 = 沿用 dns-server 配置）
func SetDNSSecServer(server string) {
	if list := validServers(splitServers(server)); len(list) > 0 {
		dnssecServers = list
	}
}

// splitServers 逗号分隔解析为地址列表（去空白、去空项）
func splitServers(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// validServers 过滤掉格式非法的地址（保留原顺序）。
// 全部非法时返回空列表，调用方据此保留原有配置。
func validServers(list []string) []string {
	out := make([]string, 0, len(list))
	for _, s := range list {
		if err := validateServerAddr(s); err != nil {
			slog.Warn("ignored invalid DNS server address", "server", s, "error", err)
			continue
		}
		out = append(out, s)
	}
	return out
}

// validateServerAddr 校验单个 DNS 服务器地址的格式：
//   - DoH URL：必须是 http/https 且 host 非空（**不限制私网** —— 节点自建内网 DoH 是合理部署）；
//   - UDP/TCP：必须是 host:port 且端口合法。
//
// 注意：UDP 形式**不禁止私有地址** —— 节点上跑本地缓存解析器（127.0.0.1:53、内网 10.x:53）是常见部署，
// 它是节点侧基础设施配置、不是被拨测目标，一并堵掉会直接让 DNS 拨测失效。
func validateServerAddr(server string) error {
	if strings.HasPrefix(server, "http://") || strings.HasPrefix(server, "https://") {
		u, err := url.Parse(server)
		if err != nil {
			return err
		}
		if u.Hostname() == "" {
			return fmt.Errorf("missing host")
		}
		return nil
	}
	host, port, err := net.SplitHostPort(server)
	if err != nil {
		return err
	}
	if strings.TrimSpace(host) == "" {
		return fmt.Errorf("missing host")
	}
	if p, err := strconv.Atoi(port); err != nil || p < 1 || p > 65535 {
		return fmt.Errorf("invalid port %q", port)
	}
	return nil
}

// ==========================================
// ⭐️ 底层核心：DoH + UDP 双通道 DNS 查询
// ==========================================

// queryDNSMsg 普通 DNS 查询：主从 failover——按配置顺序依次尝试每个服务器，
// 当前服务器查询失败（网络错误/超时）自动切换下一个；全部失败返回最后一个错误。
// 单个服务器：URL 走 DoH，ip:port 走 UDP。
func queryDNSMsg(msg *dns.Msg) (*dns.Msg, float64, error) {
	servers := dnsServers
	if len(servers) == 0 {
		// 未配置主从列表时回退单地址语义（保持原 DoH/UDP 双通道互备）
		if dnsMode == "doh" {
			resp, dur, err := queryDoHMsg(msg, dohEndpoint)
			if err == nil {
				return resp, dur, nil
			}
			slog.Warn("DoH query failed, falling back to UDP", "endpoint", dohEndpoint, "error", err)
			return queryUDPMsg(msg, dnsServer)
		}
		resp, dur, err := queryUDPMsg(msg, dnsServer)
		if err == nil {
			return resp, dur, nil
		}
		slog.Warn("UDP query failed, falling back to DoH", "server", dnsServer, "error", err)
		return queryDoHMsg(msg, dohEndpoint)
	}
	return queryWithServers(msg, servers)
}

// queryDNSSECMsg DNSSEC 查询：优先使用专用服务器（dnssec-server），未配置则沿用普通 dns-server
func queryDNSSECMsg(msg *dns.Msg) (*dns.Msg, float64, error) {
	if len(dnssecServers) > 0 {
		return queryWithServers(msg, dnssecServers)
	}
	return queryDNSMsg(msg)
}

// queryWithServers 依次尝试服务器列表，主失败切从；每个服务器按类型走 DoH 或 UDP
func queryWithServers(msg *dns.Msg, servers []string) (*dns.Msg, float64, error) {
	if len(servers) == 0 {
		servers = []string{defaultUDPServer}
	}
	var lastErr error
	for i, srv := range servers {
		var resp *dns.Msg
		var dur float64
		var err error
		if strings.HasPrefix(srv, "http://") || strings.HasPrefix(srv, "https://") {
			resp, dur, err = queryDoHMsg(msg, srv)
		} else {
			resp, dur, err = queryUDPMsg(msg, srv)
		}
		if err == nil {
			return resp, dur, nil
		}
		lastErr = err
		if i < len(servers)-1 {
			slog.Warn("DNS query failed, switching to next server", "server", srv, "error", err)
		}
	}
	slog.Warn("all DNS servers failed", "servers", servers, "error", lastErr)
	return nil, 0, lastErr
}

// queryUDPMsg 通过 UDP/TCP 向指定 DNS 服务器发送查询（miekg/dns 自动处理大响应切 TCP）
func queryUDPMsg(msg *dns.Msg, server string) (*dns.Msg, float64, error) {
	client := &dns.Client{Timeout: 5 * time.Second}
	start := time.Now()
	resp, _, err := client.Exchange(msg, server)
	duration := time.Since(start).Seconds() * 1000
	if err != nil {
		return nil, duration, err
	}
	return resp, duration, nil
}

// queryDoHMsg 通过 DoH（RFC 8484，POST application/dns-message）向指定端点发送查询。
//
// 这里**刻意不做私网/回环拦截**（与 website / ssl / speed 那三处不同）：
// endpoint 来自 dns-server 配置，而节点上自建内网 DoH（127.0.0.1 / 10.x）是合理部署，
// 堵私网会直接打死功能；配置又是管理员凭据才能改的，不构成匿名可达攻击面。
// 地址**格式**校验已在 SetDNSServer 入口由 validateServerAddr 完成（http/https + host 非空）。
func queryDoHMsg(msg *dns.Msg, endpoint string) (*dns.Msg, float64, error) {
	packedMsg, err := msg.Pack()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to pack DNS message: %v", err)
	}
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(packedMsg))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")
	client := &http.Client{Timeout: 5 * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start).Seconds() * 1000
	if err != nil {
		return nil, duration, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, duration, fmt.Errorf("DoH API returned status %d", resp.StatusCode)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, duration, fmt.Errorf("failed to read response body: %v", err)
	}
	responseMsg := new(dns.Msg)
	if err := responseMsg.Unpack(bodyBytes); err != nil {
		return nil, duration, fmt.Errorf("failed to unpack DNS response: %v", err)
	}
	return responseMsg, duration, nil
}

// ==========================================
// 业务层：构造查询并经双通道执行，提取特定记录
// ==========================================

// executeDoHQuery 构造标准 DNS 请求报文并经双通道执行。
//
// **不再把 Rcode 非 Success 当错误**：解析器明确应答（哪怕 NXDOMAIN / SERVFAIL）说明网络与
// 解析链路是通的，属合法业务结果，由上层透出 Rcode；只有传输/网络层失败才返回 error。
// 若在此处返回 error，上层 dnsQueryHandler 会回 500，前端把「域名不存在」显示成「节点故障」。
func executeDoHQuery(domain string, qtype uint16) (*dns.Msg, float64, error) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), qtype)
	msg.RecursionDesired = true // 请求递归解析

	return queryDNSMsg(msg)
}

// resolveTyped 通用记录查询：成功收到解析器应答（即便 Rcode 非 Success，如 NXDOMAIN/SERVFAIL）
// 即视为一次有效查询，将 Rcode 透出并返回 200；仅在传输/网络层失败（超时、服务器不可达、
// DoH 异常）时才返回 error（上层据此返回 500）。避免把合法的负响应误判为节点故障。
// extract 从一个 RR 中抽取记录字符串（TXT 可能含多条），返回 (values, ttl, matched)。
func resolveTyped(domain, name string, qtype uint16, extract func(dns.RR) ([]string, uint32, bool)) (DNSResult, error) {
	result := DNSResult{Domain: domain, Record: []string{}}
	response, duration, err := executeDoHQuery(name, qtype)
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query DNS", "domain", domain, "qtype", qtype, "error", err)
		return result, err
	}

	// 解析器已应答：无论 Rcode 是否为 Success，都属合法结果，不再当 error 返回
	result.Rcode = response.Rcode
	result.RcodeText = dns.RcodeToString[response.Rcode]
	for _, ans := range response.Answer {
		if values, ttl, ok := extract(ans); ok {
			result.Record = append(result.Record, values...)
			if result.TTL == 0 {
				result.TTL = ttl
			}
		}
	}
	return result, nil
}

func ResolveARecord(domain string) (DNSResult, error) {
	return resolveTyped(domain, domain, dns.TypeA, func(rr dns.RR) ([]string, uint32, bool) {
		if a, ok := rr.(*dns.A); ok {
			return []string{a.A.String()}, a.Header().Ttl, true
		}
		return nil, 0, false
	})
}

func ResolveAAAARecord(domain string) (DNSResult, error) {
	return resolveTyped(domain, domain, dns.TypeAAAA, func(rr dns.RR) ([]string, uint32, bool) {
		if a, ok := rr.(*dns.AAAA); ok {
			return []string{a.AAAA.String()}, a.Header().Ttl, true
		}
		return nil, 0, false
	})
}

func ResolveTXTRecord(domain string) (DNSResult, error) {
	return resolveTyped(domain, domain, dns.TypeTXT, func(rr dns.RR) ([]string, uint32, bool) {
		if txt, ok := rr.(*dns.TXT); ok {
			// TXT 一条 RR 可能含多段字符串，保持原有「逐段单独成项」的行为
			return txt.Txt, txt.Header().Ttl, true
		}
		return nil, 0, false
	})
}

func ResolveNSRecord(domain string) (DNSResult, error) {
	return resolveTyped(domain, domain, dns.TypeNS, func(rr dns.RR) ([]string, uint32, bool) {
		if ns, ok := rr.(*dns.NS); ok {
			return []string{ns.Ns}, ns.Header().Ttl, true
		}
		return nil, 0, false
	})
}

func ResolveCNAMERecord(domain string) (DNSResult, error) {
	return resolveTyped(domain, domain, dns.TypeCNAME, func(rr dns.RR) ([]string, uint32, bool) {
		if cname, ok := rr.(*dns.CNAME); ok {
			return []string{cname.Target}, cname.Header().Ttl, true
		}
		return nil, 0, false
	})
}

func ResolveMXRecord(domain string) (DNSResult, error) {
	return resolveTyped(domain, domain, dns.TypeMX, func(rr dns.RR) ([]string, uint32, bool) {
		if mx, ok := rr.(*dns.MX); ok {
			return []string{mx.Mx}, mx.Header().Ttl, true
		}
		return nil, 0, false
	})
}

func ResolveSRVRecord(domain string) (DNSResult, error) {
	return resolveTyped(domain, domain, dns.TypeSRV, func(rr dns.RR) ([]string, uint32, bool) {
		if srv, ok := rr.(*dns.SRV); ok {
			return []string{srv.Target}, srv.Header().Ttl, true
		}
		return nil, 0, false
	})
}

func ResolveCAARecord(domain string) (DNSResult, error) {
	return resolveTyped(domain, domain, dns.TypeCAA, func(rr dns.RR) ([]string, uint32, bool) {
		if caa, ok := rr.(*dns.CAA); ok {
			return []string{caa.Value}, caa.Header().Ttl, true
		}
		return nil, 0, false
	})
}

// ResolvePTRRecord 反查 IP 的 PTR 记录（返回的 Domain 保持为原始 IP，便于前端直接展示）
func ResolvePTRRecord(ip string) (DNSResult, error) {
	ptrName, err := dns.ReverseAddr(ip)
	if err != nil {
		slog.Warn("Invalid IP address for PTR query", "ip", ip, "error", err)
		return DNSResult{Domain: ip, Record: []string{}}, fmt.Errorf("invalid IP address: %v", err)
	}
	return resolveTyped(ip, ptrName, dns.TypePTR, func(rr dns.RR) ([]string, uint32, bool) {
		if ptr, ok := rr.(*dns.PTR); ok {
			return []string{ptr.Ptr}, ptr.Header().Ttl, true
		}
		return nil, 0, false
	})
}

// ResolveIP 通过双通道解析域名，返回指定版本（v4/v6）的 IP 地址字符串。
// EO 特供：供 tcping / speed 等需要"先把 host 归一化成 IP"的拨测复用。
func ResolveIP(host string, version string) (string, error) {
	if ip := net.ParseIP(host); ip != nil {
		if version == "v4" && ip.To4() != nil {
			return ip.String(), nil
		}
		if version == "v6" && ip.To4() == nil && ip.To16() != nil {
			return ip.String(), nil
		}
		return "", fmt.Errorf("no %s address found for %s", version, host)
	}

	var qtype uint16
	switch version {
	case "v6":
		qtype = dns.TypeAAAA
	default:
		qtype = dns.TypeA
	}

	responseMsg, _, err := executeDoHQuery(host, qtype)
	if err != nil {
		return "", err
	}

	for _, ans := range responseMsg.Answer {
		switch v := ans.(type) {
		case *dns.A:
			if qtype == dns.TypeA {
				return v.A.String(), nil
			}
		case *dns.AAAA:
			if qtype == dns.TypeAAAA {
				return v.AAAA.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no %s record found for %s", version, host)
}

// ==========================================
// 并发查询所有记录
// ==========================================

type DNSFullResult struct {
	Domain string    `json:"domain"`
	A      DNSResult `json:"a"`
	AAAA   DNSResult `json:"aaaa"`
	CNAME  DNSResult `json:"cname"`
	MX     DNSResult `json:"mx"`
	NS     DNSResult `json:"ns"`
	TXT    DNSResult `json:"txt"`
	SRV    DNSResult `json:"srv"`
	CAA    DNSResult `json:"caa"`
}

func ResolveARecordllDNSRecords(domain string) DNSFullResult {
	result := DNSFullResult{Domain: domain}
	var wg sync.WaitGroup
	wg.Add(8)

	go func() { defer wg.Done(); result.A, _ = ResolveARecord(domain) }()
	go func() { defer wg.Done(); result.AAAA, _ = ResolveAAAARecord(domain) }()
	go func() { defer wg.Done(); result.CNAME, _ = ResolveCNAMERecord(domain) }()
	go func() { defer wg.Done(); result.MX, _ = ResolveMXRecord(domain) }()
	go func() { defer wg.Done(); result.NS, _ = ResolveNSRecord(domain) }()
	go func() { defer wg.Done(); result.TXT, _ = ResolveTXTRecord(domain) }()
	go func() { defer wg.Done(); result.SRV, _ = ResolveSRVRecord(domain) }()
	go func() { defer wg.Done(); result.CAA, _ = ResolveCAARecord(domain) }()

	wg.Wait()
	return result
}
