# 中间件部署

独立中间件（middleware-go）只负责**请求转发和 Key 注入**，不参与业务计算。前端将 `/middleware/*` 请求转发到中间件，由中间件转发到对应后端节点并注入 `Authorization: Bearer <key>`。

> [!IMPORTANT]
> 项目只提供配置示例，**使用前必须先复制为 `setting.json`**（GitHub Releases 中对应文件为 `middleware.setting.json.example`）：
>
> ```bash
> # 仓库内
> cp middleware-go/setting.json.example middleware-go/setting.json
>
> # 从 Release 下载后
> cp middleware.setting.json.example setting.json
> ```
>
> 复制后按需修改 `apiBaseUrls` / `apiKeys` / `cors` 等配置项。

## 方案一：Docker

`middleware-go/` 目录提供 `Dockerfile`：

```bash
cd middleware-go
docker build -t middleware-go .
docker run -d -p 8091:8091 \
  -v $(pwd)/setting.json:/app/setting.json \
  middleware-go
```

配置通过挂载 `setting.json` 提供（`apiBaseUrls` / `apiKeys` / `cors` 等）。

## 方案二：二进制

```bash
cd middleware-go
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o middleware-go-linux-amd64 .
./middleware-go-linux-amd64          # 运行（与 setting.json 同目录）
```

所有配置可用环境变量覆盖（`API_BASE_URLS` / `IP_LOCATION_APIS` / `CORS` / `APIKEYS` 等，数组/对象用 JSON 字符串），优先级：**环境变量 > setting.json > 默认值**。

需要守护运行时，参考 [快速入门](/guide/getting-started) 中的 systemd 配置（`WorkingDirectory` 指向 `middleware-go/setting.json` 所在目录）。
