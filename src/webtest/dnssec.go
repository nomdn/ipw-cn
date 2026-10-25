package webtest

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// —— DNSSEC 默认解析服务器 ——
//
// 根因（实测）：国内公共解析器（DNSPod 119.28.28.28、doh.pub、AliDNS 223.5.5.5、114DNS）
// 即使查询带 DO 位也不在应答里返回 RRSIG——DNSKEY 能查到、RRSIG 永远缺席。而
// dnssec 查询若回退到 dns-server（默认 119.28.28.28 或系统 DNS，都是国内递归），
// 验签流程永远失败。所以 DNSSEC 的默认服务器直接内置这份实测支持 DO 位的境外列表，
// 不再回退到国内解析器；境内网络若屏蔽/劫持了这些 UDP 53，请在节点配置 dnssec-server。
//
// 顺序即主从顺序，v4/v6 交错（同厂相邻）：单栈节点上错误协议族的拨号是
// "network unreachable" 秒级失败，failover 到下一台几乎零成本，双栈节点则
// 自动绕过被劫持/丢包的那一族。国内家宽实测（江苏移动 v4）8.8.8.8 主 IP 存在
// 随机丢包，1.1.1.1 更稳故放首位。注意境外 DoH（443）在国内普遍被阻断，全部走 53。
var defaultDOCapableServers = []string{
	"1.1.1.1:53",                    // Cloudflare v4
	"[2606:4700:4700::1111]:53",     // Cloudflare v6
	"8.8.8.8:53",                    // Google v4
	"[2001:4860:4860::8888]:53",     // Google v6
	"8.8.4.4:53",                    // Google 2 v4
	"[2001:4860:4860::8844]:53",     // Google 2 v6
	"9.9.9.9:53",                    // Quad9 v4
	"[2620:fe::fe]:53",              // Quad9 v6
	"208.67.222.222:53",             // OpenDNS v4
	"[2620:119:35::35]:53",          // OpenDNS v6
	"208.67.220.220:53",             // OpenDNS 2 v4
	"149.112.112.112:53",            // Quad9 2 v4
	"[2620:fe::9]:53",               // Quad9 2 v6
	"[2620:119:53::53]:53",          // OpenDNS 2 v6
}

// queryDNSSEC DNSSEC 查询：走内置 DO 友好列表（或 dnssec-server 配置，配置优先），
// 由 queryWithServers 的主从 failover 保证可用性。
func queryDNSSEC(msg *dns.Msg) (*dns.Msg, error) {
	return queryWithServers(msg, dnssecServers)
}

// DNSSECResult 包含 DNSSEC 验证的结果
type DNSSECResult struct {
	Domain     string  `json:"domain"`
	Enabled    bool    `json:"enabled"`
	Valid      bool    `json:"valid"`
	HasRRSIG   bool    `json:"has_rrsig"`
	HasDNSKEY  bool    `json:"has_dnskey"`
	HasDS      bool    `json:"has_ds"`
	Algorithm  uint8   `json:"algorithm"`
	KeyTag     uint16  `json:"key_tag"`
	SignerName string  `json:"signer_name"`
	Validation string  `json:"validation"`
	Duration   float64 `json:"duration"`
}

