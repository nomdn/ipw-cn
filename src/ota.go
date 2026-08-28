package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"resty.dev/v3"
)

// OTA 自更新：定期检查 GitHub Release，下载与当前平台匹配的新版二进制替换自身并重启。
// 默认关闭，需显式开启（环境变量 NODE_OTA 或 setting.json 的 node-ota）。

const (
	otaReleaseAPI    = "https://api.github.com/repos/nomdn/ipw-cn/releases/latest"
	otaFirstDelay    = 5 * time.Minute  // 启动后首次检查的延迟（避免与数据库下载冲突）
	otaCheckInterval = 6 * time.Hour    // 常规检查间隔
	otaMinSize       = 1 * 1024 * 1024  // 下载文件最小体积，防止拿到错误页/占位文件
)

var (
	otaMu      sync.Mutex
	otaLastRun time.Time
)

type otaRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

// otaEnabled 判断是否启用 OTA（"true" / "1" / "yes" / "on" 视为开启）
func otaEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(NODE_OTA)) {
	case "true", "1", "yes", "on":
		return true
	}
	return false
}

// initOTA 在后台启动 OTA 检查循环，未启用时直接返回
func initOTA(ghproxy string) {
	if !otaEnabled() {
		slog.Debug("OTA disabled")
		return
	}
	slog.Info("OTA enabled", "current_version", VERSION, "check_interval", otaCheckInterval.String())

	go func() {
		// 首次延迟，避免启动瞬间与数据库下载抢占带宽或触发重启
		time.Sleep(otaFirstDelay)
		for {
			checkOTAUpdate(ghproxy)
			jitter := time.Duration(rand.Int63n(int64(time.Hour)))
			time.Sleep(otaCheckInterval + jitter)
		}
	}()
}

// otaAssetNames 按优先级返回当前平台可能的发布资产名（与 build_and_release.yml 命名一致）
func otaAssetNames() []string {
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	arch := runtime.GOARCH
	// GOARCH=arm 时无法在运行时区分 GOARM（armv7/armv6），依次尝试
	if runtime.GOOS == "linux" && arch == "arm" {
		return []string{
			"lemonipw-linux-armv7",
			"lemonipw-linux-armv6",
			"lemonipw-linux-arm",
		}
	}
	return []string{fmt.Sprintf("lemonipw-%s-%s%s", runtime.GOOS, arch, suffix)}
}

// compareVersion 比较版本号：latest > current 返回正数，相等返回 0，否则返回负数；
// 非数字段按字典序比较。latest 为空（无法获取远端版本）时返回 0，不做更新。
func compareVersion(latest, current string) int {
	if strings.TrimSpace(latest) == "" {
		return 0
	}
	trim := func(s string) string {
		s = strings.TrimSpace(s)
		s = strings.TrimPrefix(s, "v")
		s = strings.TrimPrefix(s, "V")
		if i := strings.IndexAny(s, "-+"); i >= 0 { // 忽略预发布后缀
			s = s[:i]
		}
		return s
	}
	a := strings.Split(trim(latest), ".")
	b := strings.Split(trim(current), ".")
	for i := 0; i < len(a) || i < len(b); i++ {
		var x, y int
		var xe, ye error
		if i < len(a) {
			x, xe = strconv.Atoi(a[i])
		}
		if i < len(b) {
			y, ye = strconv.Atoi(b[i])
		}
		switch {
		case xe == nil && ye == nil:
			if x != y {
				return x - y
			}
		case xe == nil: // 只有当前版本非数字
			return 1
		case ye == nil:
			return -1
		default: // 两段都非数字，按字典序
			if a[i] != b[i] {
				if a[i] > b[i] {
					return 1
				}
				return -1
			}
		}
	}
	return 0
}

// majorSame 判断 latest 与 current 的 major 段是否相同。
// 任一段无法解析（如本地构建 VERSION 为空）时返回 true（放行，不拦截更新）。
func majorSame(latest, current string) bool {
	parseMajor := func(s string) (int, bool) {
		s = strings.TrimSpace(s)
		s = strings.TrimPrefix(s, "v")
		s = strings.TrimPrefix(s, "V")
		if i := strings.IndexAny(s, "-+."); i >= 0 {
			s = s[:i]
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0, false
		}
		return n, true
	}
	lm, lok := parseMajor(latest)
	cm, cok := parseMajor(current)
	if !lok || !cok {
		return true
	}
	return lm == cm
}

// fetchLatestRelease 查询 GitHub 最新 Release 信息
func fetchLatestRelease() (*otaRelease, error) {
	client := resty.New().SetTimeout(30 * time.Second)
	defer client.Close()

	resp, err := client.R().
		SetHeader("Accept", "application/vnd.github+json").
		SetHeader("User-Agent", "lemon-ipw-ota").
		Get(otaReleaseAPI)
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode())
	}
	var rel otaRelease
	if err := json.Unmarshal(resp.Bytes(), &rel); err != nil {
		return nil, err
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("empty tag_name")
	}
	return &rel, nil
}

