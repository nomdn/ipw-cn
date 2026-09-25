# [API] 查询域名的 WHOIS 信息

## 1. 接口描述

> 该接口用于学习域名基础信息,请在中国法律许可范围内使用.

接口地址： `https://cn-jiangsu.api-ipw.wsmdn.top/v1/whois/<domain>`（公开实例，见 1.1）

请求方法：`GET`

用途：查询域名注册信息——注册局记录 + 注册商数据，结构化返回，同时附带原始文本。

本项目自有的查询流程（不是简单地把库调用包一层）：

1. **先问 IANA**：用域名后缀查 `whois.iana.org`，拿到该后缀的注册局服务器（不猜、不硬编码后缀表）；
2. **只查注册局**：关闭 whois 库自带的转介链，避免被不可达的注册商服务器拖到超时；
3. **补查注册商**：若注册局响应里给出注册商转介服务器（如 `.com` 的 thick 注册局），**限时**再查一次，把结果追加进 `raw`；这一步失败不影响已到手的注册局数据；
4. **解析失败不算请求失败**：查询/解析的错误放进响应体的 `error` 字段，HTTP 状态码仍是 `200`。

其他特性：

- **结果缓存 5 分钟**。缓存键是**传入的原始字符串**，未做归一化：`QQ.com` 与 `qq.com` 各占一份。
- **IPv4 / IPv6 双栈解析** whois 服务器地址（Happy Eyeballs，v6 延迟 150ms 起步），某一栈不通时另一栈仍可用。
- **超时预算是分层的**：库单跳 6s（主路径 IANA + 注册局两跳），注册商转介补查另有 3s 总预算且失败即弃。
- **入参不做归一化**：不剥 `www.`、不吃 `host:port`。要查的是"注册的那个域名"本身。
  - `QQ.COM`、`qq.com` 都可以（大小写不敏感，结果里 `domain` 统一大写）；
  - `www.qq.com` 会去查这个**子域**，注册信息为空——不是接口坏了；
  - `qq.com:43` 不支持，返回 `whois: no whois server found for domain: qq.com:43`。
- 免鉴权规则同所有 `/v1/*` 端点：节点配了 `access-token` 才需要 `Authorization: Bearer`。

### 1.1 公开实例

本项目对外开放了一个江苏节点，可以直接用它把本接口调通，**无需令牌**：

| 方式 | 地址 |
| --- | --- |
| 直连节点（推荐） | `https://cn-jiangsu.api-ipw.wsmdn.top/v1/whois/<domain>` |
| 经中间件转发 | `https://middleware-1.api-ipw.wsmdn.top/v1/cn-jiangsu/whois/<domain>` |

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
# 用列表里的江苏节点查一个域名
curl https://middleware-1.api-ipw.wsmdn.top/v1/cn-jiangsu/whois/qq.com

# 换成别的节点：只改中间那一段标识
curl https://middleware-1.api-ipw.wsmdn.top/v1/<节点标识>/whois/qq.com
```

> 节点有上下线，**离线的节点调用会失败**（返回 `502`）——名单上标了在线状态，挑在线的用。

## 2. 输入参数

| 参数名称 | 类型 | 描述 |
| --- | --- | --- |
| `domain`(必选，路径) | String | 要查询的域名，例如 `/v1/whois/qq.com`。写注册域名本身，不要带子域 |

## 3. 输出参数

| 参数名称 | 类型 | 描述 |
| --- | --- | --- |
| `domain` | String | 归一化后的域名（**大写**，如 `QQ.COM`） |
| `status` | Array \| null | EPP 状态码列表，如 `["clientTransferProhibited"]`；**没拿到时为 `null`**（不是空数组） |
| `registrar.name` | String | 注册商名称 |
| `registrar.ianaId` | String | 注册商 IANA ID |
| `registrant` | Object | 注册人：`name`/`org`/`phone`/`email`/`province`/`contactUri` |
| `technical` | Object | 技术联系人，字段同上 |
| `abuseContact` | Object | 滥用投诉联系人，字段同上 |
| `dates.registration` | String | 注册时间（RFC3339，缺失为 `""`） |
| `dates.expiration` | String | 到期时间（RFC3339，缺失为 `""`） |
| `dates.lastChanged` | String | 最后变更时间（RFC3339，缺失为 `""`） |
| `nameservers` | Array | NS 列表（保留注册局原始大小写） |
| `whoisServer` | String | 最终实际查询的 whois 服务器；未找到后缀对应服务器时为空串 |
| `raw` | String | 注册局响应原文（含补查到的注册商内容），排查用 |
| `error` | String | 查询/解析错误信息；成功时为空串 `""` |

判定要点：

- **字段缺失一律返回空串，不会省略键**；所以判断"有没有数据"要看内容而不是键是否存在。
- `status` 为 `null` 与 `[]` 含义不同：前者是**没取到**，后者是取到了但确实没有状态。
- 隐私保护（GDPR、注册商隐私服务）会让 `registrant` / `technical` 的字段为空或提示串（例如要求到注册商网页填表才能看邮箱），这是上游数据的样子，不是解析失败。
- 想拿"能不能注册 / 什么时候释放"，以 `dates.expiration` 与 `status`（是否 `pendingDelete`、`redemptionPeriod`）为准。

## 4. 示例

### 4.1 cURL 查询域名注册信息

```bash
curl https://cn-jiangsu.api-ipw.wsmdn.top/v1/whois/qq.com

