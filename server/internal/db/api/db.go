package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/kanzihuang/vitess/go/vt/sqlparser"
	"io"
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	msgapp "mayfly-go/internal/msg/application"
	msgdto "mayfly-go/internal/msg/application/dto"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/utils/uniqueid"
	"mayfly-go/pkg/ws"
	"strconv"
	"strings"
	"time"
)

type Db struct {
	InstanceApp  application.Instance
	DbApp        application.Db
	DbSqlExecApp application.DbSqlExec
	MsgApp       msgapp.Msg
	TagApp       tagapp.TagTree
}

// @router /api/dbs [get]
func (d *Db) Dbs(rc *req.Ctx) {
	queryCond, page := ginx.BindQueryAndPage[*entity.DbQuery](rc.GinCtx, new(entity.DbQuery))

	// 不存在可访问标签id，即没有可操作数据
	tagIds := d.TagApp.ListTagIdByAccountId(rc.LoginAccount.Id)
	if len(tagIds) == 0 {
		rc.ResData = model.EmptyPageResult[any]()
		return
	}

	queryCond.TagIds = tagIds
	rc.ResData = d.DbApp.GetPageList(queryCond, page, new([]vo.SelectDataDbVO))
}

func (d *Db) DbTags(rc *req.Ctx) {
	rc.ResData = d.TagApp.ListTagByAccountIdAndResource(rc.LoginAccount.Id, new(entity.Db))
}

func (d *Db) Save(rc *req.Ctx) {
	form := &form.DbForm{}
	db := ginx.BindJsonAndCopyTo[*entity.Db](rc.GinCtx, form, new(entity.Db))

	rc.ReqParam = form

	db.SetBaseInfo(rc.LoginAccount)
	d.DbApp.Save(db)
}

func (d *Db) DeleteDb(rc *req.Ctx) {
	idsStr := ginx.PathParam(rc.GinCtx, "dbId")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		dbId := uint64(value)
		d.DbApp.Delete(dbId)
		// 删除该库的sql执行记录
		d.DbSqlExecApp.DeleteBy(&entity.DbSqlExec{DbId: dbId})
	}
}

func (d *Db) getDbConnection(g *gin.Context) *application.DbConnection {
	return d.DbApp.GetDbConnection(getDbId(g), getDbName(g))
}

func (d *Db) TableInfos(rc *req.Ctx) {
	rc.ResData = d.getDbConnection(rc.GinCtx).GetMeta().GetTableInfos()
}

func (d *Db) TableIndex(rc *req.Ctx) {
	tn := rc.GinCtx.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")
	rc.ResData = d.getDbConnection(rc.GinCtx).GetMeta().GetTableIndex(tn)
}

func (d *Db) GetCreateTableDdl(rc *req.Ctx) {
	tn := rc.GinCtx.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")
	rc.ResData = d.getDbConnection(rc.GinCtx).GetMeta().GetCreateTableDdl(tn)
}

func (d *Db) ExecSql(rc *req.Ctx) {
	g := rc.GinCtx
	form := &form.DbSqlExecForm{}
	ginx.BindJsonAndValid(g, form)

	dbId := getDbId(g)
	dbConn := d.DbApp.GetDbConnection(dbId, form.Db)
	biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.LoginAccount.Id, dbConn.Info.TagPath), "%s")

	rc.ReqParam = fmt.Sprintf("%s\n-> %s", dbConn.Info.GetLogDesc(), form.Sql)
	biz.NotEmpty(form.Sql, "sql不能为空")

	// 去除前后空格及换行符
	sql := stringx.TrimSpaceAndBr(form.Sql)

	execReq := &application.DbSqlExecReq{
		DbId:         dbId,
		Db:           form.Db,
		Remark:       form.Remark,
		DbConn:       dbConn,
		LoginAccount: rc.LoginAccount,
	}

	sqls, err := sqlparser.SplitStatementToPieces(sql)
	biz.ErrIsNil(err, "SQL解析错误,请检查您的执行SQL")
	isMulti := len(sqls) > 1
	var execResAll *application.DbSqlExecRes
	for _, s := range sqls {
		s = stringx.TrimSpaceAndBr(s)
		// 多条执行，如果有查询语句，则跳过
		if isMulti && strings.HasPrefix(strings.ToLower(s), "select") {
			continue
		}
		execReq.Sql = s
		execRes, err := d.DbSqlExecApp.Exec(execReq)
		if err != nil {
			biz.ErrIsNilAppendErr(err, fmt.Sprintf("[%s] -> 执行失败: ", s)+"%s")
		}

		if execResAll == nil {
			execResAll = execRes
		} else {
			execResAll.Merge(execRes)
		}
	}

	colAndRes := make(map[string]any)
	colAndRes["colNames"] = execResAll.ColNames
	colAndRes["res"] = execResAll.Res
	rc.ResData = colAndRes
}

