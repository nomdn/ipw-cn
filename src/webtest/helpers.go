package webtest

import (
	"net"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

// CleanHostRecord 从 trace 的 RemoteAddr 提取纯主机名（兼容 IPv6 方括号）
func CleanHostRecord(addr string) string {
	if strings.HasPrefix(addr, "[") {
		rightBracket := strings.Index(addr, "]")
		if rightBracket != -1 {
			return addr[1:rightBracket]
		}
	}
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// measureDNSTime 当 trace 未提供 DNS 耗时时的兜底测量。req 为请求级解析配置（dns 服务器）。
func measureDNSTime(urlStr string, version string, req ReqResolve) float64 {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return 0
	}
	host := parsed.Hostname()
	if host == "" {
		return 0
	}
	// 强制解析：目标是 IP 时原域名根本不查 DNS，不能报一个凭空的耗时；
	// 目标是域名则改按新 host 计时（真实查询的是它）。
	if target := req.Override(host, version == "v6"); target != "" {
		host = strings.Trim(target, "[]")
		if net.ParseIP(host) != nil {
			return 0
		}
	}
	start := time.Now()
	var dnsErr error
	if version == "v6" {
		_, dnsErr = ResolveAAAARecord(host, req.Servers("ipv6"))
	} else {
		_, dnsErr = ResolveARecord(host, req.Servers("ipv4"))
	}
	if dnsErr != nil {
		return 0
	}
	return time.Since(start).Seconds() * 1000
}
func AddrPortsToCSV(addrs []netip.AddrPort) string {
	if len(addrs) == 0 {
		return ""
	}
	parts := make([]string, len(addrs))
	for i, addr := range addrs {
		parts[i] = addr.String() // 自动格式化为 "IP:Port"
	}
	return strings.Join(parts, ",")
}
