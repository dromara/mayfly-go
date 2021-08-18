package apis

import (
	"fmt"
	"mayfly-go/base/biz"
	"mayfly-go/base/ctx"
	"mayfly-go/base/ginx"
	"mayfly-go/base/model"
	"mayfly-go/base/utils"
	"mayfly-go/server/devops/apis/form"
	"mayfly-go/server/devops/apis/vo"
	"mayfly-go/server/devops/application"
	"mayfly-go/server/devops/domain/entity"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Db struct {
	DbApp application.Db
}

// @router /api/dbs [get]
func (d *Db) Dbs(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	m := &entity.Db{EnvId: uint64(ginx.QueryInt(g, "envId", 0)),
		ProjectId: uint64(ginx.QueryInt(g, "projectId", 0)),
		Database:  g.Query("database"),
	}
	ginx.BindQuery(g, m)
	rc.ResData = d.DbApp.GetPageList(m, ginx.GetPageParam(rc.GinCtx), new([]vo.SelectDataDbVO))
}

func (d *Db) Save(rc *ctx.ReqCtx) {
	form := &form.DbForm{}
	ginx.BindJsonAndValid(rc.GinCtx, form)

	rc.ReqParam = form

	db := new(entity.Db)
	utils.Copy(db, form)
	db.SetBaseInfo(rc.LoginAccount)
	d.DbApp.Save(db)
}

func (d *Db) DeleteDb(rc *ctx.ReqCtx) {
	d.DbApp.Delete(uint64(ginx.PathParamInt(rc.GinCtx, "id")))
}

func (d *Db) TableInfos(rc *ctx.ReqCtx) {
	rc.ResData = d.DbApp.GetDbInstance(GetDbId(rc.GinCtx)).GetTableInfos()
}

func (d *Db) TableIndex(rc *ctx.ReqCtx) {
	tn := rc.GinCtx.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")
	rc.ResData = d.DbApp.GetDbInstance(GetDbId(rc.GinCtx)).GetTableIndex(tn)
}

func (d *Db) GetCreateTableDdl(rc *ctx.ReqCtx) {
	tn := rc.GinCtx.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")
	rc.ResData = d.DbApp.GetDbInstance(GetDbId(rc.GinCtx)).GetCreateTableDdl(tn)
}

// @router /api/db/:dbId/exec-sql [get]
func (d *Db) ExecSql(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	// 去除前后空格及换行符
	sql := strings.TrimFunc(g.Query("sql"), func(r rune) bool {
		s := string(r)
		return s == " " || s == "\n"
	})
	rc.ReqParam = sql

	biz.NotEmpty(sql, "sql不能为空")
	if strings.HasPrefix(sql, "SELECT") || strings.HasPrefix(sql, "select") {
		colNames, res, err := d.DbApp.GetDbInstance(GetDbId(g)).SelectData(sql)
		if err != nil {
			panic(biz.NewBizErr(fmt.Sprintf("查询失败: %s", err.Error())))
		}
		colAndRes := make(map[string]interface{})
		colAndRes["colNames"] = colNames
		colAndRes["res"] = res
		rc.ResData = colAndRes
	} else {
		rowsAffected, err := d.DbApp.GetDbInstance(GetDbId(g)).Exec(sql)
		if err != nil {
			panic(biz.NewBizErr(fmt.Sprintf("执行失败: %s", err.Error())))
		}
		res := make([]map[string]string, 0)
		resData := make(map[string]string)
		resData["影响条数"] = fmt.Sprintf("%d", rowsAffected)
		res = append(res, resData)

		colAndRes := make(map[string]interface{})
		colAndRes["colNames"] = []string{"影响条数"}
		colAndRes["res"] = res

		rc.ResData = colAndRes
	}
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
	// 获取所有表
	tables := dbi.GetTableMetedatas()

	tableNames := make([]string, 0)
	for _, v := range tables {
		tableNames = append(tableNames, v["tableName"])
	}
	// 获取所有表下的所有列信息
	columnMds := dbi.GetColumnMetadatas(tableNames...)
	// key = 表名，value = 列名数组
	res := make(map[string][]string)

	for _, v := range columnMds {
		tName := v["tableName"]
		if res[tName] == nil {
			res[tName] = make([]string, 0)
		}

		columnName := fmt.Sprintf("%s  [%s]", v["columnName"], v["columnType"])
		comment := v["columnComment"]
		// 如果字段备注不为空，则加上备注信息
		if comment != "" {
			columnName = fmt.Sprintf("%s[%s]", columnName, comment)
		}

		res[tName] = append(res[tName], columnName)
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
