# [API] 查询指定 IP 地址的位置信息

## 1. 接口描述

> IP 库来源于网络,本项目不对结果的准确性负责,请在中国法律许可范围内使用.

接口地址： `https://cn-jiangsu.api-ipw.wsmdn.top/v1/location/<ip>`（公开实例，见 1.1）

请求方法：`GET`

用途：查询**任意 IP**（IPv4 或 IPv6）的归属地，返回与本项目其他查询同一套多源聚合结果。

- IP 写在**路径**里，不是查询串：`/v1/location/106.224.145.147`。IPv6 直接用标准写法（含 `:`）即可，无需 URL 编码以外的特殊处理。
- 与「获取客户端公网 IP 及位置」共用同一个处理器与同一份 IP 库，**区别只是"查谁"**：不带路径参数 = 查请求方自己，带 = 查指定地址。
- 出参结构与「获取客户端公网 IP 及位置」**完全一致**，见该页的字段表，此处不重复。
- 免鉴权规则同所有 `/v1/*` 端点：节点配了 `access-token` 才需要 `Authorization: Bearer`。
- 查 IP 归属地**不需要**目标是本项目节点、也不做连通性探测，只查库；因此查任何公网 IP 都是同样耗时。

### 1.1 公开实例

本项目对外开放了一个江苏节点，可以直接用它把本接口调通，**无需令牌**：

| 方式 | 地址 |
| --- | --- |
| 直连节点（推荐） | `https://cn-jiangsu.api-ipw.wsmdn.top/v1/location/<ip>` |
| 经中间件转发 | `https://middleware-1.api-ipw.wsmdn.top/v1/cn-jiangsu/location/<ip>` |

两种写法都是「节点 / 接口 / 目标」三段：直连时端点本身就是节点，经中间件时多一段节点标识。
中间件入口的 `/v1/` 前缀与 `/middleware/` 前缀等价，用哪个都行。

> 公共实例是**共享**的，调试与小流量接入没问题，请勿高频轮询、批量抓取，也不要拿它当线上服务的依赖。

### 1.2 还有别的节点 —— 拨测控制台

上面的江苏节点只是节点池里的一个。节点池的入口是拨测控制台 <https://boce.wsmdn.top>：
登录后进「可用节点」页，能看到**全部已启用节点**的节点标识、归属地、归属池（定位 / 拨测）、
协议栈与在线状态，每行都能一键复制调用前缀 —— 把上面那个中间件地址换成控制台给你的入口域名，
再接上 `/v1/<节点标识>/` 就是完整前缀，**换节点只动中间那一段标识**。

比自己搭一套省事的地方：

- **节点现成** —— 名单里的节点都是已经部署好的，不用自己找机器、装程序、配隧道。
- **入口现成** —— 统一走 HTTPS 转发，证书与反向代理不用自己操心。
- **一套地址调所有节点** —— 换节点只改标识，接口路径、参数、返回结构都不变。
- **池与栈标得很清楚** —— 定位类请求走定位池，拨测类走拨测池，两个池都有的节点两种都能接。
- **名单会自己更新** —— 新接入的节点自动出现在列表里，不需要改代码。

用它调本接口（可直接粘贴执行）：

```bash
# 用列表里的江苏节点查一个 IP
curl https://middleware-1.api-ipw.wsmdn.top/v1/cn-jiangsu/location/106.224.145.147

# 换成别的节点：只改中间那一段标识
curl https://middleware-1.api-ipw.wsmdn.top/v1/<节点标识>/location/106.224.145.147
```

> 节点有上下线，**离线的节点调用会失败**（返回 `502`）——名单上标了在线状态，挑在线的用。

## 2. 输入参数

| 参数名称 | 类型 | 描述 |
| --- | --- | --- |
| `ip`(必选，路径) | String | 要查询的 IP，IPv4 或 IPv6 均可，例如 `/v1/location/106.224.145.147` |

> 路径中的 IP 不要带方括号。IPv6 形如 `/v1/location/2400:3200::1`。

## 3. 输出参数

与「获取客户端公网 IP 及位置」相同：顶层为「源名 → 该源结果」的字典，`ip` 字段回显被查询的地址。取值约定（未加载、出错、空字段、经纬度类型）也一致。

