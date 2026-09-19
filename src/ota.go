package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

// ==================== OTA 升级（收集中心下发，节点执行） ====================
//
// 控制台（ipw-boce）经 WS 下发 type=ota 指令（或 HTTP 回退 POST /v1/ota，同 config 的鉴权语义），
// 节点下载新二进制 → 校验 → 预检 → 原子替换 → 重启。下载源二选一：
//   - url 直发：直接下载该地址
//   - version + assetBase：节点按自身平台计算资产名 lemonipw-{goos}-{goarch}[.exe]
//     （GOARCH=arm 无法在运行时区分 GOARM，依次尝试 armv7/armv6/arm），拼出
//     {assetBase}/{version}/lemonipw-... 逐一尝试
//   - sha256 提供时强校验（hex，兼容 "sha256:" 前缀）
//
// 进度经 WS 回 type=ota_result {requestId, ok, stage, error}：
//   accepted → downloading → verifying → installing → restarting（ok=false 时 error 说明原因）。
// 进程重启后连接必然断开，"最终结果"无法经原连接回传——收集中心以节点重连注册上报的
// 新版本号判定成败（见 ipw-boce ota.go 的任务追踪）。
//
// 本地开关：node-ota=false（env NODE_OTA）时节点拒绝一切 OTA 指令并回传原因，
// 供只读文件系统 / 编排托管的部署使用（替换二进制不可行，升级走各自部署渠道）。
//
// 交接逻辑与 middleware-go/ota.go 保持一致：
// 预检 → 原子替换 → 优雅停机 → 拉起新进程 → 健康检查确认 → 老进程退出；失败回滚 .old。

const (
	otaMinSize         = 1 * 1024 * 1024  // 下载文件最小体积，防止拿到错误页/占位文件
	otaDownloadTimeout = 10 * time.Minute // 单次下载超时
)

// otaAssetBaseDefault 按版本下发且未指定 assetBase 时的默认发布地址
const otaAssetBaseDefault = "https://github.com/nomdn/ipw-cn/releases/download"

// parseOTASwitch 解析 OTA 开关的字面值——启动加载（ENV / setting.json）、远端下发、运行时 PATCH
// 三处共用，保证什么算关闭的口径只有一份。
//
// 语义是缺省允许、显式关闭（见本文件头）：true/1/yes/on 与空值都算开启，
// false/0/no/off 算关闭。known=false 表示写法认不出来——运行时 PATCH 据此回
// unknown 且不覆盖原值，启动期与远端下发则按缺省（允许）处理，不因一个手滑的
// 字面值把节点锁死。
func parseOTASwitch(raw string) (enabled, known bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "true", "1", "yes", "on":
		return true, true
	case "false", "0", "no", "off":
		return false, true
	}
	return true, false
}

// otaMu 单飞：同一时刻只允许一个 OTA 任务（重复下发直接拒绝）
var otaMu sync.Mutex

// otaRequest OTA 下发报文（WS ota 消息的 data 与 POST /v1/ota 的 body 同构）
type otaRequest struct {
	RequestID string `json:"requestId"`
	URL       string `json:"url"`       // 直发下载地址（与 version 二选一，url 优先）
	Version   string `json:"version"`   // 目标版本（release tag，如 v1.2.3 / 1.2.3 均可）
	AssetBase string `json:"assetBase"` // 发布资产基址（version 模式用；空 = otaAssetBaseDefault）
	SHA256    string `json:"sha256"`    // 可选，内容校验（hex64）
}

// otaProgress 阶段回报函数：WS 触发时回 ota_result，HTTP 触发时为 nil（只记日志）
type otaProgress func(ok bool, stage, errMsg string)

// ==================== WS 入口 ====================

