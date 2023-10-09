package api

import (
	"bufio"
	"fmt"
	"github.com/lib/pq"
	"io"
	"mayfly-go/pkg/logx"

	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	msgapp "mayfly-go/internal/msg/application"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/ws"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kanzihuang/vitess/go/vt/sqlparser"
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
	dbName := g.Query("db")
	biz.NotEmpty(dbName, "db不能为空")
	return d.DbApp.GetDbConnection(getDbId(g), dbName)
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

// 执行sql文件
func (d *Db) ExecSqlFile(rc *req.Ctx) {
	g := rc.GinCtx
	fileheader, err := g.FormFile("file")
	biz.ErrIsNilAppendErr(err, "读取sql文件失败: %s")

	file, _ := fileheader.Open()
	filename := fileheader.Filename
	dbId := getDbId(g)
	dbName := getDbName(g)

	dbConn := d.getDbConnection(rc.GinCtx)
	biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.LoginAccount.Id, dbConn.Info.TagPath), "%s")
	rc.ReqParam = fmt.Sprintf("%s -> filename: %s", dbConn.Info.GetLogDesc(), filename)

	logExecRecord := true
	// 如果执行sql文件大于该值则不记录sql执行记录
	if fileheader.Size > 50*1024 {
		logExecRecord = false
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				var errInfo string
				switch t := err.(type) {
				case error:
					errInfo = t.Error()
				case string:
					errInfo = t
				}
				if len(errInfo) > 0 {
					d.MsgApp.CreateAndSend(rc.LoginAccount, ws.ErrSysMsg("sql脚本执行失败", fmt.Sprintf("[%s]%s执行失败: [%s]", filename, dbConn.Info.GetLogDesc(), errInfo)))
				}
			}
		}()

		execReq := &application.DbSqlExecReq{
			DbId:         dbId,
			Db:           dbName,
			Remark:       fileheader.Filename,
			DbConn:       dbConn,
			LoginAccount: rc.LoginAccount,
		}

		var dbNameExec string
		tokenizer := sqlparser.NewReaderTokenizer(file)
		for {
			stmt, err := sqlparser.ParseNext(tokenizer)
			if err == io.EOF {
				break
			}
			if err != nil {
				d.MsgApp.CreateAndSend(rc.LoginAccount, ws.ErrSysMsg("sql脚本执行失败", fmt.Sprintf("[%s][%s] 解析SQL失败: [%s]", filename, dbConn.Info.GetLogDesc(), err.Error())))
				return
			}
			sql := sqlparser.String(stmt)
			if strings.HasPrefix(sql, "use ") {
				dbNameExec = sql[4:]
			}
			if strings.HasPrefix(sql, "create table ") {
				logx.Info(fmt.Sprintf("exec sql in database %s: %s", dbNameExec, sql))
			}
			execReq.Sql = sql
			// 需要记录执行记录
			if logExecRecord {
				_, err = d.DbSqlExecApp.Exec(execReq)
			} else {
				_, err = dbConn.Exec(sql)
			}

			if err != nil {
				d.MsgApp.CreateAndSend(rc.LoginAccount, ws.ErrSysMsg("sql脚本执行失败", fmt.Sprintf("[%s][%s] -> sql=[%s] 执行失败: [%s]", filename, dbConn.Info.GetLogDesc(), sql, err.Error())))
				return
			}
		}
		d.MsgApp.CreateAndSend(rc.LoginAccount, ws.SuccessSysMsg("sql脚本执行成功", fmt.Sprintf("[%s]执行完成 -> %s", filename, dbConn.Info.GetLogDesc())))
	}()
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
			d.MsgApp.CreateAndSend(rc.LoginAccount, ws.ErrSysMsg("数据库导出失败", msg))
		}
		writer.Close()
	}()
	for _, dbName := range dbNames {
		d.dumpDb(writer, dbId, dbName, tables, needStruct, needData, len(dbNames) > 1)
	}

	rc.ReqParam = fmt.Sprintf("DB[id=%d, tag=%s, name=%s, databases=%s, tables=%s, dumpType=%s]", db.Id, db.TagPath, db.Name, dbNamesStr, tablesStr, dumpType)
}

func escapeSql(dbType string, sql string) string {
	switch dbType {
	case entity.DbTypePostgres:
		return pq.QuoteLiteral(sql)
	case entity.DbTypeMysql:
		sql = strings.ReplaceAll(sql, `\`, `\\`)
		sql = strings.ReplaceAll(sql, `'`, `''`)
		return "'" + sql + "'"
	default:
		return sql
	}
}

func (d *Db) dumpDb(writer *gzipWriter, dbId uint64, dbName string, tables []string, needStruct bool, needData bool, switchDb bool) {
	dbConn := d.DbApp.GetDbConnection(dbId, dbName)
	writer.WriteString("\n-- ----------------------------")
	writer.WriteString("\n-- 导出平台: mayfly-go")
	writer.WriteString(fmt.Sprintf("\n-- 导出时间: %s ", time.Now().Format("2006-01-02 15:04:05")))
	writer.WriteString(fmt.Sprintf("\n-- 导出数据库: %s ", dbName))
	writer.WriteString("\n-- ----------------------------\n")
	writer.WriteString("\nSET FOREIGN_KEY_CHECKS = 0;\n")

	if switchDb {
		switch dbConn.Info.Type {
		case entity.DbTypeMysql:
			writer.WriteString(fmt.Sprintf("USE `%s`;\n", dbName))
		default:
			biz.IsTrue(false, "数据库类型必须为 %s", entity.DbTypeMysql)
		}
	}
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
		if needStruct {
			writer.WriteString(fmt.Sprintf("\n-- ----------------------------\n-- 表结构: %s \n-- ----------------------------\n", table))
			writer.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS `%s`;\n", table))
			writer.WriteString(dbMeta.GetCreateTableDdl(table) + ";\n")
		}
		if !needData {
			continue
		}
		writer.WriteString(fmt.Sprintf("\n-- ----------------------------\n-- 表记录: %s \n-- ----------------------------\n", table))
		writer.WriteString("BEGIN;\n")
		insertSql := "INSERT INTO `%s` VALUES (%s);\n"
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
					strValue = escapeSql(dbConn.Info.Type, strValue)
					values = append(values, strValue)
				} else {
					values = append(values, stringx.AnyToStr(value))
				}
			}
			writer.WriteString(fmt.Sprintf(insertSql, table, strings.Join(values, ", ")))
		})
		writer.WriteString("COMMIT;\n")
	}
	writer.WriteString("\nSET FOREIGN_KEY_CHECKS = 1;\n")
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

// 根据;\n切割sql
func SplitSqls(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	re := regexp.MustCompile(`\s*;\s*\n`)

	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, io.EOF
		}

		match := re.FindIndex(data)

		if match != nil {
			// 如果找到了";\n"，判断是否为最后一行
			if match[1] == len(data) {
				// 如果是最后一行，则返回完整的切片
				return len(data), data, nil
			}
			// 否则，返回到";\n"之后，并且包括";\n"本身
			return match[1], data[:match[1]], nil
		}

		if atEOF {
			return len(data), data, nil
		}

		return 0, nil, nil
	})

	return scanner
}
