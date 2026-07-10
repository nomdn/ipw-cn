# DNS 解析的流程

DNS 一般分为 **本地DNS**（一般由电信运营商提供）和 **权威DNS**（由根 DNS、顶级 DNS、二级 DNS 组成）。

## 本地 DNS

本地 DNS 采用递归查询域名解析记录，比如指定一个电信 DNS（`202.101.224.69`）来解析 `ipw.cn`，在 `ANSWER SECTION` 中可以看到直接返回了 `ipw.cn` 的 A 记录。

本地 DNS 本身不会记录域名解析记录，而是把请求转发到权威 DNS，逐级递归查询，再把解析记录返回给客户端。当然它会缓存解析记录（解析记录中 `TTL` 属性控制缓存的时长）。

本地 DNS 解析示例：

```bash
~$ dig @202.101.224.69 ipw.cn
;; QUESTION SECTION:
;ipw.cn.				IN	A

;; ANSWER SECTION:
ipw.cn.			600	IN	A	106.55.75.123
```

## 权威 DNS

以查询 `ipw.cn` 为例，了解下这 3 级 DNS 的作用。

- **根 DNS**：比如 `a.root-servers.net.`，负责返回待查询域名（`ipw.cn`）所属顶级 DNS（如 `a.dns.cn.`）的解析记录 `203.119.25.1`；
- **顶级 DNS**：比如 `a.dns.cn.`，负责返回待查询域名的 DNS 地址（如 `ns3.dnsv2.com.`，登记在顶级域名服务器中，通过 `whois` 命令可以查询，在域名管理平台中可以修改）的解析记录；
- **一级 DNS**：比如 `ns3.dnsv2.com.`，负责返回具体域名的解析记录，比如 `ipw.cn` 的 A 记录为 `106.55.75.123`。

以下为查询示例：

### 查看根 DNS 的 A 记录

```bash
$ dig a.root-servers.net
;; QUESTION SECTION:
;a.root-servers.net.		IN	A

;; ANSWER SECTION:
a.root-servers.net.	81077	IN	A	198.41.0.4

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

通过根 DNS 查询 `ipw.cn`，返回了顶级 DNS `a.dns.cn.` 的地址为 `203.119.25.1`：

```bash
~$ dig @198.41.0.4 ipw.cn
;; QUESTION SECTION:
;ipw.cn.				IN	A

;; AUTHORITY SECTION:
cn.			172800	IN	NS	a.dns.cn.
cn.			172800	IN	NS	b.dns.cn.
cn.			172800	IN	NS	c.dns.cn.
cn.			172800	IN	NS	d.dns.cn.
cn.			172800	IN	NS	e.dns.cn.
cn.			172800	IN	NS	f.dns.cn.
cn.			172800	IN	NS	g.dns.cn.
cn.			172800	IN	NS	ns.cernet.net.

;; ADDITIONAL SECTION:
a.dns.cn.		172800	IN	A	203.119.25.1
b.dns.cn.		172800	IN	A	203.119.26.1
c.dns.cn.		172800	IN	A	203.119.27.1
d.dns.cn.		172800	IN	A	203.119.28.1
e.dns.cn.		172800	IN	A	203.119.29.1
f.dns.cn.		172800	IN	A	195.219.8.90
g.dns.cn.		172800	IN	A	66.198.183.65
ns.cernet.net.		172800	IN	A	202.112.0.44
a.dns.cn.		172800	IN	AAAA	2001:dc7::1
d.dns.cn.		172800	IN	AAAA	2001:dc7:1000::1
```

通过 `a.dns.cn` 查询 `ipw.cn`，返回一级 DNS 的 A 记录：

```bash
~$ dig @a.dns.cn ipw.cn

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 4096
;; QUESTION SECTION:
;ipw.cn.				IN	A

;; AUTHORITY SECTION:
ipw.cn.			86400	IN	NS	ns4.dnsv2.com.
ipw.cn.			86400	IN	NS	ns3.dnsv2.com.
```

通过 `ns4.dnsv2.com` 返回 `ipw.cn` 的 A 记录 `106.55.75.123`：

```bash
~$ dig @ns4.dnsv2.com ipw.cn
;; QUESTION SECTION:
;ipw.cn.				IN	A

;; ANSWER SECTION:
ipw.cn.			600	IN	A	106.55.75.123

;; AUTHORITY SECTION:
ipw.cn.			86400	IN	NS	ns4.dnsv2.com.
ipw.cn.			86400	IN	NS	ns3.dnsv2.com.
```