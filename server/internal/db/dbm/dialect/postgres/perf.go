package postgres

// postgres 慢查询聚合与锁等待链（可选能力 SlowQueryInspector / LockWaitInspector）。
//
// 慢查询依赖 pg_stat_statements 扩展：没装就返回「不可用 + 原因」，不退回逐条采样冒充
// 聚合结果（两条语句在同一张表上跑，采样分不清「谁更慢」，会给出误导性的排序）。
//
// 锁等待用 pg_blocking_pids(pid)：它直接给出阻塞 pid 列表，一个等待方可能被多个会话阻塞，
// 因此按「等待方 × 阻塞方」展开成多行——只报「有 N 个会话在等」定位不到源头。

import (
	"context"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/spf13/cast"
)

var (
	_ dbi.SlowQueryInspector = (*PgsqlMetadata)(nil)
	_ dbi.LockWaitInspector  = (*PgsqlMetadata)(nil)
)

// pg 13 起统计列改名为 *_exec_time，12 及以前是 *_time。两套都试一次，
// 而不是按 server_version 解析——版本字符串的格式差异（14beta、15devel）比两套列名更难伺候
const pgSlowQuerySQL = "SELECT queryid::text digest, COALESCE(query, '') sql_text, calls," +
	" COALESCE(ROUND(total_exec_time)::bigint, 0) total_ms, COALESCE(ROUND(mean_exec_time)::bigint, 0) avg_ms," +
	" COALESCE(ROUND(max_exec_time)::bigint, 0) max_ms, COALESCE(ROUND(rows / NULLIF(calls, 0))::bigint, 0) rows_avg" +
	" FROM pg_stat_statements ORDER BY total_exec_time DESC LIMIT "

const pgSlowQuerySQLLegacy = "SELECT queryid::text digest, COALESCE(query, '') sql_text, calls," +
	" COALESCE(ROUND(total_time)::bigint, 0) total_ms, COALESCE(ROUND(mean_time)::bigint, 0) avg_ms," +
	" COALESCE(ROUND(max_time)::bigint, 0) max_ms, COALESCE(ROUND(rows / NULLIF(calls, 0))::bigint, 0) rows_avg" +
	" FROM pg_stat_statements ORDER BY total_time DESC LIMIT "

// SlowQueries 按累计耗时降序返回前 limit 条归一化语句
func (pm *PgsqlMetadata) SlowQueries(ctx context.Context, limit int) ([]*dbi.SlowQuery, error) {
	if limit <= 0 {
		limit = 20
	}
	_, res, err := pm.di.QueryContext(ctx, pgSlowQuerySQL+cast.ToString(limit))
	if err != nil {
		// 可能是列名版本差异，也可能是扩展未安装；先试旧列名，仍失败再判定不可用
		_, res, err = pm.di.QueryContext(ctx, pgSlowQuerySQLLegacy+cast.ToString(limit))
		if err != nil {
			reason := strings.TrimSpace(err.Error())
			return nil, dbi.NewPerfUnavailableWithHint("slow_query", reason, pgSlowQueryHint(reason))
		}
	}

	queries := make([]*dbi.SlowQuery, 0, len(res))
	for _, row := range res {
		queries = append(queries, &dbi.SlowQuery{
			Digest:  cast.ToString(row["digest"]),
			SQL:     cast.ToString(row["sql_text"]),
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
func (pm *PgsqlMetadata) LockWaits(ctx context.Context) ([]*dbi.LockWait, error) {
	_, res, err := pm.di.QueryContext(ctx,
		"SELECT a.pid::text wait_id, COALESCE(a.usename, '') wait_user, COALESCE(a.datname, '') wait_db,"+
			" COALESCE(a.query, '') wait_query,"+
			" GREATEST(COALESCE(FLOOR(EXTRACT(EPOCH FROM (now() - a.query_start)) * 1000)::bigint, 0), 0) wait_ms,"+
			" COALESCE(a.backend_xid::text, a.backend_xmin::text, '') wait_trx,"+
			" COALESCE(a.wait_event_type || '/' || a.wait_event, '') lock_type,"+
			" b.pid::text block_id, COALESCE(b.usename, '') block_user, COALESCE(b.datname, '') block_db,"+
			" COALESCE(b.query, '') block_query"+
			" FROM pg_stat_activity a"+
			" JOIN LATERAL unnest(pg_blocking_pids(a.pid)) AS blocker(pid) ON true"+
			" JOIN pg_stat_activity b ON b.pid = blocker.pid"+
			" WHERE a.pid <> pg_backend_pid() ORDER BY wait_ms DESC")
	if err != nil {
		return nil, dbi.NewPerfUnavailable("lock_wait", strings.TrimSpace(err.Error()))
	}

	waits := make([]*dbi.LockWait, 0, len(res))
	for _, row := range res {
		waits = append(waits, &dbi.LockWait{
			WaitSessionId:  cast.ToString(row["wait_id"]),
			WaitUser:       cast.ToString(row["wait_user"]),
			WaitDb:         cast.ToString(row["wait_db"]),
			WaitQuery:      cast.ToString(row["wait_query"]),
			WaitMs:         cast.ToInt64(row["wait_ms"]),
			WaitTrx:        cast.ToString(row["wait_trx"]),
			LockType:       cast.ToString(row["lock_type"]),
			BlockSessionId: cast.ToString(row["block_id"]),
			BlockUser:      cast.ToString(row["block_user"]),
			BlockDb:        cast.ToString(row["block_db"]),
			BlockQuery:     cast.ToString(row["block_query"]),
		})
	}
	return waits, nil
}

// pgSlowQueryHint 把「慢查询不可用」翻译成可执行步骤。
//
// 三种真实成因要分开说，否则用户只会看到驱动原文：
//   - 扩展没装（relation does not exist）→ 预加载 + CREATE EXTENSION
//   - 预加载没配（must be loaded via shared_preload_libraries）→ 改配置并重启
//   - 权限不足（permission denied）→ 授予只读统计视图权限
func pgSlowQueryHint(reason string) string {
	lower := strings.ToLower(reason)
	switch {
	case strings.Contains(lower, "must be loaded via shared_preload_libraries"):
		return "启用步骤：在 postgresql.conf 设置 shared_preload_libraries = 'pg_stat_statements' 后重启实例"
	case strings.Contains(lower, "does not exist"):
		return "启用步骤：1) postgresql.conf 设置 shared_preload_libraries = 'pg_stat_statements' 并重启实例；" +
			"2) 在目标库执行 CREATE EXTENSION IF NOT EXISTS pg_stat_statements;"
	case strings.Contains(lower, "permission denied"):
		return "启用步骤：给当前连接账号授予 pg_read_all_stats（或 superuser）权限"
	}
	return ""
}
