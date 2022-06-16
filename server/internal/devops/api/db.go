package api

import (
	"fmt"
	"io/ioutil"
	"mayfly-go/internal/devops/api/form"
	"mayfly-go/internal/devops/api/vo"
	"mayfly-go/internal/devops/application"
	"mayfly-go/internal/devops/domain/entity"
	sysApplication "mayfly-go/internal/sys/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ctx"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils"
	"mayfly-go/pkg/ws"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Db struct {
	DbApp        application.Db
	DbSqlExecApp application.DbSqlExec
	MsgApp       sysApplication.Msg
	ProjectApp   application.Project
}

// @router /api/dbs [get]
func (d *Db) Dbs(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	m := &entity.Db{EnvId: uint64(ginx.QueryInt(g, "envId", 0)),
		ProjectId: uint64(ginx.QueryInt(g, "projectId", 0)),
	}
	m.CreatorId = rc.LoginAccount.Id
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
	dbId := GetDbId(rc.GinCtx)
	d.DbApp.Delete(dbId)
	// 删除该库的sql执行记录
	d.DbSqlExecApp.DeleteBy(&entity.DbSqlExec{DbId: dbId})
}

func (d *Db) TableInfos(rc *ctx.ReqCtx) {
	rc.ResData = d.DbApp.GetDbInstance(GetIdAndDb(rc.GinCtx)).GetTableInfos()
}

func (d *Db) TableIndex(rc *ctx.ReqCtx) {
	tn := rc.GinCtx.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")
	rc.ResData = d.DbApp.GetDbInstance(GetIdAndDb(rc.GinCtx)).GetTableIndex(tn)
}

func (d *Db) GetCreateTableDdl(rc *ctx.ReqCtx) {
	tn := rc.GinCtx.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")
	rc.ResData = d.DbApp.GetDbInstance(GetIdAndDb(rc.GinCtx)).GetCreateTableDdl(tn)
}

func (d *Db) ExecSql(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	form := &form.DbSqlExecForm{}
	ginx.BindJsonAndValid(g, form)

	id := GetDbId(g)
	db := form.Db
	dbInstance := d.DbApp.GetDbInstance(id, db)
	biz.ErrIsNilAppendErr(d.ProjectApp.CanAccess(rc.LoginAccount.Id, dbInstance.ProjectId), "%s")

	// 去除前后空格及换行符
	sql := strings.TrimFunc(form.Sql, func(r rune) bool {
		s := string(r)
		return s == " " || s == "\n"
	})

	rc.ReqParam = fmt.Sprintf("db: %d:%s | sql: %s", id, db, sql)

	biz.NotEmpty(sql, "sql不能为空")
	if strings.HasPrefix(sql, "SELECT") || strings.HasPrefix(sql, "select") || strings.HasPrefix(sql, "show") {
		colNames, res, err := dbInstance.SelectData(sql)
		biz.ErrIsNilAppendErr(err, "查询失败: %s")
		colAndRes := make(map[string]interface{})
		colAndRes["colNames"] = colNames
		colAndRes["res"] = res
		rc.ResData = colAndRes
	} else {
		// 根据执行sql，生成执行记录
		execRecord := d.DbSqlExecApp.GenExecLog(rc.LoginAccount, id, db, sql, dbInstance)

		rowsAffected, err := dbInstance.Exec(sql)
		biz.ErrIsNilAppendErr(err, "执行失败: %s")
		res := make([]map[string]string, 0)
		resData := make(map[string]string)
		resData["影响条数"] = fmt.Sprintf("%d", rowsAffected)
		res = append(res, resData)

		colAndRes := make(map[string]interface{})
		colAndRes["colNames"] = []string{"影响条数"}
		colAndRes["res"] = res

		rc.ResData = colAndRes
		// 保存sql执行记录
		if res[0]["影响条数"] > "0" {
			execRecord.Remark = form.Remark
			d.DbSqlExecApp.Save(execRecord)
		}
	}
}

// 执行sql文件
func (d *Db) ExecSqlFile(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	fileheader, err := g.FormFile("file")
	biz.ErrIsNilAppendErr(err, "读取sql文件失败: %s")

	// 读取sql文件并根据;切割sql语句
	file, _ := fileheader.Open()
	filename := fileheader.Filename
	bytes, _ := ioutil.ReadAll(file)
	sqlContent := string(bytes)
	sqls := strings.Split(sqlContent, ";")
	dbId, db := GetIdAndDb(g)

	go func() {
		db := d.DbApp.GetDbInstance(dbId, db)

		dbEntity := d.DbApp.GetById(dbId)
		dbInfo := fmt.Sprintf("于%s的%s环境", dbEntity.Name, dbEntity.Env)

		defer func() {
			if err := recover(); err != nil {
				switch t := err.(type) {
				case *biz.BizError:
					d.MsgApp.CreateAndSend(rc.LoginAccount, ws.ErrMsg("sql脚本执行失败", fmt.Sprintf("[%s]%s执行失败: [%s]", filename, dbInfo, t.Error())))
				}
			}
		}()

		biz.ErrIsNilAppendErr(d.ProjectApp.CanAccess(rc.LoginAccount.Id, db.ProjectId), "%s")

		for _, sql := range sqls {
			sql = strings.Trim(sql, " ")
			if sql == "" || sql == "\n" {
				continue
			}
			_, err := db.Exec(sql)
			if err != nil {
				d.MsgApp.CreateAndSend(rc.LoginAccount, ws.ErrMsg("sql脚本执行失败", fmt.Sprintf("[%s]%s执行失败: [%s]", filename, dbInfo, err.Error())))
				return
			}
		}
		d.MsgApp.CreateAndSend(rc.LoginAccount, ws.SuccessMsg("sql脚本执行成功", fmt.Sprintf("[%s]%s执行完成", filename, dbInfo)))
	}()
}

// @router /api/db/:dbId/t-metadata [get]
func (d *Db) TableMA(rc *ctx.ReqCtx) {
	dbi := d.DbApp.GetDbInstance(GetIdAndDb(rc.GinCtx))
	biz.ErrIsNilAppendErr(d.ProjectApp.CanAccess(rc.LoginAccount.Id, dbi.ProjectId), "%s")
	rc.ResData = dbi.GetTableMetedatas()
}

// @router /api/db/:dbId/c-metadata [get]
func (d *Db) ColumnMA(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	tn := g.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")

	dbi := d.DbApp.GetDbInstance(GetIdAndDb(rc.GinCtx))
	biz.ErrIsNilAppendErr(d.ProjectApp.CanAccess(rc.LoginAccount.Id, dbi.ProjectId), "%s")
	rc.ResData = dbi.GetColumnMetadatas(tn)
}

// @router /api/db/:dbId/hint-tables [get]
func (d *Db) HintTables(rc *ctx.ReqCtx) {
	dbi := d.DbApp.GetDbInstance(GetIdAndDb(rc.GinCtx))
	biz.ErrIsNilAppendErr(d.ProjectApp.CanAccess(rc.LoginAccount.Id, dbi.ProjectId), "%s")
	// 获取所有表
	tables := dbi.GetTableMetedatas()

	tableNames := make([]string, 0)
	for _, v := range tables {
		tableNames = append(tableNames, v["tableName"].(string))
	}
	// key = 表名，value = 列名数组
	res := make(map[string][]string)

	// 表为空，则直接返回
	if len(tableNames) == 0 {
		rc.ResData = res
		return
	}

	// 获取所有表下的所有列信息
	columnMds := dbi.GetColumnMetadatas(tableNames...)
	for _, v := range columnMds {
		tName := v["tableName"].(string)
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
	dbSql := &entity.DbSql{Type: dbSqlForm.Type, DbId: dbId, Name: dbSqlForm.Name, Db: dbSqlForm.Db}
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

// 获取所有保存的sql names
func (d *Db) GetSqlNames(rc *ctx.ReqCtx) {
	id, db := GetIdAndDb(rc.GinCtx)
	// 获取用于是否有该dbsql的保存记录，有则更改，否则新增
	dbSql := &entity.DbSql{Type: 1, DbId: id, Db: db}
	dbSql.CreatorId = rc.LoginAccount.Id
	var sqls []entity.DbSql
	model.ListBy(dbSql, &sqls, "id", "name")

	rc.ResData = sqls
}

// 删除保存的sql
func (d *Db) DeleteSql(rc *ctx.ReqCtx) {
	dbSql := &entity.DbSql{Type: 1, DbId: GetDbId(rc.GinCtx)}
	dbSql.CreatorId = rc.LoginAccount.Id
	dbSql.Name = rc.GinCtx.Query("name")

	model.DeleteByCondition(dbSql)

}

// @router /api/db/:dbId/sql [get]
func (d *Db) GetSql(rc *ctx.ReqCtx) {
	id, db := GetIdAndDb(rc.GinCtx)
	// 根据创建者id， 数据库id，以及sql模板名称查询保存的sql信息
	dbSql := &entity.DbSql{Type: 1, DbId: id, Db: db}
	dbSql.CreatorId = rc.LoginAccount.Id
	dbSql.Name = rc.GinCtx.Query("name")

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

func GetIdAndDb(g *gin.Context) (uint64, string) {
	db := g.Query("db")
	biz.NotEmpty(db, "db不能为空")
	return GetDbId(g), db
}
