export interface DocMeta {
  title: string
  description: string
}

export const docConfig: Record<string, DocMeta> = {
  '/doc': {
    title: 'IPv6 工具箱使用文档',
    description: 'IPv6 工具箱使用文档'
  },
  '/doc/user/enable_ipv6': {
    title: '个人宽带如何开启IPv6网络访问',
    description: '介绍个人宽带如何开启IPv6网络访问'
  },
  '/doc/user/cmd_bash_disable_ipv6': {
    title: '命令行禁用/启用IPv6本地网络',
    description: 'macOS和Window10命令行禁用/启用IPv6本地网络'
  },
  '/doc/user/cmd_getip': {
    title: '命令行(curl)获取 IPv4 和 IPv6 地址',
    description: '使用命令行curl获取IPv4和IPv6地址'
  },
  '/doc/user/view_ipv6_adress_url': {
    title: '浏览器访问 IPv6 地址',
    description: '如何在浏览器中访问IPv6地址'
  },
  '/doc/user/ipv4_ipv6_prefix_precedence': {
    title: 'Windows 10/11 设置 IPv4/IPv6 访问优先级',
    description: 'Windows 10/11 设置 IPv4/IPv6 访问优先级'
  },
  '/doc/user/ipv6_daohang': {
    title: '国内 IPv6 资源导航',
    description: '国内IPv6资源导航列表'
  },
  '/doc/user/pure_ipv6_website': {
    title: '国内纯 IPv6 网站导航',
    description: '国内纯IPv6网站导航列表'
  },
  '/doc/user/dns': {
    title: '全国各省 DNS 服务器列表',
    description: '全国各省DNS服务器列表'
  },
  '/doc/user/ipv6_dns': {
    title: 'IPv6 DNS 地址列表',
    description: 'IPv6 DNS地址列表'
  },
  '/doc/user/AliyunAuthorizeSecurityGroup': {
    title: '阿里云自动化添加安全组',
    description: '阿里云自动化添加安全组'
  },
  '/doc/user/TencentCloudAddSecurityGroup': {
    title: '腾讯云自动化添加安全组',
    description: '腾讯云自动化添加安全组'
  },
  '/doc/server/website_enable_ipv6': {
    title: '网站开启 IPv6 的三种方式',
    description: '从传统二进制部署的Nginx，到云原生部署的K8S、Istio，分别介绍网站开启IPv6的三种方式'
  },
  '/doc/server/tencent_cloud_cvm_ipv6': {
    title: '腾讯云 cvm 开启 IPv6',
    description: '一文看懂在腾讯云CVM上开启IPv6，只需7步'
  },
  '/doc/server/nginx_ipv6': {
    title: 'Nginx 开启 IPv6',
    description: '一文看懂Nginx中开启IPv6，包含设置IPv6 SSL证书'
  },
  '/doc/server/ipv6webcheck': {
    title: '如何确认一个网站是否开启 IPv6',
    description: '如何确认一个网站是否开启IPv6'
  },
  '/doc/server/ipv6_sign': {
    title: '网站增加支持IPv6访问标识',
    description: 'HTML+CSS代码，可以提醒访客本网站支持IPv6访问'
  },
  '/doc/server/ipv6_domain_record': {
    title: '如何为域名添加 IPv6 解析记录',
    description: '如何为域名添加IPv6解析记录'
  },
  '/doc/server/http2': {
    title: '原创网站如何开启 HTTP/2',
    description: '介绍网站如何开启HTTP/2协议'
  },
  '/doc/rfc/rfc8200': {
    title: 'IPv6 RFC8200 解读',
    description: 'RFC8200(Internet Protocol, Version 6 (IPv6) Specification) 是IPv6的权威定义'
  },
  '/doc/rfc/ipv6_address_format': {
    title: 'IPv6 地址标识方法',
    description: '首选格式、压缩格式、内嵌IPv4地址的IPv6地址格式'
  },
  '/doc/user/tcpdump_ipv6': {
    title: 'tcpdump 分析IPv6包',
    description: '使用tcpdump分析IPv6包'
  },
  '/doc/user/wireshark_ipv6': {
    title: 'WireShark 分析IPv6包头',
    description: '使用WireShark分析IPv6包头'
  },
  '/doc/user/ipv6_ping': {
    title: 'IPv6 Ping 检测原理',
    description: 'IPv6 Ping检测原理'
  }
}

export function getDocMeta(path: string): DocMeta {
  return docConfig[path] || { title: '', description: '' }
}
