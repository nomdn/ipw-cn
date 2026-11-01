# Changelog

本文件记录面向用户/运维的版本变更。格式参照 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本号遵循语义化版本（SemVer）。

## [v4.3.0] - 2026-11-01

> 接入自家 LemonDB 全球库（定位 + ASN 双源）、归属地可视化地图、GeoCN 镜像链修复。

### ✨ 新功能

- **接入 LemonDB 作为定位与 ASN 源**：`/v1/location` 与 `/v1/asn` 全链路新增 `lemondb` / `lemondb_asn` 输出。LemonDB（nomdn/LemonDB 月度构建）为全球打底 + 中国省市县高精度层：中国段含省/市/区县 + 运营商 + 网络类型 + GB/T 2260 区划码，v6 部分段可到区县；坐标全库统一 WGS-84，`coord_system` 字段透传
- **ASN 解析优先级重排**：`resolveASN` 改为 lemondb_asn → maxmind_asn → dbip_asn → ip2location_asn（中国段组织名更细），whois 段与 ws 同步
- **LemonDB 自动下载**：`PullDatabase` 新增 `LEMON-LOCATION.mmdb` / `LEMON-ASN.mmdb` 任务，走 ghproxy 镜像链；启动即拉 + 每日 ETag 条件请求，节点中断后恢复可在一天内追平
- **归属地地图**：location 页新增地图（高德瓦片直连 + Leaflet + 免 key），多源打点直观对比各库定位差异与离散度；坐标按源静态标记或 `coord_system` 动态判断（LemonDB），经 `wgs84ToGcj02` 统一转 GCJ-02 与底图对齐
- **地图数据源补充**：DB-IP City、GeoCN（含区划政府驻地坐标打点）
- **ASN 页节点池**：改为「双栈 ∩ IPLocationAPI」节点交集，只展示两个接口都通的节点
- **前端 LemonDB 展示**：location 表格首行 + asn 页 LemonDB ASN 行；locate.md 补字段说明

### 🛠 修复

- **GeoCN 行政区划下载单点依赖**：jsdelivr 被墙时数据永久缺失，补镜像列表兜底
- **地图最大缩放瓦片空白**：高德栅格瓦片原生上限约 17 级，声明 `maxNativeZoom=17` 后更高缩放自动放大 17 级瓦片（轻度模糊但不再空白）
- **MaxMind/DB-IP 地图打点无效**：字段名与实际 JSON 不符（小写 `latitude`/`longitude`），修正后打点恢复
- 节点池恢复西安电信节点（cn-xian）

### ⚠️ 行为变化

- `/v1/location` 输出新增 `lemondb` 键（全源之首）；`/v1/asn` 与 ws 的 asn 分支新增 `lemondb_asn`
- ASN whois 段的解析库优先级变化（见上），中国 IP 的组织名可能由 GeoLite2 值变为 LemonDB 值
- ipdb 目录新增约 70MB 的 LemonDB 两库磁盘占用（首次启动后台自动拉取，不阻塞服务）

## [v4.2.0] - 2026-10-25

> DNSSEC 从不可用变为可用、请求级解析能力同步 EO 特供、全站文档图灯箱、SEO 与部署链路整体升级。

### ✨ 新功能

- **DNSSEC 验证重写**：默认解析器内置境外 DO 友好列表（1.1.1.1 / 8.8.8.8 等，IPv4/IPv6 双栈 14 地址）；新增 DNSKEY 自验签（TLD 与无 A 记录域名可验）；DS 与 A 并行查询；解析器不支持 DNSSEC 时给出配置提示
- **SSL 检查新增 SAN 字段**：返回证书全部主体别名（DNS + IP）
- **IndexNow 接入**：key 静态文件 + 提交端点（GET 全量 / POST 增量）+ 部署后自动向 Bing/Yandex 提交 sitemap
- **文档图灯箱**：全站文档图片点击放大（el-image-viewer），支持多图切换、滚轮缩放、旋转、Esc 关闭、路由切换自动关闭
- **高级选项输入规整**：DNS 服务器填纯域名自动补全为 `https://域名/dns-query`，缺 scheme/路径/端口的写法自动补齐
- **EO 特供获得请求级解析**：`dns=` / `dns4=` / `dns6=` / `resolve=` 全套参数与主线对齐，DialContext DNS 接管 + SSRF 预检同源
- **ipdb 后台初始化**：首次启动不再被 450MB 下载阻塞，服务降级运行、库就绪一个上线一个
- **DNSSEC 原理文档**：/dnssec 页新增原理介绍、验证链图（OpenAI 风）与字段说明

### 🛠 修复

- **DNSSEC「一直用不了」**：国内公共解析器带 DO 位也不返回 RRSIG（根因，实测 6 家）；TLD/无 A 记录域没有验签路径；DS 只在失败路径查询——三项全修
- **构建 EBUSY 静默失败**：运行中的服务锁住 `.output` 导致产物不更新，`.gitignore` 同步补齐 agent 工具目录
- **内置中间件**：h3 不解码路径 `%2F`，带协议头的拨测目标经兜底转发必然拼坏——已与 fiber 语义对齐
- **/ipv6 空壳页**：改为服务端 301（消除无 h1/无 title 的低质量页）
- **sitemap**：全站显式枚举 + 权重标注
- whois 摘除自定义解析后回归系统解析器；EO 同步

### ⚡ 优化（ipdb）

- 条件下载（ETag/Last-Modified）：日更不再全量重拉 450MB
- 单文件下载超时 60s → 10 分钟（大文件慢链路不再必然失败）
- bilibili 并行查询 + 负缓存 + singleflight（/v1/location 首查 292ms → 复查 1.7ms）
- 数据库加载器原子换指针：日更重载零服务真空窗

### ⚠️ 行为变化

- `/v1/whois` 不再接受 `dns=` 参数（静默忽略）
- `/ipv6` 改为服务端 301 到 `/location`
- DNSSEC 默认走境外解析器：网络屏蔽/劫持这些 UDP 53 时需配置 `dnssec-server`
- 部署/构建前需停掉 3000/4000 端口的服务进程（EBUSY 会静默失败，本次已验证此坑）
