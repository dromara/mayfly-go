package api

import (
	"context"
	"fmt"
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/config"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/internal/event"
	msgapp "mayfly-go/internal/msg/application"
	msgdto "mayfly-go/internal/msg/application/dto"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/cryptox"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/utils/writerx"
	"mayfly-go/pkg/ws"
	"strings"
	"time"

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
	tags := d.TagApp.GetAccountTags(rc.GetLoginAccount().Id, &tagentity.TagTreeQuery{
		TypePaths:     collx.AsArray(tagentity.NewTypePaths(tagentity.TagTypeDbInstance, tagentity.TagTypeAuthCert, tagentity.TagTypeDb)),
		CodePathLikes: collx.AsArray(queryCond.TagPath),
	})
	if len(tags) == 0 {
		rc.ResData = model.EmptyPageResult[any]()
		return
	}
	queryCond.Codes = tags.GetCodes()

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
		biz.ErrIsNil(d.DbApp.Delete(ctx, cast.ToUint64(v)))
	}
}

/**  数据库操作相关、执行sql等   ***/

func (d *Db) ExecSql(rc *req.Ctx) {
	form := req.BindJsonAndValid(rc, new(form.DbSqlExecForm))

	dbId := getDbId(rc)
	dbConn, err := d.DbApp.GetDbConn(dbId, form.Db)
	biz.ErrIsNil(err)
	biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.GetLoginAccount().Id, dbConn.Info.CodePath...), "%s")

	global.EventBus.Publish(rc.MetaCtx, event.EventTopicResourceOp, dbConn.Info.CodePath[0])
	sqlStr, err := cryptox.AesDecryptByLa(form.Sql, rc.GetLoginAccount())
	biz.ErrIsNilAppendErr(err, "sql decoding failure: %s")

	rc.ReqParam = fmt.Sprintf("%s %s\n-> %s", dbConn.Info.GetLogDesc(), form.ExecId, sqlStr)
	biz.NotEmpty(form.Sql, "sql cannot be empty")

	execReq := &application.DbSqlExecReq{
		DbId:      dbId,
		Db:        form.Db,
		Remark:    form.Remark,
		DbConn:    dbConn,
		Sql:       sqlStr,
		CheckFlow: true,
	}

	ctx, cancel := context.WithTimeout(rc.MetaCtx, time.Duration(config.GetDbms().SqlExecTl)*time.Second)
	defer cancel()

	execRes, err := d.DbSqlExecApp.Exec(ctx, execReq)
	biz.ErrIsNil(err)
	rc.ResData = execRes
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
	biz.ErrIsNilAppendErr(err, "failed to read sql file: %s")
	file, err := multipart.NextPart()
	biz.ErrIsNilAppendErr(err, "failed to read sql file: %s")
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
			d.MsgApp.CreateAndSend(rc.GetLoginAccount(), msgdto.ErrSysMsg(i18n.T(imsg.SqlScriptRunFail), fmt.Sprintf("[%s][%s] execution failure: [%s]", filename, dbConn.Info.GetLogDesc(), errInfo)).WithClientId(clientId))
		}
	}()

	executedStatements := 0
	progressId := stringx.Rand(32)
	laId := rc.GetLoginAccount().Id
	defer ws.SendJsonMsg(ws.UserId(laId), clientId, msgdto.InfoSysMsg(i18n.T(imsg.SqlScripRunProgress), &progressMsg{
		Id:                 progressId,
		Title:              filename,
		ExecutedStatements: executedStatements,
		Terminated:         true,
	}).WithCategory(progressCategory))
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	err = sqlparser.SQLSplit(file, func(sql string) error {
		select {
		case <-ticker.C:
			ws.SendJsonMsg(ws.UserId(laId), clientId, msgdto.InfoSysMsg(i18n.T(imsg.SqlScripRunProgress), &progressMsg{
				Id:                 progressId,
				Title:              filename,
				ExecutedStatements: executedStatements,
				Terminated:         false,
			}).WithCategory(progressCategory))
		default:
		}

		executedStatements++
		_, err = dbConn.Exec(sql)
		return err
	})

	biz.ErrIsNilAppendErr(err, "%s")
	d.MsgApp.CreateAndSend(rc.GetLoginAccount(), msgdto.SuccessSysMsg(i18n.T(imsg.SqlScriptRunSuccess), fmt.Sprintf("execution success: %s", rc.ReqParam)).WithClientId(clientId))
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
			msg = "DB dump error: " + msg
			rc.GetWriter().Write([]byte(msg))
			d.MsgApp.CreateAndSend(la, msgdto.ErrSysMsg(i18n.T(imsg.DbDumpErr), msg))
		}
	}()

	biz.ErrIsNil(d.DbApp.DumpDb(rc.MetaCtx, &dto.DumpDb{
		DbId:     dbId,
		DbName:   dbName,
		Tables:   tables,
		DumpDDL:  needStruct,
		DumpData: needData,
		Writer:   writerx.NewGzipWriter(rc.GetWriter()),
	}))

	rc.ReqParam = collx.Kvs("db", dbConn.Info, "database", dbName, "tables", tablesStr, "dumpType", dumpType)
}

