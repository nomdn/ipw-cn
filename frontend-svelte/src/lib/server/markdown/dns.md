# DNS 解析的流程

DNS 一般分为 **本地DNS**（一般由电信运营商提供）和 **权威DNS**（由根 DNS、顶级 DNS、二级 DNS 组成）。

## 本地 DNS

本地 DNS 采用递归查询域名解析记录，比如指定阿里 DNS（`223.5.5.5`）来解析 `ipw.wsmdn.top`，在 `ANSWER SECTION` 中可以看到直接返回了 `ipw.wsmdn.top` 的解析记录。

本地 DNS 本身不会记录域名解析记录，而是把请求转发到权威 DNS，逐级递归查询，再把解析记录返回给客户端。当然它会缓存解析记录（解析记录中 `TTL` 属性控制缓存的时长）。

本地 DNS 解析示例：

```bash
~$ dig @223.5.5.5 ipw.wsmdn.top
;; QUESTION SECTION:
;ipw.wsmdn.top.			IN	A

;; ANSWER SECTION:
ipw.wsmdn.top.		1	IN	CNAME	ipw.wsmdn.top.a1.initac.com.
ipw.wsmdn.top.a1.initac.com. 76	IN	A	183.205.0.66
ipw.wsmdn.top.a1.initac.com. 76	IN	A	183.205.0.67
ipw.wsmdn.top.a1.initac.com. 76	IN	A	183.205.0.65
ipw.wsmdn.top.a1.initac.com. 76	IN	A	183.205.0.64
```

## 权威 DNS

以查询 `ipw.wsmdn.top` 为例，了解下这 3 级 DNS 的作用。

- **根 DNS**：比如 `a.root-servers.net.`，负责返回待查询域名（`ipw.wsmdn.top`）所属顶级 DNS（如 `e.zdnscloud.cn.`）的解析记录 `203.119.82.1`；
- **顶级 DNS**：比如 `e.zdnscloud.cn.`，负责返回待查询域名的 DNS 地址（如 `brodie.ns.cloudflare.com.`）的解析记录；
- **一级 DNS**：比如 `ns1.huaweicloud-dns.com.`，负责返回具体域名的解析记录，比如 `ipw.wsmdn.top` 的 CNAME 记录为 `ipw.wsmdn.top.a1.initac.com.`。

以下为查询示例：

### 查看根 DNS 的 A 记录

```bash
$ dig a.root-servers.net
;; QUESTION SECTION:
;a.root-servers.net.		IN	A

;; ANSWER SECTION:
a.root-servers.net.	2295	IN	A	198.41.0.4

;; AUTHORITY SECTION:
root-servers.net.	125280	IN	NS	h.root-servers.net.
root-servers.net.	125280	IN	NS	c.root-servers.net.
root-servers.net.	125280	IN	NS	a.root-servers.net.
root-servers.net.	125280	IN	NS	e.root-servers.net.
root-servers.net.	125280	IN	NS	k.root-servers.net.
root-servers.net.	125280	IN	NS	b.root-servers.net.
root-servers.net.	125280	IN	NS	i.root-servers.net.
root-servers.net.	125280	IN	NS	d.root-servers.net.
root-servers.net.	125280	IN	NS	l.root-servers.net.
root-servers.net.	125280	IN	NS	g.root-servers.net.
root-servers.net.	125280	IN	NS	j.root-servers.net.
root-servers.net.	125280	IN	NS	f.root-servers.net.
root-servers.net.	125280	IN	NS	m.root-servers.net.

;; ADDITIONAL SECTION:
h.root-servers.net.	517037	IN	A	198.97.190.53
c.root-servers.net.	563686	IN	A	192.33.4.12
e.root-servers.net.	601355	IN	A	192.203.230.10
g.root-servers.net.	472031	IN	A	192.112.36.4
f.root-servers.net.	553894	IN	A	192.5.5.241
k.root-servers.net.	597325	IN	A	193.0.14.129
i.root-servers.net.	517037	IN	A	192.36.148.17
l.root-servers.net.	565009	IN	A	199.7.83.42
b.root-servers.net.	559683	IN	A	199.9.14.201
d.root-servers.net.	601468	IN	A	199.7.91.13
m.root-servers.net.	603681	IN	A	202.12.27.33
j.root-servers.net.	584819	IN	A	192.58.128.30
h.root-servers.net.	569462	IN	AAAA	2001:500:1::53
c.root-servers.net.	34585	IN	AAAA	2001:500:2::c
```

通过根 DNS 查询 `wsmdn.top`，返回了顶级 DNS `e.zdnscloud.cn.` 的地址为 `203.119.82.1`：

```bash
~$ dig @198.41.0.4 wsmdn.top

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 4096
;; QUESTION SECTION:
;wsmdn.top.			IN	A

;; AUTHORITY SECTION:
top.			172800	IN	NS	e.zdnscloud.cn.
top.			172800	IN	NS	c.zdnscloud.com.
top.			172800	IN	NS	f.zdnscloud.cn.
top.			172800	IN	NS	a.zdnscloud.cn.
top.			172800	IN	NS	d.zdnscloud.com.
top.			172800	IN	NS	j.zdnscloud.com.
top.			172800	IN	NS	i.zdnscloud.cn.
top.			172800	IN	NS	b.zdnscloud.cn.

;; ADDITIONAL SECTION:
e.zdnscloud.cn.		172800	IN	A	203.119.82.1
c.zdnscloud.com.	172800	IN	A	203.99.26.1
f.zdnscloud.cn.		172800	IN	A	116.169.54.111
a.zdnscloud.cn.		172800	IN	A	203.99.24.1
d.zdnscloud.com.	172800	IN	A	203.99.27.1
j.zdnscloud.com.	172800	IN	AAAA	2401:8d00:2::1
i.zdnscloud.cn.		172800	IN	AAAA	2401:8d00:1::1
b.zdnscloud.cn.		172800	IN	A	203.99.25.1
```

通过 `e.zdnscloud.cn` 查询 `wsmdn.top`，返回一级 DNS 的 NS 记录：

```bash
~$ dig @e.zdnscloud.cn wsmdn.top

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 4096
;; QUESTION SECTION:
;wsmdn.top.			IN	A

;; AUTHORITY SECTION:
wsmdn.top.		3600	IN	NS	brodie.ns.cloudflare.com.
wsmdn.top.		3600	IN	NS	gloria.ns.cloudflare.com.
```

通过 `ns1.huaweicloud-dns.com` 返回 `ipw.wsmdn.top` 的 CNAME 记录：

```bash
~$ dig @ns1.huaweicloud-dns.com ipw.wsmdn.top

;; QUESTION SECTION:
;ipw.wsmdn.top.			IN	A

;; ANSWER SECTION:
ipw.wsmdn.top.		300	IN	CNAME	ipw.wsmdn.top.a1.initac.com.
```
