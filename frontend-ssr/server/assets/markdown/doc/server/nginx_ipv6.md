# <span style="background-color: #b95442;color: white;font-size: 0.43em;border-radius: 5px;padding: 2px 5px;">转载</span> Nginx 开启 IPv6

-   [前置条件](#%E5%89%8D%E7%BD%AE%E6%9D%A1%E4%BB%B6)
-   [Nginx 开启 IPv6](#nginx-%E5%BC%80%E5%90%AF-ipv6-1)
-   [Nginx 开启 IPv6 SSL](#nginx-%E5%BC%80%E5%90%AF-ipv6-ssl)

一文看懂 Nginx 中开启 IPv6，包含设置 IPv6 SSL证书。

## 前置条件

[所在服务器已经开启 IPv6](/server/tencent_cloud_cvm_ipv6)。

## Nginx 开启 IPv6

Nginx 默认配置中已经开启了 IPv6。

即 `listen [::]:80 default_server;`


```
    server {
        listen       80 default_server;
        listen       [::]:80 default_server;
        server_name  _;
        root         /usr/share/nginx/html;

        # Load configuration files for the default server block.
        include /etc/nginx/default.d/*.conf;

        location / {
        }

        error_page 404 /404.html;
            location = /40x.html {
        }

        error_page 500 502 503 504 /50x.html;
            location = /50x.html {
        }
    }
```


重启后，检查端口监听，发现已经监听在 `:::80` 上，即监听在 IPv6 的 80端口上。


```sh
# netstat -ntlp | grep nginx
tcp        0      0 0.0.0.0:80              0.0.0.0:*               LISTEN      29391/nginx: master
tcp6       0      0 :::80                   :::*                    LISTEN      29391/nginx: master
```


参考文档 [浏览器访问 IPv6 地址](/user/view_ipv6_adress_url)访问 IPv6 网站，验证是否访问成功。

如果访问失败，请检查 [安全组是否开放 IPv6 的 80 端口 入请求的访问策略](/server/website_enable_ipv6)。

使用 curl 访问速度测试


``` bash
$ curl http://6.wsmdn.dpdns.org -v
*   Trying 2402:4e00:1013:e500:0:9671:f018:4947...
* TCP_NODELAY set
* Connected to 6.wsmdn.dpdns.org (2402:4e00:1013:e500:0:9671:f018:4947) port 80 (#0)
> GET / HTTP/1.1
> Host: 6.wsmdn.dpdns.org
> User-Agent: curl/7.64.1
> Accept: */*
>
< HTTP/1.1 200 OK
< Server: nginx
< Date: Sun, 19 Jun 2022 13:22:01 GMT
< Content-Type: text/plain; charset=utf-8
< Content-Length: 38
< Connection: keep-alive
< Access-Control-Allow-Origin: *
<
* Connection #0 to host 6.wsmdn.dpdns.org left intact
2408:824c:200::2b8b:336f:cc9c* Closing connection 0
```


## Nginx 开启 IPv6 SSL

IPv4 中监听 HTTP 和 HTTPS 的配置是 `listen 80;`、`listen 443 ssl;`

IPv6 分别是 `listen [::]:80;` 、`listen [::]:443 ssl;`


```
server {
        listen [::]:443 ssl http2;
        listen [::]:80;
        #填写绑定证书的域名
        server_name 6.wsmdn.dpdns.org;
        #证书文件名称
        ssl_certificate ssl/6.wsmdn.dpdns.org_bundle.crt;
        #私钥文件名称
        ssl_certificate_key ssl/6.wsmdn.dpdns.org.key;
        ssl_session_timeout 5m;
        #请按照以下协议配置
        ssl_protocols TLSv1.2 TLSv1.3;
        #请按照以下套件配置，配置加密套件，写法遵循 openssl 标准。
        ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:HIGH:!aNULL:!MD5:!RC4:!DHE;
        ssl_prefer_server_ciphers on;
}
```


重启后，检查端口监听，发现已经监听在 `:::443` 上，即监听在 IPv6 的 443 端口上。


```
# netstat -ntlp | grep nginx
tcp6       0      0 :::443                  :::*                    LISTEN      29391/nginx: master
tcp6       0      0 :::80                   :::*                    LISTEN      29391/nginx: master
```


测试访问效果。


```
curl https://6.wsmdn.dpdns.org -v
*   Trying 2402:4e00:1013:e500:0:9671:f018:4947...
* TCP_NODELAY set
* Connected to 6.wsmdn.dpdns.org (2402:4e00:1013:e500:0:9671:f018:4947) port 443 (#0)
* ALPN, offering h2
* ALPN, offering http/1.1
* successfully set certificate verify locations:
*   CAfile: /etc/ssl/cert.pem
  CApath: none
* TLSv1.2 (OUT), TLS handshake, Client hello (1):
* TLSv1.2 (IN), TLS handshake, Server hello (2):
* TLSv1.2 (IN), TLS handshake, Certificate (11):
* TLSv1.2 (IN), TLS handshake, Server key exchange (12):
* TLSv1.2 (IN), TLS handshake, Server finished (14):
* TLSv1.2 (OUT), TLS handshake, Client key exchange (16):
* TLSv1.2 (OUT), TLS change cipher, Change cipher spec (1):
* TLSv1.2 (OUT), TLS handshake, Finished (20):
* TLSv1.2 (IN), TLS change cipher, Change cipher spec (1):
* TLSv1.2 (IN), TLS handshake, Finished (20):
* SSL connection using TLSv1.2 / ECDHE-RSA-AES128-GCM-SHA256
* ALPN, server accepted to use h2
* Server certificate:
*  subject: CN=6.wsmdn.dpdns.org
*  start date: Jun 19 00:00:00 2022 GMT
*  expire date: Jun 19 23:59:59 2023 GMT
*  subjectAltName: host "6.wsmdn.dpdns.org" matched cert's "6.wsmdn.dpdns.org"
*  issuer: C=CN; O=TrustAsia Technologies, Inc.; CN=TrustAsia RSA DV TLS CA G2
*  SSL certificate verify ok.
* Using HTTP2, server supports multi-use
* Connection state changed (HTTP/2 confirmed)
* Copying HTTP/2 data in stream buffer to connection buffer after upgrade: len=0
* Using Stream ID: 1 (easy handle 0x7f90a8811c00)
> GET / HTTP/2
> Host: 6.wsmdn.dpdns.org
> User-Agent: curl/7.64.1
> Accept: */*
>
* Connection state changed (MAX_CONCURRENT_STREAMS == 128)!
< HTTP/2 200
< server: nginx
< date: Sun, 19 Jun 2022 13:21:31 GMT
< content-type: text/plain; charset=utf-8
< content-length: 38
< access-control-allow-origin: *
<
* Connection #0 to host 6.wsmdn.dpdns.org left intact
2408:824c:200::2b8b:336f:cc9c* Closing connection 0
```


> listen ssl 后一个指令是 http2，在开启 ssl 的同时可以把 http2 一并开启了。在上面的测试效果可以看到 \`HTTP/2\`

