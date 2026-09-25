# [API] 获取客户端公网 IP 及位置

## 1. 接口描述

> IP 库来源于网络,本项目不对结果的准确性负责,请在中国法律许可范围内使用.

接口地址： `https://cn-jiangsu.api-ipw.wsmdn.top/v1/location`（公开实例，见 1.1；换成你自己节点的域名同样可用）

请求方法：`GET`

用途：取**请求方自己**的公网 IP，并给出 IP 归属地。

- **不带参数**：用服务端看到的客户端地址（Go `gin` 的 `ClientIP()`），因此"查自己"只需这一个请求。
- **多源聚合，不合并**：返回 10 个 IP 库各自的结论，按源名分组，便于交叉比对；本项目不替你做"以哪个源为准"的判断。
- **只查本地库**：不依赖任何第三方在线查询接口（唯一例外是 `bilibili` 源，24 小时缓存）。
- **免鉴权**：节点未配 `access-token` 时公开可访问；配了则需带 `Authorization: Bearer <access-token>`。

> 本接口属于"业务探测"端点，请求会被节点计入统计并周期上报收集中心（不落库明细）。

### 1.1 公开实例

本项目对外开放了一个江苏节点，可以直接用它把本接口调通，**无需令牌**：

| 方式 | 地址 |
| --- | --- |
| 直连节点（推荐） | `https://cn-jiangsu.api-ipw.wsmdn.top/v1/location` |
| 经中间件转发 | **不适用**——中间件路径的最后一段（拨测目标）必须非空，而本接口没有目标参数 |

> 公共实例是**共享**的，调试与小流量接入没问题，请勿高频轮询、批量抓取，也不要拿它当线上服务的依赖。

### 1.2 还有别的节点 —— 拨测控制台

上面的江苏节点只是节点池里的一个。节点池的入口是拨测控制台 <https://boce.wsmdn.top>：
登录后进「可用节点」页，能看到**全部已启用节点**的节点标识、归属地、归属池（定位 / 拨测）、
协议栈与在线状态，并附上每个节点的调用前缀。

比自己搭一套省事的地方：

- **节点现成** —— 名单里的节点都是已经部署好的，不用自己找机器、装程序、配隧道。
- **入口现成** —— 统一走 HTTPS 转发，证书与反向代理不用自己操心。
- **一套规则调所有节点** —— 接口路径与返回结构完全一致，换节点只换地址。
- **池与栈标得很清楚** —— 定位类请求走定位池，拨测类走拨测池，两个池都有的节点两种都能接。
- **名单会自己更新** —— 新接入的节点自动出现在列表里，不需要改代码。

> 本接口（"查自己"）**只能直连节点**：中间件路径的最后一段必须是拨测目标，而"查自己"没有目标参数，
> 所以走中间件转发不适用。换别的节点时，把上例的 `cn-jiangsu.api-ipw.wsmdn.top`
> 换成该节点的接入域名即可（节点标识见控制台的「可用节点」页）。
>
> 节点有上下线，列表中会标出在线状态；离线节点调用会失败。

## 2. 输入参数

无。

## 3. 输出参数

顶层是「源名 → 该源结果」的字典，另有 `ip` 回显请求方地址：

| 字段 | 类型 | 描述 |
| --- | --- | --- |
| `ip` | String | 请求方公网 IP（本节点视角） |
| `ip2region` | Object | `country`/`administrative_area`/`city`（部分记录另带 `isp`） |
| `ip2location` | Object | `country`/`country_code`/`administrative_area`/`city`/`district`/`zipcode`/`latitude`/`longitude`/`timezone`/`isp`/`asn`/`usagetype` |
| `ip2location_asn` | Object | `asn`/`as` |
| `qqwry` | Object | `country`/`country_code`/`administrative_area`/`city`/`isp` |
| `maxmind_city` | Object | `country`/`country_code`/`administrative_area`/`city`/`latitude`/`longitude` |
| `maxmind_asn` | Object | `asn`/`org` |
| `dbip_city` | Object | `country`/`country_code`/`administrative_area`/`city`/`latitude`/`longitude` |
| `dbip_asn` | Object | `asn`/`org` |
| `geocn` | Object | `division_code`（中国行政区划码）/`administrative_area`/`city`/`isp`/`type` |
| `bilibili` | Object | `country`/`administrative_area`/`city`/`isp`/`latitude`/`longitude`（在线源） |

