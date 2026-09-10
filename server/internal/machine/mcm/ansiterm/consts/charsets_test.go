package consts

import "testing"

// ==================== 字符集初始化测试 ====================

// LAT1 是恒等映射：初始化验证 + Translate 行为合并测试
func TestLAT1_MAP_IdentityAndTranslate(t *testing.T) {
	if LAT1_MAP == nil {
		t.Fatal("LAT1_MAP should not be nil")
	}
	// 验证恒等映射
	for i := 0; i < 256; i++ {
		if got := LAT1_MAP[rune(i)]; got != rune(i) {
			t.Errorf("LAT1_MAP[%d] = %d, want %d", i, got, i)
		}
	}
	// Translate 对 LAT1 应保持原文
	input := "Hello, World! 123"
	if got := Translate(input, LAT1_MAP); got != input {
		t.Errorf("LAT1 Translate = %q, want %q", got, input)
	}
}

func TestVT100_MAP_Initialized(t *testing.T) {
	if VT100_MAP == nil {
		t.Fatal("VT100_MAP should not be nil")
	}
	if len(VT100_MAP) != 256 {
		t.Errorf("VT100_MAP length = %d, want 256", len(VT100_MAP))
	}
	// VT100 特殊字符：0x60 应映射为 ♦ (0x25c6)
	if got := VT100_MAP[0x60]; got != 0x25c6 {
		t.Errorf("VT100_MAP[0x60] = %U, want U+25C6", got)
	}
}

func TestIBMPC_MAP_Initialized(t *testing.T) {
	if IBMPC_MAP == nil {
		t.Fatal("IBMPC_MAP should not be nil")
	}
	if len(IBMPC_MAP) != 256 {
		t.Errorf("IBMPC_MAP length = %d, want 256", len(IBMPC_MAP))
	}
}

// VAX42_MAP 初始化修复验证：之前 init() 中 VAX42Chars = make([]rune, 0) 覆盖了原始数组，
// 导致后续 VAX42Chars[i] = c 对空切片按索引赋值 panic
func TestVAX42_MAP_Initialized(t *testing.T) {
	if VAX42_MAP == nil {
		t.Fatal("VAX42_MAP should not be nil (init() would have panicked before fix)")
	}
	if len(VAX42_MAP) != 256 {
		t.Errorf("VAX42_MAP length = %d, want 256", len(VAX42_MAP))
	}
	// VAX42 特殊字符验证
	if got := VAX42_MAP[0x21]; got != 0x043b {
		t.Errorf("VAX42_MAP[0x21] = %U, want U+043B", got)
	}
}

func TestCHARMAPS_ContainsAllCharsets(t *testing.T) {
	charsets := []string{"B", "0", "U", "V"}
	for _, cs := range charsets {
		m, ok := CHARMAPS[cs]
		if !ok {
			t.Errorf("CHARMAPS missing charset %q", cs)
		}
		if m == nil {
			t.Errorf("CHARMAPS[%q] should not be nil", cs)
		}
	}
	// 验证各映射长度一致
	if len(CHARMAPS["B"]) != 256 {
		t.Errorf("CHARMAPS['B'] length = %d, want 256", len(CHARMAPS["B"]))
	}
	if len(CHARMAPS["0"]) != 256 {
		t.Errorf("CHARMAPS['0'] length = %d, want 256", len(CHARMAPS["0"]))
	}
}

// ==================== Translate 测试 ====================

func TestTranslate_VT100SpecialChars(t *testing.T) {
	// VT100 线画字符：0x71 应映射为 ─ (0x2500)
	input := string(rune(0x71))
	got := Translate(input, VT100_MAP)
	expected := string(rune(0x2500))
	if got != expected {
		t.Errorf("VT100 Translate(0x71) = %q, want %q", got, expected)
	}
}

func TestTranslate_UnknownCharPassthrough(t *testing.T) {
	// 不在映射表中的字符应原样通过
	input := "日本語テスト"
	got := Translate(input, LAT1_MAP)
	// LAT1 只有 0-255，日文字符不在表中，应原样返回
	if got != input {
		t.Errorf("Translate = %q, want %q", got, input)
	}
}

func TestTranslate_EmptyString(t *testing.T) {
	got := Translate("", LAT1_MAP)
	if got != "" {
		t.Errorf("Translate empty = %q, want ''", got)
	}
}
