package ipdb

import (
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"resty.dev/v3"
)

type downloadTask struct {
	url  string
	name string
}

// dbValidator 条件请求凭证（ETag / Last-Modified）：日更同步时只有真变了的文件
// 才会重新下载，没变的 304 直接跳过——原实现每 24h 无条件全量重拉 450MB。
// 凭证与「已解压到工作目录的最终文件」绑定，持久化在 ipdb-etags.json。
type dbValidator struct {
	ETag         string `json:"etag,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
}

const validatorsFile = "ipdb-etags.json"

// pullMu 串行化下载（后台首次补齐与每日同步理论上不重叠，但配置面手动触发
// 之类的新入口将来出现时，这里保证不会两个 PullDatabase 互踩同一个 tmp 目录）
var pullMu sync.Mutex

func loadValidators() map[string]dbValidator {
	m := make(map[string]dbValidator)
	data, err := os.ReadFile(validatorsFile)
	if err != nil {
		return m
	}
	_ = json.Unmarshal(data, &m)
	return m
}

func saveValidators(m map[string]dbValidator) {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		slog.Warn("failed to marshal db validators", "error", err)
		return
	}
	if err := os.WriteFile(validatorsFile, data, 0o644); err != nil {
		slog.Warn("failed to write db validators", "error", err)
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// finalFileName 计算下载产物的最终文件名（.gz / .zip 会被解掉，后缀大小写不敏感——
// IP2LOCATION 的资产名是大写 .ZIP）
func finalFileName(name string) string {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".gz") {
		return name[:len(name)-3]
	}
	if strings.HasSuffix(lower, ".zip") {
		return name[:len(name)-4]
	}
	return name
}

func PullDatabase(ghproxy string) error {
	pullMu.Lock()
	defer pullMu.Unlock()

	tasks := []downloadTask{
		{"https://cdn.jsdelivr.net/gh/ljxi/GeoCN@main/data/full.txt", "full.txt"},
		{"https://cdn.jsdelivr.net/gh/ljxi/GeoCN@main/data/short.txt", "short.txt"},
		{ghproxy + "https://raw.githubusercontent.com/lionsoul2014/ip2region/master/data/ip2region_v4.xdb", "ip2region_v4.xdb"},
		{ghproxy + "https://raw.githubusercontent.com/lionsoul2014/ip2region/master/data/ip2region_v6.xdb", "ip2region_v6.xdb"},
		{ghproxy + "https://raw.githubusercontent.com/nmgliangwei/qqwry.ipdb/main/qqwry.ipdb", "qqwry.ipdb"},
		{ghproxy + "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb", "GeoLite2-City.mmdb"},
		{ghproxy + "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-ASN.mmdb", "GeoLite2-ASN.mmdb"},
		{ghproxy + "https://github.com/ljxi/GeoCN/releases/latest/download/GeoCN.mmdb", "GeoCN.mmdb"},
		{ghproxy + "https://raw.githubusercontent.com/wp-statistics/DbIP-City-lite/master/dbip-city-lite.mmdb.gz", "dbip-city-lite.mmdb.gz"},
		{ghproxy + "https://github.com/jcjc-dev/mmdb-latest/releases/download/dbip-latest/dbip-asn-lite.mmdb", "dbip-asn-lite.mmdb"},
		{ghproxy + "https://github.com/nomdn/ip2location/releases/latest/download/IP2LOCATION-LITE-DB11.IPV6.BIN.ZIP", "IP2LOCATION-LITE-DB11.IPV6.BIN.ZIP"},
		{ghproxy + "https://github.com/nomdn/ip2location/releases/latest/download/IP2LOCATION-LITE-ASN.IPV6.BIN.zip", "IP2LOCATION-LITE-ASN.IPV6.BIN.zip"},
	}

	tmpDir := "./tmp"
	os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		slog.Error("Failed to create tmp directory", "error", err)
		return err
	}

	// 条件请求凭证：只在「最终文件还在本地」时才有意义——304 的语义是
	// 「服务器上和你手里这份一样」，本地文件没了就必须整份重下。
	validators := loadValidators()
	newValidators := make(map[string]dbValidator, len(tasks))
	for k, v := range validators {
		newValidators[k] = v
	}

	slog.Info("Starting parallel downloads...", "count", len(tasks))

	var wg sync.WaitGroup
	errs := make([]error, len(tasks))
	notModified := make([]bool, len(tasks))
	for i, t := range tasks {
		wg.Add(1)
		go func(i int, t downloadTask) {
			defer wg.Done()
			val := validators[t.name]
			if !fileExists("./" + finalFileName(t.name)) {
				val = dbValidator{}
			}
			nv, skip, err := downloadWithRetry(t, val)
			errs[i] = err
			notModified[i] = skip
			if err == nil {
				newValidators[t.name] = nv
			}
			// 失败的文件保留旧凭证：本地旧版本文件仍在，下次可继续走条件请求
		}(i, t)
	}
	wg.Wait()

	var failCount int
	for i, e := range errs {
		if e != nil {
			slog.Error("Download failed", "file", tasks[i].name, "error", e)
			failCount++
		}
	}

	slog.Info("Copying downloaded files to working directory...")
	for i, t := range tasks {
		if errs[i] != nil {
			// 下载失败（含传输中断留下的残缺文件）不覆盖现有数据库
			slog.Warn("Skipping failed download, keep existing file", "file", t.name)
			continue
		}
		if notModified[i] {
			// 304：本地文件就是最新版，无需覆盖也不必解压
			slog.Info("Unchanged since last sync, skipped", "file", t.name)
			continue
		}
		src := filepath.Join(tmpDir, t.name)
		if _, err := os.Stat(src); os.IsNotExist(err) {
			slog.Warn("Skipping missing file", "file", t.name)
			continue
		}
		dst := "./" + t.name
		if err := copyFile(src, dst); err != nil {
			slog.Error("Failed to copy file", "file", t.name, "error", err)
			continue
		}
		lower := strings.ToLower(t.name)
		// 后缀判断用小写比较：IP2LOCATION 的资产名是大写 .ZIP，原来 HasSuffix(".zip") 永远匹配不上，
		// 导致该数据库从不解压就被删除
		if strings.HasSuffix(lower, ".gz") {
			outName := t.name[:len(t.name)-3]
			slog.Info("Decompressing...", "file", t.name, "output", outName)
			if err := gunzipFile(dst, "./"+outName); err != nil {
				slog.Error("Failed to decompress", "file", t.name, "error", err)
			}
			os.Remove(dst)
		}
		if strings.HasSuffix(lower, ".zip") {
			outName := t.name[:len(t.name)-4]
			slog.Info("Unzipping...", "file", t.name, "output", outName)
			if err := unzipFile(dst, "./"+outName); err != nil {
				slog.Error("Failed to unzip", "file", t.name, "error", err)
			}
			os.Remove(dst)
		}
	}

	os.RemoveAll(tmpDir)
	saveValidators(newValidators)

	if failCount > 0 {
		slog.Error("Some downloads failed", "failed", failCount)
		return fmt.Errorf("%d downloads failed", failCount)
	}

	slog.Info("Download completed successfully!")
	return nil
}

// downloadWithRetry 带重试的条件下载。返回 (新凭证, 是否 304 跳过, 错误)。
//
// 超时 10 分钟：这是**整个请求**（含响应体下载）的超时，GeoLite2-City 这类几十 MB
// 的大文件需要持续 ~1Mbps 以上才能在 10 分钟内完成；原实现的 60s 在慢链路上
// 会让大文件 3 次重试全部超时失败。
func downloadWithRetry(t downloadTask, val dbValidator) (dbValidator, bool, error) {
	const maxRetries = 3
	client := resty.New().SetTimeout(10 * time.Minute).SetOutputDirectory("./tmp")
	defer client.Close()

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		slog.Info("Downloading...", "file", t.name, "attempt", attempt)

		req := client.R().SetOutputFileName(t.name).SetSaveResponse(true)
		if val.ETag != "" {
			req.SetHeader("If-None-Match", val.ETag)
		}
		if val.LastModified != "" {
			req.SetHeader("If-Modified-Since", val.LastModified)
		}

		resp, err := req.Get(t.url)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			slog.Warn("Download attempt failed", "file", t.name, "attempt", attempt, "error", err)
			// 传输中断会在输出文件里留下残缺内容，删掉避免被当作下载成功覆盖现有数据库
			os.Remove(filepath.Join("./tmp", t.name))
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		if resp.StatusCode() == http.StatusNotModified {
			slog.Info("Not modified since last sync", "file", t.name)
			return val, true, nil
		}
		if resp.IsError() {
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode())
			slog.Warn("Download attempt failed", "file", t.name, "attempt", attempt, "status", resp.StatusCode())
			os.Remove(filepath.Join("./tmp", t.name))
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		nv := dbValidator{
			ETag:         resp.Header().Get("ETag"),
			LastModified: resp.Header().Get("Last-Modified"),
		}
		slog.Info("Downloaded", "file", t.name)
		return nv, false, nil
	}
	return val, false, lastErr
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}

	if err := out.Sync(); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}

	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}

	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}

	return nil
}

func gunzipFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	gz, err := gzip.NewReader(in)
	if err != nil {
		return err
	}
	defer gz.Close()

	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, gz); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}

	if err := out.Sync(); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}

	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}

	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}

	return nil
}

// unzipFile 解压 zip 中与目标同名（或同后缀）的文件到 dst（dst 如 ./IP2LOCATION-LITE-DB11.IPV6.BIN）；
// 找不到匹配文件时回退解压第一个非目录文件。
func unzipFile(src, dst string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	target := filepath.Base(dst)
	var fallback *zip.File
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if fallback == nil {
			fallback = f
		}
		if f.Name == target || strings.EqualFold(f.Name, target) || strings.HasSuffix(strings.ToLower(f.Name), strings.ToLower(target)) {
			return extractZipEntry(f, dst)
		}
	}
	if fallback == nil {
		return fmt.Errorf("no file found in zip archive %s", src)
	}
	return extractZipEntry(fallback, dst)
}

func extractZipEntry(f *zip.File, dst string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, rc); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