// progressCategory sql文件执行进度消息类型
const progressCategory = "execSqlFileProgress"

// progressMsg sql文件执行进度消息
type progressMsg struct {
	Id                 uint64 `json:"id"`
	SqlFileName        string `json:"sqlFileName"`
	ExecutedStatements int    `json:"executedStatements"`
	Terminated         bool   `json:"terminated"`
}

// 执行sql文件
func (d *Db) ExecSqlFile(rc *req.Ctx) {
	g := rc.GinCtx
	multipart, err := g.Request.MultipartReader()
	biz.ErrIsNilAppendErr(err, "读取sql文件失败: %s")
	file, err := multipart.NextPart()
	biz.ErrIsNilAppendErr(err, "读取sql文件失败: %s")
	defer file.Close()
	filename := file.FileName()
	dbId := getDbId(g)
	dbName := getDbName(g)

	dbConn := d.DbApp.GetDbConnection(dbId, dbName)
	biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.LoginAccount.Id, dbConn.Info.TagPath), "%s")
	rc.ReqParam = fmt.Sprintf("filename: %s -> %s", filename, dbConn.Info.GetLogDesc())

	defer func() {
		var errInfo string
		switch t := recover().(type) {
		case error:
			errInfo = t.Error()
		case string:
			errInfo = t
		}
		if len(errInfo) > 0 {
			d.MsgApp.CreateAndSend(rc.LoginAccount, msgdto.ErrSysMsg("sql脚本执行失败", fmt.Sprintf("[%s][%s]执行失败: [%s]", filename, dbConn.Info.GetLogDesc(), errInfo)))
		}
	}()

	execReq := &application.DbSqlExecReq{
		DbId:         dbId,
		Db:           dbName,
		Remark:       filename,
		DbConn:       dbConn,
		LoginAccount: rc.LoginAccount,
	}

	progressId := uniqueid.IncrementID()
	executedStatements := 0
	defer ws.SendJsonMsg(rc.LoginAccount.Id, msgdto.InfoSysMsg("sql脚本执行进度", &progressMsg{
		Id:                 progressId,
		SqlFileName:        filename,
		ExecutedStatements: executedStatements,
		Terminated:         true,
	}).WithCategory(progressCategory))

	var sql string
	tokenizer := sqlparser.NewReaderTokenizer(file, sqlparser.WithCacheInBuffer())
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ws.SendJsonMsg(rc.LoginAccount.Id, msgdto.InfoSysMsg("sql脚本执行进度", &progressMsg{
				Id:                 progressId,
				SqlFileName:        filename,
				ExecutedStatements: executedStatements,
				Terminated:         false,
			}).WithCategory(progressCategory))
		default:
		}
		sql, err = sqlparser.SplitNext(tokenizer)
		if err == io.EOF {
			break
		}
		if err != nil {
			d.MsgApp.CreateAndSend(rc.LoginAccount, msgdto.ErrSysMsg("sql脚本解析失败", fmt.Sprintf("[%s][%s] 解析SQL失败: [%s]", filename, dbConn.Info.GetLogDesc(), err.Error())))
			return
		}
		const prefixUse = "use "
		const prefixUSE = "USE "
		if strings.HasPrefix(sql, prefixUSE) || strings.HasPrefix(sql, prefixUse) {
			var stmt sqlparser.Statement
			stmt, err = sqlparser.Parse(sql)
			if err != nil {
				d.MsgApp.CreateAndSend(rc.LoginAccount, msgdto.ErrSysMsg("sql脚本解析失败", fmt.Sprintf("[%s][%s] 解析SQL失败: [%s]", filename, dbConn.Info.GetLogDesc(), err.Error())))
			}
			stmtUse, ok := stmt.(*sqlparser.Use)
			if !ok {
				d.MsgApp.CreateAndSend(rc.LoginAccount, msgdto.ErrSysMsg("sql脚本解析失败", fmt.Sprintf("[%s][%s] 解析SQL失败: [%s]", filename, dbConn.Info.GetLogDesc(), sql)))
			}
			dbConn = d.DbApp.GetDbConnection(dbId, stmtUse.DBName.String())
			biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.LoginAccount.Id, dbConn.Info.TagPath), "%s")
			execReq.DbConn = dbConn
		}
		// 需要记录执行记录
		const maxRecordStatements = 64
		if executedStatements < maxRecordStatements {
			execReq.Sql = sql
			_, err = d.DbSqlExecApp.Exec(execReq)
		} else {
			_, err = dbConn.Exec(sql)
		}

		if err != nil {
			d.MsgApp.CreateAndSend(rc.LoginAccount, msgdto.ErrSysMsg("sql脚本执行失败", fmt.Sprintf("[%s][%s] -> sql=[%s] 执行失败: [%s]", filename, dbConn.Info.GetLogDesc(), sql, err.Error())))
			return
		}
		executedStatements++
	}
	d.MsgApp.CreateAndSend(rc.LoginAccount, msgdto.SuccessSysMsg("sql脚本执行成功", fmt.Sprintf("sql脚本执行完成：%s", rc.ReqParam)))
}

