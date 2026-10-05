package webtest

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"lemon-ipw/ssrf"

	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
)

// WhoisResult 包含 WHOIS 查询的结构化结果
// 与前端 whois.vue 的结构体保持一致
type WhoisResult struct {
	Domain       string         `json:"domain"`       // 域名（大写）
	Status       []string       `json:"status"`       // 域名状态列表
	Registrar    WhoisRegistrar `json:"registrar"`    // 注册商信息
	Registrant   WhoisContact   `json:"registrant"`   // 注册人信息
	Technical    WhoisContact   `json:"technical"`    // 技术联系人
	AbuseContact WhoisContact   `json:"abuseContact"` // Abuse 联系人
	Dates        WhoisDates     `json:"dates"`        // 关键日期
	NameServers  []string       `json:"nameservers"`  // DNS 服务器列表
	WhoisServer  string         `json:"whoisServer"`  // Whois 服务器地址
	Raw          string         `json:"raw"`          // 原始响应文本
	Error        string         `json:"error"`        // 错误信息
}

type WhoisRegistrar struct {
	Name   string `json:"name"`
	IanaId string `json:"ianaId"`
}

type WhoisContact struct {
	Name       string `json:"name"`
	Org        string `json:"org"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	Province   string `json:"province"`
	ContactUri string `json:"contactUri"`
}

type WhoisDates struct {
	Registration string `json:"registration"`
	Expiration   string `json:"expiration"`
	LastChanged  string `json:"lastChanged"`
}

// tldWhoisServers 内置常见 TLD 与对应 WHOIS 服务器的映射
// 当 IANA 查询失败时作为 fallback 使用
var tldWhoisServers = map[string]string{
	"com":   "whois.verisign-grs.com",
	"net":   "whois.verisign-grs.com",
	"org":   "whois.pir.org",
	"info":  "whois.afilias.net",
	"biz":   "whois.neulevel.biz",
	"name":  "whois.nic.name",
	"pro":   "whois.registrypro.pro",
	"io":    "whois.nic.io",
	"co":    "whois.nic.co",
	"me":    "whois.nic.me",
	"cc":    "whois.nic.cc",
	"tv":    "whois.nic.tv",
	"top":   "whois.nic.top",
	"xyz":   "whois.nic.xyz",
	"club":  "whois.nic.club",
	"online":"whois.nic.online",
	"site":  "whois.nic.site",
	"store": "whois.nic.store",
	"shop":  "whois.nic.shop",
	"app":   "whois.nic.google",
	"dev":   "whois.nic.google",
	"tech":  "whois.nic.tech",
	"cn":    "whois.cnnic.cn",
	"wang":  "whois.gtld.knet.cn",
	"ren":   "whois.renren.us",
}

// happyEyeballsV6Delay Happy Eyeballs 的 v6 启动延迟
// RFC 6555 推荐 250ms，这里缩短为 150ms，在 v6 不可达的环境下更快切到 v4
const happyEyeballsV6Delay = 150 * time.Millisecond

// whoisConnectTimeout 单 IP 连接 + 读写超时上限
const whoisConnectTimeout = 10 * time.Second

// whoisLibTimeout likexian/whois 单跳超时上限（库的超时按「跳」计，同一跳内读写共享该预算）。
// 库默认 30s：一旦注册商转介指向不可达地址（如 Cloudflare 后的网站域名，43 端口被丢弃），
// 单跳干等 30s，整条查询必然超过中心拨测默认 30s 超时。
// 主路径只剩 IANA + 注册局两跳（转介已改为独立限时补查），2×6s 仍在中心超时之内。
const whoisLibTimeout = 6 * time.Second

// whoisReferralTimeout 自行补查注册商转介的总超时。
// 转介只是「锦上添花」，失败即放弃，绝不阻塞已到手的注册局数据。
const whoisReferralTimeout = 3 * time.Second

// QueryWhois 执行 WHOIS 查询并解析结构化数据
// 主路径：likexian/whois 走「IANA → 注册局」两跳，**不追注册商转介**（转介由 chaseReferral 单独限时补），
// 这样即便转介服务器不可达也只在 ~1s 内拿到注册局记录，不会干等满库的 30s 超时。
// 兜底：主路径拿不到内容时才 fallback（内置 TLD 映射 → IANA 查服务器 → 自定义 DNS 解析 IP →
// Happy Eyeballs 双栈竞争拨号）。
func QueryWhois(domain string) (*WhoisResult, error) {
	raw, err := queryWhoisRegistry(domain)

	// 库在「连接成功但读失败」等场景会同时返回部分数据与错误，此时数据不能丢；
	// 因此只在「内容为空」或「带错误」时让 fallback 再兜一次，并取内容更长的一方。
	if err != nil || raw == "" {
		if fb, fbErr := whoisRetryWithFallback(domain, err); len(fb) > len(raw) {
			raw, err = fb, fbErr
		}
	}

	// 仅当注册局响应干净（err == nil）时才补查注册商转介
	if err == nil && raw != "" {
		if extra := chaseReferral(domain, raw); extra != "" {
			raw += extra
		}
	}

	result := parseWhoisResult(domain, raw)
	result.Error = errString(err)
	return result, nil
}

// queryWhoisRegistry 走 IANA → 注册局两跳取注册局记录，关闭库的转介链
// （转介改由 chaseReferral 限时补查，避免被不可达的转介服务器拖满 30s）
func queryWhoisRegistry(domain string) (string, error) {
	return whois.NewClient().
		SetDisableReferral(true).
		SetTimeout(whoisLibTimeout).
		Whois(domain)
}

// chaseReferral 从注册局响应里取出注册商转介服务器，限时补查一次
// 任何失败（无转介 / 不可达 / 超时）都返回空串，绝不影响已到手的注册局数据
func chaseReferral(domain, raw string) string {
	server, port := extractReferralServer(raw)
	if server == "" {
		return ""
	}
	// 转介指向的仍是刚查过的注册局服务器（库自带判断的同款保护）
	if ext := getExtension(domain); strings.EqualFold(tldWhoisServers[ext], server) {
		return ""
	}

	v4IPs, v6IPs, err := resolveWhoisServerIPs(server)
	if err != nil {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), whoisReferralTimeout)
	defer cancel()

	data, err := happyEyeballsWhoisQuery(ctx, domain, v4IPs, v6IPs, port)
	if err != nil {
		return ""
	}
	return data
}

// referralKeys 指向「下一个 WHOIS 服务器」的字段名（小写，按优先级前缀匹配）
var referralKeys = []string{"registrar whois server", "referralserver", "whois", "referral"}

// normalizeReferralValue 规整转介地址：去协议前缀 / 路径 / 认证尾巴，拆出 host 与 port
// 返回 "" 表示不是可用的 whois 转介。rwhois 是另一种协议，拨过去同样是白等，直接跳过
func normalizeReferralValue(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(strings.ToLower(value), "rwhois:") {
		return "", ""
	}
	for _, prefix := range []string{"http://", "https://", "whois://"} {
		value = strings.TrimPrefix(value, prefix)
	}
	if i := strings.IndexAny(value, "/,"); i != -1 {
		value = value[:i]
	}
	value = strings.TrimSpace(value)

	host, port := value, "43"
	if h, p, err := net.SplitHostPort(value); err == nil {
		host, port = h, p
	}
	// 合法性：必须是形如 whois.example.com 的主机名，排除 "Server: xxx" 之类的误匹配
	if host == "" || strings.ContainsAny(host, " \t:/,") || !strings.Contains(host, ".") {
		return "", ""
	}
	return host, port
}

// extractReferralServer 从注册局响应中解析注册商转介的 WHOIS 服务器与端口，取不到返回 ("", "")
// 覆盖两种写法：字段式（Registrar WHOIS Server: / ReferralServer: / whois:）
// 与注释式（%referral whois://host:4321/auth-area=.）
func extractReferralServer(raw string) (string, string) {
	for _, line := range strings.Split(raw, "\n") {
		// 归一化：去掉 % / # 注释前缀
		line = strings.TrimLeft(strings.TrimSpace(line), "%# \t")
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		for _, key := range referralKeys {
			if !strings.HasPrefix(lower, key) {
				continue
			}
			// 从 lower 切分，避免 ToLower 改变长度时错位（主机名大小写不敏感）
			rest := strings.TrimSpace(lower[len(key):])
			rest = strings.TrimSpace(strings.TrimPrefix(rest, ":"))
			if rest == "" {
				continue
			}
			if server, port := normalizeReferralValue(rest); server != "" {
				return server, port
			}
		}
	}
	return "", ""
}

// errString safely converts an error to string
func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// resolveWhoisServer 解析域名对应的 WHOIS 服务器地址
// 优先使用内置映射，失败时再查询 IANA WHOIS
func resolveWhoisServer(ext string) (string, error) {
	// 1. 优先使用内置 TLD 映射
	if server, ok := tldWhoisServers[ext]; ok && server != "" {
		return server, nil
	}

	// 2. 内置映射中没有，尝试查 IANA
	ianaResult, err := whois.Whois("." + ext)
	if err != nil {
		return "", err
	}

	server := extractWhoisServer(ianaResult)
	if server == "" {
		return "", fmt.Errorf("no whois server found in IANA response for .%s", ext)
	}
	return server, nil
}

// filterPublicIPs 过滤掉 SSRF 私网 IP（仅在 ssrf 启用时生效）
func filterPublicIPs(ips []string) []string {
	if !ssrf.Enabled() {
		return ips
	}
	filtered := make([]string, 0, len(ips))
	for _, s := range ips {
		ip := net.ParseIP(s)
		if ip != nil && ssrf.IsPrivateIP(ip) {
			continue
		}
		filtered = append(filtered, s)
	}
	return filtered
}

// resolveWhoisServerIPs 使用 webtest/dns 的统一解析接口
// 并行查询 A + AAAA（走自定义 DNS 服务器，不受系统 resolver 策略影响）
func resolveWhoisServerIPs(server string) (v4IPs, v6IPs []string, err error) {
	var (
		aResult   DNSResult
		aaaaResult DNSResult
		wg        sync.WaitGroup
		aErr      error
		aaaaErr   error
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		aResult, aErr = ResolveARecord(server)
	}()
	go func() {
		defer wg.Done()
		aaaaResult, aaaaErr = ResolveAAAARecord(server)
	}()
	wg.Wait()

	if aErr != nil && aaaaErr != nil {
		return nil, nil, fmt.Errorf("both A and AAAA DNS lookup failed for %s: A=%w, AAAA=%w", server, aErr, aaaaErr)
	}

	v4IPs = filterPublicIPs(aResult.Record)
	v6IPs = filterPublicIPs(aaaaResult.Record)

	if len(v4IPs) == 0 && len(v6IPs) == 0 {
		return nil, nil, fmt.Errorf("no reachable public IPs for %s after SSRF filter", server)
	}
	return v4IPs, v6IPs, nil
}

// happyEyeballsWhoisQuery Happy Eyeballs (RFC 6555) 简化版：
// - 立即并发拨号所有 IPv4 IP
// - 等待 happyEyeballsV6Delay 后并发拨号所有 IPv6 IP
// - 取第一个成功完成 WHOIS 查询的结果返回
// - 其他仍在运行的 goroutine 通过 context 取消尽快退出
// parent 带 deadline 时（追注册商转介），到点即整体放弃
func happyEyeballsWhoisQuery(parent context.Context, domain string, v4IPs, v6IPs []string, port string) (string, error) {
	if len(v4IPs) == 0 && len(v6IPs) == 0 {
		return "", fmt.Errorf("no IPs available for WHOIS query")
	}

	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	type outcome struct {
		raw string
		err error
	}
	// 有缓冲：保证每个 goroutine 写入不被阻塞，避免 goroutine 泄漏
	ch := make(chan outcome, len(v4IPs)+len(v6IPs))

	tryIP := func(ip string) {
		raw, err := rawWhoisQueryCtx(ctx, domain, ip, port)
		ch <- outcome{raw: raw, err: err}
	}

	// 1) 立即启动所有 v4
	for _, ip := range v4IPs {
		go tryIP(ip)
	}

	// 2) 定时器：v6 延迟启动
	var v6Timer *time.Timer
	if len(v6IPs) > 0 {
		v6Timer = time.AfterFunc(happyEyeballsV6Delay, func() {
			for _, ip := range v6IPs {
				go tryIP(ip)
			}
		})
	}
	defer func() {
		if v6Timer != nil {
			v6Timer.Stop()
		}
	}()

	// 3) 收集结果：取第一个成功；若全失败则返回最后一个错误
	var lastErr error
	for i := 0; i < len(v4IPs)+len(v6IPs); i++ {
		select {
		case r := <-ch:
			if r.err == nil {
				return r.raw, nil
			}
			lastErr = r.err
		case <-ctx.Done():
			return "", ctx.Err()
		}
		// v4 全部失败且无成功时，给延迟启动的 v6 goroutine 留一个启动 + 写入窗口
		if i == len(v4IPs)-1 && v6Timer != nil && lastErr != nil {
			// 给 v6 一个启动 + 写入窗口
			time.Sleep(happyEyeballsV6Delay + 20*time.Millisecond)
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("all WHOIS connection attempts failed")
	}
	return "", lastErr
}

// whoisRetryWithFallback 在 whois.Whois 失败后，手动解析 WHOIS 服务器所有 IP 逐个尝试
func whoisRetryWithFallback(domain string, firstErr error) (string, error) {
	ext := getExtension(domain)

	// 解析 WHOIS 服务器地址（内置映射 + IANA fallback）
	server, err := resolveWhoisServer(ext)
	if err != nil {
		return "", firstErr
	}

	// 使用 ResolveA/AAAA（自定义 DNS）并行解析 v4/v6
	v4IPs, v6IPs, err := resolveWhoisServerIPs(server)
	if err != nil {
		return "", firstErr
	}

	// Happy Eyeballs 双栈竞争拨号 + 并发多 IP
	raw, err := happyEyeballsWhoisQuery(context.Background(), domain, v4IPs, v6IPs, "43")
	if err != nil {
		return "", err
	}
	return raw, nil
}

// getExtension 提取域名后缀
func getExtension(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return domain
}

// extractWhoisServer 从 IANA 响应中提取 WHOIS 服务器地址
func extractWhoisServer(data string) string {
	for _, token := range []string{"whois: ", "Whois: "} {
		if idx := strings.Index(data, token); idx != -1 {
			start := idx + len(token)
			end := strings.Index(data[start:], "\n")
			if end == -1 {
				end = len(data) - start
			}
			server := strings.TrimSpace(data[start : start+end])
			server = strings.TrimPrefix(server, "http://")
			server = strings.TrimPrefix(server, "https://")
			server = strings.TrimPrefix(server, "whois://")
			server = strings.TrimSuffix(server, "/")
			return server
		}
	}
	return ""
}

// rawWhoisQueryCtx 带 context 的 WHOIS 查询
// context 被取消时，Dial/Read 尽快失败退出（goroutine 不泄漏）
func rawWhoisQueryCtx(ctx context.Context, domain, server, port string) (string, error) {
	d := &net.Dialer{Timeout: whoisConnectTimeout}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(server, port))
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// 1. 设置读写 deadline：取 ctx deadline 或默认上限中更早的
	deadline := time.Now().Add(whoisConnectTimeout + 5*time.Second)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	_ = conn.SetDeadline(deadline)

	// 2. 发送查询
	if _, err := conn.Write([]byte(domain + "\r\n")); err != nil {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			return "", err
		}
	}

	// 3. 读取响应：循环读到 EOF/超时。单次 Read 只能拿到一个 TCP 段，会截断大多数 WHOIS 响应
	buf := make([]byte, 65536)
	var out bytes.Buffer
	for {
		n, readErr := conn.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
		}
		if readErr != nil {
			// 有数据就返回（EOF/超时都视为读完），哪怕 readErr != nil
			if out.Len() > 0 {
				return out.String(), nil
			}
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			default:
				return "", readErr
			}
		}
		if out.Len() > 1<<20 { // 防御上限 1MB，避免异常响应无限读
			return out.String(), nil
		}
	}
}

// rawWhoisQuery 直接向指定 IP:port 发送 WHOIS 查询（向后兼容包装）
func rawWhoisQuery(domain, server, port string) (string, error) {
	return rawWhoisQueryCtx(context.Background(), domain, server, port)
}

// parseWhoisResult 用 whois-parser 解析原始响应，转换为前端需要的格式
func parseWhoisResult(domain, raw string) *WhoisResult {
	info, err := whoisparser.Parse(raw)
	result := &WhoisResult{
		Domain: strings.ToUpper(domain),
	}

	if err != nil || info.Domain == nil {
		result.Raw = raw
		return result
	}

	// Domain 字段
	result.NameServers = info.Domain.NameServers
	result.Status = info.Domain.Status
	result.Dates.Registration = info.Domain.CreatedDate
	result.Dates.Expiration = info.Domain.ExpirationDate
	result.Dates.LastChanged = info.Domain.UpdatedDate
	result.WhoisServer = info.Domain.WhoisServer

	// Registrar
	if info.Registrar != nil {
		result.Registrar.Name = info.Registrar.Name
		result.Registrar.IanaId = extractIanaIdFromRaw(raw)
	}

	// Registrant
	if info.Registrant != nil {
		result.Registrant = contactFromParser(info.Registrant)
	}

	// Technical
	if info.Technical != nil {
		result.Technical = contactFromParser(info.Technical)
	}

	// AbuseContact: 从原始响应手动提取，不用 Administrative 映射
	abuse := extractAbuseContactFromRaw(raw)
	if abuse != nil {
		result.AbuseContact = *abuse
	}

	result.Raw = raw
	return result
}

// isEmptyContact 检查联系信息是否全为空
func isEmptyContact(c WhoisContact) bool {
	return c.Name == "" && c.Org == "" && c.Phone == "" && c.Email == "" && c.Province == "" && c.ContactUri == ""
}

// abusePatterns 用于从原始 WHOIS 文本中提取 Abuse 联系信息
// 匹配字段名（不区分大小写），如:
//   Registrar Abuse Contact Email: abuse@example.com
//   Abuse Phone: +1.1234567890
var abusePatterns = map[string]*regexp.Regexp{
	"email": regexp.MustCompile(`(?i)^\s*(?:Registrar\s+Abuse\s+Contact\s+Email|Abuse\s+(?:Contact\s+)?Email)\s*[:=]\s*(.+?)\s*$`),
	"phone": regexp.MustCompile(`(?i)^\s*(?:Registrar\s+Abuse\s+Contact\s+Phone|Abuse\s+(?:Contact\s+)?Phone)\s*[:=]\s*(.+?)\s*$`),
}

// extractAbuseContactFromRaw 从原始 WHOIS 响应中手动提取 Abuse 联系人信息
// whois-parser 没有专门的 Abuse contact 字段，需要从原始文本解析
func extractAbuseContactFromRaw(raw string) *WhoisContact {
	lines := strings.Split(raw, "\n")
	var abuse WhoisContact

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if matches := abusePatterns["email"].FindStringSubmatch(line); len(matches) == 2 {
			abuse.Email = matches[1]
			continue
		}

		if matches := abusePatterns["phone"].FindStringSubmatch(line); len(matches) == 2 {
			abuse.Phone = matches[1]
			continue
		}
	}

	if !isEmptyContact(abuse) {
		return &abuse
	}
	return nil
}

// contactFromParser 将 whois-parser 的 Contact 转换为前端需要的格式
func contactFromParser(c *whoisparser.Contact) WhoisContact {
	return WhoisContact{
		Name:       c.Name,
		Org:        c.Organization,
		Phone:      c.Phone,
		Email:      c.Email,
		Province:   c.Province,
		ContactUri: c.ReferralURL,
	}
}

// ianaIdPattern 匹配 Registrar IANA ID 字段（RFC 格式）
var ianaIdPattern = regexp.MustCompile(`(?i)^\s*Registrar\s+IANA\s+ID\s*[:=]\s*(\S+)`)

// extractIanaIdFromRaw 从原始 WHOIS 响应中提取 Registrar IANA ID
// whois-parser 未提供该字段，需手动解析
func extractIanaIdFromRaw(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if matches := ianaIdPattern.FindStringSubmatch(line); len(matches) == 2 {
			return matches[1]
		}
	}
	return ""
}

// ASNWhoisResult 包含 ASN WHOIS 查询的结构化结果
type ASNWhoisResult struct {
	ASNumber    string `json:"asNumber"`
	ASName      string `json:"asName"`
	OrgName     string `json:"orgName"`
	OrgID       string `json:"orgId"`
	Country     string `json:"country"`
	RegDate     string `json:"regDate"`
	Updated     string `json:"updated"`
	AbuseName   string `json:"abuseName"`
	AbuseEmail  string `json:"abuseEmail"`
	AbusePhone  string `json:"abusePhone"`
	Raw         string `json:"raw"`
	Error       string `json:"error"`
}

// ==================== ASN WHOIS 多 RIR 解析 ====================
//
// QueryASNWhois 走 likexian/whois 的默认链路：先问 IANA（IANA 按 ASN 给出归属 RIR），
// 再取该 RIR 的记录；若 ARIN 只回了「转介桩」，库会再跟一跳取到真实记录。
// 因此 raw 里可能同时存在两段内容，且字段名随 RIR 而异：
//
//	ARIN   : ASNumber / ASName / OrgName / OrgId / Country / RegDate / Updated / OrgAbuse*
//	APNIC  : aut-num / as-name / descr / country / org / last-modified（无 org-name 时组织名在 descr）
//	RIPE   : aut-num / as-name / org / org-name / country / created / last-modified
//	LACNIC : aut-num / owner / ownerid / country / created / changed
//
// 改造前只认 ARIN 字段 ⇒ 除 ARIN 注册的 ASN（Cloudflare AS13335、Google AS15169 等）外，
// 其余 RIR 的 ASN 全部解析为空（如中国电信 AS4134 只拿到 ARIN 转介桩的 "APNIC-4134"）。
//
// 现在的规则：raw 里**存在 RIR 原生 aut-num 记录时一律以它为准，并整体忽略 ARIN 段**
// （ARIN 对非 ARIN 的 ASN 只回桩，其 ASName/OrgName/OrgAbuseEmail 都是误导信息）；
// 没有原生记录时才按 ARIN 格式解析。
var (
	// 原生（APNIC/RIPE/LACNIC/AFRINIC）字段
	reASNAutNum    = regexp.MustCompile(`(?im)^\s*aut-num:\s*AS?(\d+)\s*$`)
	reASNAsName    = regexp.MustCompile(`(?im)^\s*as-name:\s*(.+?)\s*$`)
	reASNOrgName   = regexp.MustCompile(`(?im)^\s*org-name:\s*(.+?)\s*$`)
	reASNOwner     = regexp.MustCompile(`(?im)^\s*owner:\s*(.+?)\s*$`)
	reASNOwnerID   = regexp.MustCompile(`(?im)^\s*ownerid:\s*(.+?)\s*$`)
	reASNOrgRef    = regexp.MustCompile(`(?im)^\s*org:\s*(.+?)\s*$`)
	reASNDescr     = regexp.MustCompile(`(?im)^\s*descr:\s*(.+?)\s*$`)
	reASNCreated   = regexp.MustCompile(`(?im)^\s*created:\s*(.+?)\s*$`)
	reASNRegistered = regexp.MustCompile(`(?im)^\s*registered:\s*(.+?)\s*$`)
	reASNLastMod   = regexp.MustCompile(`(?im)^\s*last-modified:\s*(.+?)\s*$`)
	reASNChanged   = regexp.MustCompile(`(?im)^\s*changed:\s*(.+?)\s*$`)
	reASNMailbox   = regexp.MustCompile(`(?im)^\s*abuse-mailbox:\s*(\S+@\S+)\s*$`)
	// APNIC/RIPE 会在响应顶部给一行注释：% Abuse contact for 'AS37963' is 'xxx@yyy'
	reASNCommentAbuse = regexp.MustCompile(`(?i)%\s*Abuse contact for '?\s*AS\d+\s*'?\s*is\s*'?([^'\s]+)'?`)

	// ARIN 字段（保持改造前语义；注意：这些正则在「多行整段」文本上用，必须带 m 标志）
	reASNNumberArin  = regexp.MustCompile(`(?im)^\s*ASNumber\s*[:=]\s*(.+?)\s*$`)
	reASNNameArin    = regexp.MustCompile(`(?im)^\s*ASName\s*[:=]\s*(.+?)\s*$`)
	reASNHandleArin  = regexp.MustCompile(`(?im)^\s*ASHandle\s*[:=]\s*(.+?)\s*$`)
	reASNRegDateArin = regexp.MustCompile(`(?im)^\s*RegDate\s*[:=]\s*(.+?)\s*$`)
	reASNUpdatedArin = regexp.MustCompile(`(?im)^\s*Updated\s*[:=]\s*(.+?)\s*$`)
	reASNOrgNameArin = regexp.MustCompile(`(?im)^\s*OrgName\s*[:=]\s*(.+?)\s*$`)
	reASNOrgIDArin   = regexp.MustCompile(`(?im)^\s*OrgId\s*[:=]\s*(.+?)\s*$`)
	reASNCountryArin = regexp.MustCompile(`(?im)^\s*Country\s*[:=]\s*([A-Z]{2})\s*$`)
	reASNAbuseName   = regexp.MustCompile(`(?im)^\s*OrgAbuseName\s*[:=]\s*(.+?)\s*$`)
	reASNAbuseEmail  = regexp.MustCompile(`(?im)^\s*OrgAbuseEmail\s*[:=]\s*(.+?)\s*$`)
	reASNAbusePhone  = regexp.MustCompile(`(?im)^\s*OrgAbusePhone\s*[:=]\s*(.+?)\s*$`)

	// RIR 通用的「非 ASN 记录块」噪音（as-block 对象的 descr，如 "APNIC ASN block"）
	reASNBlockNoise = regexp.MustCompile(`(?i)(^|\s)ASN? block\s*$`)
	// 滥用联系人的对象名：RIPE 用 irt:，APNIC 的 abuse-c 对象用 role:
	reASNRoleName = regexp.MustCompile(`(?im)^\s*(?:irt|role):\s*(.+?)\s*$`)
	reASNPhone    = regexp.MustCompile(`(?im)^\s*phone:\s*(.+?)\s*$`)
)

