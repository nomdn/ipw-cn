package main

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// literalString 是两条读取路径共用的「字面量 → 字符串」表，"布尔与字符串等价"就靠它。
func TestLiteralString(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"nil（未配置 / null）", nil, ""},
		{"bool false", false, "false"},
		{"bool true", true, "true"},
		{"字符串去首尾空白", "  no \n", "no"},
		{"JSON 数字（float64）", float64(443), "443"},
		{"int 0", 0, "0"},
	}
	for _, c := range cases {
		if got := literalString(c.in); got != c.want {
			t.Errorf("%s: literalString(%#v) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

// parseBoolSwitch 是四个开关唯一的字面值口径。
func TestParseBoolSwitch(t *testing.T) {
	for _, s := range []string{"true", "TRUE", " True ", "1", "yes", "ON", "enable", "enabled"} {
		if v, ok := parseBoolSwitch(s); !ok || !v {
			t.Errorf("parseBoolSwitch(%q) = (%v, %v), want (true, true)", s, v, ok)
		}
	}
	for _, s := range []string{"false", "FALSE", " False ", "0", "no", "off", "disable", "disabled"} {
		if v, ok := parseBoolSwitch(s); !ok || v {
			t.Errorf("parseBoolSwitch(%q) = (%v, %v), want (false, true)", s, v, ok)
		}
	}
	// 认不出来的一律 known=false —— 绝不静默当"开"
	for _, s := range []string{"", "   ", "banana", "tru", "2", "on!", "是", "否"} {
		if v, ok := parseBoolSwitch(s); ok || v {
			t.Errorf("parseBoolSwitch(%q) = (%v, %v), want (false, false)", s, v, ok)
		}
	}
}

// 启动路径：setting.json 里把开关写成 JSON 布尔 / 数字，读出来必须与字符串写法等价，
// 且不能被 configString 的"空值"判断误吞（false 变成"未配置"就等于静默变回默认的"开"）。
func TestConfigStringAcceptsJSONBoolAndNumber(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	viper.SetConfigType("json")
	if err := viper.ReadConfig(strings.NewReader(`{"ipdb": false, "access-log": 0, "port": 443, "cors": ""}`)); err != nil {
		t.Fatal(err)
	}

	if got := configString("ZZ_NO_IPDB", "ipdb", "true"); got != "false" {
		t.Errorf("configString(ipdb) = %q, want \"false\"", got)
	}
	if configBool("ZZ_NO_IPDB", "ipdb", true) {
		t.Error("configBool(ipdb) = true，JSON 布尔 false 被当成了「开」")
	}
	if configBool("ZZ_NO_ACCESS_LOG", "access-log", true) {
		t.Error("configBool(access-log) = true，数字 0 被当成了「开」")
	}
	if got := configString("ZZ_NO_PORT", "port", "8080"); got != "443" {
		t.Errorf("configString(port) = %q, want \"443\"", got)
	}
	if got := configString("ZZ_NO_CORS", "cors", "fallback"); got != "fallback" {
		t.Errorf("configString(cors) = %q, want \"fallback\"（空串应视为未配置）", got)
	}
}

// 运行路径：远端下发 / PATCH 收到布尔与字符串，行为必须一致；
// 非法值计入 unknown 且不覆盖原值。
func TestApplyConfigMapAcceptsBoolAndString(t *testing.T) {
	ipdb0, access0, ota0 := IPDB_ENABLED, ACCESS_LOG, NODE_OTA
	defer func() { IPDB_ENABLED, ACCESS_LOG, NODE_OTA = ipdb0, access0, ota0 }()

	IPDB_ENABLED = true // 先置"开"，才能验证布尔 false 真的把它关掉
	ACCESS_LOG = false  // 反向同理
	NODE_OTA = true     // 非法值不该动它

	applied, unknown, _ := applyConfigMap(map[string]any{
		"ipdb":       false,    // JSON 布尔
		"access-log": "true",   // 字符串
		"node-ota":   "banana", // 非法值
	}, nil)

	if IPDB_ENABLED {
		t.Error("ipdb：布尔 false 未生效（仍是开）")
	}
	if !ACCESS_LOG {
		t.Error("access-log：字符串 \"true\" 未生效（仍是关）")
	}
	if !NODE_OTA {
		t.Error("node-ota：收到非法值却把原值 true 改掉了")
	}
	if !contains(unknown, "node-ota") {
		t.Errorf("unknown = %v，期望包含 node-ota", unknown)
	}
	for _, k := range []string{"ipdb", "access-log"} {
		if !contains(applied, k) {
			t.Errorf("applied = %v，期望包含 %s", applied, k)
		}
	}

	// 反向对称：布尔 true 必须能打开、字符串 "false" 必须能关闭。
	// 这里专门覆盖「布尔」和「字符串」两个方向，防止只测了一个方向就以为等价。
	IPDB_ENABLED = false
	ACCESS_LOG = true
	applied, unknown, _ = applyConfigMap(map[string]any{
		"ipdb":       true,
		"access-log": "false",
	}, nil)

	if !IPDB_ENABLED {
		t.Error("ipdb：布尔 true 未生效（仍是关）")
	}
	if ACCESS_LOG {
		t.Error(`access-log：字符串 "false" 未生效（仍是开）`)
	}
	if len(unknown) != 0 {
		t.Errorf("unknown = %v，期望为空", unknown)
	}
	for _, k := range []string{"ipdb", "access-log"} {
		if !contains(applied, k) {
			t.Errorf("applied = %v，期望包含 %s", applied, k)
		}
	}
}
