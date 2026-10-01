package main

import (
	"log/slog"
	"math"
	"os"
	"runtime/debug"
	"strconv"
	"strings"

	"resty.dev/v3"
)

// 本文件集中放"资源上限"的处理：单次拨测的响应体上限、Go 运行时的软内存上限。
// 起因：1c1g 节点被单个请求打爆（响应体整块进堆，见 webtest.readBodySize 注释）。
//
// **本文件不读配置源**（不碰 os.Getenv / viper）：配置的读取口径统一在 main.go 的 readConfig，
// 这里只负责"拿到值之后怎么用" —— 字节后缀解析（parseByteSize）与应用到运行时（apply*）。

// defaultMaxResponseBody 单次拨测响应体的默认上限（32 MiB，解压后）。
//
// 取 32MB 的理由：detail/speed/ssl 都只需要一个字节数，正常网页远小于此值；
// 而节点 RAM 只有 GB 级，没有上限时一个几百 MB 的响应体就能吃掉全部内存。
// 超过上限不是截断上报，而是直接返回错误 —— 报一个"截断后的大小/速度"比报错更误导人。
const defaultMaxResponseBody int64 = 32 << 20

// zstdMaxWindow zstd 解码窗口上限（8 MiB）。
//
// klauspost/compress 默认允许 512MB 窗口、64GiB 解码量，默认并发 4。窗口是在解析帧头时
// 就按流里声明的值分配的，早于任何输出，所以 max-response-body 拦不住它 ——
// 一条声明超大窗口的恶意 zstd 流即可让节点瞬间吃掉几百 MB。
// HTTP 场景下正常服务端的窗口远小于此值（RFC 8878 同样建议流式场景限制窗口）。
const zstdMaxWindow = 8 << 20

// parseByteSize 解析字节数：支持裸数字与 K/M/G 后缀（KB/MB/GB、KiB/MiB/GiB 亦可），大小写不敏感。
// 例："33554432"、"32MB"、"8MiB"、"900M"。
func parseByteSize(s string) (int64, bool) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" {
		return 0, false
	}
	s = strings.TrimSuffix(s, "B") // 32MB -> 32M
	s = strings.TrimSuffix(s, "I") // 32Mi -> 32M
	if s == "" {
		return 0, false
	}
	mult := int64(1)
	switch s[len(s)-1] {
	case 'K':
		mult, s = 1<<10, s[:len(s)-1]
	case 'M':
		mult, s = 1<<20, s[:len(s)-1]
	case 'G':
		mult, s = 1<<30, s[:len(s)-1]
	}
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0, false
	}
	return n * mult, true
}

// applyResponseBodyLimit 把当前 MAX_RESPONSE_BODY 应用到出站客户端（启动与配置热更新都走这里）。
// SetResponseBodyLimit 内部有锁，可与在途请求并发调用。
func applyResponseBodyLimit() {
	for _, c := range []*resty.Client{V4Client, V6Client} {
		if c != nil {
			c.SetResponseBodyLimit(MAX_RESPONSE_BODY)
		}
	}
}

// applyMemoryLimit 设置 Go 运行时的软内存上限（等价于 GOMEMLIMIT，可用 debug.SetMemoryLimit 热改）。
//
// 取值口径：读全局 MEMORY_LIMIT（启动时由 resolveByteLimit 填好，配置热更新时由
// applyConfigMap 直接赋值）。**不要在这里再解析一次配置源** —— 那样配置热更新刚写进
// MEMORY_LIMIT 的值会被重新解析覆盖成旧值，PATCH 静默失效。
//
// MEMORY_LIMIT<=0 时回落到自动探测容器限额的 90%；仍探测不到就不设置（保持 Go 默认）。
// 目的：不设时 Go 只按 GOGC 比例回收，够不到"突增"；小内存机器上堆会一路涨到被内核 OOM kill，
// 中间没有减速带。设了软上限后，逼近限额时 GC 会提前加密。
// 注意软上限不是硬闸：必要时运行时仍会越过它，只是付出更多 GC 代价。
func applyMemoryLimit() {
	limit := MEMORY_LIMIT
	if limit <= 0 {
		if cg := detectCgroupMemoryLimit(); cg > 0 {
			limit = cg / 10 * 9
		}
	}
	if limit <= 0 {
		debug.SetMemoryLimit(math.MaxInt64)
		slog.Info("No memory limit configured or detected, runtime keeps default GC behavior")
		return
	}
	debug.SetMemoryLimit(limit)
	slog.Info("Runtime soft memory limit applied", "bytes", limit, "mb", limit>>20)
}

// detectCgroupMemoryLimit 读容器内存限额（字节）。不在容器里 / 未限额 / 读不到时返回 0。
// cgroup v2: /sys/fs/cgroup/memory.max（无限时值为 "max"）
// cgroup v1: /sys/fs/cgroup/memory/memory.limit_in_bytes（无限时是接近 int64 上限的大数）
func detectCgroupMemoryLimit() int64 {
	for _, p := range []string{
		"/sys/fs/cgroup/memory.max",
		"/sys/fs/cgroup/memory/memory.limit_in_bytes",
	} {
		raw, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		s := strings.TrimSpace(string(raw))
		if s == "" || s == "max" {
			return 0
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil || n <= 0 {
			continue
		}
		if n >= 1<<62 { // "无限制"的常见写法
			return 0
		}
		return n
	}
	return 0
}
