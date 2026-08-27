package webtest

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

var (
	// dnsServers 普通 DNS 查询服务器列表（主从：第一个为主，主失败自动切换后续从服务器）
	dnsServers = []string{"119.28.28.28:53"}
	// dnssecServers DNSSEC 专用服务器列表（未配置时沿用 dnsServers）
	dnssecServers []string
)

// SetDNSServer 设置DNS服务器地址（逗号分隔多地址：第一个为主，主失败自动切换后续从服务器）。
// 每项支持 "ip:port"（UDP）或 DoH URL http(s)://...
func SetDNSServer(server string) {
	if list := splitServers(server); len(list) > 0 {
		dnsServers = list
	}
}

// SetDNSSecServer 设置DNSSEC专用DNS服务器（逗号分隔主从；留空 = 沿用 dns-server 配置）
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

const (
	defaultUDPServer   = "119.28.28.28:53"
	defaultDoHEndpoint = "https://doh.pub/dns-query"
)

// queryDNS 普通 DNS 查询：主从 failover——按配置顺序依次尝试每个服务器，
// 当前服务器查询失败（网络错误/超时）自动切换下一个；全部失败返回最后一个错误。
func queryDNS(msg *dns.Msg) (*dns.Msg, error) {
	return queryWithServers(msg, dnsServers)
}

// queryDNSSEC DNSSEC 查询：优先使用专用服务器（dnssec-server），未配置则沿用普通 dns-server
func queryDNSSEC(msg *dns.Msg) (*dns.Msg, error) {
	if len(dnssecServers) > 0 {
		return queryWithServers(msg, dnssecServers)
	}
	return queryWithServers(msg, dnsServers)
}

// queryWithServers 依次尝试服务器列表，主失败切从
func queryWithServers(msg *dns.Msg, servers []string) (*dns.Msg, error) {
	if len(servers) == 0 {
		servers = []string{defaultUDPServer}
	}
	var lastErr error
	for i, srv := range servers {
		resp, err := queryOne(msg, srv)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if i < len(servers)-1 {
			slog.Warn("DNS query failed, switching to next server", "server", srv, "error", err)
		}
	}
	slog.Warn("all DNS servers failed", "servers", servers, "error", lastErr)
	return nil, lastErr
}

// queryOne 向单个 DNS 服务器查询：配置为 URL（http/https）走 DoH，否则走 UDP
func queryOne(msg *dns.Msg, server string) (*dns.Msg, error) {
	if strings.HasPrefix(server, "http://") || strings.HasPrefix(server, "https://") {
		return queryDoHMsg(msg, server)
	}
	return queryUDPMsg(msg, server)
}

// queryUDPMsg 通过 UDP/TCP 向指定 DNS 服务器发送查询（miekg/dns 自动处理大响应切 TCP）
func queryUDPMsg(msg *dns.Msg, server string) (*dns.Msg, error) {
	client := &dns.Client{Timeout: 5 * time.Second}
	resp, _, err := client.Exchange(msg, server)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// queryDoHMsg 通过 DoH（RFC 8484，POST application/dns-message）向指定端点发送查询
func queryDoHMsg(msg *dns.Msg, endpoint string) (*dns.Msg, error) {
	packedMsg, err := msg.Pack()
	if err != nil {
		return nil, fmt.Errorf("failed to pack DNS message: %v", err)
	}
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(packedMsg))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DoH API returned status %d", resp.StatusCode)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	responseMsg := new(dns.Msg)
	if err := responseMsg.Unpack(bodyBytes); err != nil {
		return nil, fmt.Errorf("failed to unpack DNS response: %v", err)
	}
	return responseMsg, nil
}

// DNSResult 统一的DNS查询结果格式
type DNSResult struct {
	Domain   string   `json:"domain"`
	Record   []string `json:"record"`
	TTL      uint32   `json:"ttl"`
	Duration float64  `json:"duration"`
}

func ResolveARecord(domain string) (DNSResult, error) {
	start := time.Now()
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	result := DNSResult{Domain: domain}

	response, err := queryDNS(msg)
	duration := time.Since(start).Seconds() * 1000
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query DNS", "domain", domain, "error", err)
		result.Record = []string{}
		return result, err
	}

	if response.Rcode != dns.RcodeSuccess {
		slog.Warn("DNS query failed with Rcode", "rcode", response.Rcode)
		return result, fmt.Errorf("DNS query failed with Rcode %d", response.Rcode)
	}

	for _, ans := range response.Answer {
		if aRecord, ok := ans.(*dns.A); ok {
			result.Record = append(result.Record, aRecord.A.String())
			if result.TTL == 0 {
				result.TTL = aRecord.Header().Ttl
			}
		}
	}
	if result.Record == nil {
		result.Record = []string{}
	}
	return result, nil
}

