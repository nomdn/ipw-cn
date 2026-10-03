# Whois 域名查询指南

## 优势

- **实时 WHOIS**：基于权威源返回域名注册商、创建/到期/更新时间、域名状态（带解释 tooltip）、NS 等关键信息。
- **网站快照**：生成首屏网页截图，辅助核验页面实际呈现与跳转等配置是否生效。
- **历史存档**：展示网站历史，包含总快照量、首次/最近快照、年度分布并支持悬停查看月份详情。
- **直观排障**：一图还原前端渲染异常、资源加载失败、跨域/混合内容、字体错位等典型问题。
- **多网络视角**：不同区域与出口路径差异一目了然，便于排查链路与 CDN 命中问题。

## 域名生命周期概述

域名从注册到最终过期或删除，会经历多个不同的状态阶段。了解这些状态对于域名管理和网站运营至关重要。

### 域名生命周期流程图

![流程](/domain-lifecycle-cloudfabric.svg)
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
