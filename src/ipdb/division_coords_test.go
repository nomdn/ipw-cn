package ipdb

import "testing"

func TestDivisionCoords(t *testing.T) {
	if len(divisionCoords) < 2800 {
		t.Fatalf("divisionCoords 区划数不足: %d", len(divisionCoords))
	}
	c, ok := divisionCoords["110101"]
	if !ok {
		t.Fatal("缺少 110101(东城区)")
	}
	if c[0] < 73 || c[0] > 136 || c[1] < 3 || c[1] > 54 {
		t.Fatalf("110101 坐标超出中国范围: %v", c)
	}
}
