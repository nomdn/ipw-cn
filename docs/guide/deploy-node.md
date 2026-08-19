# 后端节点部署

后端节点提供全部检测 API（IP 归属地 / ASN / SSL / DNS / DNSSEC / Whois / TCPing / 测速 / 截图等），可在任意地区部署，组成去中心化测试网络。

## 方案一：EdgeOne

腾讯云 EdgeOne 支持 Go 边缘函数，点击下方按钮通过 **EdgeOne Makers** 一键部署 `edgeone/` 版本（无需本地构建）：

[![使用 EdgeOne Makers 部署](https://cdnstatic.tencentcs.com/edgeone/pages/deploy.svg)](https://console.cloud.tencent.com/edgeone/makers/new?repository-url=https%3A%2F%2Fgithub.com%2Fnomdn%2Fipw-cn&root-directory=edgeone)


也可以使用 EdgeOne CLI 手动部署：

```bash
cd edgeone
npx edgeone pages deploy -n ipw-cn -t $EDGEONE_API_TOKEN
```

`edgeone/` 版本的配置来源为 环境变量 + 远端配置（`REMOTE_CONFIG_URL`）。

## 方案二：Vercel

点击下方按钮一键导入仓库（部署目标为 `edgeone/cloud-functions` 目录，即 Go 边缘函数版本）：

[![Deploy with Vercel](https://vercel.com/button)](https://vercel.com/new/clone?repository-url=https%3A%2F%2Fgithub.com%2Fnomdn%2Fipw-cn&root-directory=edgeone%2Fcloud-functions)

> [!IMPORTANT]
> **Serverless 节点均为 IPv4-only**（EdgeOne、Vercel 等平台的出站网络只提供 IPv4）。部署后需要：
>
> 1. 给节点配置 `SINGLE_STACK=ipv4`（声明节点为 IPv4 单栈），跳过 IPv6 相关测试，避免无意义的错误日志与超时
> 2. 在前端配置 `config/index.ts` 中，**该节点只能填入 `TCPing.IPv4` / `SpeedTest.IPv4` 等 IPv4 分组**，不要放进 `DualStack` / `IPv6` 分组

## 方案三：Docker

仓库根目录提供 `Dockerfile`（多阶段构建，产物基于 Alpine）：

```bash
docker build -t lemon-ipw .
docker run -d -p 8080:8080 \
  -v $(pwd)/setting.json:/app/setting.json \
  lemon-ipw
```

配置通过挂载 `setting.json` 提供，也可用环境变量覆盖（`PORTS` / `CORS` / `ACCESS_TOKEN` 等）。

## 方案四：二进制

```bash
# 编译
go build -o lemonipw main.go

# 运行（与 setting.json 同目录）
./lemonipw
```

首次启动自动下载 IP 数据库（约 200MB），之后每 24 小时自动更新一次。需要守护运行时，参考 [快速入门](/guide/getting-started) 中的 systemd 配置。

## 配置

后端配置见 [快速入门 - 配置文件](/guide/getting-started#配置文件)：`setting.json` 或环境变量，支持远端配置（`REMOTE_CONFIG_URL`，优先级：远端 > 环境变量 > setting.json）。
