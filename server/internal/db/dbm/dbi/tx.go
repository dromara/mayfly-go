package dbi

import (
	"database/sql"
	"errors"
	"strings"

	"mayfly-go/pkg/logx"
)

// CommitTargetTx 提交目标库写事务：迁移导入（transfer）与数据同步（sync）两条链路
// 共用的目标端事务边界处理（连接池持有的conn由调用方传入）。

// mssql驱动存在已知怪癖：连接池复用的conn在特定时序下commit会返回
// "no corresponding begin transaction"，但事务实际已提交；仅对该怪癖做精确
// 兼容（忽略错误），其余提交失败一律上抛，不得静默吞错
func CommitTargetTx(conn *DbConn, tx *sql.Tx) error {
	// Begin失败等异常路径下tx可能为nil，直接Commit会nil指针panic
	if tx == nil {
		return nil
	}
	if err := tx.Commit(); err != nil {
		if conn.Info.Type == ToDbType("mssql") &&
			strings.Contains(strings.ToLower(err.Error()), "no corresponding begin transaction") {
			logx.Warnf("mssql target tx commit returned driver quirk error (ignored, data committed): %s", err.Error())
			return nil
		}
		return err
	}
	return nil
}

// RollbackTx 回滚目标库事务，回滚失败仅记录日志（回滚失败常见于连接已断开，无更好处置）
func RollbackTx(tx *sql.Tx) {
	if tx == nil {
		return
	}
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		logx.Warnf("rollback target tx failed: %s", err.Error())
	}
}
