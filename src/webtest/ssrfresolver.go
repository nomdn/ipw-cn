package webtest

import (
	"context"
	"net"
	"strings"

	"lemon-ipw/ssrf"
)

// InstallSSRFResolver 让 SSRF 出站校验走本包的自定义 DNS（含请求级覆盖）。
// 必须在启动阶段调用一次（main.go）。
//
// 动机：不注入时 ssrf 用 net.LookupIP（系统解析器）做预检，而真正拨号走自定义 DNS —— 两套解析器，
// 既多一次查询，结果也可能不一致；更关键的是，当调用方用请求级 dns= 指定了服务器、且目标域名只能
// 靠该服务器解析时，系统解析器预检会先行失败，把请求直接毙掉，覆盖形同失效。
//
// 校验只需判定「是否私网」，故 A / AAAA 都查（任一命中私网即拦，口径与原先 net.LookupIP 一致）。
// 请求级配置（DNS 服务器）取自 context，与真正拨号同源。
func InstallSSRFResolver() {
	ssrf.SetResolveFunc(func(ctx context.Context, host string) ([]net.IP, error) {
		// 字面 IP 直接返回，无需解析
		if ip := net.ParseIP(host); ip != nil {
			return []net.IP{ip}, nil
		}

		req := ReqResolveFromContext(ctx)

		var ips []net.IP
		var firstErr error
		for _, wantV6 := range []bool{false, true} {
			// 强制解析命中时，预检要验「拨号真正会连的地址」，否则会拿原域名的
			// 真实解析去判定，预检与实际连接两个口径对不上（resolve= 也就白设了）。
			name := host
			if target := req.Override(name, wantV6); target != "" {
				name = strings.Trim(target, "[]")
			}
			if ip := net.ParseIP(name); ip != nil {
				ips = append(ips, ip)
				continue
			}
			var res DNSResult
			var err error
			if wantV6 {
				res, err = ResolveAAAARecord(name, req.Servers("ipv6"))
			} else {
				res, err = ResolveARecord(name, req.Servers("ipv4"))
			}
			if err != nil && firstErr == nil {
				firstErr = err
			}
			for _, s := range res.Record {
				if ip := net.ParseIP(s); ip != nil {
					ips = append(ips, ip)
				}
			}
		}

		// 两栈都解析失败才报错；只要有一栈拿到 IP 就交给上层做私网判定
		if len(ips) == 0 && firstErr != nil {
			return nil, firstErr
		}
		return ips, nil
	})
}
