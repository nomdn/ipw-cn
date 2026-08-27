package webtest

import (
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

// QueryWhois 执行 WHOIS 查询并解析结构化数据
// 使用 likexian/whois 库查询原始响应，再用 whois-parser 解析为结构化数据
// 首次失败后 fallback：内置 TLD 映射 → IANA 查询服务器地址 →
// 用 ResolveA/AAAA(自定义DNS) 解析IP → Happy Eyeballs 双栈竞争拨号
func QueryWhois(domain string) (*WhoisResult, error) {
	raw, err := whois.Whois(domain)
	if err != nil {
		raw, err = whoisRetryWithFallback(domain, err)
	}
	result := parseWhoisResult(domain, raw)
	result.Error = errString(err)
	return result, nil
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
func happyEyeballsWhoisQuery(domain string, v4IPs, v6IPs []string, port string) (string, error) {
	if len(v4IPs) == 0 && len(v6IPs) == 0 {
		return "", fmt.Errorf("no IPs available for WHOIS query")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type outcome struct {
		raw string
		err error
	}
	// 有缓冲：保证每个 goroutine 写入不被阻塞，避免 goroutine 泄漏
	ch := make(chan outcome, len(v4IPs)+len(v6IPs))
	pending := 0

	tryIP := func(ip string) {
		raw, err := rawWhoisQueryCtx(ctx, domain, ip, port)
		ch <- outcome{raw: raw, err: err}
	}

	// 1) 立即启动所有 v4
	for _, ip := range v4IPs {
		pending++
		go tryIP(ip)
	}

	// 2) 定时器：v6 延迟启动
	var v6Timer *time.Timer
	if len(v6IPs) > 0 {
		v6Timer = time.AfterFunc(happyEyeballsV6Delay, func() {
			for _, ip := range v6IPs {
				pending++
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
		// 注意：pending 是在 goroutine 内自增，这里用固定上限做兜底计数
		// 如果 v6 还没启动完，但 v4 已经全部失败也没有成功，需要等待 pending 全部发完
		// 简化起见：若 i 到达 len(v4IPs) 但 v6 还在 pending，我们再等一会
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
	raw, err := happyEyeballsWhoisQuery(domain, v4IPs, v6IPs, "43")
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

	// 3. 读取响应
	buf := make([]byte, 65536)
	n, readErr := conn.Read(buf)

	// 有数据就读到了，哪怕 readErr != nil 也先返回数据
	if n > 0 {
		return string(buf[:n]), nil
	}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		return "", readErr
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

// asnFieldPatterns 用于从原始 ASN WHOIS 文本中提取字段
var asnFieldPatterns = map[string]*regexp.Regexp{
	"asNumber":    regexp.MustCompile(`(?i)^\s*ASNumber\s*[:=]\s*(.+?)\s*$`),
	"asName":      regexp.MustCompile(`(?i)^\s*ASName\s*[:=]\s*(.+?)\s*$`),
	"asHandle":    regexp.MustCompile(`(?i)^\s*ASHandle\s*[:=]\s*(.+?)\s*$`),
	"regDate":     regexp.MustCompile(`(?i)^\s*RegDate\s*[:=]\s*(.+?)\s*$`),
	"updated":     regexp.MustCompile(`(?i)^\s*Updated\s*[:=]\s*(.+?)\s*$`),
	"orgName":     regexp.MustCompile(`(?i)^\s*OrgName\s*[:=]\s*(.+?)\s*$`),
	"orgId":       regexp.MustCompile(`(?i)^\s*OrgId\s*[:=]\s*(.+?)\s*$`),
	"country":     regexp.MustCompile(`(?i)^\s*Country\s*[:=]\s*([A-Z]{2})\s*$`),
	"abuseName":   regexp.MustCompile(`(?i)^\s*OrgAbuseName\s*[:=]\s*(.+?)\s*$`),
	"abuseEmail":  regexp.MustCompile(`(?i)^\s*OrgAbuseEmail\s*[:=]\s*(.+?)\s*$`),
	"abusePhone":  regexp.MustCompile(`(?i)^\s*OrgAbusePhone\s*[:=]\s*(.+?)\s*$`),
}

// parseASNWhoisRaw 从原始 ASN WHOIS 响应中解析结构化数据
func parseASNWhoisRaw(raw string) *ASNWhoisResult {
	result := &ASNWhoisResult{Raw: raw}
	lines := strings.Split(raw, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%") || strings.HasPrefix(line, "#") {
			continue
		}

		for field, pattern := range asnFieldPatterns {
			if matches := pattern.FindStringSubmatch(line); len(matches) == 2 {
				switch field {
				case "asNumber":
					result.ASNumber = matches[1]
				case "asName":
					result.ASName = matches[1]
				case "asHandle":
					if result.ASNumber == "" {
						result.ASNumber = strings.TrimPrefix(matches[1], "AS")
					}
				case "regDate":
					result.RegDate = matches[1]
				case "updated":
					result.Updated = matches[1]
				case "orgName":
					result.OrgName = matches[1]
				case "orgId":
					result.OrgID = matches[1]
				case "country":
					result.Country = matches[1]
				case "abuseName":
					result.AbuseName = matches[1]
				case "abuseEmail":
					result.AbuseEmail = matches[1]
				case "abusePhone":
					result.AbusePhone = matches[1]
				}
			}
		}
	}

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
