
# [原创网站如何开启 HTTP/2](#网站如何开启-http-2)

## [HTTP/2是什么？](#http-2是什么)

在[RFC 9113 HTTP/2](https://datatracker.ietf.org/doc/html/rfc9113)介绍了 HTTP/2: 通过引入字段压缩和允许在同一连接上进行多个并发交换，使网络资源得到更有效的利用，并减少了延迟。

> This specification describes an optimized expression of the semantics of the Hypertext Transfer Protocol (HTTP), referred to as HTTP version 2 (HTTP/2). HTTP/2 enables a more efficient use of network resources and a reduced latency by introducing field compression and allowing multiple concurrent exchanges on the same connection. This document obsoletes RFCs 7540 and 8740.

在之前的[RFC 7540 Hypertext Transfer Protocol Version 2 (HTTP/2)](https://www.rfc-editor.org/rfc/rfc7540#section-4.1)可以找到 Frame Format

```
4.1.  Frame Format

   All frames begin with a fixed 9-octet header followed by a variable-
   length payload.

    +-----------------------------------------------+
    |                 Length (24)                   |
    +---------------+---------------+---------------+
    |   Type (8)    |   Flags (8)   |
    +-+-------------+---------------+-------------------------------+
    |R|                 Stream Identifier (31)                      |
    +=+=============================================================+
    |                   Frame Payload (0...)                      ...
    +---------------------------------------------------------------+

                          Figure 1: Frame Layout
```

## [客户端如何检测 HTTP/2](#客户端如何检测-http-2)

### [CURL](#curl)

查看 Response Header 来确认服务器支持的 HTTP 的版本。

```
$ curl -I https://ipw.cn
HTTP/2 200 
server: nginx
content-type: text/html
accept-ranges: bytes
content-length: 11710
strict-transport-security: max-age=16070400;
```

HTTP/2 需要客户端和服务端都支持，一般浏览器都支持。

> 从 CURL 的官方文档 可以支持 curl 7.47.0 才支持 HTTP/2 访问。

Since 7.47.0, the curl tool enables HTTP/2 by default for HTTPS connections.

### [Chrome](#chrome)

在审查模式中，切换到`Network Tab`，列表页中增加`Protocol`一列，访问支持 HTTP2 的页面，可以看到值为`h2`

![http2 chrome](/doc/http2-chrome.png)

## [Wireshark 抓取并分析 HTTP2协议](#wireshark-抓取并分析-http2协议)

HTTP2 位于 TLS 协议之上，消息会被加密，需要把通信的`Pre-Master-Secret`打印出来，并被 Wireshark 读取，以便 Wireshark 能解开被 TLS 加密的上层 HTTP2 协议。

以 macOS 系统为例。

### [打印 Pre-Master-Secret](#打印-pre-master-secret)

```
mkdir -p ~/tls
echo -e "\nexport SSLKEYLOGFILE=~/tls/sslkeylog.log" 
souce  ~/.bash_profile
```

### [设置 Wireshark TLS 协议读取 Pre-Master-Secret](#设置-wireshark-tls-协议读取-pre-master-secret)

在 Wireshark Preferences 的 Protocols 中找到 TLS，做如下配置：

![wireshark http2](/doc/wireshark_tls_preferences.png)

### [启动 Chrome](#启动-chrome)

退出 Chrome，在刚才的终端中打开 Chrome，确保能加载到上面的环境变量。

```
open /Applications/Google\ Chrome.app
```

### [开启 Wireshark 抓包](#开启-wireshark-抓包)

可以看到建立了 TLS1.3 的链接后，就开始 HTTP2 的协议通信。

- Chrome 请求首页

Stream ID = 1，这也是与 HTTP/1.1 最大的区别，是流，而不是 文本。

![wireshark http2](/doc/wireshark_http2_1.png)

- 服务端返回首页

Stream ID = 1

![wireshark http2](/doc/wireshark_http2.png)

## [Nginx 开启 HTTP/2](#nginx-开启-http-2)

从 Nginx 官网[Module ngx_http_v2_module](https://nginx.org/en/docs/http/ngx_http_v2_module.html)可以了解到 Nginx 从 1.9.5 版本开始支持 HTTP/2。

目前主流发行版中的 Nginx 默认支持 HTTP2，自行编译可加上`--with-http_v2_module`参数。

> The ngx_http_v2_module module (1.9.5) provides support for HTTP/2 and supersedes the ngx_http_spdy_module module.

> This module is not built by default, it should be enabled with the --with-http_v2_module configuration parameter.

执行命令`nginx -V`看编译参数中是否带`--with-http_v2_module`来确认 Nginx 是否已经支持 HTTP2

开启 HTTP2的前提是开启 SSL，只需要在 ssl 后面加上 http2 即可。

```
server {
    listen 443 ssl http2;

    ssl_certificate server.crt;
    ssl_certificate_key server.key;
}
```

具体示例详见[Nginx 开启 IPv6](/doc/server/nginx_ipv6)


