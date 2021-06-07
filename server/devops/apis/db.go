package apis

import (
	"fmt"
	"mayfly-go/base/biz"
	"mayfly-go/base/ctx"
	"mayfly-go/base/ginx"
	"mayfly-go/base/model"
	"mayfly-go/server/devops/apis/form"
	"mayfly-go/server/devops/apis/vo"
	"mayfly-go/server/devops/application"
	"mayfly-go/server/devops/domain/entity"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Db struct {
	DbApp application.IDb
}

// @router /api/dbs [get]
func (d *Db) Dbs(rc *ctx.ReqCtx) {
	m := new(entity.Db)
	rc.ResData = d.DbApp.GetPageList(m, ginx.GetPageParam(rc.GinCtx), new([]vo.SelectDataDbVO))
}

// @router /api/db/:dbId/select [get]
func (d *Db) SelectData(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	// 去除前后空格及换行符
	selectSql := strings.TrimFunc(g.Query("selectSql"), func(r rune) bool {
		s := string(r)
		return s == " " || s == "\n"
	})
	rc.ReqParam = selectSql

	biz.NotEmpty(selectSql, "selectSql不能为空")
	res, err := d.DbApp.GetDbInstance(GetDbId(g)).SelectData(selectSql)
	if err != nil {
		panic(biz.NewBizErr(fmt.Sprintf("查询失败: %s", err.Error())))
	}
	rc.ResData = res
}

// @router /api/db/:dbId/exec-sql [post]
func (d *Db) ExecSql(g *gin.Context) {
	rc := ctx.NewReqCtxWithGin(g).WithLog(ctx.NewLogInfo("sql执行"))
	rc.Handle(func(rc *ctx.ReqCtx) {
		selectSql := g.Query("sql")
		biz.NotEmpty(selectSql, "sql不能为空")
		num, err := d.DbApp.GetDbInstance(GetDbId(rc.GinCtx)).Exec(selectSql)
		if err != nil {
			panic(biz.NewBizErr(fmt.Sprintf("执行失败: %s", err.Error())))
		}
		rc.ResData = num
	})
}

// @router /api/db/:dbId/t-metadata [get]
func (d *Db) TableMA(rc *ctx.ReqCtx) {
	rc.ResData = d.DbApp.GetDbInstance(GetDbId(rc.GinCtx)).GetTableMetedatas()
}

// @router /api/db/:dbId/c-metadata [get]
func (d *Db) ColumnMA(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	tn := g.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")
	rc.ResData = d.DbApp.GetDbInstance(GetDbId(rc.GinCtx)).GetColumnMetadatas(tn)
}

// @router /api/db/:dbId/hint-tables [get]
func (d *Db) HintTables(rc *ctx.ReqCtx) {
	dbi := d.DbApp.GetDbInstance(GetDbId(rc.GinCtx))
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
func (d *Db) SaveSql(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	account := rc.LoginAccount
	dbSqlForm := &form.DbSqlSaveForm{}
	ginx.BindJsonAndValid(g, dbSqlForm)
	rc.ReqParam = dbSqlForm

	dbId := GetDbId(g)
	// 判断dbId是否存在
	err := model.GetById(new(entity.Db), dbId)
	biz.ErrIsNil(err, "该数据库信息不存在")

	// 获取用于是否有该dbsql的保存记录，有则更改，否则新增
	dbSql := &entity.DbSql{Type: dbSqlForm.Type, DbId: dbId}
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
func (d *Db) GetSql(rc *ctx.ReqCtx) {
	// 获取用于是否有该dbsql的保存记录，有则更改，否则新增
	dbSql := &entity.DbSql{Type: 1, DbId: GetDbId(rc.GinCtx)}
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
