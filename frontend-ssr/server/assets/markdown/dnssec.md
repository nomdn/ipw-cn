DNSSEC（Domain Name System Security Extensions，DNS 安全扩展）通过**数字签名**为 DNS 应答提供来源验证与数据完整性保护：解析器收到应答后，用域名公钥本地验证签名，确认记录确实来自该域名的权威 DNS、且在传输途中没有被篡改或劫持。它防的是 DNS 欺骗与中间人攻击，但不加密内容——机密性仍由 TLS 负责。

![DNSSEC 验证链](/dnssec-trustchain-openai.svg)

## 三类关键记录

| 记录 | 发布位置 | 作用 |
| --- | --- | --- |
| `DNSKEY` | 域名自身 | 域名公钥，分 KSK（签密钥集）与 ZSK（签记录集） |
| `RRSIG` | 域名自身 | 对每类记录集（RRset）的数字签名，随应答一并返回 |
| `DS` | **父区**（如 `.top`） | 子域 DNSKEY 的摘要，是把信任链"挂"到上级的凭证 |

## 信任链如何建立

验证从根开始逐级向下：根的信任锚 → 顶级域的 `DS` → 域名的 `DNSKEY`。解析器拿到 `RRSIG` 后，用对应的 `DNSKEY` 做 `RRSIG.Verify` 本地验签；`DNSKEY` 自身的真伪再由父区 `DS` 核对，如此递归到根——任何一环对不上，应答即判定为 BOGUS（伪造）。

**本站的验证路径**：查询 `DNSKEY` → 带 DO 位查询 A 记录取回 `RRSIG` → 本地 `RRSIG.Verify` 验签（对 TLD / 无 A 记录的域，退回用 DNSKEY 自验签）→ 核对 `DS`。因国内公共 DNS（DNSPod / doh.pub / AliDNS）即使带 DO 位也不返回 RRSIG，本站默认使用实测支持 DNSSEC 的境外解析器列表（`1.1.1.1` / `8.8.8.8` 等，IPv4/IPv6 双栈）。

## 查询结果字段说明

| 字段 | 含义 |
| --- | --- |
| `enabled` | 是否启用 DNSSEC（存在 RRSIG 或 DNSKEY） |
| `valid` | 签名是否验证通过 |
| `has_rrsig` / `has_dnskey` / `has_ds` | 是否查到对应记录 |
| `algorithm` / `key_tag` | 签名算法与密钥标签（如算法 13 = ECDSAP256SHA256） |
| `validation` | 人类可读的判定说明 |

## 常见"验证失败"原因

- 域名本身**未配置 DNSSEC**（最常见——先在注册商处确认是否已开启）；
- `DNSKEY` 与 `DS` 不匹配（密钥轮换后父区 `DS` 未同步更新）；
- 解析器不支持 DNSSEC（带 DO 位也不返回 RRSIG，本站已默认规避）；
- 域名处于 ClientHold / ServerHold 暂停状态。

[DNS 解析查询](/dns) | [Whois 域名查询](/whois) | [回到顶部](#dnssec-原理介绍) | [IPv6 检测方法与工具](/ipv6check)
