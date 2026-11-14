# <span style="background-color: #b95442;color: white;font-size: 0.43em;border-radius: 5px;padding: 2px 5px;">转载</span> 命令行(curl)获取 IPv4 和 IPv6 地址

-   [1\. 查询本机外网IPv4地址](#1-查询本机外网ipv4地址)
-   [2\. 查询本机外网IPv6地址](#2-查询本机外网ipv6地址)
-   [3\. 测试网络是IPv4还是IPv6访问优先](#3-测试网络是ipv4还是ipv6访问优先)

> 本站是个人站点，用于作者实战学习前端和后台知识，请勿用于商业用途，仅供个人测试学习之用，请遵守中国法律法规

通过 curl 命令获取公网 IPv4 和 IPv6 地址，还可以返回是 IPv4 还是 IPv6 访问优先。命令行里做 IPv4 查询与 IPv6 查询不需要打开网页，适合写进监控脚本或 CI 环境。

## 1\. 查询本机外网IPv4地址


```
curl 4.wsmdn.dpdns.org
## 返回示例 106.224.145.147
```


## 2\. 查询本机外网IPv6地址


```
curl 6.wsmdn.dpdns.org
## 返回示例 2408:824c:200::2b8b:336f:cc9c
```


## 3\. 测试网络是IPv4还是IPv6访问优先

访问IPv4/IPv6双栈站点，如果返回IPv6地址，则IPv6访问优先，否则为 IPv4 地址。


```
curl test.wsmdn.dpdns.org
## 返回示例1：106.224.145.147
## 返回示例2：2408:824c:200::2b8b:336f:cc9c
```

## 相关检测工具

不想敲命令可以直接在 [首页](/) 看两个协议栈的出口地址；更多检测方式见 [IPv6 检测方法与工具](/ipv6check)。
