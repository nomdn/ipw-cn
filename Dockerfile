FROM golang:1.26-alpine AS build
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY webtest/ ./webtest/
COPY ipdb/ ./ipdb/
RUN go build -o main .

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