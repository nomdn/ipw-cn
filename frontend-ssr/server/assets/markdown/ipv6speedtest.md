指标与 [IPv4 网站测速](/speedtest) 完全一致（状态码、总时间、解析时间、连接耗时、网页大小、下载速度），差别在于探测强制走 IPv6 链路。

## 双栈环境下的三种常见情况

- **两端都可达**：客户端按 RFC 6724 前缀策略在 IPv4 / IPv6 之间选择，不同节点可能选了不同协议，结果自然有差异。
- **IPv6 较慢**：IPv6 链路未做优化、互联互通绕行，首字节耗时通常比 IPv4 高；这不一定代表配置错误。
- **IPv6 完全不通**：先看站点是否真的具备 IPv6 访问能力，再看本机是否有 IPv6 出口。

## 结果异常时的排查顺序

1. 用 [IPv6 网站检测](/ipv6webcheck) 确认 AAAA 记录、服务监听、防火墙三件事；
2. 用 [DNS 查询](/dns) 看 AAAA 记录是否指向预期地址；
3. 用 [IPv6 TCPing](/ipv6tcping) 判断是端口不通还是页面慢。

需要理解为什么系统有时优先走 IPv4，可阅读 [Windows 10/11 设置 IPv4/IPv6 访问优先级](/doc/user/ipv4_ipv6_prefix_precedence)。

## 相关工具与文档

- [IPv4 网站测速](/speedtest) · [IPv6 网站检测](/ipv6webcheck) · [IPv6 TCPing](/ipv6tcping)
- [RFC 8200：IPv6 协议规范](/doc/rfc/rfc8200)
- [IPv6 检测方法与工具](/ipv6check)：双栈优先级问题该从哪一步查起
