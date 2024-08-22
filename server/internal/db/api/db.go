package api

import (
	"context"
	"fmt"
	"io"
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/config"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/event"
	msgapp "mayfly-go/internal/msg/application"
	msgdto "mayfly-go/internal/msg/application/dto"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/cryptox"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/ws"
	"strings"
	"time"

	"github.com/kanzihuang/vitess/go/vt/sqlparser"
	"github.com/may-fly/cast"
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
	queryCond, page := req.BindQueryAndPage[*entity.DbQuery](rc, new(entity.DbQuery))

	// 不存在可访问标签id，即没有可操作数据
	codes := d.TagApp.GetAccountTagCodes(rc.GetLoginAccount().Id, int8(tagentity.TagTypeDbName), queryCond.TagPath)
	if len(codes) == 0 {
		rc.ResData = model.EmptyPageResult[any]()
		return
	}
	queryCond.Codes = codes

	var dbvos []*vo.DbListVO
	res, err := d.DbApp.GetPageList(queryCond, page, &dbvos)
	biz.ErrIsNil(err)

	instances, _ := d.InstanceApp.GetByIds(collx.ArrayMap(dbvos, func(i *vo.DbListVO) uint64 {
		return i.InstanceId
	}))
	instancesMap := collx.ArrayToMap(instances, func(i *entity.DbInstance) uint64 {
		return i.Id
	})
	for _, dbvo := range dbvos {
		di := instancesMap[dbvo.InstanceId]
		if di != nil {
			dbvo.InstanceCode = di.Code
			dbvo.InstanceType = di.Type
			dbvo.Host = di.Host
			dbvo.Port = di.Port
		}
	}

	rc.ResData = res
}

func (d *Db) Save(rc *req.Ctx) {
	form := &form.DbForm{}
	db := req.BindJsonAndCopyTo[*entity.Db](rc, form, new(entity.Db))

	rc.ReqParam = form

	biz.ErrIsNil(d.DbApp.SaveDb(rc.MetaCtx, db))
}

func (d *Db) DeleteDb(rc *req.Ctx) {
	idsStr := rc.PathParam("dbId")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	ctx := rc.MetaCtx
	for _, v := range ids {
		dbId := cast.ToUint64(v)
		biz.NotBlank(dbId, "存在错误dbId")
		biz.ErrIsNil(d.DbApp.Delete(ctx, dbId))
	}
}

/**  数据库操作相关、执行sql等   ***/

func (d *Db) ExecSql(rc *req.Ctx) {
	form := req.BindJsonAndValid(rc, new(form.DbSqlExecForm))

	ui := rc.GetLoginAccount()

	dbId := getDbId(rc)
	dbConn, err := d.DbApp.GetDbConn(dbId, form.Db)
	biz.ErrIsNil(err)
	biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.GetLoginAccount().Id, dbConn.Info.CodePath...), "%s")

	global.EventBus.Publish(rc.MetaCtx, event.EventTopicResourceOp, dbConn.Info.CodePath[0])
	sqlStr, err := cryptox.DesDecryptByToken(form.Sql, ui.Token)
	biz.ErrIsNilAppendErr(err, "sql解码失败: %s")
	// 去除前后空格及换行符
	sql := stringx.TrimSpaceAndBr(sqlStr)

	rc.ReqParam = fmt.Sprintf("%s %s\n-> %s", dbConn.Info.GetLogDesc(), form.ExecId, sql)
	biz.NotEmpty(form.Sql, "sql不能为空")

	execReq := &application.DbSqlExecReq{
		DbId:   dbId,
		Db:     form.Db,
		Remark: form.Remark,
		DbConn: dbConn,
	}

	ctx, cancel := context.WithTimeout(rc.MetaCtx, time.Duration(config.GetDbms().SqlExecTl)*time.Second)
	defer cancel()

	sqls, err := sqlparser.SplitStatementToPieces(sql, sqlparser.WithDialect(dbConn.GetMetaData().GetSqlParserDialect()))
	biz.ErrIsNil(err, "SQL解析错误,请检查您的执行SQL")
	isMulti := len(sqls) > 1
	var execResAll *application.DbSqlExecRes

	for _, s := range sqls {
		s = stringx.TrimSpaceAndBr(s)
		// 多条执行，暂不支持查询语句
		if isMulti && len(s) > 10 {
			biz.IsTrue(!strings.HasPrefix(strings.ToLower(s[:10]), "select"), "多条语句执行暂不不支持select语句")
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
	if execResAll != nil {
		colAndRes["columns"] = execResAll.Columns
		colAndRes["res"] = execResAll.Res
	}
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
	multipart, err := rc.GetRequest().MultipartReader()
	biz.ErrIsNilAppendErr(err, "读取sql文件失败: %s")
	file, err := multipart.NextPart()
	biz.ErrIsNilAppendErr(err, "读取sql文件失败: %s")
	defer file.Close()
	filename := file.FileName()
	dbId := getDbId(rc)
	dbName := getDbName(rc)
	clientId := rc.Query("clientId")

	dbConn, err := d.DbApp.GetDbConn(dbId, dbName)
	biz.ErrIsNil(err)
	biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.GetLoginAccount().Id, dbConn.Info.CodePath...), "%s")
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
		sqlparser.WithCacheInBuffer(), sqlparser.WithDialect(dbConn.GetMetaData().GetSqlParserDialect()))

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
			biz.ErrIsNilAppendErr(d.TagApp.CanAccess(laId, dbConn.Info.CodePath...), "%s")
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
	dbId := getDbId(rc)
	dbName := rc.Query("db")
	dumpType := rc.Query("type")
	tablesStr := rc.Query("tables")
	extName := rc.Query("extName")
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
	dbConn, err := d.DbApp.GetDbConn(dbId, dbName)
	biz.ErrIsNil(err)
	biz.ErrIsNilAppendErr(d.TagApp.CanAccess(la.Id, dbConn.Info.CodePath...), "%s")

	now := time.Now()
	filename := fmt.Sprintf("%s-%s.%s.sql%s", dbConn.Info.Name, dbName, now.Format("20060102150405"), extName)
	rc.Header("Content-Type", "application/octet-stream")
	rc.Header("Content-Disposition", "attachment; filename="+filename)
	if extName != ".gz" {
		rc.Header("Content-Encoding", "gzip")
	}

	var tables []string
	if len(tablesStr) > 0 {
		tables = strings.Split(tablesStr, ",")
	}

	defer func() {
		msg := anyx.ToString(recover())
		if len(msg) > 0 {
			msg = "数据库导出失败: " + msg
			rc.GetWriter().Write([]byte(msg))
			d.MsgApp.CreateAndSend(la, msgdto.ErrSysMsg("数据库导出失败", msg))
		}
	}()

	biz.ErrIsNil(d.DbApp.DumpDb(rc.MetaCtx, &dto.DumpDb{
		DbId:     dbId,
		DbName:   dbName,
		Tables:   tables,
		DumpDDL:  needStruct,
		DumpData: needData,
		Writer:   rc.GetWriter(),
	}))

	rc.ReqParam = collx.Kvs("db", dbConn.Info, "database", dbName, "tables", tablesStr, "dumpType", dumpType)
}

