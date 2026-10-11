package clickhouse

// clickhouse 会话内省与取消（可选能力 SessionInspector 的 clickhouse 实现）。
//
// 数据源用 system.processes（正在执行的查询）。clickhouse 无「空闲会话」概念——不在跑的会话不出现在
// 这张表里，故列出的都是执行中的查询，state 记为 running。无锁等待概念，LockWait 恒 false。
//
// 排除自身：clickhouse 没有「当前连接 id」这种可直接比对的量，故按语句文本排除含 system.processes 的
// 内省查询本身（与其它方言排除自身连接的目的一致）。
//
// 取消用 KILL QUERY WHERE query_id='...'：该语句不接受绑定参数，query_id 先过安全字符集校验再拼入，
// 非法标识本地拒绝（ErrInvalidSessionId），不把请求字符串交给数据库。

import (
	"context"
	"regexp"
	"time"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/spf13/cast"
)

var _ dbi.SessionInspector = (*ClickHouseMetadata)(nil)

// chQueryIdPattern clickhouse query_id 为 UUID/十六进制形态；只放行安全字符，杜绝拼进 KILL 语句的注入。
var chQueryIdPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func (cm *ClickHouseMetadata) ListSessions(ctx context.Context) ([]*dbi.SessionInfo, error) {
	_, res, err := cm.di.QueryContext(ctx, `SELECT query_id, current_database AS db, user AS username,
		query, query_start_time, toInt64(query_duration_ms) AS elapsed_ms
		FROM system.processes
		WHERE query NOT LIKE '%system.processes%'
		ORDER BY elapsed_ms DESC`)
	if err != nil {
		return nil, err
	}
	sessions := make([]*dbi.SessionInfo, 0, len(res))
	for _, row := range res {
		sessions = append(sessions, &dbi.SessionInfo{
			Id:        cast.ToString(row["query_id"]),
			Db:        cast.ToString(row["db"]),
			User:      cast.ToString(row["username"]),
			State:     "running",
			Query:     cast.ToString(row["query"]),
			StartedAt: chSessionStartTime(row["query_start_time"]),
			ElapsedMs: cast.ToInt64(row["elapsed_ms"]),
			// clickhouse 无锁等待概念，不虚构
			LockWait: false,
		})
	}
	return sessions, nil
}

// chSessionStartTime 归一 query_start_time（驱动通常给 time.Time）；拿不到时间点返回 nil，不虚构零值。
func chSessionStartTime(v any) *time.Time {
	if t, ok := v.(time.Time); ok {
		if t.IsZero() {
			return nil
		}
		return &t
	}
	return nil
}

// CancelSession 取消指定查询（KILL QUERY 中断该 query_id 正在执行的查询）。
func (cm *ClickHouseMetadata) CancelSession(ctx context.Context, id string) error {
	if !chQueryIdPattern.MatchString(id) {
		return dbi.NewInvalidSessionIdErr(string(DbTypeClickHouse), id)
	}
	_, err := cm.di.ExecContext(ctx, "KILL QUERY WHERE query_id = '"+id+"'")
	return err
}