// wsHandleOTA 处理收集中心下发的 OTA 指令（ws.go type=ota 调用）。
// 先同步回 accepted 表明节点认识该指令（老版本节点静默忽略，收集中心据此给出过旧提示），
// 再异步执行下载/替换/重启。
func wsHandleOTA(c *websocket.Conn, data []byte) {
	var req otaRequest
	if err := json.Unmarshal(data, &req); err != nil {
		otaReport(c, req.RequestID, false, "accepted", "bad payload: "+err.Error())
		return
	}
	// 本地开关（node-ota=false，只读容器等不可自更新部署）：明确拒绝并回传原因，
	// 收集中心据此立即判任务失败，而不是干等超时
	if !NODE_OTA {
		otaReport(c, req.RequestID, false, "accepted", "节点已禁用 OTA（node-ota=false）")
		return
	}
	otaReport(c, req.RequestID, true, "accepted", "")
	go otaRun(req, func(ok bool, stage, errMsg string) {
		otaReport(c, req.RequestID, ok, stage, errMsg)
	})
}

// otaReport 回一帧 ota_result（进度或失败原因）
func otaReport(c *websocket.Conn, requestID string, ok bool, stage, errMsg string) {
	if c == nil {
		return
	}
	wsSend(c, wsMsg{Type: "ota_result", Data: wsRaw(map[string]any{
		"requestId": requestID, "ok": ok, "stage": stage, "error": errMsg,
	})})
}

// ==================== HTTP 入口 ====================

// registerOTARoute 注册 POST /v1/ota（收集中心 HTTP 回退通道）。
// 鉴权与 /v1/config 一致：access-token 未配置时整个 HTTP OTA 管理面关闭（RCE 面不能裸奔）。
func registerOTARoute(r *gin.Engine) {
	g := r.Group("/v1/ota")
	if ACCESS_TOKEN == "" {
		slog.Warn("ota HTTP API disabled: access-token not set, use WS channel instead")
		g.POST("", func(c *gin.Context) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "OTA 接口已关闭：节点未配置 access-token，请通过 WS 通道下发",
			})
		})
		return
	}
	g.Use(tokenCheck())
	g.POST("", func(c *gin.Context) {
		if !NODE_OTA {
			c.JSON(http.StatusForbidden, gin.H{"error": "节点已禁用 OTA（node-ota=false）"})
			return
		}
		var req otaRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload: " + err.Error()})
			return
		}
		if err := otaValidate(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// 异步执行：下载可能耗时数分钟，HTTP 无进度通道，结果看节点日志与重连后的版本号
		go otaRun(req, nil)
		c.JSON(http.StatusAccepted, gin.H{"ok": true, "started": true})
	})
}

// otaValidate 校验下发参数（url 与 version 至少其一；sha256 合法时规整为小写 hex64）
func otaValidate(req *otaRequest) error {
	req.URL = strings.TrimSpace(req.URL)
	req.Version = strings.TrimSpace(req.Version)
	req.SHA256 = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(req.SHA256), "sha256:"))
	if req.URL == "" && req.Version == "" {
		return fmt.Errorf("url 与 version 至少填一项")
	}
	if req.URL != "" {
		if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
			return fmt.Errorf("url 必须是 http(s) 地址")
		}
	}
	if req.SHA256 != "" {
		if len(req.SHA256) != 64 {
			return fmt.Errorf("sha256 必须是 64 位 hex")
		}
		if _, err := hex.DecodeString(req.SHA256); err != nil {
			return fmt.Errorf("sha256 必须是合法 hex")
		}
		req.SHA256 = strings.ToLower(req.SHA256)
	}
	return nil
}

// ==================== 执行引擎 ====================

