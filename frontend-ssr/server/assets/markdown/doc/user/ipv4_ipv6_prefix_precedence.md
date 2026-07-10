# <span style="background-color: #b95442;color: white;font-size: 0.43em;border-radius: 5px;padding: 2px 5px;">转载</span> Windows 10/11 设置 IPv4/IPv6 访问优先级

Windows10/11开启 IPv6 后默认 IPv6 访问优先（以访问 IPv4/IPv6 双栈站点为例， 操作系统会优先访问 IPv6），如果期望 IPv4访问优先，可以通过 netsh 命令调整。

> 以下命令在 Windows10/11 验证通过。

-   [1\. 查看网络前缀访问优先级](#1-查看网络前缀访问优先级)
-   [2\. 网络前缀含义](#2-网络前缀含义)
-   [3\. 调整网络前缀优先级，让 IPv4 访问优先](#3-调整网络前缀优先级-让-ipv4-访问优先)
-   [附录](#附录)
    -   [如何重新设置 IPv6访问优先](#如何重新设置-ipv6访问优先)
    -   [netsh interface ipv6 相关命令如何使用](#netsh-interface-ipv6-相关命令如何使用)
-   [reference](#reference)

ipw.wsmdn.top 是IPv4/IPv6 双栈站点，使用 ping 命令默认会访问到 IPv6站点，如果特别指定 IPv4（ping -4）才会解析到 IPv4 站点。


```sh
Microsoft Windows [版本 10.0.19044.1766]
(c) Microsoft Corporation。保留所有权利。

C:\Windows\system32>ping ipw.wsmdn.top

正在 Ping ipw.wsmdn.top [2402:4e00:40:40::2:3b6] 具有 32 字节的数据:
来自 2402:4e00:40:40::2:3b6 的回复: 时间=13ms
来自 2402:4e00:40:40::2:3b6 的回复: 时间=18ms
来自 2402:4e00:40:40::2:3b6 的回复: 时间=10ms
来自 2402:4e00:40:40::2:3b6 的回复: 时间=16ms

2402:4e00:40:40::2:3b6 的 Ping 统计信息:
    数据包: 已发送 = 4，已接收 = 4，丢失 = 0 (0% 丢失)，
往返行程的估计时间(以毫秒为单位):
    最短 = 10ms，最长 = 18ms，平均 = 14ms

C:\Windows\system32>ping -4 ipw.wsmdn.top

正在 Ping ipw.wsmdn.top [106.55.75.123] 具有 32 字节的数据:
来自 106.55.75.123 的回复: 字节=32 时间=8ms TTL=52
来自 106.55.75.123 的回复: 字节=32 时间=7ms TTL=52
来自 106.55.75.123 的回复: 字节=32 时间=9ms TTL=52
来自 106.55.75.123 的回复: 字节=32 时间=7ms TTL=52

106.55.75.123 的 Ping 统计信息:
    数据包: 已发送 = 4，已接收 = 4，丢失 = 0 (0% 丢失)，
往返行程的估计时间(以毫秒为单位):
    最短 = 7ms，最长 = 9ms，平均 = 7ms
```


## 1\. 查看网络前缀访问优先级

我们先看下Windows10/11 中各个访问 IPv4/IPv6的优先级，可以看到 IPv6(\`::/0\`)比 IPv4(\`::ffff:0:0/96\`) 的优先级高，会被优先访问。

> 第一列优先循序越大优先级越高，会优先访问。


```sh
C:\Windows\system32>netsh interface ipv6 show prefixpolicies
查询活动状态...

优先顺序    标签   前缀
----------  -----  --------------------------------
        50      0  ::1/128
        40      1  ::/0
        35      4  ::ffff:0:0/96
        30      2  2002::/16
         5      5  2001::/32
         3     13  fc00::/7
         1     11  fec0::/10
         1     12  3ffe::/16
         1      3  ::/96
```


## 2\. 网络前缀含义

我们依次介绍下这些前缀的含义。

首先 Windows10/11 中默认的访问前缀规则是参照 [RFC6724: Default Address Selection for Internet Protocol Version 6 (IPv6)](https://www.rfc-editor.org/rfc/rfc6724#section-2.1) 实现的。

在 [IANA IPv6 Special-Purpose Address Registry](https://www.iana.org/assignments/iana-ipv6-special-registry/iana-ipv6-special-registry.xhtml) 中我们可以找到这些前缀的分配归属。

-   \`::1/128\`：本地回环地址，类似 IPv4 中的 127.0.0.1
-   \`::/0\`: IPv6 单播地址
-   \`::ffff:0:0/96\`: IPv4 地址范围
-   \`2002::/16\`: 6to4，具体详见 [RFC3056: Connection of IPv6 Domains via IPv4 Clouds](https://www.rfc-editor.org/rfc/rfc3056.html)
-   \`2001::/32\`: TEREDO，具体详见 [RFC4380: Teredo: Tunneling IPv6 over UDP through Network Address Translations (NATs)](https://www.rfc-editor.org/rfc/rfc4380.html)
-   \`fc00::/7\`: Unique-Local，具体详见 [RFC4193: Unique Local IPv6 Unicast Addresses](https://www.rfc-editor.org/rfc/rfc4193.html)
-   ...

## 3\. 调整网络前缀优先级，让 IPv4 访问优先

从前面我们知道 IPv6(\`::/0\`)比 IPv4(\`::ffff:0:0/96\`) 的优先级高，我们通过 \`netsh interface ipv6\` 命令调整优先级。

Win + R 进入运行对话框，输入 cmd，选择 \`以管理员身份运行\`，执行调整命令，可以看到 IPv4(\`::ffff:0:0/96\`) 优先级最高。


```sh
C:\Windows\system32>netsh interface ipv6 set  prefixpolicy ::ffff:0:0/96 100 4
确定。

C:\Windows\system32>netsh interface ipv6 show prefixpolicies
查询活动状态...

优先顺序    标签   前缀
----------  -----  --------------------------------
       100      4  ::ffff:0:0/96
        50      0  ::1/128
        40      1  ::/0
        30      2  2002::/16
         5      5  2001::/32
         3     13  fc00::/7
         1     11  fec0::/10
         1     12  3ffe::/16
         1      3  ::/96
```


执行 ping 、curl 命令验证，确实默认 IPv4 访问优先。

> 如果是浏览器验证可以开启 Edge 或 Chrome 的审查模式，查看网络资源的链接地址。


```sh
C:\Windows\system32>ping ipw.wsmdn.top

正在 Ping ipw.wsmdn.top [106.55.75.123] 具有 32 字节的数据:
来自 106.55.75.123 的回复: 字节=32 时间=14ms TTL=52
来自 106.55.75.123 的回复: 字节=32 时间=29ms TTL=52
来自 106.55.75.123 的回复: 字节=32 时间=14ms TTL=52
来自 106.55.75.123 的回复: 字节=32 时间=14ms TTL=52

106.55.75.123 的 Ping 统计信息:
    数据包: 已发送 = 4，已接收 = 4，丢失 = 0 (0% 丢失)，
往返行程的估计时间(以毫秒为单位):
    最短 = 14ms，最长 = 29ms，平均 = 17ms

C:\Windows\system32>curl -v  https://ipw.wsmdn.top -I
*   Trying 106.55.75.123:443...
* Connected to ipw.wsmdn.top (106.55.75.123) port 443 (#0)
* schannel: disabled automatic use of client certificate
* schannel: ALPN, offering http/1.1
* schannel: ALPN, server accepted to use http/1.1
> HEAD / HTTP/1.1
> Host: ipw.wsmdn.top
> User-Agent: curl/7.79.1
> Accept: */*
```


重启电脑后，发现还是 IPv6 访问优先。


```sh
Microsoft Windows [版本 10.0.19044.1766]
(c) Microsoft Corporation。保留所有权利。

C:\Windows\system32>ping ipw.wsmdn.top

正在 Ping ipw.wsmdn.top [2402:4e00:40:40::2:3b6] 具有 32 字节的数据:
来自 2402:4e00:40:40::2:3b6 的回复: 时间=10ms
来自 2402:4e00:40:40::2:3b6 的回复: 时间=14ms
来自 2402:4e00:40:40::2:3b6 的回复: 时间=9ms

2402:4e00:40:40::2:3b6 的 Ping 统计信息:
    数据包: 已发送 = 3，已接收 = 3，丢失 = 0 (0% 丢失)，
往返行程的估计时间(以毫秒为单位):
    最短 = 9ms，最长 = 14ms，平均 = 11ms
```


查看网络前缀访问优先级，发现只剩下 IPv4 的，可能是因为这个原因导致没生效，加回 IPv6 的网络前缀，IPv4 访问优先，达到目的。


```sh
C:\Windows\system32>netsh interface ipv6 show prefixpolicies
查询活动状态...

优先顺序    标签   前缀
----------  -----  --------------------------------
       100      4  ::ffff:0:0/96

C:\Windows\system32>netsh interface ipv6 add  prefixpolicy ::/0 40 1
确定。


C:\Windows\system32>netsh interface ipv6 show prefixpolicies
查询活动状态...

优先顺序    标签   前缀
----------  -----  --------------------------------
       100      4  ::ffff:0:0/96
        40      1  ::/0

C:\Windows\system32>ping ipw.wsmdn.top

正在 Ping ipw.wsmdn.top [106.55.75.123] 具有 32 字节的数据:
来自 106.55.75.123 的回复: 字节=32 时间=14ms TTL=52
来自 106.55.75.123 的回复: 字节=32 时间=44ms TTL=52
来自 106.55.75.123 的回复: 字节=32 时间=14ms TTL=52
来自 106.55.75.123 的回复: 字节=32 时间=14ms TTL=52

106.55.75.123 的 Ping 统计信息:
    数据包: 已发送 = 4，已接收 = 4，丢失 = 0 (0% 丢失)，
往返行程的估计时间(以毫秒为单位):
    最短 = 14ms，最长 = 44ms，平均 = 21ms
```


## 附录

### 如何重新设置 IPv6访问优先

两种方式，直接重置

#### 重置 IPv6 策略


```sh
C:\Windows\system32>netsh interface ipv6 reset
正在重置 分段转发，完成!
正在重置 分段，完成!
正在重置 控制协议，完成!
正在重置 回显顺序请求，完成!
正在重置 全局，完成!
正在重置 接口，完成!
正在重置 任意广播地址，完成!
正在重置 多播地址，完成!
正在重置 单播地址，完成!
正在重置 邻居，完成!
正在重置 路径，完成!
正在重置 潜在，完成!
正在重置 前缀策略，完成!
正在重置 代理邻居，完成!
正在重置 路由，完成!
正在重置 站点前缀，完成!
正在重置 子接口，完成!
正在重置 唤醒模式，完成!
正在重置 解析邻居，完成!
正在重置 ，完成!
正在重置 ，完成!
正在重置 ，完成!
正在重置 ，完成!
正在重置 ，失败。
拒绝访问。

正在重置 ，完成!
正在重置 ，完成!
正在重置 ，完成!
正在重置 ，完成!
正在重置 ，完成!
正在重置 ，完成!
正在重置 ，完成!
重新启动计算机来完成此操作。

C:\Windows\system32>shutdown -r -t 0
```


电脑重启后，\`ping ipw.wsmdn.top\` 会解析 IPv6 地址，重置成功。

#### 调整网络前缀优先级

调整网络前缀优先级，重启之后也是生效的。


```sh
C:\Windows\system32>netsh interface ipv6 show prefixpolicies
查询活动状态...

优先顺序    标签   前缀
----------  -----  --------------------------------
       100      4  ::ffff:0:0/96
        40      1  ::/0


C:\Windows\system32>netsh interface ipv6 set  prefixpolicy ::ffff:0:0/96 10 4
确定。


C:\Windows\system32>netsh interface ipv6 show prefixpolicies
查询活动状态...

优先顺序    标签   前缀
----------  -----  --------------------------------
        40      1  ::/0
        10      4  ::ffff:0:0/96


C:\Windows\system32>ping ipw.wsmdn.top

正在 Ping ipw.wsmdn.top [2402:4e00:40:40::2:3b6] 具有 32 字节的数据:
来自 2402:4e00:40:40::2:3b6 的回复: 时间=10ms
来自 2402:4e00:40:40::2:3b6 的回复: 时间=11ms
来自 2402:4e00:40:40::2:3b6 的回复: 时间=18ms
来自 2402:4e00:40:40::2:3b6 的回复: 时间=10ms

2402:4e00:40:40::2:3b6 的 Ping 统计信息:
    数据包: 已发送 = 4，已接收 = 4，丢失 = 0 (0% 丢失)，
往返行程的估计时间(以毫秒为单位):
    最短 = 10ms，最长 = 18ms，平均 = 12ms
```


### netsh interface ipv6 相关命令如何使用

直接回车就是帮助信息。


```sh
C:\Windows\system32>netsh interface ipv6

下列指令有效:

此上下文中的命令:
6to4           - 更改到 `netsh interface ipv6 6to4' 上下文。
?              - 显示命令列表。
add            - 在一个表格中添加一个配置项。
delete         - 从一个表格中删除一个配置项。
dump           - 显示一个配置脚本。
help           - 显示命令列表。
isatap         - 更改到 `netsh interface ipv6 isatap' 上下文。
reset          - 重置 IP 配置。
set            - 设置配置信息。
show           - 显示信息。

下列的子上下文可用:
 6to4 isatap

若需要命令的更多帮助信息，请键入命令，接着是空格，
后面跟 ?。

C:\Windows\system32>netsh interface ipv6 show

下列指令有效:

此上下文中的命令:
show addresses - 显示当前 IP 地址。
show compartments - 显示分段参数。
show destinationcache - 显示目标缓存项目。
show dnsservers - 显示 DNS 服务器地址。
show dynamicportrange - 显示动态端口范围配置参数。
show excludedportrange - 显示所有排除的端口范围。
show global    - 显示全局配置普通参数。
show interfaces - 显示接口参数。
show ipstats   - 显示 IP 统计。
show joins     - 显示加入的多播组。
show neighbors - 显示邻居缓存项。
show offload   - 显示卸载信息。
show potentialrouters - 显示潜在路由器。
show prefixpolicies - 显示前缀策略项目。
show privacy   - 显示隐私配置参数。
show route     - 显示路由表项目。
show siteprefixes - 显示站点前缀表项。
show subinterfaces - 显示子接口参数。
show tcpstats  - 显示 TCP 统计。
show teredo    - 显示 Teredo 状态。
show tfofallback - 显示各网络 TCP 快速打开回退状态。
show udpstats  - 显示 UDP 统计。

C:\Windows\system32>netsh interface ipv6 add

下列指令有效:

此上下文中的命令:
add address    - 将静态 IP 地址或默认网关添加到指定接口。
add dnsservers - 添加一个静态 DNS 服务器地址。
add excludedportrange - 为连续的端口块添加排除。
add neighbors  - 添加邻居地址。
add potentialrouter - 将路由器添加到接口的潜在路由器列表上。
add prefixpolicy - 添加前缀策略项目。
add route      - 在接口上添加路由。
add v6v4tunnel - 创建一个“IPv4 中的 IPv6”点对点隧道。

C:\Windows\system32>netsh interface ipv6 set

下列指令有效:

此上下文中的命令:
set address    - 设定通向接口的 IP 地址或默认网关。
set compartment - 修改分段配置参数。
set dnsservers - 设置 DNS 服务器模式和地址。
set dynamicportrange - 修更改态端口分配所使用端口的范围。
set global     - 修改全局配置常规参数。
set interface  - 修改 IP 的接口配置参数。
set neighbors  - 设置邻居地址。
set prefixpolicy - 修改前缀策略信息。
set privacy    - 修改隐私配置参数。
set route      - 修改路由参数。
set subinterface - 修改子接口配置参数。
set teredo     - 设定 Teredo 状态。
```


## reference

-   [RFC6724: Default Address Selection for Internet Protocol Version 6 (IPv6)](https://www.rfc-editor.org/rfc/rfc6724)
-   [IANA IPv6 Special-Purpose Address Registry](https://www.iana.org/assignments/iana-ipv6-special-registry/iana-ipv6-special-registry.xhtml)

