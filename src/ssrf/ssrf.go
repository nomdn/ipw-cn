package ssrf

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
)

// blockPrivateIPs controls whether SSRF protection is active.
// blockPrivateIPs 控制 SSRF 防护是否启用。
// Defaults to true; can be toggled via BLOCK_PRIVATE_IPS environment variable.
// 默认为 true；可通过 BLOCK_PRIVATE_IPS 环境变量切换。
var blockPrivateIPs = true

// contextKey is an unexported type for context keys to avoid collisions with other packages.
// contextKey 是用于上下文键的未导出类型，避免与其他包的键冲突。
type contextKey string

// validatedIPsKey stores the resolved IPs that have passed SSRF validation in the request context.
// validatedIPsKey 在请求上下文中存储已通过 SSRF 校验的解析 IP。
// Caching validated IPs in context avoids redundant DNS lookups in DialContext.
// 在上下文中缓存校验通过的 IP，避免 DialContext 中重复 DNS 查询。
const validatedIPsKey contextKey = "ssrf_validated_ips"

// ValidatedIPsKey returns the context key used to store validated IPs.
// ValidatedIPsKey 返回用于存储校验通过 IP 的上下文键。
func ValidatedIPsKey() contextKey {
	return validatedIPsKey
}

// Enabled returns whether SSRF protection is currently active.
// Enabled 返回 SSRF 防护当前是否启用。
func Enabled() bool {
	return blockPrivateIPs
}

// SetEnabled enables or disables SSRF protection globally.
// SetEnabled 全局启用或禁用 SSRF 防护。
// Typically called during configuration loading (readConfig).
// 通常在配置加载（readConfig）时调用。
func SetEnabled(v bool) {
	blockPrivateIPs = v
}

// Go 标准库 IsPrivate 只覆盖 RFC1918 与 fd00::/8，这里补齐其余不应出网的内部/特殊网段
var extraPrivateNets = mustParseCIDRs(
	"100.64.0.0/10", // CGNAT：运营商级 NAT、云内网、Tailscale 等
	"198.18.0.0/15", // 网络基准测试保留段
	"192.0.0.0/24",  // IETF 协议分配
	"fc00::/7",      // IPv6 ULA（IsPrivate 只认 fd00::/8）
	"64:ff9b::/96",  // NAT64：内嵌 IPv4 会被网关转换
)

func mustParseCIDRs(cidrs ...string) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, ipNet, err := net.ParseCIDR(c)
		if err != nil {
			panic(fmt.Sprintf("invalid CIDR %q: %v", c, err))
		}
		nets = append(nets, ipNet)
	}
	return nets
}

// IsPrivateIP checks if the given IP address belongs to a private or internal range.
// IsPrivateIP 检查给定的 IP 地址是否属于私有或内部地址段。
// It covers RFC 1918 private addresses, loopback, link-local, and unspecified addresses,
// plus CGNAT/ULA/NAT64/benchmark/multicast ranges that Go's IsPrivate does not cover.
// 涵盖 RFC 1918 私有地址、回环地址、链路本地地址和未指定地址，
// 以及 Go 标准库未覆盖的 CGNAT/ULA/NAT64/基准测试/组播段。
// This is the core check used to block SSRF attacks targeting internal networks.
// 这是用于阻止针对内部网络的 SSRF 攻击的核心检查。
func IsPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	if ip.Equal(net.IPv4bcast) {
		return true
	}
	for _, n := range extraPrivateNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// HasLocalOrPrivateIP checks whether the given hostname resolves to any private or internal IP address.
// HasLocalOrPrivateIP 检查给定的主机名是否解析到任何私有或内部 IP 地址。
// Returns true if at least one resolved IP is private; false if resolution fails or all IPs are public.
// 如果至少有一个解析的 IP 是私有地址则返回 true；解析失败或全部为公网 IP 时返回 false。
// Used in handlers to short-circuit requests to internal hosts before making outbound connections.
// 用于在发起出站连接前，在处理器中短路对内部主机的请求。
func HasLocalOrPrivateIP(host string) bool {
	ips, err := net.LookupIP(host)
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if IsPrivateIP(ip) {
			return true
		}
	}
	return false
}

// ValidateOutboundTarget validates the target URL against SSRF attacks before any outbound request is made.
// ValidateOutboundTarget 在发起任何出站请求之前，对目标 URL 进行 SSRF 攻击校验。
//
// Security checks performed:
// 执行的 security 检查：
//  1. Scheme whitelist — only "http" and "https" are allowed; blocks file://, gopher://, etc.
//     Scheme 白名单 — 仅允许 "http" 和 "https"；阻止 file://、gopher:// 等。
//  2. Hostname resolution — resolves the hostname via DNS.
//     主机名解析 — 通过 DNS 解析主机名。
//  3. Private IP check — verifies none of the resolved IPs are private/internal.
//     私有 IP 检查 — 验证所有解析的 IP 都不是私有/内部地址。
//  4. Context caching — validated IPs are stored in the returned context to avoid redundant DNS lookups.
//     上下文缓存 — 校验通过的 IP 存储在返回的上下文中，避免重复 DNS 查询。
//
// If validation passes, the returned context carries the resolved IPs under ValidatedIPsKey().
// 如果校验通过，返回的上下文中会携带解析的 IP，键为 ValidatedIPsKey()。
func ValidateOutboundTarget(ctx context.Context, targetURL string) (context.Context, error) {
	if !blockPrivateIPs {
		return ctx, nil
	}
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return ctx, err
	}
	host := parsed.Hostname()
	if host == "" {
		return ctx, fmt.Errorf("empty host")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ctx, fmt.Errorf("invalid scheme: %s", parsed.Scheme)
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return ctx, err
	}
	for _, ip := range ips {
		if IsPrivateIP(ip) {
			slog.Warn("Blocked request to private IP", "host", host, "ip", ip)
			return ctx, fmt.Errorf("request to private/internal address is not allowed")
		}
	}
	return context.WithValue(ctx, validatedIPsKey, ips), nil
}

// SecureCheckRedirect is a redirect policy that prevents SSRF via HTTP redirect chains.
// SecureCheckRedirect 是一个重定向策略，防止通过 HTTP 重定向链进行 SSRF 攻击。
//
// req 是即将跳转的目标（via 是已访问的链）。必须校验目标本身：
// 只检查 via 会漏掉重定向链的最后一跳。每一跳的目标都会被解析并检查
// 是否指向私有/内部 IP 段，同时限制重定向深度。
// 这阻止攻击者通过链式重定向最终到达私有 IP。
func SecureCheckRedirect(req *http.Request, via []*http.Request) error {
	if !blockPrivateIPs {
		return nil
	}
	if req == nil || req.URL == nil {
		return fmt.Errorf("invalid redirect target")
	}
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return fmt.Errorf("invalid redirect scheme: %s", req.URL.Scheme)
	}
	host := req.URL.Hostname()
	if host == "" {
		return fmt.Errorf("redirect target has empty host")
	}
	if len(via) >= 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return err
	}
	for _, ip := range ips {
		if IsPrivateIP(ip) {
			slog.Warn("Blocked redirect to private IP", "host", host, "ip", ip)
			return fmt.Errorf("redirect to private/internal address is not allowed")
		}
	}
	return nil
}