// 数据库dump
func (d *Db) DumpSql(rc *req.Ctx) {
	g := rc.GinCtx
	dbId := getDbId(g)
	dbNamesStr := g.Query("db")
	dumpType := g.Query("type")
	tablesStr := g.Query("tables")
	extName := g.Query("extName")
	switch extName {
	case ".gz", ".gzip", "gz", "gzip":
		extName = ".gz"
	default:
		extName = ""
	}

	// 是否需要导出表结构
	needStruct := dumpType == "1" || dumpType == "3"
	// 是否需要导出数据
	needData := dumpType == "2" || dumpType == "3"

	db := d.DbApp.GetById(dbId)
	biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.LoginAccount.Id, db.TagPath), "%s")

	now := time.Now()
	filename := fmt.Sprintf("%s.%s.sql%s", db.Name, now.Format("20060102150405"), extName)
	g.Header("Content-Type", "application/octet-stream")
	g.Header("Content-Disposition", "attachment; filename="+filename)
	if extName != ".gz" {
		g.Header("Content-Encoding", "gzip")
	}

	var dbNames, tables []string
	if len(dbNamesStr) > 0 {
		dbNames = strings.Split(dbNamesStr, ",")
	}
	if len(dbNames) == 1 && len(tablesStr) > 0 {
		tables = strings.Split(tablesStr, ",")
	}
	writer := newGzipWriter(g.Writer)
	defer func() {
		var msg string
		if err := recover(); err != nil {
			switch t := err.(type) {
			case error:
				msg = t.Error()
			}
		}
		if len(msg) > 0 {
			msg = "数据库导出失败: " + msg
			writer.WriteString(msg)
			d.MsgApp.CreateAndSend(rc.LoginAccount, msgdto.ErrSysMsg("数据库导出失败", msg))
		}
		writer.Close()
	}()
	for _, dbName := range dbNames {
		d.dumpDb(writer, dbId, dbName, tables, needStruct, needData, len(dbNames) > 1)
	}

	rc.ReqParam = collx.Kvs("db", db, "databases", dbNamesStr, "tables", tablesStr, "dumpType", dumpType)
}

