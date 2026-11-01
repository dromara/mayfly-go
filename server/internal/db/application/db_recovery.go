package application

// 恢复能力（PITR）探测的应用层入口。
//
// 平台自身备份是逻辑 dump：只能恢复到备份时刻。要做到「恢复到任意时刻」，
// 前提在数据库侧（MySQL binlog / PostgreSQL WAL 归档），平台只能探测并如实呈现。
// 因此这里**不做备份编排**，只回答三件事：能不能、能回溯到哪、不能的原因是什么。

import (
	"context"
	"errors"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/ioc"
)

// RecoveryApp 恢复能力探测
type RecoveryApp interface {
	// Readiness 返回实例所在库的时间点恢复前提事实
	Readiness(ctx context.Context, dbId uint64, dbName string) (*dbi.RecoveryReadiness, error)
}

func GetRecoveryApp() RecoveryApp {
	return ioc.Get[RecoveryApp]()
}

type recoveryAppImpl struct {
	dbApp Db `inject:"T"`
}

var _ RecoveryApp = (*recoveryAppImpl)(nil)

// Readiness 取连接后交给方言探测
func (r *recoveryAppImpl) Readiness(ctx context.Context, dbId uint64, dbName string) (*dbi.RecoveryReadiness, error) {
	conn, err := r.dbApp.GetDbConn(ctx, dbId, dbName)
	if err != nil {
		return nil, err
	}
	readiness, err := conn.Metadata().RecoveryReadiness(ctx)
	if err != nil {
		var bizErr *errorx.BizError
		if errors.As(err, &bizErr) {
			return nil, err
		}
		// 方言不支持或读不到：转成可展示的业务错误，不把它当成「平台故障」
		return nil, errorx.NewBizI(ctx, imsg.ErrRecoveryUnsupported, "reason", err.Error())
	}
	return readiness, nil
}
