# <span style="background-color: #b95442;color: white;font-size: 0.43em;border-radius: 5px;padding: 2px 5px;">转载</span> 如何确认一个网站是否开启 IPv6

-   [1 域名解析到 IPv6地址](#1-域名解析到-ipv6地址)
-   [2 通过 IPv6 网络是否可以访问成功](#2-通过-ipv6-网络是否可以访问成功)

在 [腾讯云 cvm 开启 IPv6](/server/tencent_cloud_cvm_ipv6) 中我们介绍了服务器如何开启 IPv6，接下来我们介绍下如何检查一个网站是否开启 IPv6。

可以直接访问 [在线 IPv6 网站检测工具](https://ipw.wsmdn.top/ipv6webcheck/) 来确认一个网站是否开 IPv6 访问。 ![ipv6 webcheck](/doc/ipv6_webcheck.png)

如果细分来看，IPv6 网站检测主要是两块，**网站域名是否有 AAAA(IPv6 地址)的解析记录**，以及 **该 IPv6 地址是否可以访问成功**。

## 1 域名解析到 IPv6地址

同一个主机名是可以同时解析到 IPv4（A 记录）、IPv6(AAAA 记录)，用户本地网络 IPv6 访问优先时，会自动解析、访问到 IPv6 地址，否则会解析到 IPv4 地址。

-   AAAA 记录 ([在线 DNS 查询工具](https://itdog.cn/dns/))

-   A 记录


## 2 通过 IPv6 网络是否可以访问成功

如果网站直接监听在 IPv6 地址上，则可以在 IPv6 地址两侧加上 `[]`, 即可[访问 IPv6 网址](/user/view_ipv6_adress_url)。

-   直接访问 IPv6 地址 （[IPv6 网站测速工具](https://itdog.cn/http_ipv6)）

-   访问解析到 IPv6 地址的域名


就是这么简单，赶紧试试吧~

