package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// OTA 自更新：定期检查 GitHub Release，下载与当前平台匹配的新版二进制替换自身并重启。
// 默认关闭，需显式开启（环境变量 OTA 或 setting.json 的 ota）。
// 交接逻辑与后端 src/ota.go 保持一致：
// 预检 → 原子替换 → 优雅停机 → 拉起新进程 → 健康检查确认 → 老进程退出；失败回滚 .old。

const (
	otaReleaseAPI    = "https://api.github.com/repos/nomdn/ipw-cn/releases/latest"
	otaFirstDelay    = 5 * time.Minute // 启动后首次检查的延迟（错开启动高峰）
	otaCheckInterval = 6 * time.Hour   // 常规检查间隔
	otaMinSize       = 1 * 1024 * 1024 // 下载文件最小体积，防止拿到错误页/占位文件
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
	switch strings.ToLower(strings.TrimSpace(OTA)) {
	case "true", "1", "yes", "on":
		return true
	}
	return false
}

// initOTA 在后台启动 OTA 检查循环，未启用时直接返回
func initOTA(ghproxy string) {
	if !otaEnabled() {
		return
	}
	log.Printf("[ota] enabled, current_version=%s check_interval=%s", VERSION, otaCheckInterval)

	go func() {
		// 首次延迟，避免启动瞬间与节点建连/数据库等启动任务抢资源
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
			"middleware-go-linux-armv7",
			"middleware-go-linux-armv6",
			"middleware-go-linux-arm",
		}
	}
	return []string{fmt.Sprintf("middleware-go-%s-%s%s", runtime.GOOS, arch, suffix)}
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
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, otaReleaseAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "middleware-go-ota")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var rel otaRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("empty tag_name")
	}
	return &rel, nil
}

// findAssetURL 在 Release 资产中查找当前平台的下载链接，同时返回资产大小用于完整性校验
func findAssetURL(rel *otaRelease) (string, string, int64) {
	for _, want := range otaAssetNames() {
		for _, a := range rel.Assets {
			if a.Name == want {
				return a.BrowserDownloadURL, a.Name, a.Size
			}
		}
	}
	return "", "", 0
}

// checkOTAUpdate 执行一次检查：有新版本则下载替换并重启
func checkOTAUpdate(ghproxy string) {
	otaMu.Lock()
	defer otaMu.Unlock()
	otaLastRun = time.Now()

	rel, err := fetchLatestRelease()
	if err != nil {
		log.Printf("[ota] WARN check failed: %v", err)
		return
	}

	if compareVersion(rel.TagName, VERSION) <= 0 {
		return
	}

	// major 版本变化（如 3.x → 4.x）通常含破坏性变更，不自动更新，需人工升级
	if !majorSame(rel.TagName, VERSION) {
		log.Printf("[ota] skipped: major version upgrade requires manual action (latest=%s current=%s)", rel.TagName, VERSION)
		return
	}

	assetURL, assetName, assetSize := findAssetURL(rel)
	if assetURL == "" {
		log.Printf("[ota] WARN no matching asset for this platform (goos=%s goarch=%s)", runtime.GOOS, runtime.GOARCH)
		return
	}

	log.Printf("[ota] new version found: tag=%s current=%s asset=%s", rel.TagName, VERSION, assetName)

	exePath, err := os.Executable()
	if err != nil {
		log.Printf("[ota] ERROR cannot locate executable: %v", err)
		return
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		log.Printf("[ota] ERROR cannot resolve executable path: %v", err)
		return
	}

	// 下载到同目录（保证与目标同分区，rename 原子替换），
	// 临时文件使用远端资产名 + .tmp 后缀，避免与运行中的 exe 冲突
	tmpPath := filepath.Join(filepath.Dir(exePath), assetName+".tmp")
	if err := downloadOTA(ghproxy+assetURL, tmpPath, assetSize); err != nil {
		log.Printf("[ota] ERROR download failed: %v", err)
		os.Remove(tmpPath)
		return
	}

	// 预检：停机前先试运行新二进制（-v 打印版本即退出），确认文件可执行、架构匹配。
	// 损坏/被杀毒隔离的文件若直到交接时才发现，子进程会秒退导致服务中断
	if err := preflightBinary(tmpPath); err != nil {
		log.Printf("[ota] ERROR preflight failed, keep current version: %v", err)
		os.Remove(tmpPath)
		return
	}

	if err := replaceBinary(tmpPath, exePath); err != nil {
		log.Printf("[ota] ERROR replace failed: %v", err)
		return
	}

	log.Printf("[ota] binary replaced, restarting (tag=%s)", rel.TagName)
	restartSelf(exePath)
}

// preflightBinary 试运行新二进制（-v 自检：打印版本后立即退出，不读配置不占端口），
// 确认文件可执行、架构匹配。10s 超时防损坏文件卡死。
func preflightBinary(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Stdout/Stderr 为 nil 时输出直接丢弃
	return exec.CommandContext(ctx, path, "-v").Run()
}

