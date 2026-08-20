package webtest

import (
	"context"
	"crypto/x509"
	"fmt"
	"time"

	lemon_ssrf "lemon-ipw/ssrf"
)

// CheckSSL SSL 检查（ssl）。客户端由 main.go 经 SetHTTPClient 注入。
func CheckSSL(url string, version string) (*SSLCheckDetail, error) {
	ctx := context.Background()
	var err error
	ctx, err = lemon_ssrf.ValidateOutboundTarget(ctx, url)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	resp, err := httpClient(version).R().EnableTrace().SetContext(ctx).Get(url)
	if err != nil {
		return nil, err
	}
	endTime := time.Now()

	trace := resp.Request.TraceInfo()
	hostRecord := CleanHostRecord(trace.RemoteAddr)

	totalTime := float64(endTime.Sub(startTime).Milliseconds())
	body := resp.Bytes()
	var downloadSpeed float64
	if totalTime > 0 {
		downloadSpeed = float64(len(body)) / 1024.0 / (totalTime / 1000.0)
	}

	rawResp := resp.RawResponse
	var cert *x509.Certificate
	var remainingDays int
	var isExpired bool
	var issuerOrganization []string
	var issuerCommonName, subjectCommonName, domain string

	if rawResp.TLS != nil && len(rawResp.TLS.PeerCertificates) > 0 {
		cert = rawResp.TLS.PeerCertificates[0]
		now := time.Now()
		remainingDays = int(cert.NotAfter.Sub(now).Hours() / 24)
		isExpired = now.After(cert.NotAfter) || now.Before(cert.NotBefore)
		issuerOrganization = cert.Issuer.Organization
		issuerCommonName = cert.Issuer.CommonName
		subjectCommonName = cert.Subject.CommonName
		domain = CleanHostRecord(cert.Subject.CommonName)
	} else {
		return nil, fmt.Errorf("no SSL certificate found")
	}

	result := &SSLCheckDetail{
		CertValidityDays:   remainingDays,
		IsExpired:          isExpired,
		CertStartTime:      cert.NotBefore,
		CertEndTime:        cert.NotAfter,
		HTTPVersion:        resp.Proto(),
		HostRecord:         hostRecord,
		HTTPSSStatusCode:   resp.StatusCode(),
		TotalTime:          totalTime,
		DownloadSpeed:      downloadSpeed,
		Domain:             domain,
		IssuerOrganization: issuerOrganization,
		IssuerCommonName:   issuerCommonName,
		SubjectCommonName:  subjectCommonName,
		IsReachable:        true,
	}

	return result, nil
}
