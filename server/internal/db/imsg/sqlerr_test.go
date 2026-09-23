package imsg

import (
	"context"
	"errors"
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// TestSplitError 覆盖「结构化领域错误 → 国际化业务错误」的归一化职责：
// 该职责已从内核 dbm 迁移到 db 业务层消息包，故其测试随迁于此。
func TestSplitError(t *testing.T) {
	// 非切割类错误（如回调抛出的执行错误）：原样透传，不改类型与错误码
	plain := errors.New("duplicate key")
	if got := SplitError(context.Background(), plain); !errors.Is(got, plain) {
		t.Fatalf("non-split error must pass through unchanged, got %v", got)
	}

	// 未闭合区域错误：转换为新的业务错误对象，而非原样返回 UnterminatedError
	ue := &tokenizer.UnterminatedError{Kind: "string", Line: 3}
	biz := SplitError(context.Background(), ue)
	if biz == nil {
		t.Fatal("UnterminatedError must be converted to a business error")
	}
	if biz == error(ue) {
		t.Fatal("UnterminatedError should be converted, not returned as-is")
	}
}
