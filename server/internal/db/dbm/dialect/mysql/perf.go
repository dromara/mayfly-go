package mysql

// mysql 慢查询聚合与锁等待链（可选能力 SlowQueryInspector / LockWaitInspector）。
//
// 数据源都是 performance_schema：
//   - 慢查询取 events_statements_summary_by_digest：它按归一化语句聚合，能直接给出
//     「这类语句一共跑了多少次、累计多慢」，比逐条 processlist 采样更接近调优时需要的信息；
//   - 锁等待取 data_lock_waits（8.0+）：它把「谁在等谁」表达成一对 id，
//     5.7 的 innodb_lock_waits 结构不同，这里不降级——返回「不可用 + 原因」比给一半数据诚实。
//
// 两者都可能因为 performance_schema 被关、或 digest 表被清空而拿不到数据：
// 一律返回 ErrPerfUnavailable 并带上原因，不返回空切片（空结果在界面上等于「没有问题」）。

import (
	"context"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/spf13/cast"
)

var (
	_ dbi.SlowQueryInspector = (*MysqlMetadata)(nil)
	_ dbi.LockWaitInspector  = (*MysqlMetadata)(nil)
)

// SlowQueries 按累计耗时降序返回前 limit 条归一化语句
func (md *MysqlMetadata) SlowQueries(ctx context.Context, limit int) ([]*dbi.SlowQuery, error) {
	if limit <= 0 {
		limit = 20
	}
	_, res, err := md.di.QueryContext(ctx,
		"SELECT IFNULL(SCHEMA_NAME, '') schema_name, IFNULL(DIGEST_TEXT, '') sql_text, COUNT_STAR calls,"+
			" ROUND(AVG_TIMER_WAIT / 1000000000) avg_ms, ROUND(MAX_TIMER_WAIT / 1000000000) max_ms,"+
			" ROUND(SUM_TIMER_WAIT / 1000000000) total_ms,"+
			" ROUND(SUM_ROWS_SENT / NULLIF(COUNT_STAR, 0)) rows_avg"+
			" FROM performance_schema.events_statements_summary_by_digest"+
			" WHERE DIGEST_TEXT IS NOT NULL ORDER BY SUM_TIMER_WAIT DESC LIMIT "+cast.ToString(limit))
	if err != nil {
		// performance_schema 关闭、或 8.0 之前没有这张表：把原因交给上层展示
		reason := strings.TrimSpace(err.Error())
		return nil, dbi.NewPerfUnavailableWithHint("slow_query", reason, mysqlSlowQueryHint(reason))
	}

	queries := make([]*dbi.SlowQuery, 0, len(res))
	for _, row := range res {
		queries = append(queries, &dbi.SlowQuery{
			Digest:  cast.ToString(row["sql_text"]),
			SQL:     cast.ToString(row["sql_text"]),
			Schema:  cast.ToString(row["schema_name"]),
			Calls:   cast.ToInt64(row["calls"]),
			AvgMs:   cast.ToInt64(row["avg_ms"]),
			MaxMs:   cast.ToInt64(row["max_ms"]),
			TotalMs: cast.ToInt64(row["total_ms"]),
			RowsAvg: cast.ToInt64(row["rows_avg"]),
		})
	}
	return queries, nil
}

// LockWaits 返回当前所有「等待方 → 阻塞方」的锁关系
func (md *MysqlMetadata) LockWaits(ctx context.Context) ([]*dbi.LockWait, error) {
	_, res, err := md.di.QueryContext(ctx,
		"SELECT IFNULL(w.REQUESTING_ENGINE_TRANSACTION_ID, '') wait_trx,"+
			" IFNULL(w.BLOCKING_ENGINE_TRANSACTION_ID, '') block_trx,"+
			" CONCAT(IFNULL(r.OBJECT_SCHEMA, ''), '.', IFNULL(r.OBJECT_NAME, '')) lock_object,"+
			" CONCAT(IFNULL(r.LOCK_TYPE, ''), ' ', IFNULL(r.LOCK_MODE, '')) lock_type,"+
			" IFNULL(wt.PROCESSLIST_ID, 0) wait_id, IFNULL(wt.PROCESSLIST_USER, '') wait_user,"+
			" IFNULL(wt.PROCESSLIST_DB, '') wait_db, IFNULL(wt.PROCESSLIST_INFO, '') wait_query,"+
			" IFNULL(wt.PROCESSLIST_TIME, 0) wait_sec,"+
			" IFNULL(bt.PROCESSLIST_ID, 0) block_id, IFNULL(bt.PROCESSLIST_USER, '') block_user,"+
			" IFNULL(bt.PROCESSLIST_DB, '') block_db, IFNULL(bt.PROCESSLIST_INFO, '') block_query"+
			" FROM performance_schema.data_lock_waits w"+
			" JOIN performance_schema.data_locks r ON r.ENGINE_LOCK_ID = w.REQUESTING_ENGINE_LOCK_ID"+
			" LEFT JOIN performance_schema.threads wt ON wt.THREAD_ID = w.REQUESTING_THREAD_ID"+
			" LEFT JOIN performance_schema.threads bt ON bt.THREAD_ID = w.BLOCKING_THREAD_ID"+
			" ORDER BY wait_sec DESC")
	if err != nil {
		// 5.7 只有 innodb_lock_waits、performance_schema 关闭时也没有 data_lock_waits
		return nil, dbi.NewPerfUnavailable("lock_wait", strings.TrimSpace(err.Error()))
	}

	waits := make([]*dbi.LockWait, 0, len(res))
	for _, row := range res {
		lockObject := strings.TrimPrefix(cast.ToString(row["lock_object"]), ".")
		lockType := strings.TrimSpace(cast.ToString(row["lock_type"]))
		if lockObject != "" {
			lockType = strings.TrimSpace(lockObject + " " + lockType)
		}
		waits = append(waits, &dbi.LockWait{
			WaitSessionId:  cast.ToString(row["wait_id"]),
			WaitUser:       cast.ToString(row["wait_user"]),
			WaitDb:         cast.ToString(row["wait_db"]),
			WaitQuery:      cast.ToString(row["wait_query"]),
			WaitMs:         cast.ToInt64(row["wait_sec"]) * 1000,
			WaitTrx:        cast.ToString(row["wait_trx"]),
			BlockTrx:       cast.ToString(row["block_trx"]),
			BlockSessionId: cast.ToString(row["block_id"]),
			BlockUser:      cast.ToString(row["block_user"]),
			BlockDb:        cast.ToString(row["block_db"]),
			BlockQuery:     cast.ToString(row["block_query"]),
			LockType:       lockType,
		})
	}
	return waits, nil
}

// mysqlSlowQueryHint 慢查询不可用时的可执行步骤
func mysqlSlowQueryHint(reason string) string {
	lower := strings.ToLower(reason)
	if strings.Contains(lower, "performance_schema") || strings.Contains(lower, "denied") {
		return "启用步骤：my.cnf 设置 performance_schema = ON 后重启实例（MySQL 8.0 默认开启），并确认账号可读 performance_schema"
	}
	return ""
}
