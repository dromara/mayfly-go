package imsg

import (
	"context"
	"errors"

	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
	"mayfly-go/pkg/errorx"
)

// SplitError 归一化 SQL 语句切割错误：未闭合的引号/注释（*tokenizer.UnterminatedError）
// 转为带行号定位的国际化业务错误；其余错误（如读取文件失败、回调执行失败）原样返回，
// 不改变其类型与错误码。
//
// 该「结构化领域错误 → 面向用户的国际化错误」的翻译职责置于 db 业务层的消息包，
// 而非内核 dbm：内核只暴露携带定位信息的结构化错误，如何呈现由业务层决定，
// 从而使内核与 i18n 解耦。
func SplitError(ctx context.Context, err error) error {
	var ue *tokenizer.UnterminatedError
	if errors.As(err, &ue) {
		return errorx.NewBizI(ctx, ErrSQLSplitUnterminated, "kind", ue.Kind, "line", ue.Line)
	}
	return err
}
