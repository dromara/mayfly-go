package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	msgapp "mayfly-go/internal/msg/application"
	msgdto "mayfly-go/internal/msg/application/dto"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/ws"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kanzihuang/vitess/go/vt/sqlparser"
)

type Db struct {
	InstanceApp  application.Instance  `inject:"DbInstanceApp"`
	DbApp        application.Db        `inject:""`
	DbSqlExecApp application.DbSqlExec `inject:""`
	MsgApp       msgapp.Msg            `inject:""`
	TagApp       tagapp.TagTree        `inject:"TagTreeApp"`
}

// @router /api/dbs [get]
func (d *Db) Dbs(rc *req.Ctx) {
	queryCond, page := ginx.BindQueryAndPage[*entity.DbQuery](rc.GinCtx, new(entity.DbQuery))

	// 不存在可访问标签id，即没有可操作数据
	codes := d.TagApp.GetAccountResourceCodes(rc.GetLoginAccount().Id, consts.TagResourceTypeDb, queryCond.TagPath)
	if len(codes) == 0 {
		rc.ResData = model.EmptyPageResult[any]()
		return
	}
	queryCond.Codes = codes

	res, err := d.DbApp.GetPageList(queryCond, page, new([]vo.DbListVO))
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (d *Db) Save(rc *req.Ctx) {
	form := &form.DbForm{}
	db := ginx.BindJsonAndCopyTo[*entity.Db](rc.GinCtx, form, new(entity.Db))

	rc.ReqParam = form

	biz.ErrIsNil(d.DbApp.SaveDb(rc.MetaCtx, db, form.TagId...))
}

func (d *Db) DeleteDb(rc *req.Ctx) {
	idsStr := ginx.PathParam(rc.GinCtx, "dbId")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	ctx := rc.MetaCtx
	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		dbId := uint64(value)
		d.DbApp.Delete(ctx, dbId)
		// 删除该库的sql执行记录
		d.DbSqlExecApp.DeleteBy(ctx, &entity.DbSqlExec{DbId: dbId})
	}
}

/**  数据库操作相关、执行sql等   ***/

func (d *Db) ExecSql(rc *req.Ctx) {
	g := rc.GinCtx
	form := &form.DbSqlExecForm{}
	ginx.BindJsonAndValid(g, form)

	dbId := getDbId(g)
	dbConn, err := d.DbApp.GetDbConn(dbId, form.Db)
	biz.ErrIsNil(err)
	biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.GetLoginAccount().Id, dbConn.Info.TagPath...), "%s")

	sqlBytes, err := base64.StdEncoding.DecodeString(form.Sql)
	biz.ErrIsNilAppendErr(err, "sql解码失败: %s")
	// 去除前后空格及换行符
	sql := stringx.TrimSpaceAndBr(string(sqlBytes))

	rc.ReqParam = fmt.Sprintf("%s %s\n-> %s", dbConn.Info.GetLogDesc(), form.ExecId, sql)
	biz.NotEmpty(form.Sql, "sql不能为空")

	execReq := &application.DbSqlExecReq{
		DbId:   dbId,
		Db:     form.Db,
		Remark: form.Remark,
		DbConn: dbConn,
	}

	// 比前端超时时间稍微快一点，可以提示到前端
	ctx, cancel := context.WithTimeout(rc.MetaCtx, 58*time.Second)
	defer cancel()

	sqls, err := sqlparser.SplitStatementToPieces(sql, sqlparser.WithDialect(dbConn.Info.Type.Dialect()))
	biz.ErrIsNil(err, "SQL解析错误,请检查您的执行SQL")
	isMulti := len(sqls) > 1
	var execResAll *application.DbSqlExecRes

	for _, s := range sqls {
		s = stringx.TrimSpaceAndBr(s)
		// 多条执行，暂不支持查询语句
		if isMulti {
			biz.IsTrue(!strings.HasPrefix(strings.ToLower(s), "select"), "多条语句执行暂不不支持select语句")
		}

		execReq.Sql = s
		execRes, err := d.DbSqlExecApp.Exec(ctx, execReq)
		biz.ErrIsNilAppendErr(err, fmt.Sprintf("[%s] -> 执行失败: ", s)+"%s")

		if execResAll == nil {
			execResAll = execRes
		} else {
			execResAll.Merge(execRes)
		}
	}

	colAndRes := make(map[string]any)
	colAndRes["columns"] = execResAll.Columns
	colAndRes["res"] = execResAll.Res
	rc.ResData = colAndRes
}

// progressCategory sql文件执行进度消息类型
const progressCategory = "execSqlFileProgress"

