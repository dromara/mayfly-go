package stringx

import "testing"

// TestRandLower 资源/标签 code 的随机源必须只产出小写字母与数字。
//
// 缺陷类：code 含大写时，会进入标签树 code_path 并被 SQL 前缀匹配与 Go 前缀比较同时使用；
// SQLite 的 LIKE 大小写不敏感、Go 的 HasPrefix 敏感，两者口径不一致会出现
// 「列表能看到、点开报无权」。从生成源头限定字符集，才能根除大小写歧义
func TestRandLower(t *testing.T) {
	const length = 256
	for i := 0; i < 50; i++ {
		s := RandLower(length)
		if len(s) != length {
			t.Fatalf("length = %d, want %d", len(s), length)
		}
		for _, r := range s {
			if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') {
				continue
			}
			t.Fatalf("unexpected char %q in %q: only [0-9a-z] allowed", r, s)
		}
	}
}
