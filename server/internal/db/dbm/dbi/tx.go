package dbi

import (
	"database/sql"
	"errors"

	"mayfly-go/pkg/logx"
)

// CommitTargetTx 提交目标库写事务：迁移导入（transfer）与数据同步（sync）两条链路
// 共用的目标端事务边界处理（连接池持有的conn由调用方传入）。
// 委托 DbBackend.CommitTargetTx，方言后端可覆写以处理各自的提交怪癖。
func CommitTargetTx(conn *DbConn, tx *sql.Tx) error {
	return conn.Info.Backend.CommitTargetTx(conn, tx)
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