// progressMsg sql文件执行进度消息
type progressMsg struct {
	Id                 string `json:"id"`
	Title              string `json:"title"`
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
	clientId := g.Query("clientId")

	dbConn, err := d.DbApp.GetDbConn(dbId, dbName)
	biz.ErrIsNil(err)
	biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.GetLoginAccount().Id, dbConn.Info.TagPath...), "%s")
	rc.ReqParam = fmt.Sprintf("filename: %s -> %s", filename, dbConn.Info.GetLogDesc())

	defer func() {
		if err := recover(); err != nil {
			errInfo := anyx.ToString(err)
			if len(errInfo) > 300 {
				errInfo = errInfo[:300] + "..."
			}
			d.MsgApp.CreateAndSend(rc.GetLoginAccount(), msgdto.ErrSysMsg("sql脚本执行失败", fmt.Sprintf("[%s][%s]执行失败: [%s]", filename, dbConn.Info.GetLogDesc(), errInfo)).WithClientId(clientId))
		}
	}()

	execReq := &application.DbSqlExecReq{
		DbId:   dbId,
		Db:     dbName,
		Remark: filename,
		DbConn: dbConn,
	}

	var sql string

	tokenizer := sqlparser.NewReaderTokenizer(file,
		sqlparser.WithCacheInBuffer(), sqlparser.WithDialect(dbConn.Info.Type.Dialect()))

	executedStatements := 0
	progressId := stringx.Rand(32)
	laId := rc.GetLoginAccount().Id
	defer ws.SendJsonMsg(ws.UserId(laId), clientId, msgdto.InfoSysMsg("sql脚本执行进度", &progressMsg{
		Id:                 progressId,
		Title:              filename,
		ExecutedStatements: executedStatements,
		Terminated:         true,
	}).WithCategory(progressCategory))
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ws.SendJsonMsg(ws.UserId(laId), clientId, msgdto.InfoSysMsg("sql脚本执行进度", &progressMsg{
				Id:                 progressId,
				Title:              filename,
				ExecutedStatements: executedStatements,
				Terminated:         false,
			}).WithCategory(progressCategory))
		default:
		}
		executedStatements++
		sql, err = sqlparser.SplitNext(tokenizer)
		if err == io.EOF {
			break
		}
		biz.ErrIsNilAppendErr(err, "%s")

		const prefixUse = "use "
		const prefixUSE = "USE "
		if strings.HasPrefix(sql, prefixUSE) || strings.HasPrefix(sql, prefixUse) {
			var stmt sqlparser.Statement
			stmt, err = sqlparser.Parse(sql)
			biz.ErrIsNilAppendErr(err, "%s")

			stmtUse, ok := stmt.(*sqlparser.Use)
			// 最终执行结果以数据库直接结果为准
			if !ok {
				logx.Warnf("sql解析失败: %s", sql)
			}
			dbConn, err = d.DbApp.GetDbConn(dbId, stmtUse.DBName.String())
			biz.ErrIsNil(err)
			biz.ErrIsNilAppendErr(d.TagApp.CanAccess(laId, dbConn.Info.TagPath...), "%s")
			execReq.DbConn = dbConn
		}
		// 需要记录执行记录
		const maxRecordStatements = 64
		if executedStatements < maxRecordStatements {
			execReq.Sql = sql
			_, err = d.DbSqlExecApp.Exec(rc.MetaCtx, execReq)
		} else {
			_, err = dbConn.Exec(sql)
		}

		biz.ErrIsNilAppendErr(err, "%s")
	}
	d.MsgApp.CreateAndSend(rc.GetLoginAccount(), msgdto.SuccessSysMsg("sql脚本执行成功", fmt.Sprintf("sql脚本执行完成：%s", rc.ReqParam)).WithClientId(clientId))
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

	la := rc.GetLoginAccount()
	db, err := d.DbApp.GetById(new(entity.Db), dbId)
	biz.ErrIsNil(err, "该数据库不存在")
	biz.ErrIsNilAppendErr(d.TagApp.CanAccess(la.Id, d.TagApp.ListTagPathByResource(consts.TagResourceTypeDb, db.Code)...), "%s")

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
		msg := anyx.ToString(recover())
		if len(msg) > 0 {
			msg = "数据库导出失败: " + msg
			writer.WriteString(msg)
			d.MsgApp.CreateAndSend(la, msgdto.ErrSysMsg("数据库导出失败", msg))
		}
		writer.Close()
	}()

	for _, dbName := range dbNames {
		d.dumpDb(writer, dbId, dbName, tables, needStruct, needData)
	}

	rc.ReqParam = collx.Kvs("db", db, "databases", dbNamesStr, "tables", tablesStr, "dumpType", dumpType)
}

