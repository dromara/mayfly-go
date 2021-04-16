package controllers

import (
	"fmt"
	"mayfly-go/base/biz"
	"mayfly-go/base/ctx"
	"mayfly-go/base/ginx"
	"mayfly-go/base/model"
	"mayfly-go/devops/controllers/form"
	"mayfly-go/devops/controllers/vo"
	"mayfly-go/devops/db"
	"mayfly-go/devops/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

// @router /api/dbs [get]
func Dbs(rc *ctx.ReqCtx) {
	m := new([]models.Db)
	// querySetter := model.QuerySetter(new(models.Db))
	rc.ResData = model.GetPage(ginx.GetPageParam(rc.GinCtx), m, new([]vo.SelectDataDbVO))
}

// @router /api/db/:dbId/select [get]
func SelectData(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	selectSql := g.Query("selectSql")
	rc.ReqParam = selectSql
	biz.NotEmpty(selectSql, "selectSql不能为空")
	res, err := db.GetDbInstance(GetDbId(g)).SelectData(selectSql)
	if err != nil {
		panic(biz.NewBizErr(fmt.Sprintf("查询失败: %s", err.Error())))
	}
	rc.ResData = res
}

// @router /api/db/:dbId/exec-sql [post]
func ExecSql(g *gin.Context) {
	rc := ctx.NewReqCtxWithGin(g).WithLog(ctx.NewLogInfo("sql执行"))
	rc.Handle(func(rc *ctx.ReqCtx) {
		selectSql := g.Query("sql")
		biz.NotEmpty(selectSql, "sql不能为空")
		num, err := db.GetDbInstance(GetDbId(g)).Exec(selectSql)
		if err != nil {
			panic(biz.NewBizErr(fmt.Sprintf("执行失败: %s", err.Error())))
		}
		rc.ResData = num
	})
}

// @router /api/db/:dbId/t-metadata [get]
func TableMA(rc *ctx.ReqCtx) {
	rc.ResData = db.GetDbInstance(GetDbId(rc.GinCtx)).GetTableMetedatas()
}

// @router /api/db/:dbId/c-metadata [get]
func ColumnMA(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	tn := g.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")
	rc.ResData = db.GetDbInstance(GetDbId(g)).GetColumnMetadatas(tn)
}

// @router /api/db/:dbId/hint-tables [get]
func HintTables(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	dbi := db.GetDbInstance(GetDbId(g))
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
	rc.ResData = res
}

// @router /api/db/:dbId/sql [post]
func SaveSql(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	account := rc.LoginAccount
	dbSqlForm := &form.DbSqlSaveForm{}
	ginx.BindJsonAndValid(g, dbSqlForm)
	rc.ReqParam = dbSqlForm

	dbId := GetDbId(g)
	// 判断dbId是否存在
	err := model.GetById(new(models.Db), dbId)
	biz.BizErrIsNil(err, "该数据库信息不存在")

	// 获取用于是否有该dbsql的保存记录，有则更改，否则新增
	dbSql := &models.DbSql{Type: dbSqlForm.Type, DbId: dbId}
	dbSql.CreatorId = account.Id
	e := model.GetBy(dbSql)

	dbSql.SetBaseInfo(account)
	// 更新sql信息
	dbSql.Sql = dbSqlForm.Sql
	if e == nil {
		model.UpdateById(dbSql)
	} else {
		model.Insert(dbSql)
	}
}

// @router /api/db/:dbId/sql [get]
func GetSql(rc *ctx.ReqCtx) {
	// 获取用于是否有该dbsql的保存记录，有则更改，否则新增
	dbSql := &models.DbSql{Type: 1, DbId: GetDbId(rc.GinCtx)}
	dbSql.CreatorId = rc.LoginAccount.Id
	e := model.GetBy(dbSql)
	if e != nil {
		return
	}
	rc.ResData = dbSql
}

func GetDbId(g *gin.Context) uint64 {
	dbId, _ := strconv.Atoi(g.Param("dbId"))
	biz.IsTrue(dbId > 0, "dbId错误")
	return uint64(dbId)
}