// splitWhoisBlocks 按空行把响应切成记录块（每个 RIR 对象一段）
func splitWhoisBlocks(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	return strings.Split(raw, "\n\n")
}

// pickField 按给定优先级在文本中取第一个命中的字段值；全不命中返回 ""
func pickField(text string, patterns ...*regexp.Regexp) string {
	for _, pattern := range patterns {
		if m := pattern.FindStringSubmatch(text); len(m) == 2 {
			if v := strings.TrimSpace(m[1]); v != "" {
				return v
			}
		}
	}
	return ""
}

// findASNNativeBlock 找出含 aut-num 的原生记录块；没有返回 ""
func findASNNativeBlock(raw string) string {
	for _, block := range splitWhoisBlocks(raw) {
		if reASNAutNum.MatchString(block) {
			return block
		}
	}
	return ""
}

// findASNOrgBlock 按原生块里的 org: 句柄定位对应的 organisation/owner 对象
// （RIPE 的 org-name 与 country 不在 aut-num 对象里，而在 organisation 对象里）
func findASNOrgBlock(raw, nativeBlock string) string {
	handle := pickField(nativeBlock, reASNOrgRef)
	if handle == "" {
		return ""
	}
	for _, block := range splitWhoisBlocks(raw) {
		if strings.Contains(block, handle) && (reASNOrgName.MatchString(block) || reASNOwner.MatchString(block)) {
			return block
		}
	}
	return ""
}

