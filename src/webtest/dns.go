package webtest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sort"
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

// NormalizeServers 解析「逗号分隔的 DNS 服务器」字符串并做格式校验，规则与配置侧一致：
// 每项为 "ip:port"（UDP/TCP）或 http(s):// DoH URL；非法项丢弃（记 Warn）。全非法时返回空。
// 供请求级参数（dns / dns4 / dns6）复用，避免两套校验漂移。
func NormalizeServers(s string) []string {
	return validServers(splitServers(s))
}

// ---- 请求级解析配置：dns*/resolve* ----
//
//	dns=1.1.1.1:53                      两栈都用这个服务器（换服务器去问）
//	dns4=1.1.1.1:53,dns6=[2400::1]:53   按栈覆盖（优先于 dns=）
//	resolve=example.com:1.2.3.4         两栈都把 example.com 解析成 1.2.3.4（直接改答案）
//	resolve4=example.com:1.2.3.4        只 IPv4 侧生效（优先于 resolve=）
//	resolve6=example.com:[::1]          只 IPv6 侧生效（优先于 resolve=）
//	resolve=www.example.com:origin.net  目标是域名时把它当作新的 host 再解析
//
// 只在「域名 → IP」这一步生效：detail/ssl/speed（经 DialContext）、tcping、SSRF 预检；
// whois 与 dns 拨测不接 resolve*（前者只做归属查询，后者问的就是真实解析）。
//
// 为什么走 context：HTTP 拨测的解析发生在 net/http 的 DialContext 里，那里只有 ctx、
// 没有直接入参；存进 ctx 后 DialContext 与 SSRF 预检便能同源读到同一份配置。

// ReqResolve 一次拨测的请求级解析配置。零值 = 没有任何请求级覆盖。
type ReqResolve struct {
	dns  []string // dns=
	dns4 []string // dns4=
	dns6 []string // dns6=

	resolve  map[string]string // resolve=：域名 → IP 或另一域名
	resolve4 map[string]string // resolve4=
	resolve6 map[string]string // resolve6=
}

// ParseReqResolve 从 query 解析全部 9 个参数（dns/dns4/dns6、resolve/resolve4/resolve6）。
// 参数存在但**全部项非法**时报错（handler 返 400），避免「看起来指定了、实际被静默忽略」；
// 缺省/为空则视为不覆盖。
func ParseReqResolve(get func(string) string) (ReqResolve, error) {
	var out ReqResolve
	// DNS 服务器：逗号分隔的 ip:port 或 DoH URL
	for _, item := range []struct {
		key string
		dst *[]string
	}{
		{"dns", &out.dns}, {"dns4", &out.dns4}, {"dns6", &out.dns6},
	} {
		if raw := strings.TrimSpace(get(item.key)); raw != "" {
			if list := NormalizeServers(raw); len(list) > 0 {
				*item.dst = list
			} else {
				return ReqResolve{}, fmt.Errorf("%s must be a comma-separated list of ip:port or DoH URLs", item.key)
			}
		}
	}
	// 强制解析：逗号分隔的 域名:IP 或 域名:域名（后者按新 host 再解析）
	for _, item := range []struct {
		key string
		dst *map[string]string
	}{
		{"resolve", &out.resolve}, {"resolve4", &out.resolve4}, {"resolve6", &out.resolve6},
	} {
		if raw := strings.TrimSpace(get(item.key)); raw != "" {
			if m := parseResolveMap(raw); len(m) > 0 {
				*item.dst = m
			} else {
				return ReqResolve{}, fmt.Errorf("%s must be a comma-separated list of host:ip or host:cname", item.key)
			}
		}
	}
	return out, nil
}

// parseResolveMap 把「域名:目标,域名:目标」切成映射表，非法项丢弃。
// 每组按**第一个冒号**切：域名不含冒号，IPv6 目标可带或不带方括号。
func parseResolveMap(s string) map[string]string {
	out := map[string]string{}
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		idx := strings.Index(item, ":")
		if idx <= 0 || idx == len(item)-1 {
			continue
		}
		host, target := canonicalHost(item[:idx]), strings.TrimSpace(item[idx+1:])
		if host == "" || strings.ContainsAny(host, " /\\") || !validResolveTarget(target) {
			continue
		}
		out[host] = target
	}
	return out
}