func (d *Db) dumpDb(writer *gzipWriter, dbId uint64, dbName string, tables []string, needStruct bool, needData bool, switchDb bool) {
	dbConn := d.DbApp.GetDbConnection(dbId, dbName)
	writer.WriteString("\n-- ----------------------------")
	writer.WriteString("\n-- 导出平台: mayfly-go")
	writer.WriteString(fmt.Sprintf("\n-- 导出时间: %s ", time.Now().Format("2006-01-02 15:04:05")))
	writer.WriteString(fmt.Sprintf("\n-- 导出数据库: %s ", dbName))
	writer.WriteString("\n-- ----------------------------\n")

	if switchDb {
		switch dbConn.Info.Type {
		case entity.DbTypeMysql:
			writer.WriteString(fmt.Sprintf("USE %s;\n", entity.DbTypeMysql.QuoteIdentifier(dbName)))
		default:
			biz.IsTrue(false, "同时导出多个数据库，数据库类型必须为 %s", entity.DbTypeMysql)
		}
	}
	writer.WriteString(dbConn.Info.Type.StmtSetForeignKeyChecks(false))

	dbMeta := dbConn.GetMeta()
	if len(tables) == 0 {
		ti := dbMeta.GetTableInfos()
		tables = make([]string, len(ti))
		for i, table := range ti {
			tables[i] = table.TableName
		}
	}

	for _, table := range tables {
		writer.TryFlush()
		quotedTable := dbConn.Info.Type.QuoteIdentifier(table)
		if needStruct {
			writer.WriteString(fmt.Sprintf("\n-- ----------------------------\n-- 表结构: %s \n-- ----------------------------\n", table))
			writer.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", quotedTable))
			writer.WriteString(dbMeta.GetCreateTableDdl(table) + ";\n")
		}
		if !needData {
			continue
		}
		writer.WriteString(fmt.Sprintf("\n-- ----------------------------\n-- 表记录: %s \n-- ----------------------------\n", table))
		writer.WriteString("BEGIN;\n")
		insertSql := "INSERT INTO %s VALUES (%s);\n"
		dbMeta.WalkTableRecord(table, func(record map[string]any, columns []string) {
			var values []string
			writer.TryFlush()
			for _, column := range columns {
				value := record[column]
				if value == nil {
					values = append(values, "NULL")
					continue
				}
				strValue, ok := value.(string)
				if ok {
					strValue = dbConn.Info.Type.QuoteLiteral(strValue)
					values = append(values, strValue)
				} else {
					values = append(values, stringx.AnyToStr(value))
				}
			}
			writer.WriteString(fmt.Sprintf(insertSql, quotedTable, strings.Join(values, ", ")))
		})
		writer.WriteString("COMMIT;\n")
	}
	writer.WriteString(dbConn.Info.Type.StmtSetForeignKeyChecks(true))
}

// @router /api/db/:dbId/t-metadata [get]
func (d *Db) TableMA(rc *req.Ctx) {
	dbi := d.getDbConnection(rc.GinCtx)
	rc.ResData = dbi.GetMeta().GetTables()
}

// @router /api/db/:dbId/c-metadata [get]
func (d *Db) ColumnMA(rc *req.Ctx) {
	g := rc.GinCtx
	tn := g.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")

	dbi := d.getDbConnection(rc.GinCtx)
	rc.ResData = dbi.GetMeta().GetColumns(tn)
}

