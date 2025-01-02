package api

import "mayfly-go/pkg/ioc"

func InitIoc() {
	ioc.Register(new(Dashbord))
	ioc.Register(new(Db))
	ioc.Register(new(Instance))
	ioc.Register(new(DbSqlExec))
	ioc.Register(new(DbSql))
	ioc.Register(new(DataSyncTask))
	ioc.Register(new(DbTransferTask))
}
