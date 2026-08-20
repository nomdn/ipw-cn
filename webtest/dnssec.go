package webtest

import (
	"fmt"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// DNSSECResult 包含 DNSSEC 验证的结果
type DNSSECResult struct {
	Domain      string  `json:"domain"`
	Enabled     bool    `json:"enabled"`
	Valid       bool    `json:"valid"`
	HasRRSIG    bool    `json:"has_rrsig"`
	HasDNSKEY   bool    `json:"has_dnskey"`
	HasDS       bool    `json:"has_ds"`
	Algorithm   uint8   `json:"algorithm"`
	KeyTag      uint16  `json:"key_tag"`
	SignerName  string  `json:"signer_name"`
	Validation  string  `json:"validation"`
	Duration    float64 `json:"duration"`
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
	for _, ans := range responseDNSKEY.Answer {
		if dnskey, ok := ans.(*dns.DNSKEY); ok {
			dnskeyRRset = append(dnskeyRRset, dnskey)
			dnskeyList = append(dnskeyList, dnskey)
			result.HasDNSKEY = true
		}
	}

	if len(dnskeyRRset) > 0 {
		result.KeyTag = dnskeyList[0].KeyTag()
		result.Algorithm = dnskeyList[0].Algorithm
	}

	// 2. 查询 A 记录（带 DO 位）
	msgA := new(dns.Msg)
	msgA.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	msgA.SetEdns0(4096, true)

	responseA, err := queryDNSSEC(msgA)
	if err == nil && responseA.Rcode == dns.RcodeSuccess {
		var aRRset []dns.RR
		for _, ans := range responseA.Answer {
			if aRecord, ok := ans.(*dns.A); ok {
				aRRset = append(aRRset, aRecord)
			}
		}

		var rrsigList []*dns.RRSIG
		for _, ans := range responseA.Answer {
			if rrsig, ok := ans.(*dns.RRSIG); ok {
				rrsigList = append(rrsigList, rrsig)
				result.HasRRSIG = true
			}
		}

		// 3. 用 DNSKEY 逐条验证 RRSIG
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

		if len(rrsigList) > 0 && len(dnskeyRRset) > 0 {
			result.Enabled = true
			result.Valid = false
			result.Validation = fmt.Sprintf("RRSIG 验证失败: %d 条 RRSIG, %d 个 DNSKEY，无匹配签名", len(rrsigList), len(dnskeyRRset))
			return result, nil
		}
	}

	// 4. 查询 DS 记录
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
		result.Validation = "存在 DNSKEY，但缺少 RRSIG"
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