// @router /api/db/:dbId/hint-tables [get]
func (d *Db) HintTables(rc *req.Ctx) {
	dbi := d.getDbConnection(rc.GinCtx)

	dm := dbi.GetMeta()
	// 获取所有表
	tables := dm.GetTables()
	tableNames := make([]string, 0)
	for _, v := range tables {
		tableNames = append(tableNames, v.TableName)
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
		tName := v.TableName
		if res[tName] == nil {
			res[tName] = make([]string, 0)
		}

		columnName := fmt.Sprintf("%s  [%s]", v.ColumnName, v.ColumnType)
		comment := v.ColumnComment
		// 如果字段备注不为空，则加上备注信息
		if comment != "" {
			columnName = fmt.Sprintf("%s[%s]", columnName, comment)
		}

		res[tName] = append(res[tName], columnName)
	}
	rc.ResData = res
}

// @router /api/db/:dbId/sql [post]
func (d *Db) SaveSql(rc *req.Ctx) {
	g := rc.GinCtx
	account := rc.LoginAccount
	dbSqlForm := &form.DbSqlSaveForm{}
	ginx.BindJsonAndValid(g, dbSqlForm)
	rc.ReqParam = dbSqlForm

	dbId := getDbId(g)
	// 判断dbId是否存在
	err := gormx.GetById(new(entity.Db), dbId)
	biz.ErrIsNil(err, "该数据库信息不存在")

	// 获取用于是否有该dbsql的保存记录，有则更改，否则新增
	dbSql := &entity.DbSql{Type: dbSqlForm.Type, DbId: dbId, Name: dbSqlForm.Name, Db: dbSqlForm.Db}
	dbSql.CreatorId = account.Id
	e := gormx.GetBy(dbSql)

	dbSql.SetBaseInfo(account)
	// 更新sql信息
	dbSql.Sql = dbSqlForm.Sql
	if e == nil {
		gormx.UpdateById(dbSql)
	} else {
		gormx.Insert(dbSql)
	}
}

// 获取所有保存的sql names
func (d *Db) GetSqlNames(rc *req.Ctx) {
	dbId := getDbId(rc.GinCtx)
	dbName := getDbName(rc.GinCtx)
	// 获取用于是否有该dbsql的保存记录，有则更改，否则新增
	dbSql := &entity.DbSql{Type: 1, DbId: dbId, Db: dbName}
	dbSql.CreatorId = rc.LoginAccount.Id
	var sqls []entity.DbSql
	gormx.ListBy(dbSql, &sqls, "id", "name")

	rc.ResData = sqls
}

// 删除保存的sql
func (d *Db) DeleteSql(rc *req.Ctx) {
	dbSql := &entity.DbSql{Type: 1, DbId: getDbId(rc.GinCtx)}
	dbSql.CreatorId = rc.LoginAccount.Id
	dbSql.Name = rc.GinCtx.Query("name")
	dbSql.Db = rc.GinCtx.Query("db")

	gormx.DeleteByCondition(dbSql)

}

// @router /api/db/:dbId/sql [get]
func (d *Db) GetSql(rc *req.Ctx) {
	dbId := getDbId(rc.GinCtx)
	dbName := getDbName(rc.GinCtx)
	// 根据创建者id， 数据库id，以及sql模板名称查询保存的sql信息
	dbSql := &entity.DbSql{Type: 1, DbId: dbId, Db: dbName}
	dbSql.CreatorId = rc.LoginAccount.Id
	dbSql.Name = rc.GinCtx.Query("name")

	e := gormx.GetBy(dbSql)
	if e != nil {
		return
	}
	rc.ResData = dbSql
}

func getDbId(g *gin.Context) uint64 {
	dbId, _ := strconv.Atoi(g.Param("dbId"))
	biz.IsTrue(dbId > 0, "dbId错误")
	return uint64(dbId)
}

func getDbName(g *gin.Context) string {
	db := g.Query("db")
	biz.NotEmpty(db, "db不能为空")
	return db
}
