package ipdb

import (
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

func PullDatabase(ghproxy string) (error, error) {
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
	}

	tmpDir := "./tmp"
	os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		slog.Error("Failed to create tmp directory", "error", err)
		return err, err
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

	if failCount > 0 {
		slog.Error("Some downloads failed, keeping tmp for inspection", "failed", failCount)
		return fmt.Errorf("%d downloads failed", failCount), fmt.Errorf("%d downloads failed", failCount)
	}

	slog.Info("All downloads succeeded, copying to working directory...")
	for _, t := range tasks {
		src := filepath.Join(tmpDir, t.name)
		dst := "./" + t.name
		if err := copyFile(src, dst); err != nil {
			slog.Error("Failed to copy file", "file", t.name, "error", err)
			return err, err
		}
		if strings.HasSuffix(t.name, ".gz") {
			outName := strings.TrimSuffix(t.name, ".gz")
			slog.Info("Decompressing...", "file", t.name, "output", outName)
			if err := gunzipFile(dst, "./"+outName); err != nil {
				slog.Error("Failed to decompress", "file", t.name, "error", err)
				return err, err
			}
			os.Remove(dst)
		}
	}

	os.RemoveAll(tmpDir)
	slog.Info("Download completed successfully!")
	return nil, nil
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

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
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

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, gz); err != nil {
		return err
	}
	return out.Close()
}