// ResolveDNSSEC 执行 DNSSEC 验证
// 流程：
//  1. 查询 DNSKEY 记录，获取密钥 RRset
//  2. 查询 A 记录（带 DO 位），获取 A RRset + RRSIG
//  3. 用 DNSKEY 中的密钥逐一对 RRSIG.Verify(dnskey, aRRset) 验证
//  4. 查询 DS 记录，检查链式信任
func ResolveDNSSEC(domain string) (*DNSSECResult, error) {
	start := time.Now()
	result := &DNSSECResult{
		Domain: domain,
	}

	// 1. 查询 DNSKEY 记录
	msgDNSKEY := new(dns.Msg)
	msgDNSKEY.SetQuestion(dns.Fqdn(domain), dns.TypeDNSKEY)
	msgDNSKEY.SetEdns0(4096, true)

	responseDNSKEY, err := queryDNSSEC(msgDNSKEY)
	result.Duration = time.Since(start).Seconds() * 1000

	if err != nil {
		result.Validation = fmt.Sprintf("DNSKEY query failed: %v", err)
		return result, nil
	}

	if responseDNSKEY.Rcode != dns.RcodeSuccess {
		result.Validation = fmt.Sprintf("DNSKEY query failed with Rcode %d", responseDNSKEY.Rcode)
		return result, nil
	}

	var dnskeyRRset []dns.RR
	var dnskeyList []*dns.DNSKEY
	// DO 位生效时，DNSKEY 应答自带 RRSIG(DNSKEY)——已签名域名的确定信号，
	// 也是 TLD / 无 A 记录域名唯一的验签素材（它们没有 A 应答可拿 RRSIG(A)）
	var rrsigDNSKEYList []*dns.RRSIG
	for _, ans := range responseDNSKEY.Answer {
		switch v := ans.(type) {
		case *dns.DNSKEY:
			dnskeyRRset = append(dnskeyRRset, v)
			dnskeyList = append(dnskeyList, v)
			result.HasDNSKEY = true
		case *dns.RRSIG:
			rrsigDNSKEYList = append(rrsigDNSKEYList, v)
		}
	}
	if len(rrsigDNSKEYList) > 0 {
		result.HasRRSIG = true
	}

	if len(dnskeyRRset) > 0 {
		result.KeyTag = dnskeyList[0].KeyTag()
		result.Algorithm = dnskeyList[0].Algorithm
	}

	// 2. A 与 DS 并行查询。DS 记录在父区（如 .com），**任何**验签结论（通过或失败）
	//    都应带上它——原实现把 DS 查询放在函数最后，验签成功时提前 return，
	//    has_ds 永远是 false（只有验证失败的路径才查得到 DS）。
	msgA := new(dns.Msg)
	msgA.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	msgA.SetEdns0(4096, true)
	msgDS := new(dns.Msg)
	msgDS.SetQuestion(dns.Fqdn(domain), dns.TypeDS)
	msgDS.SetEdns0(4096, true)

	// rrsigList 提升到函数作用域：A 应答里的 RRSIG(A) 在「A 验签失败」的最终判定里也要用
	var rrsigList []*dns.RRSIG
	var responseA *dns.Msg
	var responseDS *dns.Msg
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		responseA, err = queryDNSSEC(msgA)
	}()
	go func() {
		defer wg.Done()
		responseDS, _ = queryDNSSEC(msgDS)
	}()
	wg.Wait()

	// DS 结论先于一切 early return 就绪
	if responseDS != nil && responseDS.Rcode == dns.RcodeSuccess {
		for _, ans := range responseDS.Answer {
			if _, ok := ans.(*dns.DS); ok {
				result.HasDS = true
				break
			}
		}
	}

	if err == nil && responseA.Rcode == dns.RcodeSuccess {
		var aRRset []dns.RR
		for _, ans := range responseA.Answer {
			if aRecord, ok := ans.(*dns.A); ok {
				aRRset = append(aRRset, aRecord)
			}
		}

		for _, ans := range responseA.Answer {
			if rrsig, ok := ans.(*dns.RRSIG); ok {
				rrsigList = append(rrsigList, rrsig)
				result.HasRRSIG = true
			}
		}

		// 3. 用 DNSKEY 逐条验证 A 的 RRSIG。不在验证失败时提前返回：
		//    下面还有 DNSKEY 自验签兜底
		for _, rrsig := range rrsigList {
			for _, dnskey := range dnskeyList {
				if keyErr := rrsig.Verify(dnskey, aRRset); keyErr == nil {
					result.Enabled = true
					result.Valid = true
					result.Algorithm = dnskey.Algorithm
					result.KeyTag = dnskey.KeyTag()
					result.SignerName = rrsig.SignerName
					result.Validation = fmt.Sprintf("DNSSEC 验证通过 (算法: %d, KeyTag: %d)", dnskey.Algorithm, dnskey.KeyTag())
					return result, nil
				}
			}
		}
	}

	// 3.5 DNSKEY 自验签：用 DNSKEY 验证 DNSKEY 自己的 RRSIG（RFC 4035 §5.3.3 的
	// 自签检查）。TLD（如 cn）与无 A 记录的已签名域拿不到 RRSIG(A)，这是它们的
	// 唯一验签路径；有 A 记录但 A 验签失败的域走到这里仍失败才算真失败。
	if len(rrsigDNSKEYList) > 0 && len(dnskeyRRset) > 0 {
		for _, rrsig := range rrsigDNSKEYList {
			for _, dnskey := range dnskeyList {
				if keyErr := rrsig.Verify(dnskey, dnskeyRRset); keyErr == nil {
					result.Enabled = true
					result.Valid = true
					result.Algorithm = dnskey.Algorithm
					result.KeyTag = dnskey.KeyTag()
					result.SignerName = rrsig.SignerName
					result.Validation = fmt.Sprintf("DNSSEC 验证通过 (DNSKEY 自验签, 算法: %d, KeyTag: %d)", dnskey.Algorithm, dnskey.KeyTag())
					return result, nil
				}
			}
		}
	}

	// A 验签与 DNSKEY 自验签都未通过：只要任一应答里有 RRSIG 就说明签名存在但无效
	//（真失败，如数据被污染/密钥轮换不完整），返回明确的失败结论
	if (len(rrsigList) > 0 || len(rrsigDNSKEYList) > 0) && len(dnskeyRRset) > 0 {
		result.Enabled = true
		result.Valid = false
		result.Validation = fmt.Sprintf("RRSIG 验证失败: %d 条 RRSIG(A), %d 条 RRSIG(DNSKEY), %d 个 DNSKEY，无匹配签名", len(rrsigList), len(rrsigDNSKEYList), len(dnskeyRRset))
		return result, nil
	}

	// 4. 结论分支（DS 已在并行阶段查完并计入 HasDS）
	if result.HasRRSIG && result.HasDNSKEY {
		result.Enabled = true
		result.Valid = false
		result.Validation = "存在 RRSIG 和 DNSKEY，但签名验证未通过"
	} else if result.HasRRSIG {
		result.Enabled = true
		result.Valid = false
		result.Validation = "存在 RRSIG，但缺少 DNSKEY"
	} else if result.HasDNSKEY {
		result.Enabled = false
		result.Valid = false
		result.Validation = "存在 DNSKEY，但缺少 RRSIG（若确认该域名已签名，说明默认境外解析器在本网络不可达，请在节点配置 dnssec-server 指向可达的 DNSSEC 解析器，如 8.8.8.8:53）"
	} else {
		result.Enabled = false
		result.Valid = false
		result.Validation = "未检测到 DNSSEC 记录"
	}

	return result, nil
}

