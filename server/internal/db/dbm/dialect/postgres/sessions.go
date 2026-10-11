package postgres

// postgres 会话内省与取消（可选能力 SessionInspector 的 postgres 实现）。
//
// 数据源 pg_stat_activity 给出的是「服务端每条后端进程」的实况：pid、状态、当前语句、
// 已执行时长与等待事件类型，都可直接取用，无需像 mysql 那样从文本里猜时长。
//
// 取消只 pg_cancel_backend（中断当前语句，保留连接），不用 pg_terminate_backend：
// 后者会拆掉整条连接，池内其他请求会连带报错——用户在控制台点「取消」的意图是停掉这条查询，
// 不是把运维建的连接一并拆了

import (
	"context"
	"strconv"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/spf13/cast"
)

var _ dbi.SessionInspector = (*PgsqlMetadata)(nil)

// ListSessions 返回该实例当前的后端会话，排除本次查询自身（pg_backend_pid）。
func (pm *PgsqlMetadata) ListSessions(ctx context.Context) ([]*dbi.SessionInfo, error) {
	_, res, err := pm.di.QueryContext(ctx, "SELECT pid, COALESCE(datname, '') db, COALESCE(usename, '') username,"+
		" COALESCE(state, '') state, COALESCE(query, '') query,"+
		" COALESCE(wait_event_type, '') wait_event_type,"+
		// query_start 为空（后台进程尚未开始语句）时时长记 0，时间点也不拿 now() 去填（不虚构时间点）。
		// 取非负：now() 是本次语句开始时的快照，另一条会话的 query_start 可以比它更晚
		//（两侧时间戳来自不同快照，真库实测出现过 -7ms）——负时长对操作者毫无意义，
		// 且界面上会渲染成「已执行 -7ms」这种看起来像数据坏了的值
		" GREATEST(COALESCE(FLOOR(EXTRACT(EPOCH FROM (now() - query_start)) * 1000)::bigint, 0), 0) elapsed_ms,"+
		" query_start started_at"+
		" FROM pg_stat_activity WHERE pid <> pg_backend_pid() ORDER BY elapsed_ms DESC")
	if err != nil {
		return nil, err
	}

	sessions := make([]*dbi.SessionInfo, 0, len(res))
	for _, row := range res {
		session := &dbi.SessionInfo{
			Id:        cast.ToString(row["pid"]),
			Db:        cast.ToString(row["db"]),
			User:      cast.ToString(row["username"]),
			State:     cast.ToString(row["state"]),
			Query:     cast.ToString(row["query"]),
			ElapsedMs: cast.ToInt64(row["elapsed_ms"]),
			// pg 用结构化的等待事件类型表达等锁，不需要像 mysql 那样读文本
			LockWait: cast.ToString(row["wait_event_type"]) == "Lock",
		}
		// 只有真的拿到 query_start 才给时间点：后台进程的 query_start 为空，此时只报时长
		if startedAt := cast.ToTime(row["started_at"]); !startedAt.IsZero() {
			session.StartedAt = &startedAt
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

// CancelSession 取消指定后端会话正在执行的语句。
//
// id 先按 int64 校验（pg 的 pid 是整型），再以**绑定参数**传给函数调用，语句文本里不含任何请求输入。
// 校验失败即本地拒绝，不发任何语句给数据库
func (pm *PgsqlMetadata) CancelSession(ctx context.Context, id string) error {
	pid, err := strconv.ParseInt(id, 10, 64)
	if err != nil || pid <= 0 {
		return dbi.NewInvalidSessionIdErr(string(DbTypePostgres), id)
	}
	_, _, err = pm.di.QueryContext(ctx, "SELECT pg_cancel_backend($1)", pid)
	return err
}
