# <span style="background-color: #b95442;color: white;font-size: 0.43em;border-radius: 5px;padding: 2px 5px;">转载</span> 如何为域名添加 IPv6 解析记录
>![NOTE]
>本文的测试结果只作为示例，不代表真实数据！

-   [检查 IPv6 域名解析是否成功](#检查-ipv6-域名解析是否成功)
    -   [dig -t AAAA ipw.wsmdn.top](#dig-t-aaaa-ipw-cn)
    -   [在线多地域 DNS 解析](#在线多地域-dns-解析)
-   [IPv6工具箱 小程序 和 iOS App](#ipv6工具箱-小程序-和-ios-app)
    -   [小程序](#小程序)
    -   [苹果 iOS App](#苹果-ios-app)

IPv6 网站准备好了后，接下来可以为网站域名添加 IPv6 解析记录（AAAA)。

-   A 记录：解析为 IPv4 地址
-   AAAA 记录：解析为 IPv6 地址

![IPv6 解析记录（AAAA)](/doc/ipv6_domain_resolve.png)

## 检查 IPv6 域名解析是否成功

有 2 种方式解析 IPv6 地址：`dig` 和 `在线解析`。

### dig -t AAAA ipw.wsmdn.top


```sh
$ dig -t AAAA ipw.wsmdn.top

; <<>> DiG 9.10.6 <<>> -t AAAA ipw.wsmdn.top
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 16159
;; flags: qr rd ra; QUERY: 1, ANSWER: 1, AUTHORITY: 2, ADDITIONAL: 15

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 4096
;; QUESTION SECTION:
;ipw.wsmdn.top.				IN	AAAA

;; ANSWER SECTION:
ipw.wsmdn.top.			600	IN	AAAA	2402:4e00:40:40::2:331

;; AUTHORITY SECTION:
ipw.wsmdn.top.			74665	IN	NS	ns4.dnsv2.com.
ipw.wsmdn.top.			74665	IN	NS	ns3.dnsv2.com.

;; ADDITIONAL SECTION:
ns3.dnsv2.com.		89916	IN	AAAA	2402:4e00:1430:1102:0:9136:2b2f:bf6b
ns4.dnsv2.com.		163624	IN	AAAA	2402:4e00:1020:1264:0:9136:29bb:2f5e
ns3.dnsv2.com.		89688	IN	A	223.166.151.17
ns3.dnsv2.com.		89688	IN	A	61.129.8.141
ns3.dnsv2.com.		89688	IN	A	101.226.220.13
ns3.dnsv2.com.		89688	IN	A	129.211.176.248
ns3.dnsv2.com.		89688	IN	A	162.14.24.245
ns3.dnsv2.com.		89688	IN	A	183.192.164.118
ns4.dnsv2.com.		157105	IN	A	223.166.151.18
ns4.dnsv2.com.		157105	IN	A	58.247.212.37
ns4.dnsv2.com.		157105	IN	A	59.36.120.143
ns4.dnsv2.com.		157105	IN	A	61.151.180.45
ns4.dnsv2.com.		157105	IN	A	162.14.25.245
ns4.dnsv2.com.		157105	IN	A	183.192.164.118

;; Query time: 18 msec
;; SERVER: 240e:1f:1::1#53(240e:1f:1::1)
;; WHEN: Sat Dec 18 20:33:58 CST 2021
;; MSG SIZE  rcvd: 356
```


### 在线多地域 DNS 解析

通过 [IPv6 在线域名解析工具](https://itdog.cn/dns/) 对其解析验证。
