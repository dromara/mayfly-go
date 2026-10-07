package api

import (
	"context"
	"errors"
	"fmt"
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/config"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/imsg"
	msgdto "mayfly-go/internal/msg/application/dto"
	"mayfly-go/internal/pkg/event"
	"mayfly-go/internal/pkg/utils"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/writerx"
	"strings"
	"time"

	"github.com/spf13/cast"
)

type Db struct {
	instanceApp     application.Instance     `inject:"T"`
	dbApp           application.Db           `inject:"T"`
	dbSQLExecApp    application.DbSQLExec    `inject:"T"`
	dbDataImportApp application.DbDataImport `inject:"T"`
	tagApp          tagapp.TagTreeService    `inject:"T"`
}

func (d *Db) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 获取数据库列表
		req.NewGet("", d.Dbs),

		req.NewPost("", d.Save).Log(req.NewLogSaveI(imsg.LogDbSave)),

		req.NewDelete(":dbId", d.DeleteDb).Log(req.NewLogSaveI(imsg.LogDbDelete)),

		req.NewGet(":dbId/t-create-ddl", d.GetTableDDL),

		req.NewGet(":dbId/version", d.GetVersion),

		req.NewGet(":dbId/capabilities", d.Capabilities),

		req.NewGet(":dbId/pg/schemas", d.GetSchemas),

		req.NewPost(":dbId/exec-sql", d.ExecSQL).Log(req.NewLogI(imsg.LogDbRunSQL)),

		req.NewPost(":dbId/exec-sql-file", d.ExecSQLFile).Log(req.NewLogSaveI(imsg.LogDbRunSQLFile)).RequiredPermissionCode("db:sqlscript:run"),

		req.NewGet(":dbId/dump", d.DumpSQL).Log(req.NewLogSaveI(imsg.LogDbDump)).NoRes(),

		// 表格文件（CSV/Excel）导入数据：先预览解析结果构建列映射，再按映射批量写入目标表
		// 预览会解析上传文件，与导入同级授权，避免仅凭实例 id 就能让服务端解析任意文件
		req.NewPost(":dbId/import-data-preview", d.PreviewImportData).RequiredPermissionCode("db:sqlscript:run"),
		req.NewPost(":dbId/import-data", d.ImportData).Log(req.NewLogSaveI(imsg.LogDbImportData)).RequiredPermissionCode("db:sqlscript:run"),

		req.NewGet(":dbId/t-infos", d.TableInfos),

		req.NewGet(":dbId/meta-objects", d.MetaObjects),

		req.NewGet(":dbId/meta-object-ddl", d.MetaObjectDDL),

		req.NewGet(":dbId/t-index", d.TableIndex),

		req.NewGet(":dbId/c-metadata", d.ColumnMetadata),

		req.NewPost(":dbId/copy-table", d.CopyTable),
	}

	return req.NewConfs("/dbs", reqs[:]...)
}

// @router /api/dbs [get]
func (d *Db) Dbs(rc *req.Ctx) {
	queryCond := rc.BindQuery[entity.DbQuery]()

	// 不存在可访问标签id，即没有可操作数据
	tags := d.tagApp.GetAccountTags(rc.GetLoginAccount().Id, &tagentity.TagTreeQuery{
		TypePaths:     collx.AsArray(tagentity.NewTypePaths(tagentity.TagTypeDbInstance, tagentity.TagTypeAuthCert, tagentity.TagTypeDb)),
		CodePathLikes: collx.AsArray(queryCond.TagPath),
	})
	if len(tags) == 0 {
		rc.ResData = model.NewEmptyPageResult[any]()
		return
	}
	queryCond.Codes = tags.GetCodes()

	res, err := d.dbApp.GetPageList(queryCond)
	biz.ErrIsNil(err)
	resVo := model.PageResultConv[*entity.DbListPO, *vo.DbListVO](res)
	dbvos := resVo.List

	instances, err := d.instanceApp.GetByIds(collx.ArrayMap(dbvos, func(i *vo.DbListVO) uint64 {
		return i.InstanceId
	}))
	biz.ErrIsNil(err)
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

	rc.ResData = resVo
}

func (d *Db) Save(rc *req.Ctx) {
	form, db := rc.BindJsonAndCopyTo[form.DbForm, entity.Db]()
	rc.ReqParam = form

	biz.ErrIsNil(d.dbApp.SaveDb(rc.MetaCtx, db))
}

