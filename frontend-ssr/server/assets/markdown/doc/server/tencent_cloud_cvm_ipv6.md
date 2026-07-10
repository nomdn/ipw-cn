# <span style="background-color: #b95442;color: white;font-size: 0.43em;border-radius: 5px;padding: 2px 5px;">转载</span> 腾讯云 cvm 开启 IPv6

-   [前置条件](#%E5%89%8D%E7%BD%AE%E6%9D%A1%E4%BB%B6)
-   [1 为 VPC 分配 IPv6 CIDR](#_1-%E4%B8%BA-vpc-%E5%88%86%E9%85%8D-ipv6-cidr)
-   [2 为子网分配 IPv6 CIDR](#_2-%E4%B8%BA%E5%AD%90%E7%BD%91%E5%88%86%E9%85%8D-ipv6-cidr)
-   [3 CVM 配置 IPv6 地址](#_3-cvm-%E9%85%8D%E7%BD%AE-ipv6-%E5%9C%B0%E5%9D%80)
-   [4 为云服务器的 IPv6 地址开通公网](#_4-%E4%B8%BA%E4%BA%91%E6%9C%8D%E5%8A%A1%E5%99%A8%E7%9A%84-ipv6-%E5%9C%B0%E5%9D%80%E5%BC%80%E9%80%9A%E5%85%AC%E7%BD%91)
-   [5 配置 CVM 操作系统的 IPv6](#_5-%E9%85%8D%E7%BD%AE-cvm-%E6%93%8D%E4%BD%9C%E7%B3%BB%E7%BB%9F%E7%9A%84-ipv6)
-   [6 配置 IPv6 的安全组规则](#_6-%E9%85%8D%E7%BD%AE-ipv6-%E7%9A%84%E5%AE%89%E5%85%A8%E7%BB%84%E8%A7%84%E5%88%99)
-   [7 测试连通性](#_7-%E6%B5%8B%E8%AF%95%E8%BF%9E%E9%80%9A%E6%80%A7)
-   [reference](#reference)

一文看懂在腾讯云 CVM上开启 IPv6，只需 7 步。

## 前置条件

目前 IPv6/IPv4 双栈 VPC 功能处于内测中，如有需要，请提交 [内测申请](https://cloud.tencent.com/apply/p/a9k0gialqhj)。

> 秒批，只要申请，会自动审批。

## 1 为 VPC 分配 IPv6 CIDR

在 [私有网络](https://console.cloud.tencent.com/vpc) 菜单中，选择一个网络，`编辑 IPv6 CIDR`,为 VPC 分配一个 IPv6 网段。

![为 VPC 分配 IPv6 CIDR](/doc/16333892496060.jpg)

弹窗中点击 `获取 IPv6 CIDR`![\-w1785](/doc/16333892838706.jpg)![\-w1786](/doc/16333893415522.jpg)

## 2 为子网分配 IPv6 CIDR

选择上述 VPC 中的一个子网，点击 `获取 IPv6 CIDR`![\-w1787](/doc/16333896690329.jpg)![\-w1787](/doc/16333896988636.jpg)

## 3 CVM 配置 IPv6 地址

若云服务器在购买时未分配 IPv6 地址，可在对应云服务器实例的操作栏下，选择【更多】>【IP/网卡】>【管理IPv6地址】，为主网卡分配 IPv6 地址。

![\-w1791](/doc/16333903308731.jpg)![\-w1791](/doc/16333903920413.jpg)![\-w1788](/doc/16333904352646.jpg)

## 4 为云服务器的 IPv6 地址开通公网

在左侧目录下，选择【IP与网卡】>【弹性公网IPv6】，点击【申请】。

![\-w1790](/doc/16333906824294.jpg)![\-w1791](/doc/16333908635792.jpg)![\-w1790](/doc/16333909449752.jpg)

![\-w1791](/doc/16333909824098.jpg)

## 5 配置 CVM 操作系统的 IPv6

根据操作系统版本，参考文档 [配置 CVM 操作系统的 IPv6](https://cloud.tencent.com/document/product/1142/47666)，选择对应的方式配置。

以下是 CentOS 7 的配置示例。


```
vim /etc/sysconfig/network-scripts/ifcfg-eth0
```


新增一行配置 `DHCPV6C=yes`, 可以自动获取 IPv6 地址。

接下来，增加默认 IPv6 静态路由


```
vim /etc/sysconfig/network-scripts/route6-eth0
```


新增一行配置 `default dev eth0 via fe80::feee:ffff:feff:ffff`

重启网络。


```
systemctl restart network
```


查看 IPv6 地址是否配置成功。


```
# ifconfig eth0
eth0: flags=4163<UP,BROADCAST,RUNNING,MULTICAST>  mtu 1500
        inet 10.11.0.7  netmask 255.255.255.0  broadcast 10.12.0.255
        inet6 2402:4e00:1111:2222:0:3333:4444:6666  prefixlen 128  scopeid 0x0<global>
        inet6 fe80::5054:ff:ee3a:dbb  prefixlen 64  scopeid 0x20<link>
        ether 12:54:00:3a:0d:bb  txqueuelen 1000  (Ethernet)
        RX packets 32022 bytes 6859648 (3.8 GiB)
        RX errors 0  dropped 0  overruns 0  frame 0
        TX packets 618586  bytes 6421714 (8.0 GiB)
        TX errors 0  dropped 0 overruns 0  carrier 0  collisions 0
```


## 6 配置 IPv6 的安全组规则

安全组出策略放行。

> 源地址为 `::/0`

![\-w1790](/doc/16333930536112.jpg)

## 7 测试连通性

从 cvm ping 外部 ipv6 网址，确认 IPv6 网络可以出得去。


```
# ping6 6.wsmdn.dpdns.org
PING 6.wsmdn.dpdns.org(2402:4e00:1013:e500:0:9671:f018:4947 (2402:4e00:1013:e500:0:9671:f018:4947)) 56 data bytes
16 bytes from 2402:4e00:1013:e500:0:9671:f018:4947, icmp_seq=0 hlim=52 time=8.748 ms
16 bytes from 2402:4e00:1013:e500:0:9671:f018:4947, icmp_seq=1 hlim=52 time=8.715 ms
16 bytes from 2402:4e00:1013:e500:0:9671:f018:4947, icmp_seq=2 hlim=52 time=8.426 ms
16 bytes from 2402:4e00:1013:e500:0:9671:f018:4947, icmp_seq=3 hlim=52 time=15.139 ms
16 bytes from 2402:4e00:1013:e500:0:9671:f018:4947, icmp_seq=4 hlim=52 time=9.092 ms
```


获取本机 IPv6 地址


```
# curl 6.wsmdn.dpdns.org
240e:1f:1::1
```


当然，也可以在线 [IPv6 Ping 测试工具](https://itdog.cn/ping_ipv6/) 进行检查。

此外，通过 [IPv6 地址查询工具](https://ipw.wsmdn.top/ipv6/) 可以查询对应 IPv6 地址所属的地理位置。

当网站建设好了后，也可以通过 [工具检测网站IPv6](https://ipw.wsmdn.top/ipv6webcheck/) 是否开启成功。

## reference

-   [1] 腾讯云. [搭建 IPv6 私有网络](https://cloud.tencent.com/document/product/1142/47665#step4)

