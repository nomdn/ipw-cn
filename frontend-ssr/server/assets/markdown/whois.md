## WHOIS 域名查询能查到什么

输入域名即可完成 Whois 域名查询。本站通过 IANA 注册局 bootstrap 文件自动匹配域名对应的 WHOIS 服务器，实时向权威源查询，结果按字段展开：

- **注册信息**：域名、注册商、注册人 / 技术联系人 / abuse 联系人、DNS 服务器、WHOIS 服务器。
- **关键时间**：注册时间、到期时间、最后修改时间。判断域名是否临近续费、或刚被转手，主要看这三个字段。
- **域名状态**：`ok`、`clientTransferProhibited`、`pendingDelete`、`redemptionPeriod` 等，各状态的含义见下方生命周期表。
- **原始记录**：需要更完整的信息时，可从结果跳转到注册局官网查询。

### 查不到、或字段为空时

- 多数注册局对注册人信息做了隐私保护，这类字段会显示 `Redacted for Privacy`，并非查询失败。
- 部分国家与地区顶级域不开放公开 WHOIS，或限制查询频率，结果可能缺少部分字段。
- 新注册域名同步到 WHOIS 服务器存在延迟，短期内查不到属于正常现象。

WHOIS 查的是**域名**的注册信息，不是 IP 的归属；查 IP 请用 [IP 归属地查询](/location) 或 [ASN 查询](/asn)，查域名解析记录请用 [DNS 查询](/dns)。

## 域名生命周期概述

域名从注册到最终过期或删除，会经历多个不同的状态阶段。了解这些状态对于域名管理和网站运营至关重要。

### 域名生命周期流程图

![流程](/domain-lifecycle-openai.svg)
### 域名状态详解

| 状态 | 描述 | 特征/影响 |
|------|------|----------|
| **Available** | 域名未被注册，可在任意注册商购买 | 可立即注册 |
| **AddPeriod** | 刚注册成功的一段时间（通常 5 天） | 可撤销注册，注册商可能退回部分费用 |
| **ok** | 正常使用，无额外限制 | 已配置 NS 且可解析 |
| **inactive** | 未配置 NS 或被暂停解析 | 不进行 DNS 解析 |
| **pendingTransfer** | 正在注册商之间转移 | 转移完成后进入 transferPeriod |
| **transferPeriod** | 转移完成后的短暂状态（通常 5 天） | 期满回到 ok |
| **autoRenewPeriod** | 到期时注册局自动续一年，等待注册商扣费确认 | 支付成功回到 ok；未支付进入 redemptionPeriod |
| **renewPeriod** | 手动续费后的短暂状态（通常 5 天） | 期满回到 ok |
| **redemptionPeriod** | 逾期未续费进入赎回阶段（通常 30 天） | 可支付赎回费用恢复 |
| **pendingDelete** | 赎回期结束仍未处理，进入删除倒计时（一般 5 天） | 删除释放，状态回到 Available |
| **client/serverHold** | 客户侧/注册局侧暂停解析 | DNS 不解析，解除后恢复到之前状态 |
| **client/server\*Prohibited** | 限制操作（transfer/update/delete 三种限制的 client/server 版本） | 对应操作受限（禁止转移/更新/删除） |

### 实际案例：example.com 的生命历程

让我们以 example.com 为例，追踪一个域名的完整生命周期：

#### 时间线示例

```
2020年1月  注册成功 → AddPeriod → 配置 NS → ok
2021年中   发起注册商转移 → pendingTransfer → transferPeriod → ok
2022年     持续正常解析与内容更新，保持 ok 状态
2023年     到期 → autoRenewPeriod（未支付）→ redemptionPeriod → pendingDelete → Available
```

#### 详细状态变化过程

**第一阶段：注册与配置 (2020年1月)**

域名注册成功，进入 AddPeriod。配置 NS 记录，状态变为 ok。

**第二阶段：转移与正常运营 (2021年中-2022年)**

发起并完成注册商转移：pendingTransfer → transferPeriod → ok。持续正常解析与内容更新，保持 ok 状态。

**第三阶段：到期与删除 (2022-2023年)**

到期进入 autoRenewPeriod，未支付。进入 redemptionPeriod，仍未处理。进入 pendingDelete，最终删除释放为 Available。

## 相关工具与文档

- [DNS 查询](/dns)：查看域名当前的 A / AAAA / CNAME 解析记录
- [DNSSEC 验证](/dnssec)：检查域名是否启用 DNS 安全扩展
- [ASN 查询](/asn)：查 IP 段所属的自治系统
- [Whois 查询 API](/doc/api/whois/query)：在自己的程序里调用 WHOIS 能力
- [IPv6 检测方法与工具](/ipv6check)：域名解析与 IPv6 支持的分工