func (d *Db) DeleteDb(rc *req.Ctx) {
	idsStr := rc.PathParam("dbId")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	ctx := rc.MetaCtx
	for _, v := range ids {
		biz.ErrIsNil(d.dbApp.Delete(ctx, cast.ToUint64(v)))
	}
}

/**  数据库操作相关、执行sql等   ***/

func (d *Db) ExecSQL(rc *req.Ctx) {
	form := rc.BindJson[form.DbSQLExecForm]()

	ctx, cancel := context.WithTimeout(rc.MetaCtx, time.Duration(config.GetDbms().SQLExecTl)*time.Second)
	defer cancel()

	dbId := getDbId(rc)
	dbConn, err := d.dbApp.GetDbConn(ctx, dbId, form.Db)
	biz.ErrIsNil(err)

	biz.ErrIsNilAppendErr(d.tagApp.CanAccess(rc.GetLoginAccount().Id, dbConn.Info.CodePath...), "%s")

	event.PublishResourceOp(rc.MetaCtx, dbConn.Info.CodePath)
	sqlStr, err := utils.AesDecryptByLa(form.SQL, rc.GetLoginAccount())
	biz.ErrIsNilAppendErr(err, "sql decoding failure: %s")

	rc.ReqParam = fmt.Sprintf("%s %s\n-> %s", dbConn.Info.GetLogDesc(), form.ExecId, sqlStr)
	biz.NotEmpty(form.SQL, "sql cannot be empty")

	execReq := &dto.DbSQLExecReq{
		DbId:             dbId,
		Db:               form.Db,
		Remark:           form.Remark,
		RequireWarnAck:   form.AskWarn,
		WarnAcknowledged: form.AckWarn,
		DbConn:           dbConn,
		SQL:              sqlStr,
	}

	execRes, err := d.dbSQLExecApp.Exec(ctx, execReq)
	biz.ErrIsNil(err)
	rc.ResData = execRes
}

// 执行sql文件
func (d *Db) ExecSQLFile(rc *req.Ctx) {
	dbId := getDbId(rc)
	clientId := rc.Query("clientId")
	dbName := rc.Query("db")
	uploadId := rc.Query("uploadId")
	filename := rc.QueryDefault("filename", "sql_file.sql")

	dbConn, err := d.dbApp.GetDbConn(rc.MetaCtx, dbId, dbName)
	biz.ErrIsNil(err)
	biz.ErrIsNilAppendErr(d.tagApp.CanAccess(rc.GetLoginAccount().Id, dbConn.Info.CodePath...), "%s")
	rc.ReqParam = fmt.Sprintf("filename: %s -> %s", filename, dbConn.Info.GetLogDesc())

	body := rc.GetRequest().Body
	defer body.Close()

	// 支持 .zip / .gz 压缩包（见 newSQLFileReader）
	reader, err := newSQLFileReader(filename, body)
	biz.ErrIsNilAppendErr(err, "failed to read sql file: %s")

	biz.ErrIsNil(d.dbSQLExecApp.ExecReader(rc.MetaCtx, &dto.SQLReaderExec{
		Reader:   reader,
		Filename: filename,
		DbConn:   dbConn,
		ClientId: clientId,
		UploadId: uploadId,
	}))
}

// 数据库dump
func (d *Db) DumpSQL(rc *req.Ctx) {
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
	dbConn, err := d.dbApp.GetDbConn(rc.MetaCtx, dbId, dbName)
	biz.ErrIsNil(err)

	biz.ErrIsNilAppendErr(d.tagApp.CanAccess(la.Id, dbConn.Info.CodePath...), "%s")

	now := time.Now()
	filename := fmt.Sprintf("%s-%s.%s.sql%s", dbConn.Info.Name, dbName, now.Format("20060102150405"), extName)
	rc.Header("Content-Type", "application/octet-stream")
	rc.Header("Content-Disposition", contentDisposition(filename))
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
			global.EventBus.Publish(rc.MetaCtx, event.EventTopicMsgTmplSend, &msgdto.MsgTmplSendEvent{
				TmplChannel: msgdto.MsgTmplDbDumpFail,
				Params:      collx.M{"dbId": dbConn.Info.Id, "dbName": dbConn.Info.Name, "error": msg},
				ReceiverIds: []uint64{la.Id},
			})
		}
	}()

	gzipWriter := writerx.NewGzipWriter(rc.GetWriter())
	defer gzipWriter.Close()

	biz.ErrIsNil(d.dbApp.DumpDb(rc.MetaCtx, &dto.DumpDb{
		DbId:     dbId,
		DbName:   dbName,
		Tables:   tables,
		DumpDDL:  needStruct,
		DumpData: needData,
		Writer:   gzipWriter,
	}))

	rc.ReqParam = collx.Kvs("db", dbConn.Info, "database", dbName, "tables", tablesStr, "dumpType", dumpType)
}

