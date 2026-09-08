package transfer

import (
	"context"
	"fmt"
	"io"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
)

// importStmtBatchSize 导入侧批级提交的语句数阈值。
// 导入不再使用单一大事务：大事务对目标库undo/redo压力大，且失败后只能整体重跑；
// 批级提交配合表级断点检查点，失败重跑时按表粒度DROP重建，保证幂等。
const importStmtBatchSize = 500

// isTxnControlStmt 判断语句是否为事务控制语句（BEGIN/COMMIT/START TRANSACTION/ROLLBACK/SET autocommit）。
//
// dump产物中的事务包装语句（如mysql DefaultDumpHelper输出的BEGIN;/COMMIT;）属于"SQL文件执行"语义；
// 数据库导入路径显式管理事务（批级提交），这类语句若在显式事务内执行会被部分数据库隐式提交
// （如mysql执行BEGIN会隐式提交当前事务），破坏导入的事务语义，必须过滤。
// 注意："SET IDENTITY_INSERT ..."（mssql/dm自增列导入前置语句）不以AUTOCOMMIT匹配，不受影响。
func isTxnControlStmt(stmt string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(stmt), ";")))
	switch normalized {
	case "BEGIN", "BEGIN WORK", "BEGIN TRANSACTION",
		"COMMIT", "COMMIT WORK",
		"START TRANSACTION", "ROLLBACK", "ROLLBACK WORK":
		return true
	}
	// SET autocommit 及 SET @@session.autocommit 等变体
	return strings.HasPrefix(normalized, "SET") && strings.Contains(normalized, "AUTOCOMMIT")
}

// importDumpStream 从dump生成的SQL语句流中批量导入目标库。
//
// 事务语义：每importStmtBatchSize条语句提交一次；语句流中的事务控制语句被过滤
// （见isTxnControlStmt），事务边界完全由本函数显式管理。
// 失败时回滚当前批次并返回错误；由于表级DDL含DROP重建（dropBeforeCreate恒定生效），
// 失败表重跑时数据自清理，批级部分提交不会造成重复数据。
func (app *DbTransferAppImpl) importDumpStream(ctx context.Context, logId uint64, targetConn *dbi.DbConn, r io.Reader) error {
	splitter := targetConn.GetDialect().GetSQLSplitter()

	tx, err := targetConn.Begin()
	if err != nil {
		return fmt.Errorf("begin import transaction: %w", err)
	}
	pending := 0

	splitErr := splitter.SplitSQL(r, func(stmt string) error {
		trimmed := strings.TrimSpace(stmt)
		if trimmed == "" || isTxnControlStmt(trimmed) {
			return nil
		}
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
		return splitErr
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