// findAssetURL 在 Release 资产中查找当前平台的下载链接
func findAssetURL(rel *otaRelease) (string, string) {
	for _, want := range otaAssetNames() {
		for _, a := range rel.Assets {
			if a.Name == want {
				return a.BrowserDownloadURL, a.Name
			}
		}
	}
	return "", ""
}

// checkOTAUpdate 执行一次检查：有新版本则下载替换并重启
func checkOTAUpdate(ghproxy string) {
	otaMu.Lock()
	defer otaMu.Unlock()
	otaLastRun = time.Now()

	rel, err := fetchLatestRelease()
	if err != nil {
		slog.Warn("OTA check failed", "error", err)
		return
	}

	cmp := compareVersion(rel.TagName, VERSION)
	if cmp <= 0 {
		slog.Debug("OTA already up to date", "latest", rel.TagName, "current", VERSION)
		return
	}

	// major 版本变化（如 3.x → 4.x）通常含破坏性变更，不自动更新，需人工升级
	if !majorSame(rel.TagName, VERSION) {
		slog.Warn("OTA skipped: major version upgrade requires manual action",
			"latest", rel.TagName, "current", VERSION)
		return
	}

	assetURL, assetName := findAssetURL(rel)
	if assetURL == "" {
		slog.Warn("OTA no matching asset for this platform",
			"tag", rel.TagName, "goos", runtime.GOOS, "goarch", runtime.GOARCH)
		return
	}

	slog.Info("OTA new version found", "tag", rel.TagName, "current", VERSION, "asset", assetName)

	exePath, err := os.Executable()
	if err != nil {
		slog.Error("OTA cannot locate executable", "error", err)
		return
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		slog.Error("OTA cannot resolve executable path", "error", err)
		return
	}

	// 下载到同目录（保证与目标同分区，rename 原子替换），
	// 临时文件使用远端资产名（如 lemonipw-linux-amd64）+ .tmp 后缀，避免与运行中的 exe 冲突
	tmpPath := filepath.Join(filepath.Dir(exePath), assetName+".tmp")
	if err := downloadOTA(ghproxy+assetURL, tmpPath); err != nil {
		slog.Error("OTA download failed", "error", err)
		os.Remove(tmpPath)
		return
	}

	if err := replaceBinary(tmpPath, exePath); err != nil {
		slog.Error("OTA replace failed", "error", err)
		return
	}

	slog.Info("OTA binary replaced, restarting", "tag", rel.TagName)
	restartSelf(exePath)
}

// downloadOTA 下载二进制到临时文件，并做最小体积校验
func downloadOTA(url, dst string) error {
	client := resty.New().SetTimeout(10 * time.Minute)
	defer client.Close()

	resp, err := client.R().SetOutputFileName(dst).SetSaveResponse(true).Get(url)
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("HTTP %d", resp.StatusCode())
	}
	fi, err := os.Stat(dst)
	if err != nil {
		return err
	}
	if fi.Size() < otaMinSize {
		return fmt.Errorf("downloaded file too small: %d bytes", fi.Size())
	}
	return nil
}

// replaceBinary 用新二进制替换当前可执行文件：
// 1) 当前二进制改名为 .old（Windows 下运行中的 exe 不能删除，但可以重命名）
// 2) 新文件 rename 到原位置；失败则回滚
func replaceBinary(tmpPath, exePath string) error {
	oldPath := exePath + ".old"
	_ = os.Remove(oldPath)

	if err := os.Rename(exePath, oldPath); err != nil {
		return fmt.Errorf("backup current binary: %w", err)
	}
	if err := os.Rename(tmpPath, exePath); err != nil {
		_ = os.Rename(oldPath, exePath) // 回滚
		return fmt.Errorf("install new binary: %w", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(exePath, 0o755); err != nil {
			slog.Warn("OTA chmod failed", "error", err)
		}
	}
	return nil
}

// restartSelf 用新二进制重启进程：
// - Unix：syscall.Exec 原地替换进程镜像（PID 不变，systemd/Docker 无感）
// - Windows：另起新进程后退出当前进程
func restartSelf(exePath string) {
	if runtime.GOOS == "windows" {
		cmd := exec.Command(exePath, os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()
		if err := cmd.Start(); err != nil {
			slog.Error("OTA restart failed, waiting for supervisor", "error", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if err := syscall.Exec(exePath, os.Args, os.Environ()); err != nil {
		slog.Error("OTA exec failed, wait for supervisor to restart", "error", err)
		os.Exit(1)
	}
}