// ResolveDNSSECForRecord 针对特定记录类型进行 DNSSEC 验证
func ResolveDNSSECForRecord(domain string, recordType uint16) (*DNSSECResult, error) {
	start := time.Now()
	result := &DNSSECResult{
		Domain: domain,
	}

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), recordType)
	msg.SetEdns0(4096, true)

	response, err := queryDNSSEC(msg)
	result.Duration = time.Since(start).Seconds() * 1000

	if err != nil {
		result.Validation = fmt.Sprintf("DNS query failed: %v", err)
		return result, nil
	}

	if response.Rcode != dns.RcodeSuccess {
		result.Validation = fmt.Sprintf("DNS query failed with Rcode %d", response.Rcode)
		return result, nil
	}

	var targetRRset []dns.RR
	var rrsigList []*dns.RRSIG
	for _, ans := range response.Answer {
		switch v := ans.(type) {
		case *dns.A:
			targetRRset = append(targetRRset, v)
		case *dns.AAAA:
			targetRRset = append(targetRRset, v)
		case *dns.MX:
			targetRRset = append(targetRRset, v)
		case *dns.TXT:
			targetRRset = append(targetRRset, v)
		case *dns.NS:
			targetRRset = append(targetRRset, v)
		case *dns.CNAME:
			targetRRset = append(targetRRset, v)
		case *dns.SOA:
			targetRRset = append(targetRRset, v)
		case *dns.RRSIG:
			rrsigList = append(rrsigList, v)
		}
	}

	if len(rrsigList) > 0 {
		result.HasRRSIG = true
	}

	// 查询 DNSKEY
	msgDNSKEY := new(dns.Msg)
	msgDNSKEY.SetQuestion(dns.Fqdn(domain), dns.TypeDNSKEY)
	msgDNSKEY.SetEdns0(4096, true)

	responseDNSKEY, err := queryDNSSEC(msgDNSKEY)
	if err == nil && responseDNSKEY.Rcode == dns.RcodeSuccess {
		var dnskeyRRset []dns.RR
		for _, ans := range responseDNSKEY.Answer {
			if dnskey, ok := ans.(*dns.DNSKEY); ok {
				dnskeyRRset = append(dnskeyRRset, dnskey)
				result.HasDNSKEY = true
			}
		}

		for _, rrsig := range rrsigList {
			for _, dnskey := range dnskeyRRset {
				if keyErr := rrsig.Verify(dnskey.(*dns.DNSKEY), targetRRset); keyErr == nil {
					result.Enabled = true
					result.Valid = true
					result.Algorithm = dnskey.(*dns.DNSKEY).Algorithm
					result.KeyTag = dnskey.(*dns.DNSKEY).KeyTag()
					result.Validation = "DNSSEC 签名验证通过"
					return result, nil
				}
			}
		}

		if len(rrsigList) > 0 && len(dnskeyRRset) > 0 {
			result.Enabled = true
			result.Valid = false
			result.Validation = "RRSIG 验证失败"
		}
	}

	// 查询 DS 记录
	msgDS := new(dns.Msg)
	msgDS.SetQuestion(dns.Fqdn(domain), dns.TypeDS)
	responseDS, _ := queryDNSSEC(msgDS)
	if responseDS != nil && responseDS.Rcode == dns.RcodeSuccess {
		for _, ans := range responseDS.Answer {
			if _, ok := ans.(*dns.DS); ok {
				result.HasDS = true
				break
			}
		}
	}

	if !result.HasRRSIG {
		result.Validation = "该记录类型未启用 DNSSEC"
	}

	return result, nil
}

// FormatDNSSECResult 将 DNSSECResult 格式化为易读的字符串
func FormatDNSSECResult(r *DNSSECResult) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("域名: %s\n", r.Domain))
	sb.WriteString(fmt.Sprintf("DNSSEC 状态: %s\n", map[bool]string{true: "已启用", false: "未启用"}[r.Enabled]))
	sb.WriteString(fmt.Sprintf("验证状态: %s\n", map[bool]string{true: "通过", false: "未通过"}[r.Valid]))
	sb.WriteString(fmt.Sprintf("RRSIG 记录: %s\n", map[bool]string{true: "存在", false: "不存在"}[r.HasRRSIG]))
	sb.WriteString(fmt.Sprintf("DNSKEY 记录: %s\n", map[bool]string{true: "存在", false: "不存在"}[r.HasDNSKEY]))
	sb.WriteString(fmt.Sprintf("DS 记录: %s\n", map[bool]string{true: "存在", false: "不存在"}[r.HasDS]))
	sb.WriteString(fmt.Sprintf("验证信息: %s\n", r.Validation))
	sb.WriteString(fmt.Sprintf("查询耗时: %.2f ms\n", r.Duration))
	return sb.String()
}
