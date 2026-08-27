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
	resp, err := httpClient(version).R().EnableTrace().SetContext(ctx).Get(url)

	fallbackToHTTP := false
	if err != nil && strings.HasPrefix(url, "https://") {
		httpURL := strings.Replace(url, "https://", "http://", 1)
		startTime = time.Now()
		resp, err = httpClient(version).R().EnableTrace().SetContext(ctx).Get(httpURL)
		fallbackToHTTP = true
	}

	if err != nil {
		return nil, err
	}
	endTime := time.Now()

	body := resp.Bytes()
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
		downloadSpeed = float64(len(body)) / 1024.0 / (totalTime / 1000.0)
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
		PageSize:         int64(len(body)),
		DownloadSpeed:    downloadSpeed,
		IsReachable:      true,
	}

	return result, nil
}