func (d *Db) TableInfos(rc *req.Ctx) {
	// ?like= 为可选表名过滤：方言具备服务端下推能力则按 LIKE 查询，否则回退全量取回 + 服务端子串过滤，
	// 用于超大 schema 的资源树按需加载（避免向浏览器吐上万张表）。空 like 等价原「取全部表」语义。
	// ?limit=/?offset= 为分页续载参数：首屏与「加载更多」各取一页，不重拉前缀。
	res, err := d.getDbConn(rc).Metadata().SearchTables(rc.Query("like"), cast.ToInt(rc.Query("limit")), cast.ToInt(rc.Query("offset")))
	biz.ErrIsNilAppendErr(err, "get table error: %s")
	rc.ResData = res
}

// MetaObjects 返回指定库/schema 下某类扩展元数据对象（视图/序列等）的扁平对象列表，供资源树懒加载展开。
// 由方言 MetadataNavigator 可选能力提供；kind 不受支持时返回空列表（前端已按 /capabilities.features 决定是否展示该节点）。
func (d *Db) MetaObjects(rc *req.Ctx) {
	kind := dbi.ObjectKind(rc.Query("kind"))
	biz.NotEmpty(string(kind), "kind cannot be empty")
	schema := rc.Query("schema")
	objects, err := d.getDbConn(rc).Metadata().ListObjects(rc.MetaCtx, schema, kind)
	if errors.Is(err, dbi.ErrUnsupportedKind) {
		rc.ResData = []dbi.MetadataObject{}
		return
	}
	biz.ErrIsNilAppendErr(err, "list metadata objects error: %s")
	rc.ResData = objects
}

// MetaObjectDDL 返回单个扩展元数据对象（视图/序列…）的重建 DDL 原文，供点开对象节点时展示。
// kind/name 必填；schema 为空回退当前库/模式；方言不支持该 kind 的 DDL 时返回明确错误。
func (d *Db) MetaObjectDDL(rc *req.Ctx) {
	kind := dbi.ObjectKind(rc.Query("kind"))
	name := rc.Query("name")
	biz.NotEmpty(string(kind), "kind cannot be empty")
	biz.NotEmpty(name, "name cannot be empty")
	ddl, err := d.getDbConn(rc).Metadata().ObjectDDL(rc.MetaCtx, rc.Query("schema"), kind, name)
	biz.ErrIsNilAppendErr(err, "get metadata object ddl error: %s")
	rc.ResData = ddl
}

func (d *Db) TableIndex(rc *req.Ctx) {
	tn := rc.Query("tableName")
	biz.NotEmpty(tn, "tableName cannot be empty")
	res, err := d.getDbConn(rc).Metadata().GetTableIndex(tn)
	biz.ErrIsNilAppendErr(err, "get table index error: %s")
	rc.ResData = res
}

// ColumnMetadata 返回指定表的列元数据（列名/类型/注释/主键/默认值/可空等），供表结构与数据编辑使用。
//
// @router /api/dbs/:dbId/c-metadata [get]
func (d *Db) ColumnMetadata(rc *req.Ctx) {
	tn := rc.Query("tableName")
	biz.NotEmpty(tn, "tableName cannot be empty")

	dbConn := d.getDbConn(rc)
	res, err := dbConn.Metadata().GetColumns(tn)
	biz.ErrIsNilAppendErr(err, "get column metadata error: %s")
	rc.ResData = res
}

func (d *Db) GetTableDDL(rc *req.Ctx) {
	tn := rc.Query("tableName")
	biz.NotEmpty(tn, "tableName cannot be empty")
	res, err := d.getDbConn(rc).Metadata().GetTableDDL(tn, false)
	biz.ErrIsNilAppendErr(err, "get table DDL error: %s")
	rc.ResData = res
}

func (d *Db) GetVersion(rc *req.Ctx) {
	version := d.getDbConn(rc).Metadata().GetCompatibleDbVersion()
	rc.ResData = version
}

func (d *Db) GetSchemas(rc *req.Ctx) {
	res, err := d.getDbConn(rc).Metadata().GetSchemas()
	biz.ErrIsNilAppendErr(err, "get schemas error: %s")
	rc.ResData = res
}

