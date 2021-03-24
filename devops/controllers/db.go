package controllers

import (
	"fmt"
	"mayfly-go/base"
	"mayfly-go/base/biz"
	"mayfly-go/base/ctx"
	"mayfly-go/base/model"
	"mayfly-go/devops/controllers/form"
	"mayfly-go/devops/controllers/vo"
	"mayfly-go/devops/db"
	"mayfly-go/devops/models"
	"strconv"
)

type DbController struct {
	base.Controller
}

// @router /api/dbs [get]
func (c *DbController) Dbs() {
	c.ReturnData(ctx.NewNoLogReqCtx(true), func(account *ctx.LoginAccount) interface{} {
		m := new([]models.Db)
		querySetter := model.QuerySetter(new(models.Db))
		return model.GetPage(querySetter, c.GetPageParam(), m, new([]vo.SelectDataDbVO))
	})
}

// @router /api/db/:dbId/select [get]
func (c *DbController) SelectData() {
	rc := ctx.NewReqCtx(true, "执行数据库查询语句")
	c.ReturnData(rc, func(account *ctx.LoginAccount) interface{} {
		selectSql := c.GetString("selectSql")
		rc.ReqParam = selectSql
		biz.NotEmpty(selectSql, "selectSql不能为空")
		res, err := db.GetDbInstance(c.GetDbId()).SelectData(selectSql)
		if err != nil {
			panic(biz.NewBizErr(fmt.Sprintf("查询失败: %s", err.Error())))
		}
		return res
	})
}

// @router /api/db/:dbId/exec-sql [post]
func (c *DbController) ExecSql() {
	c.ReturnData(ctx.NewReqCtx(true, "sql执行"), func(account *ctx.LoginAccount) interface{} {
		selectSql := c.GetString("sql")
		biz.NotEmpty(selectSql, "sql不能为空")
		num, err := db.GetDbInstance(c.GetDbId()).Exec(selectSql)
		if err != nil {
			panic(biz.NewBizErr(fmt.Sprintf("执行失败: %s", err.Error())))
		}
		return num
	})
}

// @router /api/db/:dbId/t-metadata [get]
func (c *DbController) TableMA() {
	c.ReturnData(ctx.NewNoLogReqCtx(true), func(account *ctx.LoginAccount) interface{} {
		return db.GetDbInstance(c.GetDbId()).GetTableMetedatas()
	})
}

// @router /api/db/:dbId/c-metadata [get]
func (c *DbController) ColumnMA() {
	c.ReturnData(ctx.NewNoLogReqCtx(true), func(account *ctx.LoginAccount) interface{} {
		tn := c.GetString("tableName")
		biz.NotEmpty(tn, "tableName不能为空")
		return db.GetDbInstance(c.GetDbId()).GetColumnMetadatas(tn)
	})
}

// @router /api/db/:dbId/hint-tables [get]
// 数据表及字段前端提示接口
func (c *DbController) HintTables() {
	c.ReturnData(ctx.NewNoLogReqCtx(true), func(account *ctx.LoginAccount) interface{} {
		dbi := db.GetDbInstance(c.GetDbId())
		tables := dbi.GetTableMetedatas()
		res := make(map[string][]string)
		for _, v := range tables {
			tableName := v["tableName"]
			columnMds := dbi.GetColumnMetadatas(tableName)
			columnNames := make([]string, len(columnMds))
			for i, v := range columnMds {
				comment := v["columnComment"]
				if comment != "" {
					columnNames[i] = v["columnName"] + " [" + comment + "]"
				} else {
					columnNames[i] = v["columnName"]
				}
			}
			res[tableName] = columnNames
		}
		return res
	})
}

// @router /api/db/:dbId/sql [post]
func (c *DbController) SaveSql() {
	rc := ctx.NewReqCtx(true, "保存sql内容")
	c.Operation(rc, func(account *ctx.LoginAccount) {
		dbSqlForm := &form.DbSqlSaveForm{}
		c.UnmarshalBodyAndValid(dbSqlForm)
		rc.ReqParam = dbSqlForm

		dbId := c.GetDbId()
		// 判断dbId是否存在
		err := model.GetById(new(models.Db), dbId)
		biz.BizErrIsNil(err, "该数据库信息不存在")

		// 获取用于是否有该dbsql的保存记录，有则更改，否则新增
		dbSql := &models.DbSql{Type: dbSqlForm.Type, DbId: dbId}
		dbSql.CreatorId = account.Id
		e := model.GetByCondition(dbSql)

		dbSql.SetBaseInfo(account)
		// 更新sql信息
		dbSql.Sql = dbSqlForm.Sql
		if e == nil {
			model.UpdateById(dbSql)
		} else {
			model.Insert(dbSql)
		}
	})
}

// @router /api/db/:dbId/sql [get]
func (c *DbController) GetSql() {
	c.ReturnData(ctx.NewNoLogReqCtx(true), func(account *ctx.LoginAccount) interface{} {
		// 获取用于是否有该dbsql的保存记录，有则更改，否则新增
		dbSql := &models.DbSql{Type: 1, DbId: c.GetDbId()}
		dbSql.CreatorId = account.Id
		e := model.GetByCondition(dbSql)
		if e != nil {
			return nil
		}
		return dbSql
	})
}

func (c *DbController) GetDbId() uint64 {
	dbId, _ := strconv.Atoi(c.Ctx.Input.Param(":dbId"))
	biz.IsTrue(dbId > 0, "dbId错误")
	return uint64(dbId)
}
