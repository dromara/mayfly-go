package transfer

import (
	"context"
	"fmt"
	"io"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
)

// importStmtBatchSize 导入侧批级提交的语句数阈值。
// 导入不再使用单一大事务：大事务对目标库undo/redo压力大，且失败后只能整体重跑；
// 批级提交配合表级断点检查点，失败重跑时按表粒度DROP重建，保证幂等。
const importStmtBatchSize = 500

// shouldSkipImportStmt 判断切割出的语句是否不参与导入执行：空（纯注释）语句与dump事务包装语句。
//
// 关键：切割保留注释原文后，分段注释头会与其后语句同属一条（dump产物为
// "-- ----\n-- Data: t \n-- ----\nBEGIN"），直接文本匹配得不出BEGIN，从而被执行并在mysql上
// 隐式提交当前事务，使批级提交/失败回滚失效（残留部分数据），故先按方言语义归一化再判定
// （见 sqlparser.NormalizeStmtText / IsTxnControlStmtText）
func shouldSkipImportStmt(splitter sqlparser.SQLSplitter, stmt string) bool {
	normalized := sqlparser.NormalizeStmtText(splitter, stmt)
	return normalized == "" || sqlparser.IsTxnControlStmtText(normalized)
}

// iterImportStmts 切割导入SQL流，过滤掉不参与执行的语句后逐条回调 exec
//
//	切割+过滤是导入的第一个决策点，被 ImportDumpStream 与单测共用（避免测试只验证副本逻辑而与生产路径漂移）
func iterImportStmts(splitter sqlparser.SQLSplitter, r io.Reader, exec func(stmt string) error) error {
	return splitter.SplitSQL(r, func(stmt string) error {
		if shouldSkipImportStmt(splitter, stmt) {
			return nil
		}
		return exec(stmt)
	})
}

// ImportDumpStream 从dump生成的SQL语句流中批量导入目标库。
//
// 事务语义：每importStmtBatchSize条语句提交一次；语句流中的事务控制语句被过滤
// （见shouldSkipImportStmt），事务边界完全由本函数显式管理。
// 失败时回滚当前批次并返回错误；由于表级DDL含DROP重建（dropBeforeCreate恒定生效），
// 失败表重跑时数据自清理，批级部分提交不会造成重复数据。
func (app *DbTransferAppImpl) ImportDumpStream(ctx context.Context, logId uint64, targetConn *dbi.DbConn, r io.Reader) error {
	splitter := targetConn.GetDialect().GetSQLSplitter()

	tx, err := targetConn.Begin()
	if err != nil {
		return fmt.Errorf("begin import transaction: %w", err)
	}
	pending := 0

	splitErr := iterImportStmts(splitter, r, func(stmt string) error {
		if _, err := targetConn.TxExecContext(ctx, tx, stmt); err != nil {
			app.Log(ctx, logId, fmt.Sprintf("sql exec failed: %s", truncateStmtForLog(stmt)), err, nil)
			return err
		}
		pending++
		if pending >= importStmtBatchSize {
			if err := dbi.CommitTargetTx(targetConn, tx); err != nil {
				return err
			}
			if tx, err = targetConn.Begin(); err != nil {
				return fmt.Errorf("begin import transaction: %w", err)
			}
			pending = 0
		}
		return nil
	})

	if splitErr != nil {
		dbi.RollbackTx(tx)
		return sqlparser.SplitError(ctx, splitErr)
	}
	// 提交尾批（pending可能为0：空事务提交无副作用，但必须关闭tx释放连接）
	if err := dbi.CommitTargetTx(targetConn, tx); err != nil {
		dbi.RollbackTx(tx)
		return err
	}
	return nil
}

// truncateStmtForLog 日志中输出失败语句时做长度截断：
// dump产物单条INSERT可达数MB（多行值批量插入），完整写入日志会冲爆日志存储，
// 且批量数据行可能含敏感信息，截断保留语句头部（含目标表名）已足够定位问题
func truncateStmtForLog(stmt string) string {
	const maxLogStmtLen = 512
	runes := []rune(stmt)
	if len(runes) <= maxLogStmtLen {
		return stmt
	}
	return fmt.Sprintf("%s...(truncated, total %d chars)", string(runes[:maxLogStmtLen]), len(runes))
}
