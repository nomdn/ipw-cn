package webtest

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/miekg/dns"
)

var (
	dnsServer = "119.28.28.28:53"
)

// SetDNSServer 设置DNS服务器地址（格式: "ip:port"）
func SetDNSServer(server string) {
	if server != "" {
		dnsServer = server
	}
}

// DNSResult 统一的DNS查询结果格式
type DNSResult struct {
	Domain   string   `json:"domain"`
	Record   []string `json:"record"`
	TTL      uint32   `json:"ttl"`
	Duration float64  `json:"duration"`
}

func QueryA(domain string) (DNSResult, error) {
	start := time.Now()
	msg := new(dns.Msg)
	client := new(dns.Client)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	result := DNSResult{Domain: domain}

	response, _, err := client.Exchange(msg, dnsServer)
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
	client := new(dns.Client)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeAAAA)

	result := DNSResult{Domain: domain}

	response, _, err := client.Exchange(msg, dnsServer)
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
	client := new(dns.Client)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeTXT)

	result := DNSResult{Domain: domain}

	response, _, err := client.Exchange(msg, dnsServer)
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
	client := new(dns.Client)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeNS)

	result := DNSResult{Domain: domain}

	response, _, err := client.Exchange(msg, dnsServer)
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
	client := new(dns.Client)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeCNAME)

	result := DNSResult{Domain: domain}

	response, _, err := client.Exchange(msg, dnsServer)
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
	client := new(dns.Client)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeMX)

	result := DNSResult{Domain: domain}

	response, _, err := client.Exchange(msg, dnsServer)
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
	client := new(dns.Client)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeSRV)

	result := DNSResult{Domain: domain}

	response, _, err := client.Exchange(msg, dnsServer)
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
	client := new(dns.Client)
	msg.SetQuestion(ptrName, dns.TypePTR)

	result := DNSResult{Domain: ip}

	response, _, err := client.Exchange(msg, dnsServer)
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
	client := new(dns.Client)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeCAA)

	result := DNSResult{Domain: domain}

	response, _, err := client.Exchange(msg, dnsServer)
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

// QueryAllDNSRecords 并行查询所有主流DNS记录类型（不包含PTR）
func QueryAllDNSRecords(domain string) DNSFullResult {
	result := DNSFullResult{Domain: domain}

	var wg sync.WaitGroup
	wg.Add(8)

	go func() {
		defer wg.Done()
		result.A, _ = QueryA(domain)
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