**非法 IP 是"看起来成功"的**：IP 传错时 HTTP 仍返回 `200`，但每个源会各自报错，形如：

```json
{
  "ip": "not-an-ip",
  "ip2region": "error: parse ip not-an-ip: invalid ip address: not-an-ip",
  "maxmind_city": "error: ParseAddr(\"not-an-ip\"): unable to parse IP",
  "ip2location": { "country": "Invalid IP address.", "city": "Invalid IP address." },
  "ip2location_asn": { "asn": "Invalid IP address.", "as": "Invalid IP address." }
}
```

所以**判断成功与否要看字段内容，不能只看状态码**：字段是对象才有结论；是 `"error: ..."` 字符串或 `"Invalid IP address."` 就是没查到。建议调用前先自行校验 IP 格式。

## 4. 示例

### 4.1 cURL 查询指定 IP 的位置

```bash
curl https://cn-jiangsu.api-ipw.wsmdn.top/v1/location/106.224.145.147

# IPv6 同样直接拼：curl https://cn-jiangsu.api-ipw.wsmdn.top/v1/location/2400:3200::1

# 经公共中间件转发（见 1.1）：
# curl https://middleware-1.api-ipw.wsmdn.top/v1/cn-jiangsu/location/106.224.145.147
```

输出示例（节选，真实响应）：

```json
{
  "ip": "106.224.145.147",
  "ip2region": { "country": "中国", "administrative_area": "江西省", "city": "南昌市", "isp": "电信" },
  "ip2location": {
    "country": "China", "country_code": "CN", "administrative_area": "Jiangxi",
    "city": "Ji'an", "zipcode": "343000", "timezone": "+08:00",
    "latitude": "27.11716", "longitude": "114.97927"
  },
  "qqwry": { "country": "中国", "country_code": "CN", "administrative_area": "江西", "city": "吉安", "isp": "电信" },
  "geocn": { "country": "中国", "administrative_area": "江西省", "city": "南昌市", "division_code": "360100", "isp": "电信" },
  "maxmind_asn": { "asn": "AS4134", "org": "Chinanet" }
}
```

> 上面同一 IP 出现了**南昌 / 吉安**两种结论：这就是"多源并列、不做合并"的含义——
> 各库口径与更新批次不同，写法也不统一（`江西省` / `Jiangxi`、`南昌市` / `Nanchang`）。
> 想要单一结论请自行选源或做交叉投票。

### 4.2 Python

```python
#!/usr/bin/python3

import requests

queryIP = '106.224.145.147'
r = requests.get('https://cn-jiangsu.api-ipw.wsmdn.top/v1/location/' + queryIP)
data = r.json()

qqwry = data.get('qqwry', {})
if isinstance(qqwry, dict):
    print(qqwry.get('country'), qqwry.get('administrative_area'),
          qqwry.get('city'), qqwry.get('isp'))
else:
    print('该源不可用:', qqwry)
```

### 4.3 Golang

```go
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	queryIP := "106.224.145.147"

	resp, err := http.Get("https://cn-jiangsu.api-ipw.wsmdn.top/v1/location/" + queryIP)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result map[string]json.RawMessage
	if err := json.Unmarshal(body, &result); err != nil {
		panic(err)
	}

	var loc struct {
		Country string `json:"country"`
		City    string `json:"city"`
		Isp     string `json:"isp"`
	}
	if err := json.Unmarshal(result["qqwry"], &loc); err != nil {
		fmt.Println("该源不可用，原始值：", string(result["qqwry"]))
		return
	}

	fmt.Printf("%s %s %s\n", loc.Country, loc.City, loc.Isp)
}
```

## 5. 错误码

| 现象 | 说明 |
| --- | --- |
| 字段是 `"Invalid IP address."`（`ip2location` 系列） | 路径里的 IP 不合法。该库对非法输入仍返回对象，值里写明原因 |
| 字段是 `"error: ... unable to parse IP"` | 同上，其它源对非法输入直接返回错误字符串 |
| `404 page not found` | 该节点 `ipdb` 已关闭，`/v1/location/:ip` 未注册 |
| `401` | 节点配了 `access-token` 但未携带正确的 `Authorization: Bearer <access-token>` |
| 结果与常识不符 | 免费 IP 库本身有偏差；本项目是**多源并列**，可交叉比对，不保证单一源正确 |
