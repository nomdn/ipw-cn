package webtest

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

var (
	// dnsServers 普通 DNS 查询服务器列表（主从：第一个为主，主失败自动切换后续从服务器）
	dnsServers = []string{"119.28.28.28:53"}
	// dnssecServers DNSSEC 专用服务器列表（未配置时沿用 dnsServers）
	dnssecServers []string
)

// SetDNSServer 设置DNS服务器地址（逗号分隔多地址：第一个为主，主失败自动切换后续从服务器）。
// 每项支持 "ip:port"（UDP）或 DoH URL http(s)://...
func SetDNSServer(server string) {
	if list := validServers(splitServers(server)); len(list) > 0 {
		dnsServers = list
	}
}

// SetDNSSecServer 设置DNSSEC专用DNS服务器（逗号分隔主从；留空 = 沿用 dns-server 配置）
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
//
// dns-server / dnssec-server 既能由本地管理面 PATCH 改，也能由远端托管配置下发 —— 都是外部可控输入。
// 在这里先挡掉畸形值，总比留到查询时才以"莫名其妙的超时/报错"暴露好排查。
// 全部非法时返回空列表，调用方据此保留原有配置（与"空串不覆盖"的既有语义一致）。
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

// queryDoHMsg 通过 DoH（RFC 8484，POST application/dns-message）向指定端点发送查询。
//
// 这里**刻意不做私网/回环拦截**（与 website / ssl / speed 那三处不同）：
// endpoint 来自 dns-server 配置，而节点上自建内网 DoH（127.0.0.1 / 10.x）是合理部署，
// 堵私网会直接打死功能；配置又是管理员凭据才能改的，不构成匿名可达攻击面。
// 地址**格式**校验已在 SetDNSServer 入口由 validateServerAddr 完成（http/https + host 非空）。
// 对应 CodeQL go/request-forgery（#24）按误报处理。
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
	Domain    string   `json:"domain"`
	Record    []string `json:"record"`
	TTL       uint32   `json:"ttl"`
	Duration  float64  `json:"duration"`
	Rcode     int      `json:"rcode"`      // 解析器应答的 Rcode（0=Success, 3=NXDOMAIN）
	RcodeText string   `json:"rcode_text"` // Rcode 可读名称，如 NXDOMAIN / SERVFAIL
}

// resolveTyped 通用记录查询：成功收到解析器应答（即便 Rcode 非 Success，如 NXDOMAIN/SERVFAIL）
// 即视为一次有效查询，将 Rcode 透出并返回 200；仅在传输/网络层失败（超时、服务器不可达、
// DoH 异常）时才返回 error（上层据此返回 500）。避免把合法的负响应误判为节点故障。
// extract 从一个 RR 中抽取记录字符串（TXT 可能含多条），返回 (values, ttl, matched)。
func resolveTyped(domain, name string, qtype uint16, extract func(dns.RR) ([]string, uint32, bool)) (DNSResult, error) {
	start := time.Now()
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(name), qtype)
	result := DNSResult{Domain: domain}

	response, err := queryDNS(msg)
	result.Duration = time.Since(start).Seconds() * 1000
	if err != nil {
		slog.Warn("Failed to query DNS", "domain", domain, "qtype", qtype, "error", err)
		result.Record = []string{}
		return result, err
	}

	// 解析器已应答：无论 Rcode 是否为 Success，都属合法结果，不再当 error 返回
	result.Rcode = response.Rcode
	result.RcodeText = dns.RcodeToString[response.Rcode]
	result.Record = []string{}
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
