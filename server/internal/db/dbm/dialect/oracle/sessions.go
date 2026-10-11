package oracle

// oracle 会话内省与取消（可选能力 SessionInspector 的 oracle 实现）。
//
// 数据源用 v$session（实例级会话视图）左连 v$sql 取正在执行语句全文。仅取 type='USER' 的用户会话，
// 并排除 SYS_CONTEXT('USERENV','SID') 即本次内省自身。oracle 是单库实例，会话不携带「库名」语义，
// Db 留空不虚构；last_call_et 是「距上次调用秒数」而非时间点，故只填 ElapsedMs 不造 StartedAt。
//
// 取消用 ALTER SYSTEM KILL SESSION 'sid,serial'：sid/serial# 均为正整数，标识形态 "sid,serial"，
// 先按形态校验再重新格式化为整数拼入语句，请求侧原始字符串进不了 SQL。

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/spf13/cast"
)

var _ dbi.SessionInspector = (*OracleMetadata)(nil)

// oracleSessionIdPattern KILL SESSION 需要的 "sid,serial" 形态，两段均为正整数
var oracleSessionIdPattern = regexp.MustCompile(`^\d{1,10},\d{1,10}$`)

func (od *OracleMetadata) ListSessions(ctx context.Context) ([]*dbi.SessionInfo, error) {
	// 语句全文不能写 COALESCE(q.sql_fulltext, '')：v$sql.sql_fulltext 是 CLOB，而 Oracle 的空串是
	// CHAR(0)，两者不可统一类型，整条内省语句在准备阶段就报 ORA-00932（真库实测）——表现为
	// Oracle 实例上「活动会话」永远列表为空。这里直接交出 CLOB，未关联到语句时行本就是 NULL，
	// 由 cast.ToString 归成空串
	_, res, err := od.di.QueryContext(ctx, `SELECT s.sid AS "sid", s.serial# AS "serialNum",
		COALESCE(s.username, '') AS "username",
		COALESCE(s.status, '') AS "state",
		q.sql_fulltext AS "query",
		COALESCE(s.last_call_et, 0) AS "elapsedSec",
		CASE WHEN s.blocking_session IS NOT NULL THEN 1 ELSE 0 END AS "blocking"
		FROM v$session s
		LEFT JOIN v$sql q ON q.sql_id = s.sql_id AND q.address = s.sql_address
		WHERE s.type = 'USER' AND s.sid <> SYS_CONTEXT('USERENV', 'SID')
		ORDER BY s.last_call_et DESC`)
	if err != nil {
		return nil, err
	}
	sessions := make([]*dbi.SessionInfo, 0, len(res))
	for _, row := range res {
		sessions = append(sessions, &dbi.SessionInfo{
			Id:        cast.ToString(row["sid"]) + "," + cast.ToString(row["serialNum"]),
			User:      cast.ToString(row["username"]),
			State:     cast.ToString(row["state"]),
			Query:     cast.ToString(row["query"]),
			ElapsedMs: cast.ToInt64(row["elapsedSec"]) * 1000,
			LockWait:  cast.ToInt64(row["blocking"]) != 0,
		})
	}
	return sessions, nil
}

// CancelSession 取消指定会话。id 必须是 "sid,serial" 形态；校验后以 %d 重新格式化，杜绝注入。
func (od *OracleMetadata) CancelSession(ctx context.Context, id string) error {
	if !oracleSessionIdPattern.MatchString(id) {
		return dbi.NewInvalidSessionIdErr(string(DbTypeOracle), id)
	}
	parts := strings.SplitN(id, ",", 2)
	sid, err1 := strconv.Atoi(parts[0])
	serial, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || sid <= 0 || serial <= 0 {
		return dbi.NewInvalidSessionIdErr(string(DbTypeOracle), id)
	}
	_, err := od.di.ExecContext(ctx, fmt.Sprintf("ALTER SYSTEM KILL SESSION '%d,%d' IMMEDIATE", sid, serial))
	// ORA-00031（session marked for kill）不是失败：Oracle 的 KILL 是协作式的，目标会话正卡在不可中断
	// 的调用里时，服务端先把它「标记待杀」并以此码回报发起方，该会话随后就会以同一码终止。
	// 把它当错误抛回界面就变成「点取消弹红色报错、用户以为没生效而反复点」，而取消其实已受理。
	// 只按 ORA 码判读（码是 Oracle 对外稳定的契约）而不匹配提示正文，正文会随语言与版本变化
	if err != nil && !strings.Contains(err.Error(), "ORA-00031") {
		return err
	}
	return nil
}