func (d *Db) TableInfos(rc *req.Ctx) {
	res, err := d.getDbConn(rc).GetMetaData().GetTables()
	biz.ErrIsNilAppendErr(err, "获取表信息失败: %s")
	rc.ResData = res
}

func (d *Db) TableIndex(rc *req.Ctx) {
	tn := rc.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")
	res, err := d.getDbConn(rc).GetMetaData().GetTableIndex(tn)
	biz.ErrIsNilAppendErr(err, "获取表索引信息失败: %s")
	rc.ResData = res
}

// @router /api/db/:dbId/c-metadata [get]
func (d *Db) ColumnMA(rc *req.Ctx) {
	tn := rc.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")

	dbi := d.getDbConn(rc)
	res, err := dbi.GetMetaData().GetColumns(tn)
	biz.ErrIsNilAppendErr(err, "获取数据库列信息失败: %s")
	rc.ResData = res
}

// @router /api/db/:dbId/hint-tables [get]
func (d *Db) HintTables(rc *req.Ctx) {
	dbi := d.getDbConn(rc)

	metadata := dbi.GetMetaData()
	// 获取所有表
	tables, err := metadata.GetTables()
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
	columnMds, err := metadata.GetColumns(tableNames...)
	biz.ErrIsNil(err)
	for _, v := range columnMds {
		tName := v.TableName
		if res[tName] == nil {
			res[tName] = make([]string, 0)
		}

		columnName := fmt.Sprintf("%s  [%s]", v.ColumnName, v.GetColumnType())
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
	tn := rc.Query("tableName")
	biz.NotEmpty(tn, "tableName不能为空")
	res, err := d.getDbConn(rc).GetMetaData().GetTableDDL(tn, false)
	biz.ErrIsNilAppendErr(err, "获取表ddl失败: %s")
	rc.ResData = res
}

func (d *Db) GetSchemas(rc *req.Ctx) {
	res, err := d.getDbConn(rc).GetMetaData().GetSchemas()
	biz.ErrIsNilAppendErr(err, "获取schemas失败: %s")
	rc.ResData = res
}

func (d *Db) CopyTable(rc *req.Ctx) {
	form := &form.DbCopyTableForm{}
	copy := req.BindJsonAndCopyTo[*dbi.DbCopyTable](rc, form, new(dbi.DbCopyTable))

	conn, err := d.DbApp.GetDbConn(form.Id, form.Db)
	biz.ErrIsNilAppendErr(err, "拷贝表失败: %s")

	err = conn.GetDialect().CopyTable(copy)
	if err != nil {
		logx.Errorf("拷贝表失败: %s", err.Error())
	}
	biz.ErrIsNilAppendErr(err, "拷贝表失败: %s")
}

func getDbId(rc *req.Ctx) uint64 {
	dbId := rc.PathParamInt("dbId")
	biz.IsTrue(dbId > 0, "dbId错误")
	return uint64(dbId)
}

func getDbName(rc *req.Ctx) string {
	db := rc.Query("db")
	biz.NotEmpty(db, "db不能为空")
	return db
}

func (d *Db) getDbConn(rc *req.Ctx) *dbi.DbConn {
	dc, err := d.DbApp.GetDbConn(getDbId(rc), getDbName(rc))
	biz.ErrIsNil(err)
	return dc
}