func (d *Db) TableInfos(rc *req.Ctx) {
	res, err := d.getDbConn(rc).GetMetadata().GetTables()
	biz.ErrIsNilAppendErr(err, "get table error: %s")
	rc.ResData = res
}

func (d *Db) TableIndex(rc *req.Ctx) {
	tn := rc.Query("tableName")
	biz.NotEmpty(tn, "tableName cannot be empty")
	res, err := d.getDbConn(rc).GetMetadata().GetTableIndex(tn)
	biz.ErrIsNilAppendErr(err, "get table index error: %s")
	rc.ResData = res
}

// @router /api/db/:dbId/c-metadata [get]
func (d *Db) ColumnMA(rc *req.Ctx) {
	tn := rc.Query("tableName")
	biz.NotEmpty(tn, "tableName cannot be empty")

	dbi := d.getDbConn(rc)
	res, err := dbi.GetMetadata().GetColumns(tn)
	biz.ErrIsNilAppendErr(err, "get column metadata error: %s")
	rc.ResData = res
}

// @router /api/db/:dbId/hint-tables [get]
func (d *Db) HintTables(rc *req.Ctx) {
	dbi := d.getDbConn(rc)

	metadata := dbi.GetMetadata()
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
	biz.NotEmpty(tn, "tableName cannot be empty")
	res, err := d.getDbConn(rc).GetMetadata().GetTableDDL(tn, false)
	biz.ErrIsNilAppendErr(err, "get table DDL error: %s")
	rc.ResData = res
}

func (d *Db) GetVersion(rc *req.Ctx) {
	version := d.getDbConn(rc).GetMetadata().GetCompatibleDbVersion()
	rc.ResData = version
}

func (d *Db) GetSchemas(rc *req.Ctx) {
	res, err := d.getDbConn(rc).GetMetadata().GetSchemas()
	biz.ErrIsNilAppendErr(err, "get schemas error: %s")
	rc.ResData = res
}

func (d *Db) CopyTable(rc *req.Ctx) {
	form := &form.DbCopyTableForm{}
	copy := req.BindJsonAndCopyTo[*dbi.DbCopyTable](rc, form, new(dbi.DbCopyTable))

	conn, err := d.DbApp.GetDbConn(form.Id, form.Db)
	biz.ErrIsNilAppendErr(err, "copy table error: %s")

	err = conn.GetDialect().CopyTable(copy)
	if err != nil {
		logx.Errorf("copy table error: %s", err.Error())
	}
	biz.ErrIsNilAppendErr(err, "copy table error: %s")
}

func getDbId(rc *req.Ctx) uint64 {
	dbId := rc.PathParamInt("dbId")
	biz.IsTrue(dbId > 0, "dbId error")
	return uint64(dbId)
}

func getDbName(rc *req.Ctx) string {
	db := rc.Query("db")
	biz.NotEmpty(db, "db cannot be empty")
	return db
}

func (d *Db) getDbConn(rc *req.Ctx) *dbi.DbConn {
	dc, err := d.DbApp.GetDbConn(getDbId(rc), getDbName(rc))
	biz.ErrIsNil(err)
	return dc
}
