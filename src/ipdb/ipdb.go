package ipdb

import (
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
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

func PullDatabase(ghproxy string) error {
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

	slog.Info("Starting parallel downloads...", "count", len(tasks))

	var wg sync.WaitGroup
	errs := make([]error, len(tasks))
	for i, t := range tasks {
		wg.Add(1)
		go func(i int, t downloadTask) {
			defer wg.Done()
			errs[i] = downloadWithRetry(t.url, t.name)
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

	if failCount > 0 {
		slog.Error("Some downloads failed", "failed", failCount)
		return fmt.Errorf("%d downloads failed", failCount)
	}

	slog.Info("Download completed successfully!")
	return nil
}

func downloadWithRetry(url, name string) error {
	const maxRetries = 3
	client := resty.New().SetTimeout(60 * time.Second).SetOutputDirectory("./tmp")
	defer client.Close()

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		slog.Info("Downloading...", "file", name, "attempt", attempt)

		resp, err := client.R().SetOutputFileName(name).SetSaveResponse(true).Get(url)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			slog.Warn("Download attempt failed", "file", name, "attempt", attempt, "error", err)
			// 传输中断会在输出文件里留下残缺内容，删掉避免被当作下载成功覆盖现有数据库
			os.Remove(filepath.Join("./tmp", name))
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		if resp.IsError() {
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode())
			slog.Warn("Download attempt failed", "file", name, "attempt", attempt, "status", resp.StatusCode())
			os.Remove(filepath.Join("./tmp", name))
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		slog.Info("Downloaded", "file", name)
		return nil
	}
	return lastErr
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
