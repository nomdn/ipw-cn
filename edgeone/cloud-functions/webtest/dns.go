package webtest

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// ==========================================
// ⭐️ 核心重构：严格遵循 RFC 8484 标准 DoH (二进制报文)
// ==========================================

// 阿里云标准 DoH 端点
var dohEndpoint string = "https://doh.pub/dns-query"

type DNSResult struct {
	Domain   string   `json:"domain"`
	Duration float64  `json:"duration"`
	Record   []string `json:"record"`
	TTL      uint32   `json:"ttl"`
}

func SetDNSServer(server string) {
	if server == "" {
		server = "https://doh.pub/dns-query"
	}
	dohEndpoint = server
}

// ==========================================
// ⭐️ 底层核心：构造、发送并解析二进制 DNS 报文
// ==========================================
func executeDoHQuery(domain string, qtype uint16) (*dns.Msg, float64, error) {
	// 1. 构造标准的 DNS 请求报文
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), qtype)
	msg.RecursionDesired = true // 请求递归解析

	// 2. 将 DNS 报文打包成二进制字节流
	packedMsg, err := msg.Pack()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to pack DNS message: %v", err)
	}

	// 3. 构造 HTTP POST 请求，Body 为二进制报文
	req, err := http.NewRequest("POST", dohEndpoint, bytes.NewReader(packedMsg))
	if err != nil {
		return nil, 0, err
	}

	// ⭐️ 灵魂 Header：RFC 8484 标准规定的 MIME 类型
	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")

	// 4. 发起 HTTPS 请求 (走 443 端口)
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

	// 5. 读取返回的二进制响应报文
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, duration, fmt.Errorf("failed to read response body: %v", err)
	}

	// 6. 使用 miekg/dns 解包二进制响应
	responseMsg := new(dns.Msg)
	if err := responseMsg.Unpack(bodyBytes); err != nil {
		return nil, duration, fmt.Errorf("failed to unpack DNS response: %v", err)
	}

	// 7. 检查 DNS 响应码 (Rcode)
	if responseMsg.Rcode != dns.RcodeSuccess {
		return responseMsg, duration, fmt.Errorf("DNS query failed with Rcode %d", responseMsg.Rcode)
	}

	return responseMsg, duration, nil
}

// ==========================================
// 业务层：调用底层函数并提取特定记录
// ==========================================

func QueryA(domain string) (DNSResult, error) {
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

func QueryAllDNSRecords(domain string) DNSFullResult {
	result := DNSFullResult{Domain: domain}
	var wg sync.WaitGroup
	wg.Add(8)

	go func() { defer wg.Done(); result.A, _ = QueryA(domain) }()
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
