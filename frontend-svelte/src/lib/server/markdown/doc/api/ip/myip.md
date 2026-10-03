# [API] 获取客户端公网 IP 地址

## 1. 接口描述

> 该接口用于网络排障,请在中国法律许可范围内使用.

接口地址： `https://test.wsmdn.top/`（本项目公共实例，双栈；另有仅 IPv4 / 仅 IPv6 的实例，见下文）

请求方法：`GET`

用途：**原样**返回访问者的公网 IP——纯文本、无 JSON 包裹、无参数、无鉴权、不查任何 IP 库。

实现是本项目自带的两个边缘函数（任选其一部署，都只做这一件事）：

| 形态 | 代码 | 取 IP 的方式 |
| --- | --- | --- |
| Cloudflare Workers | `serverless/lemon-getip` | `CF-Connecting-IP` 请求头 |
| EdgeOne Edge Functions | `serverless/edgeone-getip` | `request.eo.clientIp` |

两者都返回 `content-type: text/plain` 并带 `Access-Control-Allow-Origin: *`，所以浏览器里可以直接跨域取、也可以 `curl`。

**按协议栈部署三份**，域名的解析记录决定拿到哪一族的地址：

| 用途 | DNS 记录 | 行为 |
| --- | --- | --- |
| 双栈 | A + AAAA | 返回客户端**实际使用**的那一族地址 |
| 仅 IPv4 | 只配 A | 强制走 v4 出口，返回 IPv4 地址 |
| 仅 IPv6 | 只配 AAAA | 纯 IPv6 网络下返回 IPv6 地址；本机没有 IPv6 出口时域名**解析不出来**（请求直接失败，而不是返回空） |

本项目公共实例即按此部署：`4.wsmdn.top`（仅 IPv4）、`6.wsmdn.top`（仅 IPv6）、`test.wsmdn.top`（双栈）。

> 想把"本机 IP"和"归属地"一次拿全，用同一套体系里的 `/v1/location`（见「获取客户端公网 IP 及位置」），
> 它的 `ip` 字段与这里返回的是同一个地址。江苏节点公开实例：
> `curl https://cn-jiangsu.api-ipw.wsmdn.top/v1/location`（无需令牌；该接口没有目标参数，
> 只能直连节点，见该页的「公开实例」）。

## 2. 输入参数

无。

## 3. 输出参数

| 参数名称 | 类型 | 描述 |
| --- | --- | --- |
| 无 | String | 响应体**本身就是**客户端公网 IP 文本，例如 `223.68.219.100` |

判定要点：

- 只有 IP 一个 token，没有换行、没有 JSON 包裹；末尾是否带 `\n` 取决于边缘平台，取值前先 `strip()`。
- 不查库 ⇒ 不会出现 `"not loaded"` 之类的占位串，也没有归属地信息。
- 边缘函数按平台提供的客户端地址取值，**不解析 `X-Forwarded-For`**：它本身就是直接面向公网的入口。若你在它前面又套了一层自建代理，拿到的是代理地址。

## 4. 示例

### 4.1 cURL 获取本机公网 IP

```bash
# 双栈（返回实际使用的栈）
curl https://test.wsmdn.top

# 只要 IPv4 / 只要 IPv6
curl https://4.wsmdn.top
curl https://6.wsmdn.top
```

输出示例：

```
223.68.219.100
```

### 4.2 Python

```python
#!/usr/bin/python3

import requests

r = requests.get('https://test.wsmdn.top')
clientIP = r.text.strip()

print(clientIP)
```

### 4.3 Golang

```go
package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func main() {
	resp, err := http.Get("https://test.wsmdn.top")
	if err != nil {
		fmt.Println("获取外网 IP 失败，请检查网络")
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	clientIP := strings.TrimSpace(string(body))

	fmt.Println(clientIP)
}
```

### 4.4 浏览器里直接取（CORS 已放开）

```js
const ip = await fetch('https://test.wsmdn.top').then(r => r.text())
console.log(ip.trim())
```

> 要判断"当前访问更偏向 IPv4 还是 IPv6"，可以同时取双栈域名与两个单栈域名做对比，
> 思路见「JS 检查网络是 IPv4 还是 IPv6」。

## 5. 错误码

| 现象 | 说明 |
| --- | --- |
| 域名解析失败 / 连接超时 | 用的是**仅 IPv6** 的域名，而本机没有 IPv6 出口。改用双栈或仅 IPv4 的域名 |
| 返回 `Unable to determine IP address` | 边缘平台没有提供客户端地址。常见于本地直连调试（`wrangler dev` / EdgeOne 本地调试）或自建前置代理未透传 |
| 返回的是代理地址 | 你在边缘函数前面又套了一层代理；该函数只认平台提供的客户端地址，不解析 `X-Forwarded-For` |
| 拿到私有地址（如 `192.168.x.x`） | 本地/内网调试环境，边缘平台看到的就是内网出口 |