# 经公共中间件转发（见 1.1）：
# curl https://middleware-1.api-ipw.wsmdn.top/v1/cn-jiangsu/whois/qq.com
```

输出示例（真实响应，节选）：

```json
{
  "domain": "QQ.COM",
  "status": [
    "clientDeleteProhibited", "clientTransferProhibited", "clientUpdateProhibited",
    "serverDeleteProhibited", "serverTransferProhibited", "serverUpdateProhibited"
  ],
  "registrar": { "name": "MarkMonitor Information Technology (Shanghai) Co., Ltd.", "ianaId": "3838" },
  "registrant": {
    "name": "",
    "org": "深圳市腾讯计算机系统有限公司",
    "phone": "",
    "email": "select request email form at https://domains.markmonitor.com/whois/qq.com",
    "province": "",
    "contactUri": ""
  },
  "abuseContact": { "name": "", "org": "", "phone": "+1.2086851750", "email": "abusecomplaints@markmonitor.com" },
  "dates": {
    "registration": "1995-05-04T04:00:00Z",
    "expiration": "2034-07-27T02:09:19Z",
    "lastChanged": "2026-09-23T10:00:17Z"
  },
  "nameservers": ["ns1.qq.com", "ns2.qq.com", "ns3.qq.com", "ns4.qq.com"],
  "whoisServer": "whois.markmonitor.com",
  "error": ""
}
```

### 4.2 Python

```python
#!/usr/bin/python3

import requests

queryDomain = 'qq.com'
r = requests.get('https://cn-jiangsu.api-ipw.wsmdn.top/v1/whois/' + queryDomain)
data = r.json()

if data.get('error'):
    print('查询出错:', data['error'])
else:
    print('Domain Name: %s' % data['domain'])
    print('Status: %s' % data.get('status'))
    print('Registrar: %s' % data['registrar']['name'])
    print('Registration Time: %s' % data['dates']['registration'])
    print('Expiration Time: %s' % data['dates']['expiration'])
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
	queryDomain := "qq.com"

	resp, err := http.Get("https://cn-jiangsu.api-ipw.wsmdn.top/v1/whois/" + queryDomain)
	if err != nil {
		fmt.Println("请检查网络")
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Domain    string   `json:"domain"`
		Status    []string `json:"status"`
		Nameservers []string `json:"nameservers"`
		Dates     struct {
			Registration string `json:"registration"`
			Expiration   string `json:"expiration"`
		} `json:"dates"`
		Registrar struct {
			Name string `json:"name"`
		} `json:"registrar"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		panic(err)
	}

	if result.Error != "" {
		fmt.Println("查询出错:", result.Error)
		return
	}

	fmt.Printf("Domain: %s\n", result.Domain)
	fmt.Printf("Registrar: %s\n", result.Registrar.Name)
	fmt.Printf("Expiration: %s\n", result.Dates.Expiration)
}
```

## 5. 错误码

| 现象 | 原因与处理 |
| --- | --- |
| `400 {"error":"Domain parameter is required"}` | 路径里没有域名 |
| `401` | 节点配了 `access-token` 而未携带正确的 `Authorization: Bearer <access-token>` |
| `500 {"error":"..."}` | 上游查询异常（网络不可达等），可稍后重试 |
| `200` 且 `error` 为 `whois: no whois server found for domain: xxx` | 该后缀**没有 whois 服务器**（国家后缀常见）或不支持。换用该后缀的注册局查询入口 |
| `200` 且各字段全空、`status` 为 `null` | 域名格式不合法（如 `not_a_domain`），IANA 查不到对应后缀；或传了子域（如 `www.qq.com`），因注册信息不在子域上故为空 |
| `200` 但 `registrant`/`technical` 为空或提示串 | 上游做了隐私保护；`raw` 里能看到原文措辞 |
| 改了域名的注册信息但查询结果是旧的 | 命中 5 分钟缓存；等待或用另一个节点查（缓存是**节点本地**的） |
