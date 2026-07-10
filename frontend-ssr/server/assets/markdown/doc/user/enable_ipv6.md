# <span style="background-color: #b95442;color: white;font-size: 0.43em;border-radius: 5px;padding: 2px 5px;">转载</span> 个人宽带如何开启IPv6网络访问

- [1. 修改光猫：开启桥接和 IPv6 设置](#1-修改光猫-开启桥接和-ipv6-设置)
- [2. 设置路由器PPPoE拨号和开启 IPv6设置](#2-设置路由器pppoe拨号和开启-ipv6设置)
- [3. 验证 IPv6 网络](#3-验证-ipv6-网络)
    - [3.1 网页访问验证](#31-网页访问验证)
    - [3.2 域名访问验证](#32-域名访问验证)
    - [3.3 命令行验证](#33-命令行访问验证)
      
IPv6是大势所趋，就在前段时间湖南联通发布公告，对家庭宽带提供 IPv6 地址，不再提供 IPv4地址，那本文就介绍 **个人宽带如何开启 IPv6网络访问**。

![湖南联通停止向普通家庭宽带用户提供公网 IPv4 地址](/doc/16355852767627.jpg)

先给大家看下开启 IPv6 网络访问后的效果。

这是一个 [IPv6 地址查询](https://ipw.wsmdn.top/ipv6) 的网站，可以看到页面提示 您的网络 IPv6 访问优先。 
![您的网络 IPv6 访问优先](/doc/v6test.png)

如果使用 Google Chrome 浏览器，右键网页开启审查模式，也可以看到网络请求的是 IPv6 地址。 
![网络请求的是 IPv6 地址](/doc/v6connect.png)

至于 IPv6 有哪些好的资源，大家可以去搜索 IPv6资源导航。

接下来我们介绍下，**个人宽带如何开启 IPv6.**

主要修改两个配置：**光猫(开启桥接和IPv6）、路由器（PPPoe拨号和 IPv6网络设置）。**

接下来以电信天翼光猫和小米4A千兆路由器为例，介绍如何开启 IPv6 网络访问。

## 1. 修改光猫：开启桥接和 IPv6 设置
> 光猫：在电信宽带中就是 天翼宽带家庭网关。

查看光猫背面的光猫 IP 地址，使用管理员账号密码登入，进入光猫后台。

- 地址：http://192.168.1.1
- 用户名：useradmin
- 密码：nE7jA%5 （注意：不是普通账号，而是管理员账号，不知道就搜一下或问客服） 
![开启桥接和 IPv6 设置](/doc/16355624511052.jpg)
  

可以看到下面这个页面，证明光猫管理端进入成功。 
![光猫管理端进入成功](/doc/16355616705145.jpg)

点击网络，会看到 `网络连接` 页面。
![网络连接](/doc/16355617569448.jpg)

**连接名称** 选择 `3_INTERNET_R_VID_41`，**连接模式** 选择 `桥接`，IP模式为 `IPv4&IPv6`。点击 `保存/应用`，大约等待 10s 会生效。 
![3_INTERNET_R_VID_41](/doc/16355621400735.jpg)

此外，如果你 **忘记了 PPPoE 拨号的账号密码**，也可以在这个页面查看，把 **连接模式** 选择为 `PPPoE`，可以查看曾经保存的账号密码，密码看不到的话，用 Chrome 的审查模式，右键密码的输入框，在下图中将 password 删除，这样就能查看拨号密码了。

> 这个PPPoe 拨号账号密码会在路由器中拨号使用。 这个PPPoe 拨号账号密码会在路由器中拨号使用
> ![教科书级别的前端失误](/doc/16355621400735.jpg)

接下来，在 **状态** 菜单中，可以看到 IPv4 和 IPv6 已经开启成功。 IPv4 和 IPv6 已经开启成功
![开启](/doc/16355623081368.jpg)

光猫设置好了，接下来设置路由器。

## 2. 设置路由器PPPoE拨号和开启 IPv6设置
查看路由器背后的管理地址，进入路由器后台。 进入路由器后台
![路由器后台管理页面](/doc/16355626382824.jpg)

在上网设置中，**上网方式** 选择 `PPPoE`，填写宽带拨号账号密码。（如果不记得，可以在光猫后台找到，或者电话客服。） 上网方式选择PPPoE
![PPPoe](/doc/16355626574718.jpg)
开启**IPv6网络设置，上网方式选择 Native。**

手动配置DNS，比如广东地区是 240e:1f:1::1 和 240e:1f:1::33，更多查看 IPv6 DNS 地址列表，然后点击 应用。 
![手动配置DNS](/doc/16355698585824.jpg)

在上网设置页面顶部就能看到当前的上网信息了，可以看到 WAN 口已经获取到 IPv6 地址，而且还有 公网IPv6 前缀。 
![WAN 口已经获取到 IPv6 地址2](/doc/16355909063872.jpg)

IPv6 网络开启成功后，我们需要验证下是否真正开启成功。

## 3. 验证 IPv6 网络
以下几种验证方式，任选其一。

### 3.1 网页访问验证
这是一个 [IPv6 地址查询](https://ipw.wsmdn.top/ipv6) 的网站，可以看到上面提示 您的网络 IPv6 访问优先。

也可以对自己的公网 IPv6 地址进行 [在线 Ping](https://itdog.cn/ping_ipv6)。

### 3.2 域名访问验证
打开 [https://6.wsmdn.dpdns.org/](https://6.wsmdn.dpdns.org/)，如果能访问成功，那么证明 IPv6 网络开启成功。
>该站点仅支持 IPv6 网络访问，IPv4 网络无法访问。

```json
// https://6.ipw.cn/
240e:3b7:3b7:3b7::3b7
```
打开 [https://test.wsmdn.dpdns.org/](https://test.wsmdn.dpdns.org/)，**如果返回的 IPVersion 字段为 IPv6，则当前网络 IPv6 访问优先，**如果返回的 IPVersion 字段为 IPv4，则当前网络 IPv4 访问优先。
>[test.wsmdn.dpdns.org](https://test.wsmdn.dpdns.org/) 支持 IPv4/IPv6双栈访问。
```json
// https://test.ipw.cn/
240e:3b7:3b7:3b7::3b7
```

### 3.3 命令行验证
#查看本机 IPv6 地址
- Windows
```sh
C:\Users\>ipconfig/all
无线局域网适配器 WLAN:

   连接特定的 DNS 后缀 . . . . . . . :
   描述. . . . . . . . . . . . . . . : Intel(R) Dual Band Wireless-AC 7265
   物理地址. . . . . . . . . . . . . : 48-45-45-45-45-48
   DHCP 已启用 . . . . . . . . . . . : 是
   自动配置已启用. . . . . . . . . . : 是
   IPv6 地址 . . . . . . . . . . . . : 240e:3b7:3b7:3b7::3b7(首选)
   获得租约的时间  . . . . . . . . . : 2021年10月31日 7:32:02
   租约过期的时间  . . . . . . . . . : 2021年10月31日 9:32:02
   IPv6 地址 . . . . . . . . . . . . : 240e:3b7:3b7:3b7::3b7(首选)
   临时 IPv6 地址. . . . . . . . . . : 240e:3b7:3b7:3b7::3b7(首选)
   本地链接 IPv6 地址. . . . . . . . : fe80::c77:4078:c21a:b17e%19(首选)
   IPv4 地址 . . . . . . . . . . . . : 192.168.31.31(首选)
   子网掩码  . . . . . . . . . . . . : 255.255.255.0
   获得租约的时间  . . . . . . . . . : 2021年10月31日 7:32:01
   租约过期的时间  . . . . . . . . . : 2021年10月31日 19:32:01
   默认网关. . . . . . . . . . . . . : fe80::5648:f6ff:feb3:f1e8%19
                                       192.168.31.1
   DHCP 服务器 . . . . . . . . . . . : 192.168.31.1
   DHCPv6 IAID . . . . . . . . . . . : 155555232
   DHCPv6 客户端 DUID  . . . . . . . : 00-01-00-80-24-D2-80-80-80-80-80-80-80-80
   DNS 服务器  . . . . . . . . . . . : 240e:1f:1::1
                                       192.168.31.1
   TCPIP 上的 NetBIOS  . . . . . . . : 已启用
```
- macOS
```sh
$ ifconfig  en0
en0: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1500
	options=400<CHANNEL_IO>
	ether 88:66:66:66:66:88
	inet 192.168.31.215 netmask 0xffffff00 broadcast 192.168.31.255
	inet6 fe80::181a:8937:988d:ae46%en0 prefixlen 64 secured scopeid 0x6
	inet6 240e:3b7:3b7:3b7::3b7 prefixlen 64 autoconf secured
	inet6 240e:3b7:3b7:3b7::3b7 prefixlen 64 autoconf temporary
	inet6 240e:3b7:3b7:3b7::3b7 prefixlen 64 dynamic
	nd6 options=201<PERFORMNUD,DAD>
	media: autoselect
	status: active
```
#### Ping IPv6 网站
- Windows
```sh
PS C:\Users\Administrator> ping -6 ipw.wsmdn.top

正在 Ping ipw.wsmdn.top [2606:4700:3037::ac43:88b8] 具有 32 字节的数据:
来自 2606:4700:3037::ac43:88b8 的回复: 时间=190ms
来自 2606:4700:3037::ac43:88b8 的回复: 时间=190ms
来自 2606:4700:3037::ac43:88b8 的回复: 时间=193ms
来自 2606:4700:3037::ac43:88b8 的回复: 时间=193ms

2606:4700:3037::ac43:88b8 的 Ping 统计信息:
    数据包: 已发送 = 4，已接收 = 4，丢失 = 0 (0% 丢失)，
往返行程的估计时间(以毫秒为单位):
    最短 = 190ms，最长 = 193ms，平均 = 191ms
PS C:\Users\Administrator> ping ipw.wsmdn.top

正在 Ping ipw.wsmdn.top [2606:4700:3037::ac43:88b8] 具有 32 字节的数据:
来自 2606:4700:3037::ac43:88b8 的回复: 时间=190ms
来自 2606:4700:3037::ac43:88b8 的回复: 时间=195ms
来自 2606:4700:3037::ac43:88b8 的回复: 时间=189ms
来自 2606:4700:3037::ac43:88b8 的回复: 时间=193ms

2606:4700:3037::ac43:88b8 的 Ping 统计信息:
    数据包: 已发送 = 4，已接收 = 4，丢失 = 0 (0% 丢失)，
往返行程的估计时间(以毫秒为单位):
    最短 = 189ms，最长 = 195ms，平均 = 191ms
PS C:\Users\Administrator> ping -4 ipw.wsmdn.top

正在 Ping ipw.wsmdn.top [172.67.136.184] 具有 32 字节的数据:
来自 172.67.136.184 的回复: 字节=32 时间=211ms TTL=52
来自 172.67.136.184 的回复: 字节=32 时间=212ms TTL=52
来自 172.67.136.184 的回复: 字节=32 时间=212ms TTL=52
来自 172.67.136.184 的回复: 字节=32 时间=212ms TTL=52

172.67.136.184 的 Ping 统计信息:
    数据包: 已发送 = 4，已接收 = 4，丢失 = 0 (0% 丢失)，
往返行程的估计时间(以毫秒为单位):
    最短 = 211ms，最长 = 212ms，平均 = 211ms
```
- macOS
```sh
$ ping ipw.wsmdn.top
PING ipw.wsmdn.top (159.75.190.197): 56 data bytes
64 bytes from 159.75.190.197: icmp_seq=0 ttl=53 time=14.769 ms
64 bytes from 159.75.190.197: icmp_seq=1 ttl=53 time=10.403 ms
64 bytes from 159.75.190.197: icmp_seq=2 ttl=53 time=10.125 ms
64 bytes from 159.75.190.197: icmp_seq=3 ttl=53 time=17.507 ms

$ ping6 ipw.wsmdn.top
PING6(56=40+8+8 bytes)  --> 2402:4e00:40:40::2:331
16 bytes from 2402:4e00:40:40::2:331, icmp_seq=0 hlim=53 time=18.052 ms
16 bytes from 2402:4e00:40:40::2:331, icmp_seq=1 hlim=53 time=16.393 ms
16 bytes from 2402:4e00:40:40::2:331, icmp_seq=2 hlim=53 time=14.736 ms
16 bytes from 2402:4e00:40:40::2:331, icmp_seq=3 hlim=53 time=14.062 ms
 --- ipw.cn ping6 statistics ---
4 packets transmitted, 4 packets received, 0.0% packet loss
round-trip min/avg/max/std-dev = 14.062/15.811/18.052/1.547 ms
```
好的，那就先介绍到这里了，预祝大家开启 IPv6 网络成功。