# <span style="background-color: #b95442;color: white;font-size: 0.43em;border-radius: 5px;padding: 2px 5px;">转载</span> IPv6 Ping 检测原理

-   [1\. IPv6 Ping 地址抓包分析](#1-ipv6-ping-%E5%9C%B0%E5%9D%80%E6%8A%93%E5%8C%85%E5%88%86%E6%9E%90)
-   [2\. 了解ICMPv6 报文格式](#2-%E4%BA%86%E8%A7%A3icmpv6-%E6%8A%A5%E6%96%87%E6%A0%BC%E5%BC%8F)

## 1\. IPv6 Ping 地址抓包分析

-   协议标准：[RFC4443](https://www.rfc-editor.org/rfc/rfc4443.html) Internet Control Message Protocol (ICMPv6) for the Internet Protocol Version 6 (IPv6) Specification
-   先熟悉下 IPv6 报文格式


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
   |                                                               |
   +                                                               +
   |                                                               |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |                                                               |
   +                                                               +
   |                                                               |
   +                      Destination Address                      +
   |                                                               |
   +                                                               +
   |                                                               |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```


看下 IPv6 包头，其中 \`Next Header: ICMPv6 (58)\` 表示包的内容为 ICMPv6包


```
Internet Protocol Version 6, Src: 240e::, Dst: 2402:4e00:40:40::2:3b6
    0110 .... = Version: 6
    .... 0000 0000 .... .... .... .... .... = Traffic Class: 0x00 (DSCP: CS0, ECN: Not-ECT)
    .... 1110 0000 1010 0000 0000 = Flow Label: 0xe0a00
    Payload Length: 16
    Next Header: ICMPv6 (58)
    Hop Limit: 64
    Source Address: 240e::
    Destination Address: 2402:4e00:40:40::2:3b6
```


## 2\. 了解ICMPv6 报文格式


```
      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
      |     Type      |     Code      |          Checksum             |
      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
      |                                                               |
      +                         Message Body                          +
      |                                                               |
```


第 1 个是 Ping 本站地址的请求包

\`Type: Echo (ping) request (128)\` 表示 \`发起 Ping 请求包\`


```
Internet Control Message Protocol v6
    Type: Echo (ping) request (128)
    Code: 0
    Checksum: 0xa7b5 [correct]
    [Checksum Status: Good]
    Identifier: 0x0484
    Sequence: 0
    [Response In: 4]
    Data (8 bytes)
        Data: 62e681f7000d7bfa
        [Length: 8]
```


第 2 个是 Ping 本站地址 返回包

\`Type: Echo (ping) reply (129)\`，表示 \`包成功返回，Ping 成功了\`


```
 Internet Control Message Protocol v6
    Type: Echo (ping) reply (129)
    Code: 0
    Checksum: 0xa6b5 [correct]
    [Checksum Status: Good]
    Identifier: 0x0484
    Sequence: 0
    [Response To: 3]
    [Response Time: 16.419 ms]
    Data (8 bytes)
        Data: 62e681f7000d7bfa
        [Length: 8]
```


第4个是 \`Ping 2402:4e00:40:40::2:3b7\` 的返回包（ping 失败了）

\`Type: Time Exceeded (3)\`，表示 \`时间超时，目标地址不可到达\`


```
Internet Control Message Protocol v6
    Type: Time Exceeded (3)
    Code: 0 (hop limit exceeded in transit)
    Checksum: 0xbf81 [correct]
    [Checksum Status: Good]
    Reserved: 00000000
    Internet Protocol Version 6, Src: 240e::, Dst: 2402:4e00:40:40::2:3b7
    Internet Control Message Protocol v6
        Type: Echo (ping) request (128)
        Code: 0
        Checksum: 0x00aa [unverified] [in ICMP error packet]
        [Checksum Status: Unverified]
        Identifier: 0x0489
        Sequence: 0
        Data (8 bytes)
            Data: 62e681fd000c22fb
            [Length: 8]
```