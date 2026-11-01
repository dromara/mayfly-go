package application

// 性能诊断入口：慢查询聚合与锁等待链。
//
// 这一层只做三件事：拿连接（内含资源可达校验）、把「能力缺失」与「数据源不可用」翻译成
// 可读的业务错误、把方言事实原样透出。判定与取数一律留在方言里——
// 上层再用文本猜「这条是不是在等锁」会让判据在多个入口各写一份。

import (
	"context"
	"errors"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/i18n"
)

// perfLimitMax 慢查询返回条数上限：这不是分页接口，而是「看一眼最贵的语句」，
// 超过这个量人也不会读，且 digest 文本本身可能很长
const perfLimitMax = 100

// SlowQueries 返回该库所属实例的慢查询聚合（按累计耗时降序）
func (d *dbAppImpl) SlowQueries(ctx context.Context, dbId uint64, dbName string, limit int) ([]*dbi.SlowQuery, error) {
	if limit <= 0 || limit > perfLimitMax {
		limit = 20
	}
	conn, err := d.GetDbConn(ctx, dbId, dbName)
	if err != nil {
		return nil, err
	}
	queries, err := conn.Metadata().SlowQueries(ctx, limit)
	if err != nil {
		return nil, perfBizError(ctx, err, imsg.ErrSlowQueryUnavailable)
	}
	return queries, nil
}

// LockWaits 返回该库所属实例当前的锁等待链
func (d *dbAppImpl) LockWaits(ctx context.Context, dbId uint64, dbName string) ([]*dbi.LockWait, error) {
	conn, err := d.GetDbConn(ctx, dbId, dbName)
	if err != nil {
		return nil, err
	}
	waits, err := conn.Metadata().LockWaits(ctx)
	if err != nil {
		return nil, perfBizError(ctx, err, imsg.ErrLockWaitUnavailable)
	}
	return waits, nil
}

// perfBizError 把方言侧的「能力不支持」与「数据源不可用」翻译成业务错误。
//
// 两者必须区分：方言不支持是产品边界（换方言才能解决），数据源不可用是部署开关
// （开 performance_schema 就能解决）——给出同一条提示会让运维按错误的方向排查。
func perfBizError(ctx context.Context, err error, unavailableMsgID i18n.MsgId) error {
	if errors.Is(err, dbi.ErrUnsupportedKind) {
		return errorx.NewBizI(ctx, imsg.ErrPerfUnsupported)
	}
	if errors.Is(err, dbi.ErrPerfUnavailable) {
		// 启用步骤由方言给出（PG 要预加载 + CREATE EXTENSION，MySQL 要开 performance_schema），
		// 这里只做拼接：放应用层按方言类型拼会让同一份知识在两处各说各话
		reason := perfReason(err)
		if hint := dbi.PerfHint(err); hint != "" {
			reason += "\n" + hint
		}
		return errorx.NewBizI(ctx, unavailableMsgID, "reason", reason)
	}
	return err
}

// perfReason 去掉包装前缀，只留方言给的原始原因
func perfReason(err error) string {
	text := err.Error()
	if idx := strings.LastIndex(text, "("); idx >= 0 && strings.HasSuffix(text, ")") {
		return text[idx+1 : len(text)-1]
	}
	return text
}