// findASNOrgName 取组织名：organisation 对象的 org-name/owner（RIPE/APNIC/JPNIC）
// → 原生块首行 descr（APNIC 常见，如 "Hangzhou Alibaba Advertising Co.,Ltd."），并过滤 block 噪音
func findASNOrgName(orgBlock, nativeBlock string) string {
	if v := pickField(orgBlock, reASNOrgName, reASNOwner); v != "" {
		return v
	}
	for _, m := range reASNDescr.FindAllStringSubmatch(nativeBlock, -1) {
		v := strings.TrimSpace(m[1])
		if v == "" || reASNBlockNoise.MatchString(v) {
			continue
		}
		return v
	}
	return ""
}

// findASNCommentAbuseEmail 取响应顶部注释里的 Abuse 邮箱（APNIC/RIPE 提供）
func findASNCommentAbuseEmail(raw string) string {
	if m := reASNCommentAbuse.FindStringSubmatch(raw); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// findASNMailboxAbuse 取含 abuse-mailbox 的记录块里的联系人名与电话（RIPE 的 irt:、APNIC 的 role:）
func findASNMailboxAbuse(raw string) (name, email, phone string) {
	for _, block := range splitWhoisBlocks(raw) {
		mailbox := pickField(block, reASNMailbox)
		if mailbox == "" {
			continue
		}
		return pickField(block, reASNRoleName), mailbox, pickField(block, reASNPhone)
	}
	return "", "", ""
}

// parseASNWhoisRaw 从原始 ASN WHOIS 响应中解析结构化数据（多 RIR）
func parseASNWhoisRaw(raw string) *ASNWhoisResult {
	result := &ASNWhoisResult{Raw: raw}

	// 情况一：有 RIR 原生记录（APNIC/RIPE/LACNIC/AFRINIC），只认它
	if nativeBlock := findASNNativeBlock(raw); nativeBlock != "" {
		orgBlock := findASNOrgBlock(raw, nativeBlock)
		result.ASNumber = pickField(nativeBlock, reASNAutNum, reASNNumberArin)
		result.ASName = pickField(nativeBlock, reASNAsName)
		result.OrgName = findASNOrgName(orgBlock, nativeBlock)
		result.OrgID = pickField(nativeBlock, reASNOrgRef, reASNOwnerID)
		result.Country = pickField(nativeBlock, reASNCountryArin)
		if result.Country == "" {
			result.Country = pickField(orgBlock, reASNCountryArin)
		}
		result.RegDate = pickField(nativeBlock, reASNCreated, reASNRegistered, reASNRegDateArin)
		result.Updated = pickField(nativeBlock, reASNLastMod, reASNUpdatedArin, reASNChanged)

		// 滥用联系人：先用注释行（最权威），再用 abuse-mailbox 所在对象
		result.AbuseEmail = findASNCommentAbuseEmail(raw)
		name, email, phone := findASNMailboxAbuse(raw)
		if result.AbuseName == "" {
			result.AbuseName = name
		}
		if result.AbuseEmail == "" {
			result.AbuseEmail = email
		}
		result.AbusePhone = phone
		return result
	}

	// 情况二：ARIN 格式（ARIN 自有的 ASN），保持改造前行为
	result.ASNumber = pickField(raw, reASNNumberArin)
	if result.ASNumber == "" {
		result.ASNumber = strings.TrimPrefix(pickField(raw, reASNHandleArin), "AS")
	}
	result.ASName = pickField(raw, reASNNameArin)
	result.OrgName = pickField(raw, reASNOrgNameArin)
	result.OrgID = pickField(raw, reASNOrgIDArin)
	result.Country = pickField(raw, reASNCountryArin)
	result.RegDate = pickField(raw, reASNRegDateArin)
	result.Updated = pickField(raw, reASNUpdatedArin)
	result.AbuseName = pickField(raw, reASNAbuseName)
	result.AbuseEmail = pickField(raw, reASNAbuseEmail)
	result.AbusePhone = pickField(raw, reASNAbusePhone)

	return result
}

// QueryASNWhois 执行 ASN WHOIS 查询并解析结构化数据
// 使用 likexian/whois 库查询 ARIN WHOIS 服务器获取 ASN 详细信息
func QueryASNWhois(asn string) (*ASNWhoisResult, error) {
	// 确保 ASN 格式为 "AS" + 数字
	asn = strings.TrimSpace(asn)
	if !strings.HasPrefix(strings.ToUpper(asn), "AS") {
		asn = "AS" + asn
	}

	raw, err := whois.Whois(asn)
	if err != nil {
		return &ASNWhoisResult{
			ASNumber: strings.TrimPrefix(asn, "AS"),
			Error:    err.Error(),
		}, nil
	}

	result := parseASNWhoisRaw(raw)
	result.Raw = raw
	return result, nil
}
