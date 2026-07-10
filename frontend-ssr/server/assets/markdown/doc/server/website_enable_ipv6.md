# <span style="background-color: #b95442;color: white;font-size: 0.43em;border-radius: 5px;padding: 2px 5px;">转载</span> 网站开启 IPv6 的三种方式

-   [1.Nginx 如何开启 IPv6](#1-nginx-%E5%A6%82%E4%BD%95%E5%BC%80%E5%90%AF-ipv6)
    -   [前置条件](#%E5%89%8D%E7%BD%AE%E6%9D%A1%E4%BB%B6)
    -   [1.1 启动监听 IPv6 地址的 Nginx](#11-%E5%90%AF%E5%8A%A8%E7%9B%91%E5%90%AC-ipv6-%E5%9C%B0%E5%9D%80%E7%9A%84-nginx)
    -   [1.2 服务器验证 IPv6 访问](#12-%E6%9C%8D%E5%8A%A1%E5%99%A8%E9%AA%8C%E8%AF%81-ipv6-%E8%AE%BF%E9%97%AE)
    -   [1.3 安全组对 IPv6 来源地址授权访问](#13-%E5%AE%89%E5%85%A8%E7%BB%84%E5%AF%B9-ipv6-%E6%9D%A5%E6%BA%90%E5%9C%B0%E5%9D%80%E6%8E%88%E6%9D%83%E8%AE%BF%E9%97%AE)
    -   [1.4 本地电脑验证 IPv6 访问](#14-%E6%9C%AC%E5%9C%B0%E7%94%B5%E8%84%91%E9%AA%8C%E8%AF%81-ipv6-%E8%AE%BF%E9%97%AE)
    -   [1.5 域名添加 IPv6 地址解析](#15-%E5%9F%9F%E5%90%8D%E6%B7%BB%E5%8A%A0-ipv6-%E5%9C%B0%E5%9D%80%E8%A7%A3%E6%9E%90)
    -   [1.6 IPv6 网站开启验证](#16-ipv6-%E7%BD%91%E7%AB%99%E5%BC%80%E5%90%AF%E9%AA%8C%E8%AF%81)
-   [2\. K8S 如何开启 IPv6](#2-k8s-%E5%A6%82%E4%BD%95%E5%BC%80%E5%90%AF-ipv6)
    -   [2.1 创建 IPv6 NAT64 版本的 Ingress](#21-%E5%88%9B%E5%BB%BA-ipv6-nat64-%E7%89%88%E6%9C%AC%E7%9A%84-ingress)
    -   [2.2 验证 IPv6 网站是否开启](#22-%E9%AA%8C%E8%AF%81-ipv6-%E7%BD%91%E7%AB%99%E6%98%AF%E5%90%A6%E5%BC%80%E5%90%AF)
-   [3\. Istio 服务网格如何开启 IPv6](#3-istio-%E6%9C%8D%E5%8A%A1%E7%BD%91%E6%A0%BC%E5%A6%82%E4%BD%95%E5%BC%80%E5%90%AF-ipv6)
    -   [3.1 创建 IPv6 边缘代理网关](#31-%E5%88%9B%E5%BB%BA-ipv6-%E8%BE%B9%E7%BC%98%E4%BB%A3%E7%90%86%E7%BD%91%E5%85%B3)
    -   [3.2 创建 IPv6 Gateway](#32-%E5%88%9B%E5%BB%BA-ipv6-gateway)
    -   [3.3 创建 IPv6 Virtual Service](#33-%E5%88%9B%E5%BB%BA-ipv6-virtual-service)
    -   [3.4 验证 IPv6 网站是否开启](#34-%E9%AA%8C%E8%AF%81-ipv6-%E7%BD%91%E7%AB%99%E6%98%AF%E5%90%A6%E5%BC%80%E5%90%AF)

从传统二进制部署的 Nginx，到云原生部署的 K8S、Istio，分别介绍网站开启 IPv6 的三种方式。

## 1.Nginx 如何开启 IPv6

### 前置条件

-   [服务器已开启 IPv6](/server/tencent_cloud_cvm_ipv6.html)

### 1.1 启动监听 IPv6 地址的 Nginx

默认 Nginx 的配置文件（/etc/nginx/nginx.conf）已经开启 IPv6 访问，启动 Nginx。


```
    server {
        listen       80;
        listen       [::]:80;  ## 监听 IPv6 的 80 端口
        ...
    }
```


可以看到 Ngnix 同时监听在 IPv4 和 IPv6 地址上。


```
# netstat -ntlp | grep nginx
tcp        0      0 0.0.0.0:80              0.0.0.0:*               LISTEN      138362/nginx: maste
tcp6       0      0 :::80                   :::*                    LISTEN      138362/nginx: maste
```


### 1.2 服务器验证 IPv6 访问

在服务器上通过 curl 命令可以验证 IPv6 访问情况


```
# curl -g http://[2402:4e00:1013:e500:0:9671:f018:4947] -I
HTTP/1.1 200 OK
Server: nginx/1.20.1
Date: Sat, 04 Dec 2021 13:58:37 GMT
Content-Type: text/html
Content-Length: 4833
Last-Modified: Fri, 16 May 2014 15:12:48 GMT
Connection: keep-alive
ETag: "53762af0-12e1"
Accept-Ranges: bytes
```


### 1.3 安全组对 IPv6 来源地址授权访问

接下来在 \`安全组\` 中对 监听在 IPv6 地址上的 80 端口授权访问。

![安全组-IPv6](/doc/securityGroupEnableIPv6.png)

### 1.4 本地电脑验证 IPv6 访问

如果 [本地网络已开启 IPv6访问](/user/enable_ipv6.html)，接下来参照 [浏览器访问 IPv6 地址](/user/view_ipv6_adress_url.html)，在本地电脑通过浏览器可以验证 IPv6 访问。

### 1.5 域名添加 IPv6 地址解析

[给域名添加 IPv6 解析记录（AAAA）](/server/ipv6_domain_record.html)![IPv6 解析记录（AAAA)](/doc/ipv6_domain_resolve.png)

添加完后可以通过 [IPv6 在线域名解析工具](https://ipw.wsmdn.top/dns/) 对其解析验证。

> 同一个主机名可以同时解析 IPv4（A 记录） 和 IPv6（AAAA 记录)，本地浏览器一般 IPv6 优先访问。

### 1.6 IPv6 网站开启验证

通过 [网站IPv6开启验证工具](https://ipw.wsmdn.top/ipv6webcheck/) 来验证 IPv6 是否开启成功。

## 2\. K8S 如何开启 IPv6

在 K8S 中对 Web 服务来说，一般通过 Ingress 对用户提供流量访问，所以 K8S 中开启 IPv6，就是 Ingress 支持 IPv6。

> 此处以 腾讯云容器服务为例，介绍如何开启 IPv6。

参照文章 [腾讯云 cvm 开启 IPv6](/server/tencent_cloud_cvm_ipv6.html) ，提交 IPv6 内测申请。

### 2.1 创建 \`IPv6 NAT64\` 版本的 Ingress

创建 \`IPv6 NAT64\` 版本的 Ingress。

> IPv6 NAT64：用户与 \`IPv6 NAT64\` 的 LB 之间是 IPv6 访问，\`IPv6 NAT64\` 的 LB 与后端 Service、Pod 通信是 IPv4。 优点：业务程序无需改造网络，可支持 IPv6 网络访问。 缺点： 无法获取客户端真实 IPv6 地址。

![K8S 如何开启 IPv6](/doc/k8s-ipv6-ingress.png)

接下来参照上文，对该 Ingress 背后的负载均衡器添加对来源 IPv6 地址访问的安全组策略即可。

### 2.2 验证 IPv6 网站是否开启

验证方法同上，此处不再赘述。

## 3\. Istio 服务网格如何开启 IPv6

服务网格 istio，比 K8S 原生的 Ingress 管理流量更方便，如果你使用了服务网格，就不需要对 K8S 的 ingress 开启 IPv6 访问，直接在服务网格中开启 IPv6 访问。

### 3.1 创建 IPv6 边缘代理网关

创建边缘代理网关时，选择提前创建好 IPv6 LB。 ![创建 IPv6 边缘代理网关](/doc/istio-ingress-ipv6.png)

### 3.2 创建 IPv6 Gateway

创建 IPv6 Gateway 时，关联刚创建的 IPv6 边缘代理网关。 ![istio-gateway-ipv6](/doc/istio-gateway-ipv6.png)

如果开启 HTTPS 访问，则添加好证书即可。 ![istio-gateway-ipv6-https](/doc/istio-gateway-ipv6-https.png)

### 3.3 创建 IPv6 Virtual Service

创建 IPv6 Virtual Service，挂载上面创建的 IPv6 Gateway。

### 3.4 验证 IPv6 网站是否开启

验证方法同上，此处不再赘述。

IPv6 大势所趋，广大站长赶紧行动吧。
