package webtest

import (
	"context"
	"net/http/httputil"
	"strings"
	"time"

	lemon_ssrf "lemon-ipw/ssrf"
)

// SpeedTest 网站测速（speed）。客户端由 main.go 经 SetHTTPClient 注入。
func SpeedTest(url string, version string) (*WebsiteSpeedTestResult, error) {
	ctx := context.Background()
	var err error
	ctx, err = lemon_ssrf.ValidateOutboundTarget(ctx, url)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	resp, err := newProbeRequest(version, ctx).Get(url)

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
	// 必须放在 endTime 之前：改造前 resty 是在 Get() 内部读完整个响应体的，
	// 所以 total_time 一直包含响应体下载耗时，download_speed 才是"字节数/总耗时"。
	// 读体挪到 Get() 之外后若不在此处计入，total_time 会只剩到首字节的时间，
	// download_speed 会算出几百 GB/s 这种离谱值。
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
	dumpBytes, _ := httputil.DumpResponse(resp.RawResponse, false)
	httpStatus := resp.StatusCode()
	httpsStatus := resp.StatusCode()
	if fallbackToHTTP {
		httpsStatus = 0
	}
	result := &WebsiteSpeedTestResult{
		Version:          version,
		Headers:          string(dumpBytes),
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
