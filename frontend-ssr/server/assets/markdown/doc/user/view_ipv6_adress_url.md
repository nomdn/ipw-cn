# <span style="background-color: #b95442;color: white;font-size: 0.43em;border-radius: 5px;padding: 2px 5px;">转载</span> 浏览器访问 IPv6 地址
> [!WARNING]  
> 镜像版本使用Cloudflare Workers，无固定IP，本文内容仅供参考！！

在 IPv6 地址两边加上中括号，就可以通过浏览器访问了

-   [浏览器访问](#浏览器访问)
-   [Ping IPv6 地址](#ping-ipv6-地址)
-   [终端 CURL 访问](#终端-curl-访问)
    -   [MacOS](#macos)
    -   [Linux](#linux)
    -   [curl 指定 IPv6 或 IPv4 访问](#curl-指定-ipv6-或-ipv4-访问)
-   [telnet ipv6 地址](#telnet-ipv6-地址)
-   [IPv6工具箱 小程序 和 iOS App](#ipv6工具箱-小程序-和-ios-app)

## 浏览器访问

访问 `http://[2402:4e00:1013:e500:0:9671:f018:4947]` ，会返回本机 IPv6 IP地址。

![](https://static.ipw.wsmdn.top/images/doc/ipv6/user/media/view_ipv6_adress_url.png)

如果有端口，在 IP 地址后面添加端口，比如

`http://[2402:4e00:1013:e500:0:9671:f018:4947]:80/`

## Ping IPv6 地址

> [6.wsmdn.dpdns.org](https://6.wsmdn.dpdns.org/) 域名解析的 AAAA 记录就是上面这个 IPv6 IP。


```
~$ ping6 6.wsmdn.dpdns.org
PING6(56=40+8+8 bytes) 1111:1111:30c0:9556:b807:e464:1111:1111 --> > 2402:4e00:1013:e500:0:9671:f018:4947
16 bytes from 2402:4e00:1013:e500:0:9671:f018:4947, icmp_seq=0 hlim=52 time=8.748 ms
16 bytes from 2402:4e00:1013:e500:0:9671:f018:4947, icmp_seq=1 hlim=52 time=8.715 ms
16 bytes from 2402:4e00:1013:e500:0:9671:f018:4947, icmp_seq=2 hlim=52 time=8.426 ms
16 bytes from 2402:4e00:1013:e500:0:9671:f018:4947, icmp_seq=3 hlim=52 time=15.139 ms
16 bytes from 2402:4e00:1013:e500:0:9671:f018:4947, icmp_seq=4 hlim=52 time=9.092 ms
```


## 终端 CURL 访问

### MacOS


```
curl [2402:4e00:1013:e500:0:9671:f018:4947]
1111:1111:30c0:9556:b807:e464:1111:1111


curl http://[2402:4e00:1013:e500:0:9671:f018:4947]/
1111:1111:30c0:9556:b807:e464:1111:1111
```


### Linux

如果在 Linux 下，还需要增加 \`-g\` 参数，否则会提示 \`curl: (3) \[globbing\] error: bad range specification after pos\`。


```
 -g, --globoff
    his  option  switches off the "URL globbing parser". When you set this option, you can specify URLs that contain the letters {}[] without having them being interpreted by curl itself. Note that these letters are not normal legal URL contents but they should be encoded according to the URI standard.
```


```
curl -g http://[2402:4e00:1013:e500:0:9671:f018:4947]/
```


### curl 指定 IPv6 或 IPv4 访问

如果同一个 host 同时解析到 IPv6 和 IPv4 地址，即 IPv4/IPv6 双栈，则 curl 使用参数可指定 IP 协议的版本。

> \-4, --ipv4
>
> If curl is capable of resolving an address to multiple IP versions (which it is if it is IPv6-capable), this option tells curl to resolve names to IPv4 addresses only.
>
> \-6, --ipv6
>
> If curl is capable of resolving an address to multiple IP versions (which it is if it is IPv6-capable), this option tells curl to resolve names to IPv6 addresses only.

比如 test.wsmdn.dpdns.org 同时解析到 IPv4 和 IPv6 地址。


```
curl -4 test.wsmdn.dpdns.org
106.224.145.147

curl -6 test.wsmdn.dpdns.org
2408:824c:200::2b8b:336f:cc9c
```


## telnet ipv6 地址


```
telnet -6 2402:4e00:1013:e500:0:940e:29d7:3443 80
Trying 2402:4e00:1013:e500:0:940e:29d7:3443...
Connected to 2402:4e00:1013:e500:0:940e:29d7:3443.
Escape character is '^]'.
```


