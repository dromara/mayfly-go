package api

import (
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
)

type DbSQL struct {
	dbSQLApp application.DbSQL `inject:"T"`
}

func (d *DbSQL) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 用户sql相关
		req.NewPost(":dbId/sql", d.SaveSQL),

		req.NewGet(":dbId/sql", d.GetSQL),

		req.NewDelete(":dbId/sql", d.DeleteSQL),

		req.NewGet(":dbId/sql-names", d.GetSQLNames),
	}

	return req.NewConfs("/dbs", reqs[:]...)
}

// @router /api/dbs/:dbId/sql [post]
func (d *DbSQL) SaveSQL(rc *req.Ctx) {
	dbSQLForm := rc.BindJson[form.DbSQLSaveForm]()
	rc.ReqParam = dbSQLForm

	dbId := getDbId(rc)

	account := rc.GetLoginAccount()
	// 获取用于是否有该dbsql的保存记录，有则更改，否则新增
	dbSQL := &entity.DbSQL{Type: dbSQLForm.Type, DbId: dbId, Name: dbSQLForm.Name, Db: dbSQLForm.Db}
	dbSQL.CreatorId = account.Id
	e := d.dbSQLApp.GetByCond(dbSQL)

	// 更新sql信息
	dbSQL.SQL = dbSQLForm.SQL
	if e == nil {
		d.dbSQLApp.UpdateById(rc.MetaCtx, dbSQL)
	} else {
		d.dbSQLApp.Insert(rc.MetaCtx, dbSQL)
	}
}

// 获取所有保存的sql names
func (d *DbSQL) GetSQLNames(rc *req.Ctx) {
	dbId := getDbId(rc)
	dbName := getDbName(rc)
	// 获取用于是否有该dbsql的保存记录，有则更改，否则新增
	dbSQL := &entity.DbSQL{Type: 1, DbId: dbId, Db: dbName}
	dbSQL.CreatorId = rc.GetLoginAccount().Id
	sqls, err := d.dbSQLApp.ListByCond(model.NewModelCond(dbSQL).Columns("id", "name"))
	biz.ErrIsNil(err)

	rc.ResData = sqls
}

// 删除保存的sql
func (d *DbSQL) DeleteSQL(rc *req.Ctx) {
	dbSQL := &entity.DbSQL{Type: 1, DbId: getDbId(rc)}
	dbSQL.CreatorId = rc.GetLoginAccount().Id
	dbSQL.Name = rc.Query("name")
	dbSQL.Db = rc.Query("db")

	biz.ErrIsNil(d.dbSQLApp.DeleteByCond(rc.MetaCtx, dbSQL))
}

// @router /api/dbs/:dbId/sql [get]
func (d *DbSQL) GetSQL(rc *req.Ctx) {
	dbId := getDbId(rc)
	dbName := getDbName(rc)
	// 根据创建者id， 数据库id，以及sql模板名称查询保存的sql信息
	dbSQL := &entity.DbSQL{Type: 1, DbId: dbId, Db: dbName}
	dbSQL.CreatorId = rc.GetLoginAccount().Id
	dbSQL.Name = rc.Query("name")

	e := d.dbSQLApp.GetByCond(dbSQL)
	if e != nil {
		return
	}
	rc.ResData = dbSQL
}
