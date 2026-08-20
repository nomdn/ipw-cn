package webtest

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
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

type DNSResult struct {
	Domain   string   `json:"domain"`
	Duration float64  `json:"duration"`
	Record   []string `json:"record"`
	TTL      uint32   `json:"ttl"`
}

// SetDNSServer 设置 DNS 服务器：支持逗号分隔多地址主从 failover（第一个为主，主失败自动切换后续从服务器）。
// 每项支持 "ip:port"（UDP）或 DoH URL http(s)://...；单地址时保持原有语义。
func SetDNSServer(server string) {
	if server == "" {
		return
	}
	list := splitServers(server)
	dnsServers = list
	// 兼容单地址的旧语义：URL → DoH 主通道；ip:port → UDP 主通道
	if len(list) == 1 {
		if strings.HasPrefix(server, "http://") || strings.HasPrefix(server, "https://") {
			dohEndpoint = server
			dnsMode = "doh"
		} else {
			dnsServer = server
			dnsMode = "udp"
		}
	}
}

// SetDNSSecServer 设置 DNSSEC 专用 DNS 服务器（逗号分隔主从；留空 = 沿用 dns-server 配置）
func SetDNSSecServer(server string) {
	if list := splitServers(server); len(list) > 0 {
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

// defaultUDPServer 兜底 UDP 服务器
const defaultUDPServer = "119.28.28.28:53"

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

// queryDoHMsg 通过 DoH（RFC 8484，POST application/dns-message）向指定端点发送查询
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

func executeDoHQuery(domain string, qtype uint16) (*dns.Msg, float64, error) {
	// 1. 构造标准的 DNS 请求报文
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), qtype)
	msg.RecursionDesired = true // 请求递归解析

	// 2. 双通道查询（DoH 主 + UDP 备，或反之）
	responseMsg, duration, err := queryDNSMsg(msg)
	if err != nil {
		return nil, duration, err
	}

	// 3. 检查 DNS 响应码 (Rcode)
	if responseMsg.Rcode != dns.RcodeSuccess {
		return responseMsg, duration, fmt.Errorf("DNS query failed with Rcode %d", responseMsg.Rcode)
	}

	return responseMsg, duration, nil
}

// ==========================================
// 业务层：调用底层函数并提取特定记录
// ==========================================

func ResolveARecord(domain string) (DNSResult, error) {
	result := DNSResult{Domain: domain, Record: []string{}}
	responseMsg, duration, err := executeDoHQuery(domain, dns.TypeA)
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query A", "domain", domain, "err", err)
		return result, err
	}
	for _, ans := range responseMsg.Answer {
		if aRecord, ok := ans.(*dns.A); ok {
			result.Record = append(result.Record, aRecord.A.String())
			if result.TTL == 0 {
				result.TTL = aRecord.Header().Ttl
			}
		}
	}
	return result, nil
}

func ResolveAAAARecord(domain string) (DNSResult, error) {
	result := DNSResult{Domain: domain, Record: []string{}}
	responseMsg, duration, err := executeDoHQuery(domain, dns.TypeAAAA)
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query AAAA", "domain", domain, "err", err)
		return result, err
	}
	for _, ans := range responseMsg.Answer {
		if aRecord, ok := ans.(*dns.AAAA); ok {
			result.Record = append(result.Record, aRecord.AAAA.String())
			if result.TTL == 0 {
				result.TTL = aRecord.Header().Ttl
			}
		}
	}
	return result, nil
}

func ResolveTXTRecord(domain string) (DNSResult, error) {
	result := DNSResult{Domain: domain, Record: []string{}}
	responseMsg, duration, err := executeDoHQuery(domain, dns.TypeTXT)
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query TXT", "domain", domain, "err", err)
		return result, err
	}
	for _, ans := range responseMsg.Answer {
		if txtRecord, ok := ans.(*dns.TXT); ok {
			result.Record = append(result.Record, txtRecord.Txt...)
			if result.TTL == 0 {
				result.TTL = txtRecord.Header().Ttl
			}
		}
	}
	return result, nil
}

