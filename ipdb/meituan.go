// ipdb/meituan.go
package ipdb

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"sync"
	"time"

	"resty.dev/v3"
)

// ========== 配置常量 ==========
const (
	locateAPI = "https://apimobile.meituan.com/locate/v2/ip/loc"
	cityAPI   = "https://apimobile.meituan.com/group/v1/city/latlng/%s,%s?tag=0"

	cacheTTL       = 24 * time.Hour  // 缓存有效期
	requestTimeout = 5 * time.Second // 请求超时
)

// ========== 返回结果结构体 ==========
// MeituanResult 统一返回结果（仅包含用户需要的字段）
type MeituanResult struct {
	Country   string  `json:"country"`   // 国家
	Province  string  `json:"region"`    // 省份
	City      string  `json:"city"`      // 城市
	District  string  `json:"district"`  // 区县
	Detail    string  `json:"detail"`    // 详细地址/街道
	Latitude  float64 `json:"latitude"`  // 纬度
	Longitude float64 `json:"longitude"` // 经度
}

// ========== 内部结构体（匹配 API 响应） ==========
type locateResp struct {
	Data struct {
		Lng  float64 `json:"lng"`
		Lat  float64 `json:"lat"`
		IP   string  `json:"ip"`
		Rgeo *rgeo   `json:"rgeo,omitempty"`
	} `json:"data"`
}

type rgeo struct {
	Country  string `json:"country"`
	Province string `json:"province"`
	AdCode   string `json:"adcode"`
	City     string `json:"city"`
	District string `json:"district"`
}

type cityResp struct {
	Data struct {
		Country  string `json:"country"`
		Province string `json:"province"`
		City     string `json:"city"`
		District string `json:"district"`
		AreaName string `json:"areaName"`
		Detail   string `json:"detail"`
	} `json:"data"`
}

// ========== 全局状态 ==========
var (
	client   *resty.Client
	clientMu sync.Once
	cache    = &sync.Map{} // map[string]cacheEntry
)

type cacheEntry struct {
	result *MeituanResult
	expire time.Time
}

// ========== 核心接口：MeituanSearch ==========
// MeituanSearch 统一入口：输入 IP，返回地理位置信息
//
// 功能：
//   - 自动缓存（24 小时）
//   - 两级查询：定位 → 城市详情
//   - 降级策略：城市接口失败时用定位数据填充
//   - 空指针防护 + 手动 JSON 解析（兼容 text/html）
//
// 返回：
//   - *MeituanResult: 包含 country/province/city/district/detail/lat/lng
//   - error: 网络/解析/参数错误
func MeituanSearch(ip string) (*MeituanResult, error) {
	// 1. 初始化 Resty 客户端（单例）
	clientMu.Do(func() {
		client = resty.New().
			SetTimeout(requestTimeout).
			SetHeader("User-Agent", "IP-Tool/1.0").
			SetHeader("Accept", "application/json").
			SetRetryCount(2).
			SetRetryWaitTime(100 * time.Millisecond)
	})

	// 2. 参数校验
	if net.ParseIP(ip) == nil {
		return nil, fmt.Errorf("invalid ip: %s", ip)
	}

	// 3. 缓存检查
	if entry, ok := cache.Load(ip); ok {
		if e := entry.(cacheEntry); time.Now().Before(e.expire) {
			slog.Debug("MeituanSearch cache hit", "ip", ip)
			return e.result, nil
		}
		cache.Delete(ip) // 过期清理
	}

	// 4. 执行查询
	result, err := doSearch(ip)
	if err != nil {
		return nil, err
	}

	// 5. 写入缓存
	cache.Store(ip, cacheEntry{
		result: result,
		expire: time.Now().Add(cacheTTL),
	})

	return result, nil
}

// doSearch 内部实现：两级查询 + 结果整合
func doSearch(ip string) (*MeituanResult, error) {
	// ── 第一级：定位接口（获取经纬度 + adcode）──
	locate, err := fetchLocate(ip)
	if err != nil {
		return nil, fmt.Errorf("locate failed: %w", err)
	}

	// ── 第二级：城市接口（获取详细地址，可选降级）──
	var city *cityResp
	if locate.Data.Lat != 0 && locate.Data.Lng != 0 {
		city, _ = fetchCity(locate.Data.Lat, locate.Data.Lng)
		// ⚠️ 城市接口失败不中断，用定位数据兜底
	}

	// ── 整合结果（城市数据优先，定位数据兜底）──
	result := &MeituanResult{
		Latitude:  locate.Data.Lat,
		Longitude: locate.Data.Lng,
	}

	// 优先用城市接口数据（更详细）
	if city != nil {
		fillFromCity(result, &city.Data)
	}
	// 用定位数据填充空缺字段（降级）
	if locate.Data.Rgeo != nil {
		fillFromRgeo(result, locate.Data.Rgeo)
	}

	return result, nil
}

// fetchLocate 调用定位 API（兼容 text/html，使用 Resty v3 正确 API）
func fetchLocate(ip string) (*locateResp, error) {
	resp, err := client.R().
		SetQueryParam("rgeo", "true").
		SetQueryParam("ip", ip).
		Get(locateAPI)

	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode())
	}

	// 🔥 Resty v3 正确写法：使用 resp.Bytes()
	bodyBytes := resp.Bytes()
	if len(bodyBytes) == 0 {
		return nil, fmt.Errorf("empty response body")
	}

	var result locateResp
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		// 打印前 200 字节用于调试
		preview := string(bodyBytes[:min(200, len(bodyBytes))])
		slog.Warn("Meituan locate parse failed", "error", err, "body_preview", preview)
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// 基础校验
	if result.Data.IP == "" || result.Data.Lng == 0 || result.Data.Lat == 0 {
		return nil, fmt.Errorf("invalid response: missing required fields")
	}

	return &result, nil
}

// fetchCity 调用城市详情 API
func fetchCity(lat, lng float64) (*cityResp, error) {
	latStr := url.PathEscape(fmt.Sprintf("%.6f", lat))
	lngStr := url.PathEscape(fmt.Sprintf("%.6f", lng))
	cityURL := fmt.Sprintf(cityAPI, latStr, lngStr)

	resp, err := client.R().Get(cityURL)
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, nil // 降级
	}

	// 🔥 Resty v3 正确写法
	bodyBytes := resp.Bytes()
	if len(bodyBytes) == 0 {
		return nil, nil
	}

	var result cityResp
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, nil // 降级
	}

	if result.Data.Country == "" && result.Data.City == "" {
		return nil, nil
	}

	return &result, nil
}

// fillFromCity 用城市数据填充结果
func fillFromCity(r *MeituanResult, c *struct {
	Country  string `json:"country"`
	Province string `json:"province"`
	City     string `json:"city"`
	District string `json:"district"`
	AreaName string `json:"areaName"`
	Detail   string `json:"detail"`
}) {
	if c.Country != "" {
		r.Country = c.Country
	}
	if c.Province != "" {
		r.Province = c.Province
	}
	if c.City != "" {
		r.City = c.City
	}
	if c.District != "" {
		r.District = c.District
	}
	if c.Detail != "" {
		r.Detail = c.Detail
	}
}

// fillFromRgeo 用定位数据填充空缺字段（降级）
func fillFromRgeo(r *MeituanResult, g *rgeo) {
	if r.Country == "" && g.Country != "" {
		r.Country = g.Country
	}
	if r.Province == "" && g.Province != "" {
		r.Province = g.Province
	}
	if r.City == "" && g.City != "" {
		r.City = g.City
	}
	if r.District == "" && g.District != "" {
		r.District = g.District
	}
}

// 工具函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