// otaRun 执行 OTA：解析下载源 → 下载校验 → 预检 → 替换 → 重启。
// 每个 ok=false 的阶段回报后终止；重启前的最后回报是 "restarting"（连接随后断开）。
func otaRun(req otaRequest, progress otaProgress) {
	report := func(ok bool, stage, errMsg string) {
		if errMsg == "" {
			slog.Info("[ota]", "stage", stage, "ok", ok)
		} else {
			slog.Info("[ota]", "stage", stage, "ok", ok, "msg", errMsg)
		}
		if progress != nil {
			progress(ok, stage, errMsg)
		}
	}

	if !otaMu.TryLock() {
		report(false, "accepted", "已有 OTA 任务在执行中，请稍后再试")
		return
	}
	defer otaMu.Unlock()

	if err := otaValidate(&req); err != nil {
		report(false, "accepted", err.Error())
		return
	}

	exePath, err := os.Executable()
	if err != nil {
		report(false, "accepted", "cannot locate executable: "+err.Error())
		return
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		report(false, "accepted", "cannot resolve executable path: "+err.Error())
		return
	}

	// 下载到与目标同分区（保证 rename 原子替换）
	tmpPath := filepath.Join(filepath.Dir(exePath), "ota-update.tmp")
	report(true, "downloading", "")
	if err := otaDownload(req, tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		report(false, "downloading", err.Error())
		return
	}

	// 预检：试运行新二进制（-v 打印版本即退出），损坏/架构不符的文件在停机前就拦下
	report(true, "verifying", "")
	if err := preflightBinary(tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		report(false, "verifying", err.Error())
		return
	}

	report(true, "installing", "")
	if err := replaceBinary(tmpPath, exePath); err != nil {
		report(false, "installing", err.Error())
		return
	}

	report(true, "restarting", "")
	// 给 WS 把 restarting 帧写出去留一点时间，然后交接（连接随进程结束断开）
	time.Sleep(300 * time.Millisecond)
	slog.Info("[ota] binary replaced, restarting")
	restartSelf(exePath)
}

// otaAssetNames 按优先级返回当前平台可能的发布资产名（与 build_and_release.yml 命名一致）
func otaAssetNames() []string {
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	// GOARCH=arm 时无法在运行时区分 GOARM（armv7/armv6），依次尝试
	if runtime.GOOS == "linux" && runtime.GOARCH == "arm" {
		return []string{"lemonipw-linux-armv7", "lemonipw-linux-armv6", "lemonipw-linux-arm"}
	}
	return []string{fmt.Sprintf("lemonipw-%s-%s%s", runtime.GOOS, runtime.GOARCH, suffix)}
}

// otaDownload 把下发参数解析成候选下载地址并下载到 dst（url 直发只有一个候选；
// version 模式按平台资产名生成多个候选逐一尝试）
func otaDownload(req otaRequest, dst string) error {
	var urls []string
	if req.URL != "" {
		urls = []string{req.URL}
	} else {
		base := strings.TrimSuffix(strings.TrimSpace(req.AssetBase), "/")
		if base == "" {
			base = otaAssetBaseDefault
		}
		tag := req.Version
		if !strings.HasPrefix(tag, "v") {
			tag = "v" + tag // 本仓库 release 一律 v 前缀（CI 由 v* 标签触发），宽容无前缀写法
		}
		for _, name := range otaAssetNames() {
			u := fmt.Sprintf("%s/%s/%s", base, tag, name)
			// 延续原版自更新的加速语义：GitHub 官方基址且节点配了 gh-proxy 时自动加前缀；
			// url 直发不改写（管理员指定的地址保持原样）
			if GH_PROXY != "" && strings.Contains(base, "github.com") {
				u = strings.TrimRight(GH_PROXY, "/") + "/" + u
			}
			urls = append(urls, u)
		}
	}

	var lastErr error
	for _, u := range urls {
		err := otaDownloadOne(u, dst, req.SHA256)
		if err == nil {
			return nil
		}
		slog.Warn("[ota] download candidate failed", "url", u, "error", err)
		lastErr = err
		_ = os.Remove(dst)
	}
	return lastErr
}