// downloadOTA 下载二进制到临时文件，并做体积校验（Release 声明的资产大小精确比对 + 最小体积兜底）。
// 注意：GitHub Release 未提供签名/哈希清单，无法做内容级认证，体积校验只能防截断与错装文件。
func downloadOTA(url, dst string, expectSize int64) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil || closeErr != nil {
		os.Remove(dst) // 残缺文件不留
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}
	fi, err := os.Stat(dst)
	if err != nil {
		return err
	}
	if fi.Size() < otaMinSize {
		os.Remove(dst)
		return fmt.Errorf("downloaded file too small: %d bytes", fi.Size())
	}
	if expectSize > 0 && fi.Size() != expectSize {
		os.Remove(dst)
		return fmt.Errorf("downloaded size mismatch: got %d, want %d", fi.Size(), expectSize)
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
			log.Printf("[ota] WARN chmod failed: %v", err)
		}
	}
	return nil
}

// restartSelf 用新二进制重启进程：
//   - Unix：syscall.Exec 原地替换进程镜像（PID 不变，systemd/Docker 无感；Go 的 socket 带
//     CLOEXEC，exec 瞬间旧监听关闭、新进程正常重绑）
//   - Windows：运行中的 exe 只能重命名不能替换，采用优雅交接：
//     1) 优雅停机：停止接收新请求并等待在途请求完成，端口释放
//     2) 拉起新进程：端口已空闲，子进程可正常绑定
//     3) 轮询子进程健康检查通过后老进程才退出；失败回滚 .old 恢复服务
func restartSelf(exePath string) {
	if runtime.GOOS == "windows" {
		gracefulShutdown()

		cmd := exec.Command(exePath, os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()
		if err := cmd.Start(); err != nil {
			log.Printf("[ota] ERROR restart failed, waiting for supervisor: %v", err)
			os.Exit(1)
		}

		if waitChildReady(cmd, 10*time.Minute) {
			log.Printf("[ota] new process ready, old process exiting (new_pid=%d)", cmd.Process.Pid)
			os.Exit(0)
		}
		log.Printf("[ota] ERROR new process not ready (exited or timed out), rolling back (new_pid=%d)", cmd.Process.Pid)

		// 回滚：新版本起不来，用 .old 备份恢复服务，而不是陪它一起下线
		if rollbackToOld(exePath) {
			log.Printf("[ota] WARN rolled back to previous version, old process exiting")
			os.Exit(0)
		}
		os.Exit(1)
	}

	if err := syscall.Exec(exePath, os.Args, os.Environ()); err != nil {
		log.Printf("[ota] ERROR exec failed, wait for supervisor to restart: %v", err)
		os.Exit(1)
	}
}

// gracefulShutdown 优雅停止 HTTP 服务：停止接收新请求，等待在途请求完成（上限 30s）。
// 供 Windows OTA 在拉起新进程前调用，确保监听端口先释放、子进程能正常绑定。
func gracefulShutdown() {
	if fiberApp == nil {
		return
	}
	otaShutdownStarted.Store(true)
	log.Printf("[ota] graceful shutdown started, waiting for in-flight requests")
	if err := fiberApp.ShutdownWithTimeout(30 * time.Second); err != nil {
		log.Printf("[ota] WARN graceful shutdown: %v", err)
	}
}

// waitChildReady 轮询新进程的健康检查接口（GET /），确认其完成端口绑定并对外服务。
// 返回 true = 就绪；false = 子进程启动期间退出（立即判定失败）或超时未就绪。
// 判定失败的依据是"子进程退出"（进程死了立即返回），而不是固定超时——
// 10 分钟兜底只为覆盖慢启动场景。
func waitChildReady(cmd *exec.Cmd, timeout time.Duration) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	url := "http://127.0.0.1:" + PORT + "/"
	deadline := time.Now().Add(timeout)

	// 监听子进程退出：起不来直接失败，不用干等超时
	exited := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(exited)
	}()

	for {
		if resp, err := client.Get(url); err == nil {
			resp.Body.Close()
			return true
		}
		select {
		case <-exited:
			return false
		case <-time.After(300 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			return false
		}
	}
}

// rollbackToOld 用 .old 备份恢复服务：新版本起不来时，把旧二进制换回原位并重新拉起。
// 返回 true 表示旧版本已就绪。恢复失败时尽力把新二进制挪回原位后返回 false。
func rollbackToOld(exePath string) bool {
	oldPath := exePath + ".old"
	if _, err := os.Stat(oldPath); err != nil {
		log.Printf("[ota] ERROR rollback skipped: no .old backup: %v", err)
		return false
	}
	failed := exePath + ".failed"
	_ = os.Remove(failed)
	if err := os.Rename(exePath, failed); err != nil {
		log.Printf("[ota] ERROR rollback: cannot move failed binary: %v", err)
		return false
	}
	if err := os.Rename(oldPath, exePath); err != nil {
		log.Printf("[ota] ERROR rollback: cannot restore old binary: %v", err)
		_ = os.Rename(failed, exePath) // 尽力恢复现场
		return false
	}
	cmd := exec.Command(exePath, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		log.Printf("[ota] ERROR rollback: cannot start old binary: %v", err)
		return false
	}
	if waitChildReady(cmd, 10*time.Minute) {
		log.Printf("[ota] WARN rollback serving traffic on previous version (pid=%d)", cmd.Process.Pid)
		return true
	}
	log.Printf("[ota] ERROR rollback: old binary also failed to become ready")
	return false
}
