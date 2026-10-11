package dm

// dm（达梦）会话内省与取消（可选能力 SessionInspector 的 dm 实现）。
//
// 数据源用 V$SESSIONS（达梦实例级会话视图）。达梦按 database/schema 组织，会话的「库」即当前模式，
// 这里以 CURR_SCH 归入 Db。取消用系统过程 SP_CLOSE_SESSION(sess_id)：sess_id 为整数，先校验再格式化拼入。
//
// ⚠️ 验证状态：本环境无可用达梦实例，V$SESSIONS 的列名（SESS_ID/STATE/USER_NAME/SQL_TEXT/CURR_SCH）
// 与 SP_CLOSE_SESSION 签名按达梦 8 文档编写，接入真实 DM8 前需逐列核对，缺失列会使整条查询报错。
// 排除自身：按语句文本排除含 V$SESSIONS 的内省查询本身（达梦无稳定的「当前会话 id」函数可直连比对）。

import (
	"context"
	"fmt"
	"strconv"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/spf13/cast"
)

var _ dbi.SessionInspector = (*DMMetadata)(nil)

func (dd *DMMetadata) ListSessions(ctx context.Context) ([]*dbi.SessionInfo, error) {
	_, res, err := dd.di.QueryContext(ctx, `SELECT SESS_ID AS "sessId",
		COALESCE(STATE, '') AS "state",
		COALESCE(USER_NAME, '') AS "username",
		COALESCE(SQL_TEXT, '') AS "query",
		COALESCE(CURR_SCH, '') AS "db"
		FROM V$SESSIONS
		WHERE SQL_TEXT NOT LIKE '%V$SESSIONS%'
		ORDER BY SESS_ID`)
	if err != nil {
		return nil, err
	}
	sessions := make([]*dbi.SessionInfo, 0, len(res))
	for _, row := range res {
		sessions = append(sessions, &dbi.SessionInfo{
			Id:    cast.ToString(row["sessId"]),
			Db:    cast.ToString(row["db"]),
			User:  cast.ToString(row["username"]),
			State: cast.ToString(row["state"]),
			Query: cast.ToString(row["query"]),
			// 达梦 V$SESSIONS 未给出稳定的语句开始时间/已运行时长列，不虚构
		})
	}
	return sessions, nil
}

// CancelSession 关闭指定会话。sess_id 必须是正整数；校验后以 %d 重新格式化，杜绝注入。
func (dd *DMMetadata) CancelSession(ctx context.Context, id string) error {
	sessId, err := strconv.ParseUint(id, 10, 64)
	if err != nil || sessId == 0 {
		return dbi.NewInvalidSessionIdErr(string(DbTypeDM), id)
	}
	_, err = dd.di.ExecContext(ctx, fmt.Sprintf("CALL SP_CLOSE_SESSION(%d)", sessId))
	return err
}
