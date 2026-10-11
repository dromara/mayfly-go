package mssql

// mssql 会话内省与取消（可选能力 SessionInspector 的 mssql 实现）。
//
// 数据源用 sys.dm_exec_sessions 左连 sys.dm_exec_requests：dm_exec_sessions 给出所有用户会话
// （含空闲），dm_exec_requests 只在有正在执行的请求时出行，故 LEFT JOIN 后空闲会话的请求列为 NULL。
// OUTER APPLY sys.dm_exec_sql_text(r.sql_handle) 取该请求正在跑的语句文本；sql_handle 为空时
// OUTER APPLY 返回空（不报错），空闲会话语句文本自然为空。
//
// 排除 @@SPID（本次查询自身连接）与 is_user_process=0（系统内部会话），与 mysql/pg 一致，避免列表混入探测本身。
// 取消用 KILL <session_id>：该语句不接受绑定参数，故先把 id 解析成整数再按 %d 重新格式化，
// 非法标识本地拒绝、不把请求字符串拼进语句交给数据库。

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/spf13/cast"
)

var _ dbi.SessionInspector = (*MssqlMetadata)(nil)

// ListSessions 返回该 mssql 实例当前的用户会话，排除本次查询自身连接。
//
// ElapsedMs 取当前请求 start_time 至今的毫秒差；空闲会话无请求，start_time 为 NULL → 0，
// 不虚构时间点（StartedAt 亦为 nil）。取非负：SYSDATETIME() 是本次语句的快照时间，另一条会话的
// start_time 可以比它更晚（两者来自不同快照），测得的负值对操作者无意义。
// 等锁判据：dm_exec_requests.blocking_session_id <> 0。
func (md *MssqlMetadata) ListSessions(ctx context.Context) ([]*dbi.SessionInfo, error) {
	_, res, err := md.di.QueryContext(ctx, `SELECT s.session_id,
		ISNULL(DB_NAME(ISNULL(r.database_id, s.database_id)), '') AS db,
		ISNULL(s.login_name, '') AS username,
		ISNULL(r.status, s.status) AS state,
		ISNULL(t.text, '') AS query,
		r.start_time AS start_time,
		ISNULL(r.blocking_session_id, 0) AS blocking,
		CASE WHEN r.start_time IS NULL OR r.start_time > SYSDATETIME() THEN 0 ELSE DATEDIFF(ms, r.start_time, SYSDATETIME()) END AS elapsed_ms
	FROM sys.dm_exec_sessions s
	LEFT JOIN sys.dm_exec_requests r ON s.session_id = r.session_id
	OUTER APPLY sys.dm_exec_sql_text(r.sql_handle) t
	WHERE s.is_user_process = 1 AND s.session_id <> @@SPID
	ORDER BY r.start_time DESC`)
	if err != nil {
		return nil, err
	}

	sessions := make([]*dbi.SessionInfo, 0, len(res))
	for _, row := range res {
		sessions = append(sessions, &dbi.SessionInfo{
			Id:        cast.ToString(row["session_id"]),
			Db:        cast.ToString(row["db"]),
			User:      cast.ToString(row["username"]),
			State:     cast.ToString(row["state"]),
			Query:     cast.ToString(row["query"]),
			StartedAt: mssqlSessionStartTime(row["start_time"]),
			ElapsedMs: cast.ToInt64(row["elapsed_ms"]),
			LockWait:  cast.ToInt64(row["blocking"]) != 0,
		})
	}
	return sessions, nil
}

// mssqlSessionStartTime 把 datetime 结果归一成 *time.Time；驱动可能返回 time.Time 或字符串，
// 拿不到时间点时返回 nil（不虚构零值时间）。
func mssqlSessionStartTime(v any) *time.Time {
	switch t := v.(type) {
	case time.Time:
		if t.IsZero() {
			return nil
		}
		return &t
	case *time.Time:
		if t == nil || t.IsZero() {
			return nil
		}
		cp := *t
		return &cp
	default:
		s := cast.ToString(v)
		if s == "" {
			return nil
		}
		if parsed, err := time.Parse("2006-01-02T15:04:05Z07:00", s); err == nil {
			return &parsed
		}
		if parsed, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
			return &parsed
		}
		return nil
	}
}

// CancelSession 取消指定会话（KILL 终止该会话上正在执行的语句并断开连接）。
//
// KILL 的 session id 不接受绑定参数，故先解析成整数再按 %d 生成语句：拼进语句的永远是本函数
// 自己格式化的整数，请求侧原始字符串进不了 SQL。解析失败视为非法标识，本地拒绝、不发任何语句。
func (md *MssqlMetadata) CancelSession(ctx context.Context, id string) error {
	sessionId, err := strconv.ParseUint(id, 10, 64)
	if err != nil || sessionId == 0 {
		return dbi.NewInvalidSessionIdErr(string(DbTypeMssql), id)
	}
	_, err = md.di.ExecContext(ctx, fmt.Sprintf("KILL %d", sessionId))
	return err
}
