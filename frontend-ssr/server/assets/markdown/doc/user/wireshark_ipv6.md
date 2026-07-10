# <span style="background-color: #b95442;color: white;font-size: 0.43em;border-radius: 5px;padding: 2px 5px;">转载</span> Wireshare 分析 IPv6 包

>[WARNING]
>本文的部分URL已失效，但为了保留原文，未作修改

## 场景

执行 `curl 6.ipw.cn`，查看本机的 IPv6 通信包。

> 如果有 DNS 缓存，可以执行 `sudo killall -HUP mDNSResponder;sudo killall mDNSResponderHelper;sudo dscacheutil -flushcache` 清理缓存

## 理论支撑

[RFC8200](https://www.rfc-editor.org/rfc/rfc8200.html) 的 IPv6 Header Format 章节


```
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |Version| Traffic Class |           Flow Label                  |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |         Payload Length        |  Next Header  |   Hop Limit   |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |                                                               |
   +                                                               +
   |                                                               |
   +                         Source Address                        +
   |          请求的源地址，如 2408:824c:200::2b8b:336f:cc9c          |
   +                                                               +
   |                                                               |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |                                                               |
   +                                                               +
   |                                                               |
   +                      Destination Address                      +
   |         目标地址，例如 2402:4e00:1013:e500:0:940e:29d7:3443     |
   +                                                               +
   |                                                               |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```


```
  Version             4-bit Internet Protocol version number = 6.

  Traffic Class       8-bit Traffic Class field.  See Section 7.

  Flow Label          20-bit flow label.  See Section 6.

  Payload Length      16-bit unsigned integer.  Length of the IPv6
                      payload, i.e., the rest of the packet
                      following this IPv6 header, in octets.  (Note
                      that any extension headers (see Section 4)
                      present are considered part of the payload,
                      i.e., included in the length count.)

  Next Header         8-bit selector.  Identifies the type of header
                      immediately following the IPv6 header.  Uses
                      the same values as the IPv4 Protocol field
                      [IANA-PN].

  Hop Limit           8-bit unsigned integer.  Decremented by 1 by
                      each node that forwards the packet.  When
                      forwarding, the packet is discarded if Hop
                      Limit was zero when received or is decremented
                      to zero.  A node that is the destination of a
                      packet should not discard a packet with Hop
                      Limit equal to zero; it should process the
                      packet normally.

  Source Address      128-bit address of the originator of the
                      packet.  See [RFC4291].

  Destination Address 128-bit address of the intended recipient of
                      the packet (possibly not the ultimate
                      recipient, if a Routing header is present).
                      See [RFC4291] and Section 4.4.
```


## 实战

使用 wireshark 开启抓包，命令访问 \`curl 6.wsmdn.dpdns.org\`，抓包截图如下：

![wireshark_ipv6](/doc/wireshark_ipv6.png)

-   DNS 解析（2332-2335）

    -   编号为2332：向 DNS 服务器 \`240e:1f:1::1\` （广东电信 IPv6 DNS）请求 6.wsmdn.dpdns.org 的 A 记录（即IPv4 记录）
    -   2333： 请求 6.wsmdn.dpdns.org 的 AAAA 记录（即IPv6 记录）
    -   2334: DNS 服务器返回 6.wsmdn.dpdns.org 的 AAAA 记录为 \`2402:4e00:1013:e500:0:940e:29d7:3443\`
    -   2335: 返回 6.wsmdn.dpdns.org 的 A 记录为空（实际上 6.wsmdn.dpdns.org 只添加 AAAA 记录，未添加 A 记录）
-   TCP 三次握手（2336-2338）

-   HTTP 传输（2339-2341）

-   TCP 四次挥手再见（2342-2345）


![wireshark\_ipv6](/doc/wireshark_ipv6_tcp_ack.png)

在上图中可以看到 IPv6 包头。

> 下次解释下，这次太晚了，先睡觉了。
