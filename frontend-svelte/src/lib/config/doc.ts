export interface DocMeta {
  title: string
  description: string
}

export const docConfig: Record<string, DocMeta> = {
  '/doc': {
    title: 'IPv6 工具箱使用文档',
    description: '致力于普及 IPv6，推进 IPv6 规模部署和应用，以全面推进 IPv6 技术创新与融合应用为主线，以提升应用广度深度为主攻方向。支持 IPv6 本地网络（宽带）开启验证、IPv6 网站开启检测、IPv6 Ping 延迟测试、IPv6 网站测速、IPv6 地址查询、IPv6 DNS 解析。'
  },
  '/doc/user/enable_ipv6': {
    title: '个人宽带如何开启IPv6网络访问',
    description: 'IPv6 是大势所趋，就在前段时间湖南联通发布公告，对家庭宽带提供 IPv6 地址，不再提供 IPv4 地址，那本文就介绍个人宽带如何开启 IPv6 网络访问。'
  },
  '/doc/user/cmd_bash_disable_ipv6': {
    title: '命令行禁用/启用IPv6本地网络',
    description: '图形化禁用 IPv6 网络比较繁琐，以下是 macOS 和 Window 10 命令行下禁用/启用 IPv6 本地网络的方法。'
  },
  '/doc/user/cmd_getip': {
    title: '命令行(curl)获取 IPv4 和 IPv6 地址',
    description: '通过 curl 命令获取公网 IPv4 和 IPv6 地址，还可以返回是 IPv4 还是 IPv6 访问优先。'
  },
  '/doc/user/view_ipv6_adress_url': {
    title: '浏览器访问 IPv6 地址',
    description: '在 IPv6 地址两边加上中括号，就可以通过浏览器访问了，比如 http://[2402:4e00:1013:e500:0:9671:f018:4947]/'
  },
  '/doc/user/ipv4_ipv6_prefix_precedence': {
    title: 'Windows 10/11 设置 IPv4/IPv6 访问优先级',
    description: 'Windows 10/11 开启 IPv6 后默认 IPv6 访问优先（以访问 IPv4/IPv6 双栈站点为例，操作系统会优先访问 IPv6），如果期望 IPv4 访问优先，可以通过 netsh 命令调整。'
  },
  '/doc/user/ipv6_daohang': {
    title: '国内 IPv6 资源导航',
    description: 'IPv6 网络开启成功后，可以访问一些 IPv6 网站资源，来体验下 IPv6 的魅力。不过目前纯 IPv6 网站比较少，目前网站支持 IPv6 的主流方式为 IPv4/IPv6 双栈访问，毕竟要兼顾 IPv4 网络用户的访问需求。'
  },
  '/doc/user/pure_ipv6_website': {
    title: '国内纯 IPv6 网站导航',
    description: 'IPv6 网络开启成功后，可以访问一些纯 IPv6 网站资源，来体验下 IPv6 的魅力。'
  },
  '/doc/user/dns': {
    title: '全国各省 DNS 服务器列表',
    description: '全国各省电信、联通、移动的首选 IPv4 DNS 服务器和备用 DNS 服务器，以及公共 DNS 服务器列表，最近更新为 2021-07-25。'
  },
  '/doc/user/ipv6_dns': {
    title: 'IPv6 DNS 地址列表',
    description: 'IPv6 DNS 地址列表，请求 IPv6 网络需要设置 IPv6 DNS。'
  },
  '/doc/user/AliyunAuthorizeSecurityGroup': {
    title: '阿里云自动化添加安全组',
    description: 'Golang 代码实操阿里云自动化添加安全组'
  },
  '/doc/user/TencentCloudAddSecurityGroup': {
    title: '腾讯云自动化添加安全组',
    description: 'Golang 代码实操腾讯云自动化添加安全组'
  },
  '/doc/server/website_enable_ipv6': {
    title: '网站开启 IPv6 的三种方式',
    description: '从传统二进制部署的 Nginx，到云原生部署的 K8S、Istio，分别介绍网站开启 IPv6 的三种方式。'
  },
  '/doc/server/tencent_cloud_cvm_ipv6': {
    title: '腾讯云 cvm 开启 IPv6',
    description: '一文看懂在腾讯云 CVM 上开启 IPv6，只需 7 步。'
  },
  '/doc/server/nginx_ipv6': {
    title: 'Nginx 开启 IPv6',
    description: '一文看懂 Nginx 中开启 IPv6，包含设置 IPv6 SSL 证书。'
  },
  '/doc/server/ipv6webcheck': {
    title: '如何确认一个网站是否开启 IPv6',
    description: '通过 IPv6 网站检测工具，一键检查网站是否开启 IPv6 访问。'
  },
  '/doc/server/ipv6_sign': {
    title: '网站增加支持IPv6访问标识',
    description: 'HTML+CSS 代码，可以提醒访客本网站支持 IPv6 访问，同时可以跳转到 IPv6 网站检测页面。'
  },
  '/doc/server/ipv6_domain_record': {
    title: '如何为域名添加 IPv6 解析记录',
    description: 'IPv6 网站准备好了后，接下来可以为网站域名添加 IPv6 解析记录（AAAA）。'
  },
  '/doc/server/http2': {
    title: '网站如何开启 HTTP/2',
    description: 'HTTP/2 通过引入字段压缩和允许在同一连接上进行多个并发交换，使网络资源得到更有效的利用，并减少了延迟。'
  },
  '/doc/rfc/rfc8200': {
    title: 'IPv6 RFC8200 解读',
    description: 'RFC8200（Internet Protocol, Version 6 (IPv6) Specification）是 IPv6 的权威定义，接下来在解读 RFC8200 文档中的过程中，加深对 IPv6 的理解。'
  },
  '/doc/rfc/ipv6_address_format': {
    title: 'IPv6 地址标识方法',
    description: '首选格式、压缩格式、内嵌 IPv4 地址的 IPv6 地址格式。'
  },
  '/doc/user/tcpdump_ipv6': {
    title: 'tcpdump 分析 IPv6 包',
    description: 'tcpdump 分析 IPv6 包'
  },
  '/doc/user/wireshark_ipv6': {
    title: 'WireShark 分析IPv6包头',
    description: '使用 WireShark 分析 IPv6 包头'
  },
  '/doc/user/ipv6_ping': {
    title: 'IPv6 Ping 检测原理',
    description: '协议标准：Internet Control Message Protocol (ICMPv6) for the Internet Protocol Version 6 (IPv6) Specification'
  },
  '/doc/api/ip/locate': {
    title: '获取客户端公网 IP 及位置',
    description: '调用本项目的 /v1/location 接口，一次取回访问者自己的公网 IP 与 10 套 IP 库的归属地结果。'
  },
  '/doc/api/ip/myip': {
    title: '获取客户端公网 IP 地址',
    description: '用自带边缘函数获取客户端公网 IP，纯文本返回、支持跨域，可按 IPv4 / IPv6 / 双栈分别部署。'
  },
  '/doc/api/ip/query': {
    title: '查询指定 IP 地址的位置信息',
    description: '调用本项目的 /v1/location/<ip> 接口查询任意 IPv4 / IPv6 地址的归属地，返回多源聚合结果。'
  },
  '/doc/api/whois/query': {
    title: '查询域名的 WHOIS 信息',
    description: '调用本项目的 /v1/whois/<domain> 接口查询域名注册信息：IANA → 注册局 → 注册商转介三跳，返回状态、注册商、关键日期与 NS。'
  },
  '/doc/server/check_userip_ipv4_ipv6': {
    title: 'JS 检查网络是 IPv4 还是 IPv6',
    description: '通过 JS 代码检查客户端是 IPv4 还是 IPv6 访问优先。'
  },
  '/doc/server/tls': {
    title: '网站如何开启 TLS',
    description: '用 Wireshark 抓包分析 TLS 1.3 的 TCP 三次握手与 TLS 协商全过程。'
  },
  '/doc/user/code_getip': {
    title: 'Python/Go 获取 IPv4 和 IPv6 地址',
    description: '通过 Python/Golang 获取公网 IPv4 和 IPv6 地址，还可以返回是 IPv4 还是 IPv6 访问优先。'
  },
  '/doc/user/dns_dig': {
    title: 'DNS 解析流程',
    description: '剖析 DNS 解析流程'
  }
}

export function getDocMeta(path: string): DocMeta {
  return docConfig[path] || { title: '', description: '' }
}