// otaDownloadOne 下载单个地址到 dst：体积下限 + 可选 sha256 强校验
func otaDownloadOne(url, dst, wantSHA string) error {
	client := &http.Client{Timeout: otaDownloadTimeout}
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
	hash := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(f, hash), resp.Body)
	closeErr := f.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(dst)
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
		_ = os.Remove(dst)
		return fmt.Errorf("downloaded file too small: %d bytes", fi.Size())
	}
	if wantSHA != "" {
		got := hex.EncodeToString(hash.Sum(nil))
		if !strings.EqualFold(got, wantSHA) {
			_ = os.Remove(dst)
			return fmt.Errorf("sha256 mismatch: got %s", got)
		}
	}
	// 下载文件默认不带执行位（os.Create 跟随 umask，通常 0644），而 preflightBinary
	// 在 replaceBinary 的 chmod 之前就要试运行它；Linux 下不显式加 +x 会 EACCES（Permission denied）。
	// 故此处下载落盘后即补执行位，确保 preflight 与最终 replace 都能 exec。
	if runtime.GOOS != "windows" {
		if err := os.Chmod(dst, 0o755); err != nil {
			_ = os.Remove(dst)
			return fmt.Errorf("chmod downloaded binary: %w", err)
		}
	}
	return nil
}

// preflightBinary 试运行新二进制（-v 自检：打印版本后立即退出，不读配置不占端口），
// 确认文件可执行、架构匹配。10s 超时防损坏文件卡死。
func preflightBinary(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, path, "-v").Run()
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
			slog.Warn("[ota] chmod failed", "error", err)
		}
	}
	return nil
}

// restartSelf 用新二进制重启进程：
//   - Unix：syscall.Exec 原地替换进程镜像（PID 不变，systemd/Docker 无感）
//   - Windows：运行中的 exe 只能重命名不能替换，采用优雅交接：
//     优雅停机（端口释放）→ 拉起新进程 → 健康检查通过后老进程退出；失败回滚 .old
func restartSelf(exePath string) {
	if runtime.GOOS == "windows" {
		gracefulShutdown()

		cmd := exec.Command(exePath, os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()
		if err := cmd.Start(); err != nil {
			slog.Error("[ota] restart failed, waiting for supervisor", "error", err)
			os.Exit(1)
		}

		if otaWaitChildReady(cmd, 10*time.Minute) {
			slog.Info("[ota] new process ready, old process exiting", "new_pid", cmd.Process.Pid)
			os.Exit(0)
		}
		slog.Error("[ota] new process not ready (exited or timed out), rolling back", "new_pid", cmd.Process.Pid)

		if otaRollbackToOld(exePath) {
			slog.Warn("[ota] rolled back to previous version, old process exiting")
			os.Exit(0)
		}
		os.Exit(1)
	}

	if err := syscall.Exec(exePath, os.Args, os.Environ()); err != nil {
		slog.Error("[ota] exec failed, wait for supervisor to restart", "error", err)
		os.Exit(1)
	}
}

// otaWaitChildReady 轮询新进程的健康检查接口（GET /），确认其完成端口绑定并对外服务。
// 返回 true = 就绪；false = 子进程启动期间退出（立即判定失败）或超时未就绪。
func otaWaitChildReady(cmd *exec.Cmd, timeout time.Duration) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	url := "http://127.0.0.1:" + PORTS + "/"
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

// otaRollbackToOld 用 .old 备份恢复服务：新版本起不来时，把旧二进制换回原位并重新拉起。
// 返回 true 表示旧版本已就绪。
func otaRollbackToOld(exePath string) bool {
	oldPath := exePath + ".old"
	if _, err := os.Stat(oldPath); err != nil {
		slog.Error("[ota] rollback skipped: no .old backup", "error", err)
		return false
	}
	failed := exePath + ".failed"
	_ = os.Remove(failed)
	if err := os.Rename(exePath, failed); err != nil {
		slog.Error("[ota] rollback: cannot move failed binary", "error", err)
		return false
	}
	if err := os.Rename(oldPath, exePath); err != nil {
		slog.Error("[ota] rollback: cannot restore old binary", "error", err)
		_ = os.Rename(failed, exePath) // 尽力恢复现场
		return false
	}
	cmd := exec.Command(exePath, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		slog.Error("[ota] rollback: cannot start old binary", "error", err)
		return false
	}
	if otaWaitChildReady(cmd, 10*time.Minute) {
		slog.Warn("[ota] rollback serving traffic on previous version", "pid", cmd.Process.Pid)
		return true
	}
	slog.Error("[ota] rollback: old binary also failed to become ready")
	return false
}
