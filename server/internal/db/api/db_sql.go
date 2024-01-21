package api

import (
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
)

type DbSql struct {
	DbSqlApp application.DbSql `inject:""`
}

// @router /api/db/:dbId/sql [post]
func (d *DbSql) SaveSql(rc *req.Ctx) {
	g := rc.GinCtx
	dbSqlForm := &form.DbSqlSaveForm{}
	ginx.BindJsonAndValid(g, dbSqlForm)
	rc.ReqParam = dbSqlForm

	dbId := getDbId(g)

	account := rc.GetLoginAccount()
	// 获取用于是否有该dbsql的保存记录，有则更改，否则新增
	dbSql := &entity.DbSql{Type: dbSqlForm.Type, DbId: dbId, Name: dbSqlForm.Name, Db: dbSqlForm.Db}
	dbSql.CreatorId = account.Id
	e := d.DbSqlApp.GetBy(dbSql)

	// 更新sql信息
	dbSql.Sql = dbSqlForm.Sql
	if e == nil {
		d.DbSqlApp.UpdateById(rc.MetaCtx, dbSql)
	} else {
		d.DbSqlApp.Insert(rc.MetaCtx, dbSql)
	}
}

// 获取所有保存的sql names
func (d *DbSql) GetSqlNames(rc *req.Ctx) {
	dbId := getDbId(rc.GinCtx)
	dbName := getDbName(rc.GinCtx)
	// 获取用于是否有该dbsql的保存记录，有则更改，否则新增
	dbSql := &entity.DbSql{Type: 1, DbId: dbId, Db: dbName}
	dbSql.CreatorId = rc.GetLoginAccount().Id
	var sqls []entity.DbSql
	d.DbSqlApp.ListByCond(dbSql, &sqls, "id", "name")

	rc.ResData = sqls
}

// 删除保存的sql
func (d *DbSql) DeleteSql(rc *req.Ctx) {
	dbSql := &entity.DbSql{Type: 1, DbId: getDbId(rc.GinCtx)}
	dbSql.CreatorId = rc.GetLoginAccount().Id
	dbSql.Name = rc.GinCtx.Query("name")
	dbSql.Db = rc.GinCtx.Query("db")

	biz.ErrIsNil(d.DbSqlApp.DeleteByCond(rc.MetaCtx, dbSql))
}

// @router /api/db/:dbId/sql [get]
func (d *DbSql) GetSql(rc *req.Ctx) {
	dbId := getDbId(rc.GinCtx)
	dbName := getDbName(rc.GinCtx)
	// 根据创建者id， 数据库id，以及sql模板名称查询保存的sql信息
	dbSql := &entity.DbSql{Type: 1, DbId: dbId, Db: dbName}
	dbSql.CreatorId = rc.GetLoginAccount().Id
	dbSql.Name = rc.GinCtx.Query("name")

	e := d.DbSqlApp.GetBy(dbSql)
	if e != nil {
		return
	}
	rc.ResData = dbSql
}
