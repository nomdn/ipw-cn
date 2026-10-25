export interface DocMeta {
  title: string
  description: string
  // —— SEO 专用字段（可选）——
  // 审计（Bing 站长工具）判定 10 个页面标题过短、13 个页面 meta description 过短，
  // 全是文档页：上面的 title/description 同时用于侧栏菜单与卡片展示，不适合拉长，
  // 因此文档页的 <title> 与 meta 优先取这里的加长版（见 doc 两个页面的 useHead）。
  seoTitle?: string
  seoDescription?: string
}

export const docConfig: Record<string, DocMeta> = {
  '/doc': {
    title: 'IPv6 工具箱使用文档',
    description: '致力于普及 IPv6，推进 IPv6 规模部署和应用，以全面推进 IPv6 技术创新与融合应用为主线，以提升应用广度深度为主攻方向。支持 IPv6 本地网络（宽带）开启验证、IPv6 网站开启检测、IPv6 Ping 延迟测试、IPv6 网站测速、IPv6 地址查询、IPv6 DNS 解析。',
    seoTitle: 'IPv6 工具箱使用文档：宽带开启、网站检测、测速与 DNS 解析教程',
    seoDescription: 'IPv6 工具箱完整使用文档：涵盖家庭宽带开启 IPv6、网站 IPv6 支持检测、IPv6 Ping 延迟测试、IPv6 网站测速、IP 归属地查询与 IPv6 DNS 解析等教程，致力于普及 IPv6、推进 IPv6 规模部署和应用。'
  },
  '/doc/user/enable_ipv6': {
    title: '个人宽带如何开启IPv6网络访问',
    description: 'IPv6 是大势所趋，就在前段时间湖南联通发布公告，对家庭宽带提供 IPv6 地址，不再提供 IPv4 地址，那本文就介绍个人宽带如何开启 IPv6 网络访问。',
    seoTitle: '个人宽带如何开启 IPv6 网络访问：光猫与路由器设置教程',
    seoDescription: '手把手教你为家庭宽带开启 IPv6 网络访问：从光猫与路由器的设置，到向运营商申请（湖南联通等已对家庭宽带提供 IPv6 地址、不再提供 IPv4），附开启后的 IPv6 地址验证方法与常见问题排查。'
  },
  '/doc/user/cmd_bash_disable_ipv6': {
    title: '命令行禁用/启用IPv6本地网络',
    description: '图形化禁用 IPv6 网络比较繁琐，以下是 macOS 和 Window 10 命令行下禁用/启用 IPv6 本地网络的方法。',
    seoTitle: 'macOS / Windows 命令行禁用或启用 IPv6 本地网络',
    seoDescription: '图形化开关 IPv6 网络比较繁琐，本文整理 macOS（networksetup）与 Windows 10/11（netsh、PowerShell）命令行下禁用/启用 IPv6 本地网络的完整命令，可直接复制执行，便于脚本化批量管理多台机器。'
  },
  '/doc/user/cmd_getip': {
    title: '命令行(curl)获取 IPv4 和 IPv6 地址',
    description: '通过 curl 命令获取公网 IPv4 和 IPv6 地址，还可以返回是 IPv4 还是 IPv6 访问优先。',
    seoTitle: 'curl 命令获取公网 IPv4 / IPv6 地址及访问优先级',
    seoDescription: '通过 curl 命令查询本机公网 IPv4 和 IPv6 地址：分别指定 -4 / -6 强制协议栈，还可一键返回当前网络是 IPv4 还是 IPv6 访问优先。适用于 Linux、macOS、Windows 终端以及监控脚本与 CI 环境调用。'
  },
  '/doc/user/view_ipv6_adress_url': {
    title: '浏览器访问 IPv6 地址',
    description: '在 IPv6 地址两边加上中括号，就可以通过浏览器访问了，比如 http://[2402:4e00:1013:e500:0:9671:f018:4947]/',
    seoTitle: '浏览器访问 IPv6 地址：中括号写法与端口注意事项',
    seoDescription: '在 IPv6 地址两边加上中括号即可用浏览器访问，例如 http://[2402:4e00:1013:e500:0:9671:f018:4947]/。本文介绍浏览器访问 IPv6 地址的正确写法、端口号的书写位置，以及无法访问时的排查思路。'
  },
  '/doc/user/ipv4_ipv6_prefix_precedence': {
    title: 'Windows 10/11 设置 IPv4/IPv6 访问优先级',
    description: 'Windows 10/11 开启 IPv6 后默认 IPv6 访问优先（以访问 IPv4/IPv6 双栈站点为例，操作系统会优先访问 IPv6），如果期望 IPv4 访问优先，可以通过 netsh 命令调整。',
    seoTitle: 'Windows 10/11 设置 IPv4/IPv6 访问优先级（netsh 前缀策略）',
    seoDescription: 'Windows 10/11 开启 IPv6 后默认 IPv6 访问优先（访问 IPv4/IPv6 双栈站点时操作系统会优先走 IPv6）。如果期望 IPv4 访问优先，可通过 netsh interface ipv6 修改前缀策略调整，本文提供可直接复制的命令与恢复默认的方法。'
  },
  '/doc/user/ipv6_daohang': {
    title: '国内 IPv6 资源导航',
    description: 'IPv6 网络开启成功后，可以访问一些 IPv6 网站资源，来体验下 IPv6 的魅力。不过目前纯 IPv6 网站比较少，目前网站支持 IPv6 的主流方式为 IPv4/IPv6 双栈访问，毕竟要兼顾 IPv4 网络用户的访问需求。',
    seoTitle: '国内 IPv6 资源导航：支持 IPv6 访问的网站大全',
    seoDescription: 'IPv6 网络开启成功后，可以访问这些国内 IPv6 网站资源体验下一代互联网。目前纯 IPv6 网站还比较少，主流网站支持 IPv6 的方式为 IPv4/IPv6 双栈访问，以兼顾 IPv4 网络用户的访问需求。'
  },
  '/doc/user/pure_ipv6_website': {
    title: '国内纯 IPv6 网站导航',
    description: 'IPv6 网络开启成功后，可以访问一些纯 IPv6 网站资源，来体验下 IPv6 的魅力。',
    seoTitle: '国内纯 IPv6 网站导航：仅支持 IPv6 访问的站点合集',
    seoDescription: '收录一批仅提供 IPv6 访问的国内纯 IPv6 网站：家庭宽带开启 IPv6 网络后即可直接访问，用来验证本机 IPv6 连通性，或体验真正的 IPv6-only 互联网服务，列表持续更新。'
  },
  '/doc/user/dns': {
    title: '全国各省 DNS 服务器列表',
    description: '全国各省电信、联通、移动的首选 IPv4 DNS 服务器和备用 DNS 服务器，以及公共 DNS 服务器列表，最近更新为 2021-07-25。',
    seoTitle: '全国各省电信/联通/移动 DNS 服务器地址列表（IPv4）',
    seoDescription: '全国各省电信、联通、移动的首选与备用 IPv4 DNS 服务器地址汇总，外加 114DNS、阿里 DNS、DNSPod 等公共 DNS 列表，按省份与运营商分类整理，方便网络排障时就近选用延迟最低的 DNS 服务器。'
  },
  '/doc/user/ipv6_dns': {
    title: 'IPv6 DNS 地址列表',
    description: 'IPv6 DNS 地址列表，请求 IPv6 网络需要设置 IPv6 DNS。',
    seoTitle: 'IPv6 DNS 服务器地址列表：公共 IPv6 DNS 推荐',
    seoDescription: 'IPv6 网络需要设置 IPv6 DNS 才能正常解析域名。本文汇总国内外常用 IPv6 DNS 服务器地址：包括阿里、DNSPod、Google、Cloudflare 等公共 IPv6 DNS，可直接复制到光猫、路由器或网卡设置中。'
  },
  '/doc/user/AliyunAuthorizeSecurityGroup': {
    title: '阿里云自动化添加安全组',
    description: 'Golang 代码实操阿里云自动化添加安全组',
    seoTitle: 'Golang 实现阿里云 ECS 安全组规则自动化添加',
    seoDescription: '使用 Golang 调用阿里云 ECS OpenAPI 实现安全组放行规则的自动化添加：包含 SDK 初始化、AccessKey 鉴权、AuthorizeSecurityGroup 接口的参数说明与完整示例代码，适合批量管理多台 ECS 实例的场景。'
  },
  '/doc/user/TencentCloudAddSecurityGroup': {
    title: '腾讯云自动化添加安全组',
    description: 'Golang 代码实操腾讯云自动化添加安全组',
    seoTitle: 'Golang 实现腾讯云 CVM 安全组规则自动化添加',
    seoDescription: '使用 Golang 调用腾讯云 CVM OpenAPI 实现安全组规则的自动化添加：涵盖 SDK 安装、密钥配置、安全组策略接口的调用示例与完整源码，便于在脚本中批量放行端口、管理大量云主机实例。'
  },
  '/doc/server/website_enable_ipv6': {
    title: '网站开启 IPv6 的三种方式',
    description: '从传统二进制部署的 Nginx，到云原生部署的 K8S、Istio，分别介绍网站开启 IPv6 的三种方式。',
    seoTitle: '网站开启 IPv6 的三种方式：Nginx、K8s 与 Istio',
    seoDescription: '从传统二进制部署的 Nginx 监听 IPv6 地址，到云原生部署的 K8s Service 双栈与 Istio 网关，一文讲清网站开启 IPv6 的三种方式及各自适用场景，附关键配置片段与开启后的验证方法。'
  },
  '/doc/server/tencent_cloud_cvm_ipv6': {
    title: '腾讯云 cvm 开启 IPv6',
    description: '一文看懂在腾讯云 CVM 上开启 IPv6，只需 7 步。',
    seoTitle: '腾讯云 CVM 开启 IPv6 完整教程（只需 7 步）',
    seoDescription: '一文看懂在腾讯云 CVM 上开启 IPv6：从 VPC 与子网启用 IPv6、给实例分配 IPv6 地址、安全组放行，到系统内网卡配置与连通性验证，共 7 步完成，附每步操作说明与常见报错的处理办法。'
  },
  '/doc/server/nginx_ipv6': {
    title: 'Nginx 开启 IPv6',
    description: '一文看懂 Nginx 中开启 IPv6，包含设置 IPv6 SSL 证书。',
    seoTitle: 'Nginx 开启 IPv6：listen 监听与 IPv6 SSL 证书配置',
    seoDescription: '一文看懂 Nginx 中开启 IPv6：listen [::] 监听写法、与 IPv4 共存的配置示例、IPv6 SSL 证书的设置，以及防火墙放行 IPv6 流量，配置完成后可用 IPv6 网站检测工具一键验证。'
  },
  '/doc/server/ipv6webcheck': {
    title: '如何确认一个网站是否开启 IPv6',
    description: '通过 IPv6 网站检测工具，一键检查网站是否开启 IPv6 访问。',
    seoTitle: '如何确认一个网站是否开启 IPv6：一键检测方法',
    seoDescription: '使用本站的 IPv6 网站检测工具，输入域名即可一键检查网站是否开启 IPv6 访问：返回 IPv4 与 IPv6 双栈的 HTTP/HTTPS 状态码、DNS 解析与连接耗时对比数据，并可申请 IPv6 徽标认证。'
  },
  '/doc/server/ipv6_sign': {
    title: '网站增加支持IPv6访问标识',
    description: 'HTML+CSS 代码，可以提醒访客本网站支持 IPv6 访问，同时可以跳转到 IPv6 网站检测页面。',
    seoTitle: '网站添加「支持 IPv6 访问」标识（HTML+CSS 代码）',
    seoDescription: '提供一段现成的 HTML+CSS 代码，在网站页脚展示「本网站支持 IPv6 访问」标识，点击即可跳转到 IPv6 网站检测页面实时验证，让访客直观感知网站的 IPv6 支持状态。'
  },
  '/doc/server/ipv6_domain_record': {
    title: '如何为域名添加 IPv6 解析记录',
    description: 'IPv6 网站准备好了后，接下来可以为网站域名添加 IPv6 解析记录（AAAA）。',
    seoTitle: '如何为域名添加 IPv6 解析记录（AAAA 记录）',
    seoDescription: '网站支持 IPv6 后，还需为域名添加 IPv6 解析记录（AAAA）。本文演示在常见 DNS 服务商控制台添加 AAAA 记录的操作步骤、TTL 建议值，以及添加后用 dig / nslookup 验证解析是否生效。'
  },
  '/doc/server/http2': {
    title: '网站如何开启 HTTP/2',
    description: 'HTTP/2 通过引入字段压缩和允许在同一连接上进行多个并发交换，使网络资源得到更有效的利用，并减少了延迟。',
    seoTitle: '网站如何开启 HTTP/2：Nginx 配置与抓包验证',
    seoDescription: 'HTTP/2 通过引入字段压缩和允许在同一连接上进行多个并发交换，使网络资源得到更有效的利用并减少延迟。本文介绍网站在 Nginx 中开启 HTTP/2 的配置方法，并用浏览器与 Wireshark 抓包验证协商结果。'
  },
  '/doc/rfc/rfc8200': {
    title: 'IPv6 RFC8200 解读',
    description: 'RFC8200（Internet Protocol, Version 6 (IPv6) Specification）是 IPv6 的权威定义，接下来在解读 RFC8200 文档中的过程中，加深对 IPv6 的理解。',
    seoTitle: 'IPv6 RFC 8200 解读：IPv6 协议规范逐节精读',
    seoDescription: 'RFC 8200（Internet Protocol, Version 6 (IPv6) Specification）是 IPv6 协议的权威定义。本文逐节解读其报文格式、扩展报头、地址语义等核心内容，在精读文档的过程中建立对 IPv6 的完整理解。'
  },
  '/doc/rfc/ipv6_address_format': {
    title: 'IPv6 地址标识方法',
    description: '首选格式、压缩格式、内嵌 IPv4 地址的 IPv6 地址格式。',
    seoTitle: 'IPv6 地址表示方法：首选格式、压缩格式与内嵌 IPv4',
    seoDescription: 'IPv6 地址共 128 位，本文讲解三种标准表示方法：首选格式（8 组 4 位十六进制数）、零压缩格式（:: 缩写规则）以及内嵌 IPv4 地址的混合表示法，并给出每组示例与常见的书写误区。'
  },
  '/doc/user/tcpdump_ipv6': {
    title: 'tcpdump 分析 IPv6 包',
    description: 'tcpdump 分析 IPv6 包',
    seoTitle: 'tcpdump 抓包分析 IPv6 数据包与常用过滤命令',
    seoDescription: '使用 tcpdump 抓取并分析 IPv6 数据包：整理 ip6、icmp6、TCP/UDP 端口等常用过滤表达式，配合 -nn -vv 等参数看清 IPv6 报文结构，适用于排查 IPv6 连通性与邻居发现（NDP）协议问题。'
  },
  '/doc/user/wireshark_ipv6': {
    title: 'WireShark 分析IPv6包头',
    description: '使用 WireShark 分析 IPv6 包头',
    seoTitle: 'WireShark 分析 IPv6 包头：各字段逐项讲解',
    seoDescription: '使用 WireShark 逐字段分析 IPv6 包头：版本号、流量类别（Traffic Class）、流标签（Flow Label）、有效载荷长度、下一报头、跳数限制与源/目的地址，配合显示过滤器快速定位 IPv6 通信问题。'
  },
  '/doc/user/ipv6_ping': {
    title: 'IPv6 Ping 检测原理',
    description: '协议标准：Internet Control Message Protocol (ICMPv6) for the Internet Protocol Version 6 (IPv6) Specification',
    seoTitle: 'IPv6 Ping 检测原理：ICMPv6 回显请求与应答',
    seoDescription: 'IPv6 Ping 基于 ICMPv6 协议（Internet Control Message Protocol for IPv6）的回显请求（Echo Request）与应答（Echo Reply）实现。本文讲解 ICMPv6 报文结构与类型字段，以及 IPv6 为何用多播取代广播探测。'
  },
  '/doc/api/ip/locate': {
    title: '获取客户端公网 IP 及位置',
    description: '调用本项目的 /v1/location 接口，一次取回访问者自己的公网 IP 与 10 套 IP 库的归属地结果。',
    seoTitle: 'API 文档：获取客户端公网 IP 与多库归属地结果',
    seoDescription: '调用本项目的 /v1/location 接口，一次取回访问者自己的公网 IPv4/IPv6 地址与 10 套 IP 库（MaxMind、IP2Region、GeoCN 等）的归属地结果，返回 JSON、支持跨域，附请求示例与响应字段说明。'
  },
  '/doc/api/ip/myip': {
    title: '获取客户端公网 IP 地址',
    description: '用自带边缘函数获取客户端公网 IP，纯文本返回、支持跨域，可按 IPv4 / IPv6 / 双栈分别部署。',
    seoTitle: 'API 文档：边缘函数获取客户端公网 IP（纯文本跨域）',
    seoDescription: '使用本项目自带的边缘函数获取客户端公网 IP：纯文本直接返回、支持 CORS 跨域，可按 IPv4-only / IPv6-only / 双栈三种方式分别部署，适合脚本、服务器监控与 CI 环境快速取公网 IP。'
  },
  '/doc/api/ip/query': {
    title: '查询指定 IP 地址的位置信息',
    description: '调用本项目的 /v1/location/<ip> 接口查询任意 IPv4 / IPv6 地址的归属地，返回多源聚合结果。',
    seoTitle: 'API 文档：查询指定 IPv4/IPv6 地址的归属地信息',
    seoDescription: '调用本项目的 /v1/location/<ip> 接口查询任意 IPv4 / IPv6 地址的归属地：返回多套 IP 数据库聚合的地理位置、运营商、ASN 等结果，支持节点直连与中间件转发两种调用方式，附完整响应示例。'
  },
  '/doc/api/whois/query': {
    title: '查询域名的 WHOIS 信息',
    description: '调用本项目的 /v1/whois/<domain> 接口查询域名注册信息：IANA → 注册局 → 注册商转介三跳，返回状态、注册商、关键日期与 NS。',
    seoTitle: 'API 文档：查询域名 WHOIS 注册信息（三跳转介）',
    seoDescription: '调用本项目的 /v1/whois/<domain> 接口查询域名注册信息：按 IANA → 注册局 → 注册商三跳转介获取完整 WHOIS 数据，返回查询状态、注册商、注册/到期等关键日期与 NS 服务器字段。'
  },
  '/doc/server/check_userip_ipv4_ipv6': {
    title: 'JS 检查网络是 IPv4 还是 IPv6',
    description: '通过 JS 代码检查客户端是 IPv4 还是 IPv6 访问优先。',
    seoTitle: 'JS 检查客户端网络是 IPv4 还是 IPv6 访问优先',
    seoDescription: '通过一段前端 JS 代码检查客户端当前是 IPv4 还是 IPv6 访问、以及双栈网络下的访问优先级：分别向 IPv4/IPv6 探测端点发起请求并按可达性判定，可嵌入任意网页实时展示访客的网络栈状态。'
  },
  '/doc/server/tls': {
    title: '网站如何开启 TLS',
    description: '用 Wireshark 抓包分析 TLS 1.3 的 TCP 三次握手与 TLS 协商全过程。',
    seoTitle: '网站如何开启 TLS：TLS 1.3 握手过程抓包分析',
    seoDescription: '用 Wireshark 抓包分析 TLS 1.3 的 TCP 三次握手与 TLS 协商全过程：ClientHello、密钥共享（Key Share）、证书验证与 Finished 报文逐个拆解，理解网站开启 TLS 加密后的完整建连链路。'
  },
  '/doc/user/code_getip': {
    title: 'Python/Go 获取 IPv4 和 IPv6 地址',
    description: '通过 Python/Golang 获取公网 IPv4 和 IPv6 地址，还可以返回是 IPv4 还是 IPv6 访问优先。',
    seoTitle: 'Python / Golang 代码获取公网 IPv4 与 IPv6 地址',
    seoDescription: '用 Python 和 Golang 获取本机公网 IPv4 和 IPv6 地址：分别请求 IPv4、IPv6 探测端点并判定当前网络的访问优先栈，附可直接运行的完整示例代码，适合嵌入监控脚本或后端服务使用。'
  },
  '/doc/user/dns_dig': {
    title: 'DNS 解析流程',
    description: '剖析 DNS 解析流程',
    seoTitle: 'DNS 解析流程详解：递归查询与迭代查询全过程',
    seoDescription: '剖析一次完整的 DNS 解析流程：浏览器缓存 → 操作系统缓存 → 本地 DNS 服务器的递归查询 → 根域名 / 顶级域 / 权威服务器的迭代过程，配合 dig 命令逐步演示，理解域名解析为何时快时慢。'
  }
}

export function getDocMeta(path: string): DocMeta {
  return docConfig[path] || { title: '', description: '' }
}