// canonicalHost 归一化主机名：去首尾空白、转小写、去结尾的点，便于与查询域名匹配。
func canonicalHost(host string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
}

// validResolveTarget 目标必须是 IP 或形如域名的字符串（含点、无空格、无端口）。
func validResolveTarget(target string) bool {
	t := strings.Trim(target, "[]")
	if net.ParseIP(t) != nil {
		return true
	}
	// 非 IP 却含冒号：视为「带端口」的误用，丢弃（映射不需要端口）
	if strings.Contains(t, ":") || !strings.Contains(t, ".") || strings.ContainsAny(t, " /\\") {
		return false
	}
	for _, r := range t {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' ||
			r == '.' || r == '-' || r == '_' || r == '*' {
			continue
		}
		return false
	}
	return true
}

// Empty 无任何请求级覆盖（此时才允许走共享缓存）。
func (r ReqResolve) Empty() bool {
	return len(r.dns)+len(r.dns4)+len(r.dns6)+len(r.resolve)+len(r.resolve4)+len(r.resolve6) == 0
}

// Servers 按栈名取本次生效的 DNS 服务器：stack 为 "ipv4"/"ipv6" 时优先取对应覆盖，
// 其余（含空串）取 dns= 默认；最终为 nil 表示用配置 dns-server。
// 供 whois、dns 拨测等按记录类型选栈的调用方使用。
func (r ReqResolve) Servers(stack string) []string {
	switch stack {
	case "ipv4":
		if len(r.dns4) > 0 {
			return r.dns4
		}
	case "ipv6":
		if len(r.dns6) > 0 {
			return r.dns6
		}
	}
	return r.dns
}

// Override 返回该域名在指定族下的强制解析目标；"" 表示未命中（照常走 DNS）。
// 优先级与 dns* 对称：resolve4/resolve6 非空时覆盖 resolve。
// 目标是 IP 还是域名由调用方判断——IP 直连、域名当作新 host 再解析。
func (r ReqResolve) Override(host string, wantV6 bool) string {
	m := r.resolve
	if wantV6 && len(r.resolve6) > 0 {
		m = r.resolve6
	} else if !wantV6 && len(r.resolve4) > 0 {
		m = r.resolve4
	}
	return m[canonicalHost(host)]
}

