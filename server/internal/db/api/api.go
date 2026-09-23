package api

import "mayfly-go/pkg/ioc"

func InitIoc() {
	ioc.Register(new(Db))
	ioc.Register(new(Instance))
	ioc.Register(new(DbSQLExec))
	ioc.Register(new(DbSQL))
	ioc.Register(new(DataSyncTask))
	ioc.Register(new(DbTransferTask))
	ioc.Register(new(DbMask))
}
