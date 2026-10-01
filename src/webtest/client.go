package webtest

import (
	"context"
	"errors"
	"fmt"
	"io"

	"resty.dev/v3"
)

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

// newProbeRequest 构造拨测请求：开 trace（要 DNS/TCP/TLS 各段计时）、绑 context，
// 并关掉 resty 的自动解析（SetDoNotParseResponse）—— 这一条是关键，见 readBodySize。
//
// 必须走这个构造函数，不要在调用点手写请求链：漏掉 DoNotParseResponse 时
// resty 的 AutoParseResponseMiddleware 会替我们把整个响应体读进 bodyBytes
// （Content-Type 不是 JSON/XML 时它就 readAll），响应体上限和流式计数全部失效。
func newProbeRequest(version string, ctx context.Context) *resty.Request {
	return httpClient(version).R().
		EnableTrace().
		SetContext(ctx).
		SetDoNotParseResponse(true)
}

// readBodySize 流式读完响应体，返回**解压后**的字节数。
//
// 不用 resp.Bytes()：它只是把 resty 已经缓存的 bodyBytes 返回给调用方，而缓存动作
// 发生在 Get() 内部 —— resty 的 AutoParseResponseMiddleware 对没注册解码器的
// Content-Type（text/html、text/plain、octet-stream……差不多就是全部被测站点）
// 会直接 res.readAll()，等价于 io.ReadAll，整个响应体留在堆上。
// detail / speed / ssl 要的只是一个计数（page_size、download_speed），而响应体大小
// 完全由被测站点决定：一个压缩炸弹（Content-Encoding: gzip/zstd）能把几百 KB 的响应
// 放大到几百 MB，1c1g 节点单次请求即可 OOM。
// 配合 newProbeRequest 的 DoNotParseResponse，这里边读边丢，内存恒定，
// 与响应体大小无关。
//
// 兜底上限由客户端侧的 SetResponseBodyLimit（配置项 max-response-body）提供：
// 超限时 resty 的 limitReadCloser 会返回 ErrReadExceedsThresholdLimit，
// 这里转成可读的错误，而不是悄悄把截断的字节数当成页面大小报出去。
func readBodySize(resp *resty.Response) (int64, error) {
	if resp == nil || resp.Body == nil {
		return 0, nil
	}
	n, err := io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		if errors.Is(err, resty.ErrReadExceedsThresholdLimit) {
			return n, fmt.Errorf("response body exceeds the limit of %d bytes (max-response-body)", responseBodyLimit(resp))
		}
		return n, err
	}
	return n, nil
}

// responseBodyLimit 取本次请求实际生效的响应体上限（<=0 表示不限制）
func responseBodyLimit(resp *resty.Response) int64 {
	if resp == nil || resp.Request == nil {
		return 0
	}
	return resp.Request.ResponseBodyLimit
}

// closeResponseBody 尽力释放失败请求的响应体。
//
// 为什么需要：resty 除了传输错误，还会在**响应体解压失败**时提前返回
// （例如目标回了未注册的 Content-Encoding，client.go 里 wrapContentDecompresser 直接返回
// ErrContentDecompresserNotFound）。这时 Response.Body 已经挂上原始响应体但没有任何人关闭，
// 而我们随后就走 `if err != nil { return nil, err }` 把 Response 丢掉 ——
// 每个这样的请求漏一个连接/FD，且因为 DisableKeepAlives，要等 GC 终结器才回收。
// 目标站点只要固定回一个没注册的 Content-Encoding，就能一点点耗掉节点的 FD。
func closeResponseBody(resp *resty.Response) {
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}