func ResolveNSRecord(domain string) (DNSResult, error) {
	result := DNSResult{Domain: domain, Record: []string{}}
	responseMsg, duration, err := executeDoHQuery(domain, dns.TypeNS)
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query NS", "domain", domain, "err", err)
		return result, err
	}
	for _, ans := range responseMsg.Answer {
		if nsRecord, ok := ans.(*dns.NS); ok {
			result.Record = append(result.Record, nsRecord.Ns)
			if result.TTL == 0 {
				result.TTL = nsRecord.Header().Ttl
			}
		}
	}
	return result, nil
}

func ResolveCNAMERecord(domain string) (DNSResult, error) {
	result := DNSResult{Domain: domain, Record: []string{}}
	responseMsg, duration, err := executeDoHQuery(domain, dns.TypeCNAME)
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query CNAME", "domain", domain, "err", err)
		return result, err
	}
	for _, ans := range responseMsg.Answer {
		if cnameRecord, ok := ans.(*dns.CNAME); ok {
			result.Record = append(result.Record, cnameRecord.Target)
			if result.TTL == 0 {
				result.TTL = cnameRecord.Header().Ttl
			}
		}
	}
	return result, nil
}

func ResolveMXRecord(domain string) (DNSResult, error) {
	result := DNSResult{Domain: domain, Record: []string{}}
	responseMsg, duration, err := executeDoHQuery(domain, dns.TypeMX)
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query MX", "domain", domain, "err", err)
		return result, err
	}
	for _, ans := range responseMsg.Answer {
		if mxRecord, ok := ans.(*dns.MX); ok {
			result.Record = append(result.Record, mxRecord.Mx)
			if result.TTL == 0 {
				result.TTL = mxRecord.Header().Ttl
			}
		}
	}
	return result, nil
}

func ResolveSRVRecord(domain string) (DNSResult, error) {
	result := DNSResult{Domain: domain, Record: []string{}}
	responseMsg, duration, err := executeDoHQuery(domain, dns.TypeSRV)
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query SRV", "domain", domain, "err", err)
		return result, err
	}
	for _, ans := range responseMsg.Answer {
		if srvRecord, ok := ans.(*dns.SRV); ok {
			result.Record = append(result.Record, srvRecord.Target)
			if result.TTL == 0 {
				result.TTL = srvRecord.Header().Ttl
			}
		}
	}
	return result, nil
}

func ResolvePTRRecord(ip string) (DNSResult, error) {
	result := DNSResult{Domain: ip, Record: []string{}}
	ptrName, err := dns.ReverseAddr(ip)
	if err != nil {
		return result, fmt.Errorf("invalid IP: %v", err)
	}

	responseMsg, duration, err := executeDoHQuery(ptrName, dns.TypePTR)
	result.Duration = duration
	result.Domain = ip // 保持返回的 Domain 为原始 IP

	if err != nil {
		slog.Warn("Failed to query PTR", "ip", ip, "err", err)
		return result, err
	}
	for _, ans := range responseMsg.Answer {
		if ptrRecord, ok := ans.(*dns.PTR); ok {
			result.Record = append(result.Record, ptrRecord.Ptr)
			if result.TTL == 0 {
				result.TTL = ptrRecord.Header().Ttl
			}
		}
	}
	return result, nil
}

func ResolveCAARecord(domain string) (DNSResult, error) {
	result := DNSResult{Domain: domain, Record: []string{}}
	responseMsg, duration, err := executeDoHQuery(domain, dns.TypeCAA)
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query CAA", "domain", domain, "err", err)
		return result, err
	}
	for _, ans := range responseMsg.Answer {
		if caaRecord, ok := ans.(*dns.CAA); ok {
			result.Record = append(result.Record, caaRecord.Value)
			if result.TTL == 0 {
				result.TTL = caaRecord.Header().Ttl
			}
		}
	}
	return result, nil
}

// ResolveIP 通过 DoH 解析域名，返回指定版本（v4/v6）的 IP 地址字符串
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
