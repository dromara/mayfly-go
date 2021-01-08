package controllers

import (
	"fmt"
	"mayfly-go/base"
	"mayfly-go/base/ctx"
	"mayfly-go/base/model"
	"mayfly-go/controllers/form"
	"mayfly-go/controllers/vo"
	"mayfly-go/db"
	"mayfly-go/models"
	"strconv"
)

type DbController struct {
	base.Controller
}

// @router /api/dbs [get]
func (c *DbController) Dbs() {
	c.ReturnData(false, func(account *ctx.LoginAccount) interface{} {
		m := new([]models.Db)
		querySetter := model.QuerySetter(new(models.Db))
		return model.GetPage(querySetter, c.GetPageParam(), m, new([]vo.SelectDataDbVO))
	})
}

// @router /api/db/:dbId/select [get]
func (c *DbController) SelectData() {
	c.ReturnData(false, func(account *ctx.LoginAccount) interface{} {
		selectSql := c.GetString("selectSql")
		model.NotEmpty(selectSql, "selectSql不能为空")
		res, err := db.GetDbInstance(c.GetDbId()).SelectData(selectSql)
		if err != nil {
			panic(model.NewBizErr(fmt.Sprintf("查询失败: %s", err.Error())))
		}
		return res
	})
}

// @router /api/db/:dbId/exec-sql [post]
func (c *DbController) ExecSql() {
	c.ReturnData(false, func(account *ctx.LoginAccount) interface{} {
		selectSql := c.GetString("sql")
		model.NotEmpty(selectSql, "sql不能为空")
		num, err := db.GetDbInstance(c.GetDbId()).Exec(selectSql)
		if err != nil {
			panic(model.NewBizErr(fmt.Sprintf("执行失败: %s", err.Error())))
		}
		return num
	})
}

// @router /api/db/:dbId/t-metadata [get]
func (c *DbController) TableMA() {
	c.ReturnData(false, func(account *ctx.LoginAccount) interface{} {
		return db.GetDbInstance(c.GetDbId()).GetTableMetedatas()
	})
}

// @router /api/db/:dbId/c-metadata [get]
func (c *DbController) ColumnMA() {
	c.ReturnData(false, func(account *ctx.LoginAccount) interface{} {
		tn := c.GetString("tableName")
		model.NotEmpty(tn, "tableName不能为空")
		return db.GetDbInstance(c.GetDbId()).GetColumnMetadatas(tn)
	})
}

// @router /api/db/:dbId/hint-tables [get]
// 数据表及字段前端提示接口
func (c *DbController) HintTables() {
	c.ReturnData(false, func(account *ctx.LoginAccount) interface{} {
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
	c.Operation(true, func(account *ctx.LoginAccount) {
		dbSqlForm := &form.DbSqlSaveForm{}
		c.UnmarshalBodyAndValid(dbSqlForm)

		dbId := c.GetDbId()
		// 判断dbId是否存在
		err := model.GetById(new(models.Db), dbId)
		model.BizErrIsNil(err, "该数据库信息不存在")

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
	c.ReturnData(true, func(account *ctx.LoginAccount) interface{} {
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
	model.IsTrue(dbId > 0, "dbId错误")
	return uint64(dbId)
}
