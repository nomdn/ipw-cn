package webtest

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/miekg/dns"
)

var (
	// ⭐️ 建议更换为稳定的公共 DNS，避免特定运营商 DNS 在云环境中不可达
	dnsServer = "119.29.29.29:53"
)

type DNSResult struct {
	Domain   string   `json:"domain"`
	Record   []string `json:"record"`
	TTL      uint32   `json:"ttl"`
	Duration float64  `json:"duration"`
}

// ==========================================
// ⭐️ 核心重构：提取通用的底层 DNS 查询函数
// ==========================================
func executeQuery(domain string, qtype uint16) (*dns.Msg, float64, error) {
	msg := new(dns.Msg)
	client := new(dns.Client)

	// ⭐️ 核心修复：强制使用 TCP，并设置超时
	client.Net = "tcp"
	client.Timeout = 3 * time.Second

	msg.SetQuestion(dns.Fqdn(domain), qtype)

	start := time.Now()
	response, _, err := client.Exchange(msg, dnsServer)
	duration := time.Since(start).Seconds() * 1000

	if err != nil {
		return nil, duration, err
	}
	if response.Rcode != dns.RcodeSuccess {
		return nil, duration, fmt.Errorf("DNS query failed with Rcode %d", response.Rcode)
	}

	return response, duration, nil
}

// ==========================================
// 业务层：各种记录类型的解析（代码大幅精简）
// ==========================================

func QueryA(domain string) (DNSResult, error) {
	result := DNSResult{Domain: domain, Record: []string{}}
	response, duration, err := executeQuery(domain, dns.TypeA)
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query A for %s: %v", domain, err)
		return result, err
	}
	for _, ans := range response.Answer {
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
	response, duration, err := executeQuery(domain, dns.TypeAAAA)
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query AAAA for %s: %v", domain, err)
		return result, err
	}
	for _, ans := range response.Answer {
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
	response, duration, err := executeQuery(domain, dns.TypeTXT)
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query TXT for %s: %v", domain, err)
		return result, err
	}
	for _, ans := range response.Answer {
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
	response, duration, err := executeQuery(domain, dns.TypeNS)
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query NS for %s: %v", domain, err)
		return result, err
	}
	for _, ans := range response.Answer {
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
	response, duration, err := executeQuery(domain, dns.TypeCNAME)
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query CNAME for %s: %v", domain, err)
		return result, err
	}
	for _, ans := range response.Answer {
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
	response, duration, err := executeQuery(domain, dns.TypeMX)
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query MX for %s: %v", domain, err)
		return result, err
	}
	for _, ans := range response.Answer {
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
	response, duration, err := executeQuery(domain, dns.TypeSRV)
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query SRV for %s: %v", domain, err)
		return result, err
	}
	for _, ans := range response.Answer {
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
		return result, fmt.Errorf("invalid IP address: %v", err)
	}

	// PTR 查询特殊处理：直接调用 executeQuery，但 domain 参数传 ptrName
	response, duration, err := executeQuery(ptrName, dns.TypePTR)
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query PTR for %s: %v", ip, err)
		return result, err
	}
	for _, ans := range response.Answer {
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
	response, duration, err := executeQuery(domain, dns.TypeCAA)
	result.Duration = duration
	if err != nil {
		slog.Warn("Failed to query CAA for %s: %v", domain, err)
		return result, err
	}
	for _, ans := range response.Answer {
		if caaRecord, ok := ans.(*dns.CAA); ok {
			result.Record = append(result.Record, caaRecord.Value)
			if result.TTL == 0 {
				result.TTL = caaRecord.Header().Ttl
			}
		}
	}
	return result, nil
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