// Capabilities 方言能力协商端点：一次性吐出该库方言的静态能力、扩展对象类别、命名空间层次。
//
// 供前端数据驱动渲染（资源树按 features 决定 view/relation 等节点显隐、DDL/分页等按能力位），
// 使「新增方言或能力」无需前端改动——这正是能力枚举 SupportedFeatures 作为单一事实源的价值。
// 全部取自方言静态能力声明（Metadata.GetCapabilities），不执行任何 SQL 查询。
func (d *Db) Capabilities(rc *req.Ctx) {
	dbConn := d.getDbConn(rc)
	caps := dbConn.Metadata().GetCapabilities()
	rc.ResData = map[string]any{
		"dbType":    string(dbConn.Info.Type),
		"features":  caps.SupportedFeatures(),
		"namespace": caps.NamespaceHierarchy,
	}
}

func (d *Db) CopyTable(rc *req.Ctx) {
	form, copy := rc.BindJsonAndCopyTo[form.DbCopyTableForm, dbi.DbCopyTable]()

	conn, err := d.dbApp.GetDbConn(rc.MetaCtx, form.Id, form.Db)
	biz.ErrIsNilAppendErr(err, "copy table error: %s")

	err = conn.GetDialect().CopyTable(copy)
	if err != nil {
		logx.Errorf("copy table error: %s", err.Error())
	}
	biz.ErrIsNilAppendErr(err, "copy table error: %s")
	// 复制表的建表语句经 dbi 执行层发出，结构变更后的元数据缓存失效已由该层统一负责
}

// contentDisposition 构造安全的Content-Disposition响应头。
// 文件名由实例名与库名拼接而成，两者均可能被用户改动或直接通过查询参数传入，未消毒直接拼接存在三类问题：
//   - 含双引号、分号、反斜杠会截断甚至伪造filename参数（如 name";x=y 使解析结果只剩前半段）；
//   - 含控制字符（含CR/LF）时Go会判定整个头值非法并静默丢弃该响应头，浏览器只能退回默认文件名；
//   - 中文等非ASCII字符既未加引号也未做RFC 5987编码，各浏览器按不同字符集解码，下载名乱码。
//
// 因此同时输出ASCII回退名（危险与不可打印字符统一替换为下划线）与RFC 5987的filename*（UTF-8百分号编码），
// 现代浏览器优先取filename*，旧客户端退回ASCII名。
func contentDisposition(filename string) string {
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, asciiFallbackFilename(filename), rfc5987Encode(filename))
}

// asciiFallbackFilename 生成仅含可见ASCII字符的回退文件名
func asciiFallbackFilename(filename string) string {
	var sb strings.Builder
	for _, rn := range filename {
		// 0x20以下控制字符（含CR/LF/TAB/NUL）、空格、DEL以及双引号、分号、反斜杠、路径分隔符一律替换
		if rn <= 0x20 || rn == 0x7f || rn > 0x7f || strings.IndexByte(`";\/`, byte(rn)) >= 0 {
			sb.WriteByte('_')
			continue
		}
		sb.WriteRune(rn)
	}
	// 全部字符都被替换时兜底，避免得到一个纯下划线或空的文件名
	if strings.Trim(sb.String(), "_") == "" {
		return "dump.sql"
	}
	return sb.String()
}

// rfc5987Encode 按RFC 8187对ext-value做百分号编码：仅保留attr-char，其余字节按UTF-8逐字节编码。
// 注意'与%本身也需编码，否则文件名中的%s会被对端解析成非法的扩展值
func rfc5987Encode(val string) string {
	var sb strings.Builder
	for i := 0; i < len(val); i++ {
		b := val[i]
		isAlphaNum := (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
		if isAlphaNum || strings.IndexByte("!#$&*+-.^_`|~", b) >= 0 {
			sb.WriteByte(b)
			continue
		}
		sb.WriteString(fmt.Sprintf("%%%02X", b))
	}
	return sb.String()
}

func getDbId(rc *req.Ctx) uint64 {
	dbId := cast.ToUint64(rc.PathParam("dbId"))
	biz.IsTrue(dbId > 0, "dbId error")
	return dbId
}

func getDbName(rc *req.Ctx) string {
	db := rc.Query("db")
	biz.NotEmpty(db, "db cannot be empty")
	return db
}

func (d *Db) getDbConn(rc *req.Ctx) *dbi.DbConn {
	dc, err := d.dbApp.GetDbConn(rc.MetaCtx, getDbId(rc), getDbName(rc))
	biz.ErrIsNil(err)
	return dc
}
