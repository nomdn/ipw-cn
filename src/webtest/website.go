package webtest

import (
	"context"
	"strings"
	"time"

	lemon_ssrf "lemon-ipw/ssrf"
)

// CheckWebsite 网站检测（detail）。客户端由 main.go 经 SetHTTPClient 注入。
func CheckWebsite(url string, version string) (*WebsiteCheckDetail, error) {
	ctx := context.Background()
	var err error
	ctx, err = lemon_ssrf.ValidateOutboundTarget(ctx, url)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	resp, err := newProbeRequest(version, ctx).Get(url)

	// HTTPS 请求失败时 fallback 到 HTTP
	fallbackToHTTP := false
	if err != nil && strings.HasPrefix(url, "https://") {
		closeResponseBody(resp) // 丢弃本次失败的响应，避免连接/FD 悬挂（见 closeResponseBody）
		httpURL := strings.Replace(url, "https://", "http://", 1)
		startTime = time.Now()
		resp, err = newProbeRequest(version, ctx).Get(httpURL)
		fallbackToHTTP = true
	}

	if err != nil {
		closeResponseBody(resp)
		return nil, err
	}

	// 流式计数，不把响应体留在堆上（详见 readBodySize）。
	// 放在 endTime 之前：改造前 resty 在 Get() 内部就读完了整个响应体，
	// total_time 一直包含响应体下载耗时，见 speed.go 同处注释。
	bodySize, err := readBodySize(resp)
	if err != nil {
		return nil, err
	}
	endTime := time.Now()

	trace := resp.Request.TraceInfo()

	hostRecord := CleanHostRecord(trace.RemoteAddr)

	dnsLookupTime := trace.DNSLookup.Seconds() * 1000
	if dnsLookupTime == 0 {
		dnsLookupTime = measureDNSTime(url, version)
	}
	tcpConnectTime := trace.TCPConnTime.Seconds() * 1000
	httpConnectTime := trace.ConnTime.Seconds() * 1000
	firstByteTime := trace.ServerTime.Seconds() * 1000

	totalTime := float64(endTime.Sub(startTime).Milliseconds())
	var downloadSpeed float64
	if totalTime > 0 {
		downloadSpeed = float64(bodySize) / 1024.0 / (totalTime / 1000.0)
	}

	httpStatus := resp.StatusCode()
	httpsStatus := resp.StatusCode()
	if fallbackToHTTP {
		httpsStatus = 0
	}

	result := &WebsiteCheckDetail{
		HostRecord:       hostRecord,
		HTTPStatusCode:   httpStatus,
		HTTPSSStatusCode: httpsStatus,
		DNSLookupTime:    dnsLookupTime,
		TCPConnectTime:   tcpConnectTime,
		HTTPConnectTime:  httpConnectTime,
		FirstByteTime:    firstByteTime,
		TotalTime:        totalTime,
		PageSize:         bodySize,
		DownloadSpeed:    downloadSpeed,
		IsReachable:      true,
	}

	return result, nil
}
