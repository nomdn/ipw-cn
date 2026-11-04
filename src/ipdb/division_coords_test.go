package ipdb

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDivisionCoords(t *testing.T) {
	dir := t.TempDir()

	// 正常解析：键保留、坐标按 [lng, lat] 原样读出
	p := filepath.Join(dir, "division-coords.json")
	if err := os.WriteFile(p, []byte(`{"110101":[116.416334,39.928359],"321324":[118.699761,33.721266]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := loadDivisionCoords(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 2 {
		t.Fatalf("应解析出 2 条，实际 %d 条", len(m))
	}
	if c := m["110101"]; c[0] != 116.416334 || c[1] != 39.928359 {
		t.Fatalf("110101 坐标不符: %v", c)
	}

	// 坏 JSON 必须返回错误（防止静默空表被当成"无坐标"钉进缓存）
	if err := os.WriteFile(p, []byte(`{"110101":`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadDivisionCoords(p); err == nil {
		t.Fatal("坏 JSON 应返回错误")
	}

	// 缺文件返回错误（调用方降级为无坐标，不影响区划名解析）
	if _, err := loadDivisionCoords(filepath.Join(dir, "absent.json")); err == nil {
		t.Fatal("缺文件应返回错误")
	}
}