// CacheKey 返回请求级配置的规范化片段，拼在拨测缓存键末尾。
// 零值返回空串（缓存键与「没有任何请求级参数」时完全一致）；
// 否则形如 ";dns=1.1.1.1:53,8.8.8.8:53;resolve6=example.com:[::1]"。
//
// 逐字段拼接而不是 join 结构体：字段名与顺序由这里固定，避免参数书写顺序
// （dns= 在前还是 dns6= 在前）把同一个请求拆成两条缓存。
func (r ReqResolve) CacheKey() string {
	var b strings.Builder
	for _, f := range []struct {
		name string
		vals []string
	}{
		{"dns", r.dns}, {"dns4", r.dns4}, {"dns6", r.dns6},
	} {
		if len(f.vals) == 0 {
			continue
		}
		b.WriteString(";")
		b.WriteString(f.name)
		b.WriteString("=")
		b.WriteString(strings.Join(f.vals, ","))
	}
	for _, f := range []struct {
		name string
		vals map[string]string
	}{
		{"resolve", r.resolve}, {"resolve4", r.resolve4}, {"resolve6", r.resolve6},
	} {
		if len(f.vals) == 0 {
			continue
		}
		// map 迭代顺序随机：必须按键排序，否则同一请求每次拼出的键都不同，缓存永远命不中
		keys := make([]string, 0, len(f.vals))
		for k := range f.vals {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		pairs := make([]string, 0, len(keys))
		for _, k := range keys {
			pairs = append(pairs, k+":"+f.vals[k])
		}
		b.WriteString(";")
		b.WriteString(f.name)
		b.WriteString("=")
		b.WriteString(strings.Join(pairs, ","))
	}
	return b.String()
}

// reqResolveCtxKey 请求级解析配置在 context 中的键。
type reqResolveCtxKey struct{}

// WithReqResolve 把请求级解析配置写进 context（空配置原样返回，零开销）。
func WithReqResolve(ctx context.Context, r ReqResolve) context.Context {
	if r.Empty() {
		return ctx
	}
	return context.WithValue(ctx, reqResolveCtxKey{}, r)
}

// ReqResolveFromContext 取出请求级解析配置（零值 = 无覆盖）。
func ReqResolveFromContext(ctx context.Context) ReqResolve {
	if ctx == nil {
		return ReqResolve{}
	}
	if v, ok := ctx.Value(reqResolveCtxKey{}).(ReqResolve); ok {
		return v
	}
	return ReqResolve{}
}

// queryDNS 普通 DNS 查询：使用配置的 dns-server（主从 failover，见 queryWithServers）。
// resolveTyped 未收到请求级 servers 时走这里。
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
// servers 为请求级 DNS 覆盖：非空时用它（queryWithServers），为空则用配置 dns-server（queryDNS）。
func resolveTyped(domain, name string, qtype uint16, servers []string, extract func(dns.RR) ([]string, uint32, bool)) (DNSResult, error) {
	start := time.Now()
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(name), qtype)
	result := DNSResult{Domain: domain}

	// 请求级 servers 优先；未传（nil/空）走配置 dns-server。
	var response *dns.Msg
	var err error
	if len(servers) > 0 {
		response, err = queryWithServers(msg, servers)
	} else {
		response, err = queryDNS(msg)
	}
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

// ---- 记录解析：每类一个函数，第三个参数 servers 为「请求级 DNS 覆盖」 ----
//
// servers 为空 → 走 resolveTyped 的 queryDNS（配置 dns-server）；非空 → queryWithServers（请求级覆盖）。

func ResolveARecord(domain string, servers []string) (DNSResult, error) {
	return resolveTyped(domain, domain, dns.TypeA, servers, func(rr dns.RR) ([]string, uint32, bool) {
		if a, ok := rr.(*dns.A); ok {
			return []string{a.A.String()}, a.Header().Ttl, true
		}
		return nil, 0, false
	})
}

func ResolveAAAARecord(domain string, servers []string) (DNSResult, error) {
	return resolveTyped(domain, domain, dns.TypeAAAA, servers, func(rr dns.RR) ([]string, uint32, bool) {
		if a, ok := rr.(*dns.AAAA); ok {
			return []string{a.AAAA.String()}, a.Header().Ttl, true
		}
		return nil, 0, false
	})
}

func ResolveTXTRecord(domain string, servers []string) (DNSResult, error) {
	return resolveTyped(domain, domain, dns.TypeTXT, servers, func(rr dns.RR) ([]string, uint32, bool) {
		if txt, ok := rr.(*dns.TXT); ok {
			// TXT 一条 RR 可能含多段字符串，保持原有「逐段单独成项」的行为
			return txt.Txt, txt.Header().Ttl, true
		}
		return nil, 0, false
	})
}

func ResolveNSRecord(domain string, servers []string) (DNSResult, error) {
	return resolveTyped(domain, domain, dns.TypeNS, servers, func(rr dns.RR) ([]string, uint32, bool) {
		if ns, ok := rr.(*dns.NS); ok {
			return []string{ns.Ns}, ns.Header().Ttl, true
		}
		return nil, 0, false
	})
}

func ResolveCNAMERecord(domain string, servers []string) (DNSResult, error) {
	return resolveTyped(domain, domain, dns.TypeCNAME, servers, func(rr dns.RR) ([]string, uint32, bool) {
		if cname, ok := rr.(*dns.CNAME); ok {
			return []string{cname.Target}, cname.Header().Ttl, true
		}
		return nil, 0, false
	})
}

func ResolveMXRecord(domain string, servers []string) (DNSResult, error) {
	return resolveTyped(domain, domain, dns.TypeMX, servers, func(rr dns.RR) ([]string, uint32, bool) {
		if mx, ok := rr.(*dns.MX); ok {
			return []string{mx.Mx}, mx.Header().Ttl, true
		}
		return nil, 0, false
	})
}

func ResolveSRVRecord(domain string, servers []string) (DNSResult, error) {
	return resolveTyped(domain, domain, dns.TypeSRV, servers, func(rr dns.RR) ([]string, uint32, bool) {
		if srv, ok := rr.(*dns.SRV); ok {
			return []string{srv.Target}, srv.Header().Ttl, true
		}
		return nil, 0, false
	})
}

func ResolveCAARecord(domain string, servers []string) (DNSResult, error) {
	return resolveTyped(domain, domain, dns.TypeCAA, servers, func(rr dns.RR) ([]string, uint32, bool) {
		if caa, ok := rr.(*dns.CAA); ok {
			return []string{caa.Value}, caa.Header().Ttl, true
		}
		return nil, 0, false
	})
}

func ResolvePTRRecord(ip string, servers []string) (DNSResult, error) {
	ptrName, err := dns.ReverseAddr(ip)
	if err != nil {
		slog.Warn("Invalid IP address for PTR query", "ip", ip, "error", err)
		return DNSResult{Domain: ip, Record: []string{}}, fmt.Errorf("invalid IP address: %v", err)
	}
	return resolveTyped(ip, ptrName, dns.TypePTR, servers, func(rr dns.RR) ([]string, uint32, bool) {
		if ptr, ok := rr.(*dns.PTR); ok {
			return []string{ptr.Ptr}, ptr.Header().Ttl, true
		}
		return nil, 0, false
	})
}

// ResolveByType 按记录类型名（a/aaaa/cname/mx/ns/txt/srv/ptr/caa）解析，servers 为请求级覆盖。
// DNS 拨测的 HTTP 路由与 WS 报文统一走这里，避免两处各维护一套 switch。
func ResolveByType(domain, recodeType string, servers []string) (DNSResult, error) {
	switch recodeType {
	case "a":
		return ResolveARecord(domain, servers)
	case "aaaa":
		return ResolveAAAARecord(domain, servers)
	case "cname":
		return ResolveCNAMERecord(domain, servers)
	case "mx":
		return ResolveMXRecord(domain, servers)
	case "ns":
		return ResolveNSRecord(domain, servers)
	case "srv":
		return ResolveSRVRecord(domain, servers)
	case "txt":
		return ResolveTXTRecord(domain, servers)
	case "ptr":
		return ResolvePTRRecord(domain, servers)
	case "caa":
		return ResolveCAARecord(domain, servers)
	}
	return DNSResult{}, fmt.Errorf("unsupported record type: %s", recodeType)
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

// ResolveARecordllDNSRecords 并行查询所有主流DNS记录类型（不包含PTR）。
// servers 为请求级 DNS 覆盖（空则用配置 dns-server）。
func ResolveARecordllDNSRecords(domain string, servers []string) DNSFullResult {
	result := DNSFullResult{Domain: domain}

	var wg sync.WaitGroup
	wg.Add(8)

	go func() {
		defer wg.Done()
		result.A, _ = ResolveARecord(domain, servers)
	}()

	go func() {
		defer wg.Done()
		result.AAAA, _ = ResolveAAAARecord(domain, servers)
	}()

	go func() {
		defer wg.Done()
		result.CNAME, _ = ResolveCNAMERecord(domain, servers)
	}()

	go func() {
		defer wg.Done()
		result.MX, _ = ResolveMXRecord(domain, servers)
	}()

	go func() {
		defer wg.Done()
		result.NS, _ = ResolveNSRecord(domain, servers)
	}()

	go func() {
		defer wg.Done()
		result.TXT, _ = ResolveTXTRecord(domain, servers)
	}()

	go func() {
		defer wg.Done()
		result.SRV, _ = ResolveSRVRecord(domain, servers)
	}()

	go func() {
		defer wg.Done()
		result.CAA, _ = ResolveCAARecord(domain, servers)
	}()

	wg.Wait()

	return result
}
