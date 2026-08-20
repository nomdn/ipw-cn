package webtest

import "resty.dev/v3"

// 出站 HTTP 客户端（定义与初始化仍在 main.go，启动时经 SetHTTPClient 注入）
var v4Client, v6Client *resty.Client

// SetHTTPClient 注入 v4 / v6 出站客户端（main.go 启动时调用）
func SetHTTPClient(v4, v6 *resty.Client) {
	v4Client = v4
	v6Client = v6
}

// httpClient 按版本返回出站客户端
func httpClient(version string) *resty.Client {
	if version == "v6" {
		return v6Client
	}
	return v4Client
}
