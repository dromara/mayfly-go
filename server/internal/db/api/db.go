package api

import (
	"fmt"
	"io"
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	sysapp "mayfly-go/internal/sys/application"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ctx"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils"
	"mayfly-go/pkg/ws"
	"strconv"
	"strings"
	"time"

	b64 "encoding/base64"
	"github.com/gin-gonic/gin"
	"github.com/xwb1989/sqlparser"
)

type Db struct {
	DbApp        application.Db
	DbSqlExecApp application.DbSqlExec
	MsgApp       sysapp.Msg
	TagApp       tagapp.TagTree
}

const DEFAULT_ROW_SIZE = 1800

// @router /api/dbs [get]
func (d *Db) Dbs(rc *ctx.ReqCtx) {
	condition := new(entity.DbQuery)
	condition.TagPathLike = rc.GinCtx.Query("tagPath")

	// 不存在可访问标签id，即没有可操作数据
	tagIds := d.TagApp.ListTagIdByAccountId(rc.LoginAccount.Id)
	if len(tagIds) == 0 {
		rc.ResData = model.EmptyPageResult()
		return
	}
	condition.TagIds = tagIds
	rc.ResData = d.DbApp.GetPageList(condition, ginx.GetPageParam(rc.GinCtx), new([]vo.SelectDataDbVO))
}

func (d *Db) Save(rc *ctx.ReqCtx) {
	form := &form.DbForm{}
	ginx.BindJsonAndValid(rc.GinCtx, form)

	db := new(entity.Db)
	utils.Copy(db, form)

	// 密码解密，并使用解密后的赋值
	originPwd, err := utils.DefaultRsaDecrypt(form.Password, true)
	biz.ErrIsNilAppendErr(err, "解密密码错误: %s")
	db.Password = originPwd

	// 密码脱敏记录日志
	form.Password = "****"
	rc.ReqParam = form

	db.SetBaseInfo(rc.LoginAccount)
	d.DbApp.Save(db)
}

// 获取数据库实例密码，由于数据库是加密存储，故提供该接口展示原文密码
func (d *Db) GetDbPwd(rc *ctx.ReqCtx) {
	dbId := GetDbId(rc.GinCtx)
	dbEntity := d.DbApp.GetById(dbId, "Password")
	dbEntity.PwdDecrypt()
	rc.ResData = dbEntity.Password
}

// 获取数据库实例的所有数据库名
func (d *Db) GetDatabaseNames(rc *ctx.ReqCtx) {
	form := &form.DbForm{}
	ginx.BindJsonAndValid(rc.GinCtx, form)

	db := new(entity.Db)
	utils.Copy(db, form)

	// 密码解密，并使用解密后的赋值
	originPwd, err := utils.DefaultRsaDecrypt(form.Password, true)
	biz.ErrIsNilAppendErr(err, "解密密码错误: %s")
	db.Password = originPwd

	// 如果id不为空，并且密码为空则从数据库查询
	if form.Id != 0 && db.Password == "" {
		db = d.DbApp.GetById(form.Id)
	}
	rc.ResData = d.DbApp.GetDatabases(db)
}

func (d *Db) DeleteDb(rc *ctx.ReqCtx) {
	dbId := GetDbId(rc.GinCtx)
	d.DbApp.Delete(dbId)
	// 删除该库的sql执行记录
	d.DbSqlExecApp.DeleteBy(&entity.DbSqlExec{DbId: dbId})
}

func (d *Db) TableInfos(rc *ctx.ReqCtx) {
	rc.ResData = d.DbApp.GetDbInstance(GetIdAndDb(rc.GinCtx)).GetMeta().GetTableInfos()
}

func (d *Db) TableIndex(rc *ctx.ReqCtx) {
	tn := rc.GinCtx.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")
	rc.ResData = d.DbApp.GetDbInstance(GetIdAndDb(rc.GinCtx)).GetMeta().GetTableIndex(tn)
}

func (d *Db) GetCreateTableDdl(rc *ctx.ReqCtx) {
	tn := rc.GinCtx.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")
	rc.ResData = d.DbApp.GetDbInstance(GetIdAndDb(rc.GinCtx)).GetMeta().GetCreateTableDdl(tn)
}