func ResolveAAAARecord(domain string) (DNSResult, error) {
	start := time.Now()
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeAAAA)

	result := DNSResult{Domain: domain}

	response, err := queryDNS(msg)
	duration := time.Since(start).Seconds() * 1000
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query DNS", "domain", domain, "error", err)
		result.Record = []string{}
		return result, err
	}

	if response.Rcode != dns.RcodeSuccess {
		slog.Warn("DNS query failed with Rcode", "rcode", response.Rcode)
		return result, fmt.Errorf("DNS query failed with Rcode %d", response.Rcode)
	}

	for _, ans := range response.Answer {
		if aRecord, ok := ans.(*dns.AAAA); ok {
			result.Record = append(result.Record, aRecord.AAAA.String())
			if result.TTL == 0 {
				result.TTL = aRecord.Header().Ttl
			}
		}
	}
	if result.Record == nil {
		result.Record = []string{}
	}
	return result, nil
}

func ResolveTXTRecord(domain string) (DNSResult, error) {
	start := time.Now()
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeTXT)

	result := DNSResult{Domain: domain}

	response, err := queryDNS(msg)
	duration := time.Since(start).Seconds() * 1000
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query DNS", "domain", domain, "error", err)
		result.Record = []string{}
		return result, err
	}

	if response.Rcode != dns.RcodeSuccess {
		slog.Warn("DNS query failed with Rcode", "rcode", response.Rcode)
		return result, fmt.Errorf("DNS query failed with Rcode %d", response.Rcode)
	}

	for _, ans := range response.Answer {
		if aRecord, ok := ans.(*dns.TXT); ok {
			for _, txt := range aRecord.Txt {
				result.Record = append(result.Record, txt)
			}
			if result.TTL == 0 {
				result.TTL = aRecord.Header().Ttl
			}
		}
	}
	if result.Record == nil {
		result.Record = []string{}
	}
	return result, nil
}

func ResolveNSRecord(domain string) (DNSResult, error) {
	start := time.Now()
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeNS)

	result := DNSResult{Domain: domain}

	response, err := queryDNS(msg)
	duration := time.Since(start).Seconds() * 1000
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query DNS", "domain", domain, "error", err)
		result.Record = []string{}
		return result, err
	}

	if response.Rcode != dns.RcodeSuccess {
		slog.Warn("DNS query failed with Rcode", "rcode", response.Rcode)
		return result, fmt.Errorf("DNS query failed with Rcode %d", response.Rcode)
	}

	for _, ans := range response.Answer {
		if aRecord, ok := ans.(*dns.NS); ok {
			result.Record = append(result.Record, aRecord.Ns)
			if result.TTL == 0 {
				result.TTL = aRecord.Header().Ttl
			}
		}
	}
	if result.Record == nil {
		result.Record = []string{}
	}
	return result, nil
}

func ResolveCNAMERecord(domain string) (DNSResult, error) {
	start := time.Now()
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeCNAME)

	result := DNSResult{Domain: domain}

	response, err := queryDNS(msg)
	duration := time.Since(start).Seconds() * 1000
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query CNAME", "domain", domain, "error", err)
		result.Record = []string{}
		return result, err
	}

	if response.Rcode != dns.RcodeSuccess {
		slog.Warn("CNAME query failed with Rcode", "rcode", response.Rcode)
		return result, fmt.Errorf("CNAME query failed with Rcode %d", response.Rcode)
	}

	for _, ans := range response.Answer {
		if cnameRecord, ok := ans.(*dns.CNAME); ok {
			result.Record = append(result.Record, cnameRecord.Target)
			if result.TTL == 0 {
				result.TTL = cnameRecord.Header().Ttl
			}
		}
	}
	if result.Record == nil {
		result.Record = []string{}
	}
	return result, nil
}

func ResolveMXRecord(domain string) (DNSResult, error) {
	start := time.Now()
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeMX)

	result := DNSResult{Domain: domain}

	response, err := queryDNS(msg)
	duration := time.Since(start).Seconds() * 1000
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query MX", "domain", domain, "error", err)
		result.Record = []string{}
		return result, err
	}

	if response.Rcode != dns.RcodeSuccess {
		slog.Warn("MX query failed with Rcode", "rcode", response.Rcode)
		return result, fmt.Errorf("MX query failed with Rcode %d", response.Rcode)
	}

	for _, ans := range response.Answer {
		if mxRecord, ok := ans.(*dns.MX); ok {
			result.Record = append(result.Record, mxRecord.Mx)
			if result.TTL == 0 {
				result.TTL = mxRecord.Header().Ttl
			}
		}
	}
	if result.Record == nil {
		result.Record = []string{}
	}
	return result, nil
}

