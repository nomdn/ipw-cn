# 快速开始

> 本章节将指导您以最低成本部署一个功能完整的测试站

## 尝试一下

我们的开发团队已经部署了一个功能完整的预览站：
[预览站点](https://ipw.wsmdn.top/)

## 前置准备

- 一台 Linux 服务器（需 root 权限 + systemd）
- 服务器上可访问 GitHub（下载 release 二进制）
- 无需本地编译环境，无需准备配置文件

## 开始部署

### 交互式一键部署（推荐）

```bash
# 获取安装脚本（任选其一）
git clone https://github.com/nomdn/ipw-cn && cd ipw-cn   # 或
curl -LO https://raw.githubusercontent.com/nomdn/ipw-cn/main/install.sh

# 执行安装（root 权限）
sudo bash install.sh
```

脚本会自动完成：检测架构 → 下载最新 release 二进制 → **交互式输入配置** → 生成并启动 systemd 守护进程。

**交互式配置项**（直接回车使用默认值）：

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| 安装目录 | `/opt/lemon-ipw` | 二进制与工作目录 |
| 监听端口 | `8080` | 后端 HTTP 端口 |
| access_token | 留空 | 留空 = 不启用鉴权 |
| DNS 服务器 | `119.28.28.28:53` | 主从逗号分隔（如 `119.28.28.28:53,223.5.5.5:53`） |
| DNSSEC 专用 DNS | 留空 | 留空 = 沿用 dns-server |
| IP 数据库 | `Y` | 首次启动下载约 200MB |
| CORS | 留空 | 逗号分隔允许来源 |
| 远端配置地址 | 留空 | `REMOTE_CONFIG_URL` |
| WS 通道接入 | `N` | 选 `y` 后输入 WS_URL / NODE_ID / NODE_KEY |
| 其他环境变量 | 无 | 每行一个 `K=V`，空行结束 |

**WS 通道接入规则**（选 `y` 时）：

- `WS_URL` 留空自动使用默认值 `wss://middleware-1.api-ipw.wsmdn.top`，可改填自己的中间件地址
- `NODE_ID` **强制自动生成 UUID**（无需输入）
- `NODE_KEY` **必填**：不加 key 禁止启用 WS；安装完成后脚本会提示把 `"<NODE_ID>": "<NODE_KEY>"` 加入中间件 `apiKeys`

### 验证安装

```bash
systemctl status lemon-ipw        # 服务状态（active running）
curl http://127.0.0.1:8080/       # 应返回 {"status":"ok"}
journalctl -u lemon-ipw -f        # 实时日志
```

### 修改配置

```bash
sudo systemctl edit --full lemon-ipw   # 编辑环境变量
sudo systemctl daemon-reload && sudo systemctl restart lemon-ipw
```

## 下一步

- 部署**独立中间件**（WS 通道 / 多节点转发）：见 [中间件部署](/guide/deploy-middleware)
- 部署**前端**（Cloudflare Workers / EdgeOne Makers 等）：见 [前端部署](/guide/deploy-frontend)
- 完整**配置说明**（后端 / 前端 / 中间件 / 远端配置）：见 [配置文件](/guide/config)
- 手动**编译部署**（源码构建三个组件）：见 [后端节点部署](/guide/deploy-node) / [中间件部署](/guide/deploy-middleware) / [前端部署](/guide/deploy-frontend)