func (d *Db) ExecSql(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	form := &form.DbSqlExecForm{}
	ginx.BindJsonAndValid(g, form)

	id := GetDbId(g)
	db := form.Db
	dbInstance := d.DbApp.GetDbInstance(id, db)
	biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.LoginAccount.Id, dbInstance.Info.TagPath), "%s")

	rc.ReqParam = fmt.Sprintf("%s -> %s", dbInstance.Info.GetLogDesc(), form.Sql)
	biz.NotEmpty(form.Sql, "sql不能为空")

	// 去除前后空格及换行符
	sqlBytes, err := b64.StdEncoding.DecodeString(strings.Replace(form.Sql, "MAGIC", "", 1))
	biz.ErrIsNilAppendErr(err, "sql base64解码错误: %s")
	sql := utils.StrTrimSpaceAndBr(string(sqlBytes))

	execReq := &application.DbSqlExecReq{
		DbId:         id,
		Db:           db,
		Remark:       form.Remark,
		DbInstance:   dbInstance,
		LoginAccount: rc.LoginAccount,
	}

	sqls := strings.Split(sql, ";\n")
	isMulti := len(sqls) > 1
	var execResAll *application.DbSqlExecRes
	for _, s := range sqls {
		s = utils.StrTrimSpaceAndBr(s)
		// 多条执行，如果有查询语句，则跳过
		if isMulti && strings.HasPrefix(strings.ToLower(s), "select") {
			continue
		}
		execReq.Sql = s
		execRes, err := d.DbSqlExecApp.Exec(execReq)
		biz.ErrIsNilAppendErr(err, "执行失败: %s")

		if execResAll == nil {
			execResAll = execRes
		} else {
			execResAll.Merge(execRes)
		}
	}

	colAndRes := make(map[string]interface{})
	colAndRes["colNames"] = execResAll.ColNames
	colAndRes["res"] = execResAll.Res
	rc.ResData = colAndRes
}

// 执行sql文件
func (d *Db) ExecSqlFile(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	fileheader, err := g.FormFile("file")
	biz.ErrIsNilAppendErr(err, "读取sql文件失败: %s")

	file, _ := fileheader.Open()
	filename := fileheader.Filename
	dbId, db := GetIdAndDb(g)

	rc.ReqParam = fmt.Sprintf("dbId: %d, db: %s, filename: %s", dbId, db, filename)

	go func() {
		db := d.DbApp.GetDbInstance(dbId, db)

		dbEntity := d.DbApp.GetById(dbId)
		dbInfo := fmt.Sprintf("于%s的%s环境", dbEntity.Name, dbEntity.TagPath)

		defer func() {
			if err := recover(); err != nil {
				switch t := err.(type) {
				case *biz.BizError:
					d.MsgApp.CreateAndSend(rc.LoginAccount, ws.ErrMsg("sql脚本执行失败", fmt.Sprintf("[%s]%s执行失败: [%s]", filename, dbInfo, t.Error())))
				}
			}
		}()

		biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.LoginAccount.Id, db.Info.TagPath), "%s")

		tokens := sqlparser.NewTokenizer(file)
		for {
			stmt, err := sqlparser.ParseNext(tokens)
			if err == io.EOF {
				break
			}
			sql := sqlparser.String(stmt)
			_, err = db.Exec(sql)
			if err != nil {
				d.MsgApp.CreateAndSend(rc.LoginAccount, ws.ErrMsg("sql脚本执行失败", fmt.Sprintf("[%s]%s执行失败: [%s]", filename, dbInfo, err.Error())))
				return
			}
		}
		d.MsgApp.CreateAndSend(rc.LoginAccount, ws.SuccessMsg("sql脚本执行成功", fmt.Sprintf("[%s]%s执行完成", filename, dbInfo)))
	}()
}

