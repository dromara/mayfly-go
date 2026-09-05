package tokenizer

import (
	"testing"
)

// TestKeywordsPaginationAndConflict 验证分页（FETCH/OFFSET相关）与冲突处理（ON CONFLICT）关键字已注册。
// 这些关键字缺失曾导致 ORDER BY ... FETCH 的 SkipExpr 无法判断表达式边界而死循环
func TestKeywordsPaginationAndConflict(t *testing.T) {
	required := []string{
		// 分页相关
		"FETCH", "FIRST", "NEXT", "ROW", "ROWS", "TIES",
		// 冲突处理相关
		"CONFLICT", "DO", "NOTHING", "CONSTRAINT",
	}

	for _, kw := range required {
		if !Keywords[kw] {
			t.Errorf("keyword [%s] should be registered in tokenizer.Keywords", kw)
		}
	}
}

func TestTokenizerFetchClauseTokens(t *testing.T) {
	sql := "SELECT id FROM users ORDER BY id OFFSET 20 ROWS FETCH FIRST 10 ROWS ONLY"
	tok := New(sql, DialectConfig{})

	found := map[string]int{}
	for _, t2 := range tok.Tokens {
		if Keywords[t2.Value] {
			found[t2.Value]++
		}
	}
	if found["OFFSET"] == 0 {
		t.Fatal("OFFSET should be tokenized")
	}
	if found["FETCH"] == 0 {
		t.Fatal("FETCH should be tokenized")
	}
	if found["ROWS"] != 2 {
		t.Fatalf("expected 2 ROWS tokens, got %d", found["ROWS"])
	}
}