取值约定：

- 某源**未加载**时该键为字符串 `"not loaded"`；该源**查询出错**时为字符串 `"error: ..."`。
  单个源失败**不会**让整个请求失败——HTTP 状态码仍是 `200`。
- 库里没有的字段返回空串 `""`。IP2Location LITE 免费版缺失的字段（如 `isp`、`asn`）已统一过滤，不会出现 `"This parameter is unavailable"`。
- 经纬度类型各源不一致：`ip2location` 是**字符串**，`maxmind_*` / `dbip_*` 是**数字**；`bilibili` 也是字符串。
- ASN 带前缀与否各源不一致：`maxmind_asn` / `dbip_asn` 形如 `AS13335`，`ip2location_asn.asn` 是裸数字 `13335`。

## 4. 示例

### 4.1 cURL 获取本机公网 IP 及位置

```bash
curl https://cn-jiangsu.api-ipw.wsmdn.top/v1/location
```

输出示例（节选）：

```json
{
  "ip": "1.1.1.1",
  "ip2region": { "country": "Australia", "administrative_area": "Queensland", "city": "Brisbane" },
  "ip2location": {
    "country": "Australia", "country_code": "AU", "administrative_area": "Queensland",
    "city": "Brisbane", "zipcode": "4000", "timezone": "+10:00",
    "latitude": "-27.46754", "longitude": "153.02809"
  },
  "maxmind_asn": { "asn": "AS13335", "org": "Cloudflare, Inc." },
  "dbip_city": {
    "country": "澳大利亚", "country_code": "AU", "administrative_area": "New South Wales",
    "city": "Sydney", "latitude": -33.8688, "longitude": 151.209
  },
  "qqwry": { "country": "澳大利亚", "country_code": "AU", "administrative_area": "", "city": "", "isp": "APNIC" },
  "geocn": { "division_code": "0", "isp": "", "type": "" },
  "bilibili": { "country": "CLOUDFLARE.COM", "administrative_area": "CLOUDFLARE.COM", "city": "", "isp": "" }
}
```

### 4.2 Python

```python
#!/usr/bin/python3

import json
import requests

r = requests.get('https://cn-jiangsu.api-ipw.wsmdn.top/v1/location')
data = json.loads(r.text)

print(data['ip'])                                    # 本机公网 IP
print(data['ip2location']['country'],
      data['ip2location']['administrative_area'],
      data['ip2location']['city'])                   # 取某一个源
print(data['maxmind_asn']['org'])                    # 运营商 / 组织
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
	resp, err := http.Get("https://cn-jiangsu.api-ipw.wsmdn.top/v1/location")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// 各源字段不一致，逐源取用；此处只解出需要的部分
	var result map[string]json.RawMessage
	if err := json.Unmarshal(body, &result); err != nil {
		panic(err)
	}

	var ip string
	_ = json.Unmarshal(result["ip"], &ip)

	var loc struct {
		Country string `json:"country"`
		City    string `json:"city"`
	}
	_ = json.Unmarshal(result["ip2location"], &loc)

	fmt.Println(ip, loc.Country, loc.City)
}
```

## 5. 错误码

| 现象 | 说明 |
| --- | --- |
| `404 page not found` | 该节点 `ipdb` 已关闭：IP 库不加载时 `/v1/location`、`/v1/asn/:ip` 路由不注册 |
| `401` | 节点配了 `access-token` 而未携带正确的 `Authorization: Bearer <access-token>` |
| 全部源都是 `"error: ..."` / `"not loaded"` | IP 库仍在下载/加载，或本地库文件缺失。节点启动时自动拉取全套库（首次约 450MB），之后每 24h 更新一次 |
| `ip` 是内网地址（如 `10.x`） | 直连节点、未经代理时看到的就是内网地址，属预期 |
| `ip` 是代理/CDN 的地址 | 节点前置了 Nginx/CDN 但未配 `trusted-proxies`，`X-Forwarded-For` 不被信任。配置方式见《节点 API 与协议参考》 |