// 数据库dump
func (d *Db) DumpSql(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	dbId, db := GetIdAndDb(g)
	dumpType := g.Query("type")
	tablesStr := g.Query("tables")
	biz.NotEmpty(tablesStr, "请选择要导出的表")
	tables := strings.Split(tablesStr, ",")

	// 是否需要导出表结构
	needStruct := dumpType == "1" || dumpType == "3"
	// 是否需要导出数据
	needData := dumpType == "2" || dumpType == "3"

	dbInstance := d.DbApp.GetDbInstance(dbId, db)
	biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.LoginAccount.Id, dbInstance.Info.TagPath), "%s")

	now := time.Now()
	filename := fmt.Sprintf("%s.%s.sql", db, now.Format("200601021504"))
	g.Header("Content-Type", "application/octet-stream")
	g.Header("Content-Disposition", "attachment; filename="+filename)

	writer := g.Writer
	writer.WriteString("-- ----------------------------")
	writer.WriteString("\n-- 导出平台: mayfly-go")
	writer.WriteString(fmt.Sprintf("\n-- 导出时间: %s ", now.Format("2006-01-02 15:04:05")))
	writer.WriteString(fmt.Sprintf("\n-- 导出数据库: %s ", db))
	writer.WriteString("\n-- ----------------------------\n")

	dbmeta := d.DbApp.GetDbInstance(GetIdAndDb(rc.GinCtx)).GetMeta()
	for _, table := range tables {
		if needStruct {
			writer.WriteString(fmt.Sprintf("\n-- ----------------------------\n-- 表结构: %s \n-- ----------------------------\n", table))
			writer.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS `%s`;\n", table))
			writer.WriteString(dbmeta.GetCreateTableDdl(table)[0]["Create Table"].(string) + ";\n")
		}

		if !needData {
			continue
		}

		writer.WriteString(fmt.Sprintf("\n-- ----------------------------\n-- 表记录: %s \n-- ----------------------------\n", table))
		writer.WriteString("BEGIN;\n")

		countSql := fmt.Sprintf("SELECT COUNT(*) count FROM %s", table)
		_, countRes, _ := dbInstance.SelectData(countSql)
		// 查询出所有列信息总数，手动分页获取所有数据
		maCount := 0
		// 查询出所有列信息总数，手动分页获取所有数据
		if count64, is64 := countRes[0]["count"].(int64); is64 {
			maCount = int(count64)
		} else {
			maCount = countRes[0]["count"].(int)
		}
		// 计算需要查询的页数
		pageNum := maCount / DEFAULT_ROW_SIZE
		if maCount%DEFAULT_ROW_SIZE > 0 {
			pageNum++
		}

		var sqlTmp string
		switch dbInstance.Info.Type {
		case entity.DbTypeMysql:
			sqlTmp = "SELECT * FROM %s LIMIT %d, %d"
		case entity.DbTypePostgres:
			sqlTmp = "SELECT * FROM %s OFFSET %d LIMIT %d"
		}
		for index := 0; index < pageNum; index++ {
			sql := fmt.Sprintf(sqlTmp, table, index*DEFAULT_ROW_SIZE, DEFAULT_ROW_SIZE)
			columns, result, _ := dbInstance.SelectData(sql)

			insertSql := "INSERT INTO `%s` VALUES (%s);\n"
			for _, res := range result {
				var values []string
				for _, column := range columns {
					value := res[column]
					if value == nil {
						values = append(values, "NULL")
						continue
					}
					strValue, ok := value.(string)
					if ok {
						values = append(values, fmt.Sprintf("%#v", strValue))
					} else {
						values = append(values, utils.ToString(value))
					}
				}
				writer.WriteString(fmt.Sprintf(insertSql, table, strings.Join(values, ", ")))
			}
		}

		writer.WriteString("COMMIT;\n")
	}
	rc.NoRes = true

	rc.ReqParam = fmt.Sprintf("dbId: %d, db: %s, tables: %s, dumpType: %s", dbId, db, tablesStr, dumpType)
}

// @router /api/db/:dbId/t-metadata [get]
func (d *Db) TableMA(rc *ctx.ReqCtx) {
	dbi := d.DbApp.GetDbInstance(GetIdAndDb(rc.GinCtx))
	biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.LoginAccount.Id, dbi.Info.TagPath), "%s")
	rc.ResData = dbi.GetMeta().GetTables()
}

// @router /api/db/:dbId/c-metadata [get]
func (d *Db) ColumnMA(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	tn := g.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")

	dbi := d.DbApp.GetDbInstance(GetIdAndDb(rc.GinCtx))
	// biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.LoginAccount.Id, dbInstance.TagPath), "%s")
	rc.ResData = dbi.GetMeta().GetColumns(tn)
}

// @router /api/db/:dbId/hint-tables [get]
func (d *Db) HintTables(rc *ctx.ReqCtx) {
	dbi := d.DbApp.GetDbInstance(GetIdAndDb(rc.GinCtx))
	// biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.LoginAccount.Id, dbInstance.TagPath), "%s")

	dm := dbi.GetMeta()
	// 获取所有表
	tables := dm.GetTables()
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
	columnMds := dm.GetColumns(tableNames...)
	for _, v := range columnMds {
		tName := v["tableName"].(string)
		if res[tName] == nil {
			res[tName] = make([]string, 0)
		}

		columnName := fmt.Sprintf("%s  [%s]", v["columnName"], v["columnType"])
		comment := v["columnComment"]
		// 如果字段备注不为空，则加上备注信息
		if comment != nil && comment != "" {
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
