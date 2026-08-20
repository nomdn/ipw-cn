package webtest

import (
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

// measureDNSTime 当 trace 未提供 DNS 耗时时的兜底测量
func measureDNSTime(urlStr string, version string) float64 {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return 0
	}
	host := parsed.Hostname()
	if host == "" {
		return 0
	}
	start := time.Now()
	var dnsErr error
	if version == "v6" {
		_, dnsErr = ResolveAAAARecord(host)
	} else {
		_, dnsErr = ResolveARecord(host)
	}
	if dnsErr != nil {
		return 0
	}
	return time.Since(start).Seconds() * 1000
}
