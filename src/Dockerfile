# 多架构构建：buildx 会注入 TARGETOS/TARGETARCH，构建阶段固定在构建机原生平台
# （--platform=$BUILDPLATFORM）交叉编译，目标架构无需 QEMU 模拟；未用 buildx 时
# TARGETOS/TARGETARCH 为空，Go 默认取宿主机平台，普通 docker build 仍然可用
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
# 版本信息：CI（docker.yml）注入，与 build_and_release.yml 的二进制产物同源；
# BUILD_TIME 每次构建都变，会使 go build 层缓存失效（go mod download 层不受影响），
# 未传时用兜底值，本地 docker build 也能识别出容器构建
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH}
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY webtest/ ./webtest/
COPY ipdb/ ./ipdb/
COPY ssrf/ ./ssrf/
# TARGETVARIANT 是 buildx 的 "v7"/"v6" 写法，Go 的 GOARM 只要数字；不映射时 GOARCH=arm 默认按 GOARM=6 编译
RUN set -eux; \
    case "$TARGETVARIANT" in \
      v7) export GOARM=7 ;; \
      v6) export GOARM=6 ;; \
    esac; \
    go build -ldflags "-X main.VERSION=${VERSION} -X main.COMMIT=${COMMIT} -X main.BUILD_TIME=${BUILD_TIME}" -o main .

# 运行阶段
FROM alpine:latest

# 添加必要的证书
RUN apk --no-cache add ca-certificates

# 创建非root用户
RUN adduser -D appuser
USER appuser

WORKDIR /home/appuser

# 从构建阶段复制二进制文件
COPY --from=build /app/main .

EXPOSE 8080
CMD ["./main"]