func (d *Db) dumpDb(writer *gzipWriter, dbId uint64, dbName string, tables []string, needStruct bool, needData bool) {
	dbConn, err := d.DbApp.GetDbConn(dbId, dbName)
	biz.ErrIsNil(err)
	writer.WriteString("\n-- ----------------------------")
	writer.WriteString("\n-- 导出平台: mayfly-go")
	writer.WriteString(fmt.Sprintf("\n-- 导出时间: %s ", time.Now().Format("2006-01-02 15:04:05")))
	writer.WriteString(fmt.Sprintf("\n-- 导出数据库: %s ", dbName))
	writer.WriteString("\n-- ----------------------------\n\n")

	writer.WriteString(dbConn.Info.Type.StmtUseDatabase(dbName))
	writer.WriteString(dbConn.Info.Type.StmtSetForeignKeyChecks(false))

	dbMeta := dbConn.GetDialect()
	if len(tables) == 0 {
		ti, err := dbMeta.GetTables()
		biz.ErrIsNil(err)
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
			ddl, err := dbMeta.GetTableDDL(table)
			biz.ErrIsNil(err)
			writer.WriteString(ddl + "\n")
		}
		if !needData {
			continue
		}
		writer.WriteString(fmt.Sprintf("\n-- ----------------------------\n-- 表记录: %s \n-- ----------------------------\n", table))

		// 达梦不支持begin语句
		if dbConn.Info.Type != dbi.DbTypeDM {
			writer.WriteString("BEGIN;\n")
		}
		insertSql := "INSERT INTO %s VALUES (%s);\n"
		dbConn.WalkTableRows(context.TODO(), table, func(record map[string]any, columns []*dbi.QueryColumn) error {
			var values []string
			writer.TryFlush()
			for _, column := range columns {
				value := record[column.Name]
				if value == nil {
					values = append(values, "NULL")
					continue
				}
				strValue, ok := value.(string)
				if ok {
					strValue = dbConn.Info.Type.QuoteLiteral(strValue)
					values = append(values, strValue)
				} else {
					values = append(values, anyx.ToString(value))
				}
			}
			writer.WriteString(fmt.Sprintf(insertSql, quotedTable, strings.Join(values, ", ")))
			return nil
		})
		writer.WriteString("COMMIT;\n")
	}
	writer.WriteString(dbConn.Info.Type.StmtSetForeignKeyChecks(true))
}

func (d *Db) TableInfos(rc *req.Ctx) {
	res, err := d.getDbConn(rc.GinCtx).GetDialect().GetTables()
	biz.ErrIsNilAppendErr(err, "获取表信息失败: %s")
	rc.ResData = res
}

func (d *Db) TableIndex(rc *req.Ctx) {
	tn := rc.GinCtx.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")
	res, err := d.getDbConn(rc.GinCtx).GetDialect().GetTableIndex(tn)
	biz.ErrIsNilAppendErr(err, "获取表索引信息失败: %s")
	rc.ResData = res
}

// @router /api/db/:dbId/c-metadata [get]
func (d *Db) ColumnMA(rc *req.Ctx) {
	g := rc.GinCtx
	tn := g.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")

	dbi := d.getDbConn(rc.GinCtx)
	res, err := dbi.GetDialect().GetColumns(tn)
	biz.ErrIsNilAppendErr(err, "获取数据库列信息失败: %s")
	rc.ResData = res
}

// @router /api/db/:dbId/hint-tables [get]
func (d *Db) HintTables(rc *req.Ctx) {
	dbi := d.getDbConn(rc.GinCtx)

	dm := dbi.GetDialect()
	// 获取所有表
	tables, err := dm.GetTables()
	biz.ErrIsNil(err)
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
	columnMds, err := dm.GetColumns(tableNames...)
	biz.ErrIsNil(err)
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

func (d *Db) GetTableDDL(rc *req.Ctx) {
	tn := rc.GinCtx.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")
	res, err := d.getDbConn(rc.GinCtx).GetDialect().GetTableDDL(tn)
	biz.ErrIsNilAppendErr(err, "获取表ddl失败: %s")
	rc.ResData = res
}

func (d *Db) GetSchemas(rc *req.Ctx) {
	res, err := d.getDbConn(rc.GinCtx).GetDialect().GetSchemas()
	biz.ErrIsNilAppendErr(err, "获取schemas失败: %s")
	rc.ResData = res
}

func (d *Db) CopyTable(rc *req.Ctx) {
	form := &form.DbCopyTableForm{}
	copy := ginx.BindJsonAndCopyTo[*dbi.DbCopyTable](rc.GinCtx, form, new(dbi.DbCopyTable))

	conn, err := d.DbApp.GetDbConn(form.Id, form.Db)
	biz.ErrIsNilAppendErr(err, "拷贝表失败: %s")

	err = conn.GetDialect().CopyTable(copy)
	if err != nil {
		logx.Errorf("拷贝表失败: %s", err.Error())
	}
	biz.ErrIsNilAppendErr(err, "拷贝表失败: %s")
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

func (d *Db) getDbConn(g *gin.Context) *dbi.DbConn {
	dc, err := d.DbApp.GetDbConn(getDbId(g), getDbName(g))
	biz.ErrIsNil(err)
	return dc
}

// GetRestoreTask 获取数据库备份任务
// @router /api/instances/:instance/restore-task [GET]
func (d *Db) GetRestoreTask(rc *req.Ctx) {
	// todo get restore task
	panic("implement me")
}

// SaveRestoreTask 设置数据库备份任务
// @router /api/instances/:instance/restore-task [POST]
func (d *Db) SaveRestoreTask(rc *req.Ctx) {
	// todo set restore task
	panic("implement me")
}

// GetRestoreHistories 获取数据库备份历史
// @router /api/instances/:instance/restore-histories [GET]
func (d *Db) GetRestoreHistories(rc *req.Ctx) {
	// todo get restore histories
	panic("implement me")
}