func ResolveSRVRecord(domain string) (DNSResult, error) {
	start := time.Now()
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeSRV)

	result := DNSResult{Domain: domain}

	response, err := queryDNS(msg)
	duration := time.Since(start).Seconds() * 1000
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query SRV", "domain", domain, "error", err)
		result.Record = []string{}
		return result, err
	}

	if response.Rcode != dns.RcodeSuccess {
		slog.Warn("SRV query failed with Rcode", "rcode", response.Rcode)
		return result, fmt.Errorf("SRV query failed with Rcode %d", response.Rcode)
	}

	for _, ans := range response.Answer {
		if srvRecord, ok := ans.(*dns.SRV); ok {
			result.Record = append(result.Record, srvRecord.Target)
			if result.TTL == 0 {
				result.TTL = srvRecord.Header().Ttl
			}
		}
	}
	if result.Record == nil {
		result.Record = []string{}
	}
	return result, nil
}

func ResolvePTRRecord(ip string) (DNSResult, error) {
	start := time.Now()
	ptrName, err := dns.ReverseAddr(ip)
	if err != nil {
		slog.Warn("Invalid IP address for PTR query", "ip", ip, "error", err)
		result := DNSResult{Domain: ip, Record: []string{}}
		return result, fmt.Errorf("invalid IP address: %v", err)
	}

	msg := new(dns.Msg)
	msg.SetQuestion(ptrName, dns.TypePTR)

	result := DNSResult{Domain: ip}

	response, err := queryDNS(msg)
	duration := time.Since(start).Seconds() * 1000
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query PTR", "ip", ip, "error", err)
		result.Record = []string{}
		return result, err
	}

	if response.Rcode != dns.RcodeSuccess {
		slog.Warn("PTR query failed with Rcode", "rcode", response.Rcode)
		return result, fmt.Errorf("PTR query failed with Rcode %d", response.Rcode)
	}

	for _, ans := range response.Answer {
		if ptrRecord, ok := ans.(*dns.PTR); ok {
			result.Record = append(result.Record, ptrRecord.Ptr)
			if result.TTL == 0 {
				result.TTL = ptrRecord.Header().Ttl
			}
		}
	}
	if result.Record == nil {
		result.Record = []string{}
	}
	return result, nil
}

func ResolveCAARecord(domain string) (DNSResult, error) {
	start := time.Now()
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeCAA)

	result := DNSResult{Domain: domain}

	response, err := queryDNS(msg)
	duration := time.Since(start).Seconds() * 1000
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query CAA", "domain", domain, "error", err)
		result.Record = []string{}
		return result, err
	}

	if response.Rcode != dns.RcodeSuccess {
		slog.Warn("CAA query failed with Rcode", "rcode", response.Rcode)
		return result, fmt.Errorf("CAA query failed with Rcode %d", response.Rcode)
	}

	for _, ans := range response.Answer {
		if caaRecord, ok := ans.(*dns.CAA); ok {
			result.Record = append(result.Record, caaRecord.Value)
			if result.TTL == 0 {
				result.TTL = caaRecord.Header().Ttl
			}
		}
	}
	if result.Record == nil {
		result.Record = []string{}
	}
	return result, nil
}

// DNSFullResult 完整的DNS查询结果（不包含PTR，PTR需单独查询）
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

// ResolveARecordllDNSRecords 并行查询所有主流DNS记录类型（不包含PTR）
func ResolveARecordllDNSRecords(domain string) DNSFullResult {
	result := DNSFullResult{Domain: domain}

	var wg sync.WaitGroup
	wg.Add(8)

	go func() {
		defer wg.Done()
		result.A, _ = ResolveARecord(domain)
	}()

	go func() {
		defer wg.Done()
		result.AAAA, _ = ResolveAAAARecord(domain)
	}()

	go func() {
		defer wg.Done()
		result.CNAME, _ = ResolveCNAMERecord(domain)
	}()

	go func() {
		defer wg.Done()
		result.MX, _ = ResolveMXRecord(domain)
	}()

	go func() {
		defer wg.Done()
		result.NS, _ = ResolveNSRecord(domain)
	}()

	go func() {
		defer wg.Done()
		result.TXT, _ = ResolveTXTRecord(domain)
	}()

	go func() {
		defer wg.Done()
		result.SRV, _ = ResolveSRVRecord(domain)
	}()

	go func() {
		defer wg.Done()
		result.CAA, _ = ResolveCAARecord(domain)
	}()

	wg.Wait()

	return result
}
