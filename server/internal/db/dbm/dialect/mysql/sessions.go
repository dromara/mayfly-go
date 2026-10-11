package mysql

// mysql 会话内省与取消（可选能力 SessionInspector 的 mysql 实现）。
//
// 数据源用 information_schema.processlist 而非 SHOW FULL PROCESSLIST：前者可以按列名取值、
// 也能在 WHERE 里排除本次查询自身的连接（SHOW 做不到按列过滤，列表里会混进这条查询本身）。
//
// 取消用 KILL <id>：该语句不接受绑定参数，因此 id 必须先过 dbi.IsNumericSessionId 字符集校验，
// 不合法直接本地拒绝，不把请求参数拼进语句交给数据库报错。

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/spf13/cast"
)

var _ dbi.SessionInspector = (*MysqlMetadata)(nil)

// mysql 等锁判据：processlist.STATE 是执行阶段文本，出现 lock 即在等锁
// （如 Waiting for table metadata lock、Waiting for row lock 等）。
// 该判定只在这里出现一次，上层不做字符串猜测，避免同一条判据在多方言/多层重复漂移
func mysqlSessionLockWait(state string) bool {
	return strings.Contains(strings.ToLower(state), "lock")
}

// ListSessions 返回该 mysql 实例当前的会话，排除本次查询自身的连接。
//
// TIME 是「当前状态已持续秒数」而非时间戳，故只填 ElapsedMs 不填 StartedAt（不虚构时间点）
func (md *MysqlMetadata) ListSessions(ctx context.Context) ([]*dbi.SessionInfo, error) {
	_, res, err := md.di.QueryContext(ctx, "SELECT id, IFNULL(db, '') db, user, IFNULL(state, '') state, IFNULL(info, '') query, time"+
		" FROM information_schema.processlist WHERE id <> CONNECTION_ID() ORDER BY time DESC")
	if err != nil {
		return nil, err
	}

	sessions := make([]*dbi.SessionInfo, 0, len(res))
	for _, row := range res {
		state := cast.ToString(row["state"])
		sessions = append(sessions, &dbi.SessionInfo{
			Id:        cast.ToString(row["id"]),
			Db:        cast.ToString(row["db"]),
			User:      cast.ToString(row["user"]),
			State:     state,
			Query:     cast.ToString(row["query"]),
			ElapsedMs: cast.ToInt64(row["time"]) * 1000,
			LockWait:  mysqlSessionLockWait(state),
		})
	}
	return sessions, nil
}

// CancelSession 取消指定会话（KILL 会终止该连接上正在执行的语句）。
//
// KILL 的连接 id 无法用绑定参数传，因此这里先把 id 解析成 uint64 再按 %d 重新生成语句文本：
// 拼进语句的永远是一个由本函数自己格式化出的整数，请求侧的原始字符串无论如何都进不了 SQL。
// 解析失败视为非法会话标识，本地拒绝，不发任何语句给数据库
func (md *MysqlMetadata) CancelSession(ctx context.Context, id string) error {
	connId, err := strconv.ParseUint(id, 10, 64)
	if err != nil || connId == 0 {
		return dbi.NewInvalidSessionIdErr(string(DbTypeMysql), id)
	}
	_, err = md.di.ExecContext(ctx, fmt.Sprintf("KILL %d", connId))
	return err
}
