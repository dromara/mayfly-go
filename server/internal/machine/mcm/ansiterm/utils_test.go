package ansiterm

import "testing"

// ==================== WidthOfRune 测试（CJK 宽度逻辑） ====================

func TestWidthOfRune(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want int
	}{
		// ASCII
		{"space", ' ', 1},
		{"letter", 'A', 1},
		{"digit", '0', 1},

		// CJK 全角字符
		{"CJK unified", '中', 2},
		{"Hiragana", 'あ', 2},
		{"Katakana", 'ア', 2},
		{"Hangul", '한', 2},

		// 控制字符
		{"NUL", 0x00, 0},
		{"ESC", 0x1b, 0},
		{"DEL", 0x7f, 0},

		// C1 控制字符
		{"C1 start", 0x80, 0},
		{"C1 end", 0x9f, 0},

		// 其他可打印字符
		{"Latin accented", 'é', 1},
		{"Euro sign", '€', 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WidthOfRune(tt.r)
			if got != tt.want {
				t.Errorf("WidthOfRune(%U) = %d, want %d", tt.r, got, tt.want)
			}
		})
	}
}

// ==================== DecodeUTF8WithReplacement 测试 ====================

func TestDecodeUTF8WithReplacement(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"valid ASCII", []byte("hello"), "hello"},
		{"multi-byte CJK", []byte("你好"), "你好"},
		{"invalid bytes", []byte{0xff, 0xfe}, "\uFFFD\uFFFD"},
		{"mixed valid/invalid", []byte("a\xffb"), "a\uFFFDb"},
		{"empty", []byte{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeUTF8WithReplacement(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// ==================== Pop 测试 ====================

func TestPop(t *testing.T) {
	tests := []struct {
		name    string
		initial []int
		wantVal int
		wantOk  bool
		wantLen int
	}{
		{"normal", []int{1, 2, 3}, 3, true, 2},
		{"single element", []int{42}, 42, true, 0},
		{"empty", []int{}, 0, false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.initial
			val, ok := Pop(&s)
			if ok != tt.wantOk {
				t.Errorf("ok = %v, want %v", ok, tt.wantOk)
			}
			if val != tt.wantVal {
				t.Errorf("val = %d, want %d", val, tt.wantVal)
			}
			if len(s) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(s), tt.wantLen)
			}
		})
	}
}

// ==================== BytesToString 测试 ====================

func TestBytesToString(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"normal", []byte{0x41, 0x42, 0x43}, "ABC"},
		{"empty", []byte{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BytesToString(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
