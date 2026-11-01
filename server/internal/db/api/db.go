package api

import (
	"context"
	"errors"
	"fmt"
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/application/sqlexec"
	"mayfly-go/internal/db/application/sqlutil"
	"mayfly-go/internal/db/config"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/imsg"
	msgdto "mayfly-go/internal/msg/application/dto"
	"mayfly-go/internal/pkg/event"
	"mayfly-go/internal/pkg/utils"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
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
	instanceApp     application.Instance          `inject:"T"`
	dbApp           application.Db                `inject:"T"`
	dbSQLExecApp    sqlexec.DbSQLExec             `inject:"T"`
	dbDataImportApp application.DbDataImport      `inject:"T"`
	tagApp          tagapp.TagTreeService         `inject:"T"`
	exportTicketApp application.DbExportTicketApp `inject:"T"`
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

		// DDL 预检：只读，不参与放行判定（策略管「要不要批」，它管「批的是什么」）
		req.NewPost(":dbId/ddl-precheck", d.DdlPrecheck).RequiredPermissionCode("db:sqlscript:run"),
		req.NewPost(":dbId/exec-sql-file", d.ExecSQLFile).Log(req.NewLogSaveI(imsg.LogDbRunSQLFile)).RequiredPermissionCode("db:sqlscript:run"),

		req.NewGet(":dbId/dump", d.DumpSQL).Log(req.NewLogSaveI(imsg.LogDbDump)).RequiredPermissionCode("db:data:export").NoRes(),

		// 凭已批准的导出工单下载：权限码与即席导出同档（能导出的人才能用凭据），
		// 但不再过触发策略——工单本身就是那道审批
		req.NewGet(":dbId/export-tickets/:procinstId/download", d.DownloadExportTicket).
			Log(req.NewLogSaveI(imsg.LogDbDump)).RequiredPermissionCode("db:data:export").NoRes(),

		// 表格文件（CSV/Excel）导入数据：先预览解析结果构建列映射，再按映射批量写入目标表
		// 预览会解析上传文件，与导入同级授权，避免仅凭实例 id 就能让服务端解析任意文件
		req.NewPost(":dbId/import-data-preview", d.PreviewImportData).RequiredPermissionCode("db:sqlscript:run"),
		req.NewPost(":dbId/import-data", d.ImportData).Log(req.NewLogSaveI(imsg.LogDbImportData)).RequiredPermissionCode("db:sqlscript:run"),

		req.NewGet(":dbId/t-infos", d.TableInfos),

		req.NewGet(":dbId/meta-objects", d.MetaObjects),

		req.NewGet(":dbId/meta-object-ddl", d.MetaObjectDDL),

		req.NewGet(":dbId/t-index", d.TableIndex),

		// 表的主键/唯一键约束（含约束名与键内列序）：表编辑撤销主键时要引用约束名（pg/mssql 无 DROP PRIMARY KEY）
		req.NewGet(":dbId/t-keys", d.TableKeys),

		req.NewGet(":dbId/c-metadata", d.ColumnMetadata),
		req.NewPost(":dbId/classify", d.ClassifySQL),

		// 数据字典导出：只导出结构事实（表/列/索引/注释），不含数据；格式 csv|md
		req.NewGet(":dbId/data-dict", d.DataDict).Log(req.NewLogSaveI(imsg.LogDbDump)).NoRes(),

		// 列值采样（编辑器值补全）：与其余元数据读取同档（靠 GetDbConn 里的资源可达校验），
		// 不另开权限码：开了一个库里没建行的码，功能上线即除 admin 外全体 403
		req.NewGet(":dbId/column-values", d.ColumnValues),

		// 敏感列扫描：读的是列元信息而非数据，与其余元数据读取同档
		req.NewGet(":dbId/sensitive-columns", d.ScanSensitiveColumns),

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
	sqlStr, err := utils.AesDecryptByLa(form.SQL, rc.GetLoginAccount())
	biz.ErrIsNilAppendErr(err, "sql decoding failure: %s")
	biz.NotEmpty(form.SQL, "sql cannot be empty")

	execReq := &dto.DbSQLExecReq{
		ExecId:           form.ExecId,
		DbId:             dbId,
		Db:               form.Db,
		Remark:           form.Remark,
		RequireWarnAck:   form.AskWarn,
		WarnAcknowledged: form.AckWarn,
		Atomic:           form.Atomic,
		SQL:              sqlStr,
	}

	// 先登记再取连接：取连接（冷池拨号/建隧道）可能超过取消侧的有界等待，
	// 登记晚于取连接时，拨号期间到达的「取消」会误报「未找到进行中的执行」而语句随后照跑。
	// 用独立变量接登记的 runCtx：跨属主碰撞时 register 会返回 nil ctx，直接覆盖 ctx 会埋下 nil-ctx 雷，
	// 故仅在 biz.ErrIsNil 放行（必无错）后才把可取消的 runCtx 交回 ctx 供下游使用
	runCtx, done, err := d.dbSQLExecApp.RegisterExec(ctx, execReq)
	biz.ErrIsNil(err)
	defer done()
	ctx = runCtx

	dbConn, err := d.dbApp.GetDbConn(ctx, dbId, form.Db)
	biz.ErrIsNil(err)
	biz.ErrIsNilAppendErr(d.tagApp.CanAccess(rc.GetLoginAccount().Id, dbConn.Info.CodePath...), "%s")
	execReq.DbConn = dbConn

	event.PublishResourceOp(rc.MetaCtx, dbConn.Info.CodePath)
	rc.ReqParam = fmt.Sprintf("%s %s\n-> %s", dbConn.Info.GetLogDesc(), form.ExecId, sqlStr)

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

// dumpRequest 一次导出的全部输入。
//
// 抽成结构体是为了让两条入口（即席导出 / 导出工单下载）共用同一段流式实现：
// 响应头、脱敏、行数统计与失败通知都不能有第二份实现，否则两条通道的产物形态会漂移。
type dumpRequest struct {
	DbConn   *dbi.DbConn
	DbName   string
	DumpType string
	Tables   []string
	ExtName  string
	// SkipPolicy 跳过触发策略：只有「导出工单已批准」这一条通道会置 true——
	// 策略那关已经由人工审批替代，再判一次会把工单通道直接堵死。
	// 表级/列级权限不在此列：那是硬边界，任何通道都必须判
	SkipPolicy bool
	// ReqParam 需要写进操作日志的原始入参（工单通道记录工单 id）
	AuditExtra collx.M
}

// 数据库dump
func (d *Db) DumpSQL(rc *req.Ctx) {
	dbId := getDbId(rc)
	dbName := rc.Query("db")
	dbConn, err := d.dbApp.GetDbConn(rc.MetaCtx, dbId, dbName)
	biz.ErrIsNil(err)
	biz.ErrIsNilAppendErr(d.tagApp.CanAccess(rc.GetLoginAccount().Id, dbConn.Info.CodePath...), "%s")

	d.streamDump(rc, &dumpRequest{
		DbConn:   dbConn,
		DbName:   dbName,
		DumpType: rc.Query("type"),
		Tables:   splitTables(rc.Query("tables")),
		ExtName:  normalizeDumpExt(rc.Query("extName")),
	})
}

// DownloadExportTicket 凭已批准的导出工单下载它授权的那一次导出。
//
// 与 DumpSQL 的区别只有一处：**不再过触发策略**（工单就是那道审批）。
// 表/列权限、资源可达性、脱敏、审计全部照旧——工单只能解决「要不要人批」，
// 不能解决「这个人有没有权看这张表」。
// @router /api/dbs/:dbId/export-tickets/:procinstId/download [get]
func (d *Db) DownloadExportTicket(rc *req.Ctx) {
	dbId := getDbId(rc)
	ticket, err := d.exportTicketApp.ValidateForDownload(rc.MetaCtx, cast.ToUint64(rc.PathParam("procinstId")))
	biz.ErrIsNil(err)

	// 导出范围以工单为准：请求方不能用一张小范围工单去导整库
	if ticket.DbId != dbId {
		biz.ErrIsNil(errorx.NewBizI(rc.MetaCtx, imsg.ErrExportTicketScopeMismatch))
	}
	dbName := ticket.DbName
	dbConn, err := d.dbApp.GetDbConn(rc.MetaCtx, dbId, dbName)
	biz.ErrIsNil(err)
	biz.ErrIsNilAppendErr(d.tagApp.CanAccess(rc.GetLoginAccount().Id, dbConn.Info.CodePath...), "%s")

	d.streamDump(rc, &dumpRequest{
		DbConn:     dbConn,
		DbName:     dbName,
		DumpType:   ticket.DumpType,
		Tables:     ticket.Tables,
		SkipPolicy: true,
		AuditExtra: collx.Kvs("procinstId", cast.ToUint64(rc.PathParam("procinstId"))),
	})
}

// splitTables 逗号分隔的表清单；空串表示整库
func splitTables(tablesStr string) []string {
	if tablesStr == "" {
		return nil
	}
	return strings.Split(tablesStr, ",")
}

// normalizeDumpExt 只接受 .gz 这一种额外封装（其余一律按原生 SQL 处理）
func normalizeDumpExt(extName string) string {
	switch extName {
	case ".gz", ".gzip", "gz", "gzip":
		return ".gz"
	default:
		return ""
	}
}

// streamDump 流式导出：先判定（权限 → 策略），再写响应头与产物。
//
// 判定必须早于设置响应头：被拦时要返回 JSON 错误，而不是一个已经写了头、解压即坏的文件。
func (d *Db) streamDump(rc *req.Ctx, req *dumpRequest) {
	la := rc.GetLoginAccount()
	dbConn, dbName := req.DbConn, req.DbName
	dbId := dbConn.Info.Id

	// 是否需要导出表结构 / 数据
	needStruct := req.DumpType == "1" || req.DumpType == "3"
	needData := req.DumpType == "2" || req.DumpType == "3"

	// 表/列权限是硬边界：任何导出形态（含只导结构）都必须先判。
	//
	// 只导结构时若跳过判定，受保护表的表名、列名、列类型、默认值、注释会被完整取走——
	// 这与本函数"权限不因通道而异"的约定冲突，也与 SQL 编辑器/查询导出的行为不一致。
	exportTables := req.Tables
	if len(exportTables) == 0 {
		// 整库导出：表清单取库侧元数据（与导出本体同一来源），逐表等价于一次 SELECT *
		ts, listErr := dbConn.Metadata().GetTables()
		biz.ErrIsNilAppendErr(listErr, "list tables for the export policy check failed: %s")
		exportTables = collx.ArrayMap(ts, func(t dbi.Table) string { return t.TableName })
	}
	// 列级权限：整表导出会把无权列一起写进文件，且文件看不出缺列，只能整条拒绝。
	// 工单通道同样要过——权限是硬边界，审批不能替数据所有者做主
	biz.ErrIsNil(sqlexec.AssertDumpTablesAllowed(rc.MetaCtx, sqlexec.GetTableAclApp(), dbId, exportTables))

	if needData {

		if !req.SkipPolicy {
			// 数据导出是一次大范围读取，与执行 / 导入 / 查询导出同一道闸门
			checks := sqlutil.ExportSelects(dbConn.GetDialect().Quoter(), exportTables)
			if len(checks) > 0 {
				notices, checkErr := d.dbSQLExecApp.CheckSqlsWithoutTicket(rc.MetaCtx, dbConn, checks)
				biz.ErrIsNil(checkErr)
				for _, notice := range notices {
					if notice != "" {
						// 下载通道没有能回显提醒的界面：落日志，避免「仅提醒」级别静默消失
						logx.WarnfContext(rc.MetaCtx, "db dump policy notice: %s", notice)
					}
				}
			}
		}
	}

	now := time.Now()
	extName := req.ExtName
	filename := fmt.Sprintf("%s-%s.%s.sql%s", dbConn.Info.Name, dbName, now.Format("20060102150405"), extName)
	rc.Header("Content-Type", "application/octet-stream")
	rc.Header("Content-Disposition", contentDisposition(filename))
	if extName != ".gz" {
		rc.Header("Content-Encoding", "gzip")
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

	// 按表记录导出数据行数（每张表写完自己那一段时上报一次最终值），用于回答「这次被拉走了多少数据」。
	// 取 end=true 的那次上报而非累加：进度回调对同一表会多次报当前计数，累加会把同一行重复计数
	rowCounts := map[string]int{}
	var exportRows int
	countProgress := func(table string, stmtType dbi.DumpKind, count int, currentStmtTypeEnd bool) {
		if !currentStmtTypeEnd || stmtType != dbi.DumpKindInsert {
			return
		}
		if prev, ok := rowCounts[table]; !ok || count > prev {
			rowCounts[table] = count
		}
	}

	biz.ErrIsNil(d.dbApp.DumpDb(rc.MetaCtx, &dto.DumpDb{
		DbId:     dbId,
		DbName:   dbName,
		Tables:   req.Tables,
		DumpDDL:  needStruct,
		DumpData: needData,
		// 即席导出是给人看的数据副本，必须过与交互式查询同一套脱敏规则：
		// 否则同一个脱敏列「在网格里是掩码、被导出却是明文」，脱敏等于没有
		Mask:     true,
		Writer:   gzipWriter,
		Progress: countProgress,
	}))

	for _, cnt := range rowCounts {
		exportRows += cnt
	}
	params := collx.Kvs("db", dbConn.Info, "database", dbName, "tables", strings.Join(req.Tables, ","), "dumpType", req.DumpType, "exportRows", exportRows)
	for key, value := range req.AuditExtra {
		params[key] = value
	}
	rc.ReqParam = params
}

func (d *Db) TableInfos(rc *req.Ctx) {
	// ?like= 为可选表名过滤：方言具备服务端下推能力则按 LIKE 查询，否则回退全量取回 + 服务端子串过滤，
	// 用于超大 schema 的资源树按需加载（避免向浏览器吐上万张表）。空 like 等价原「取全部表」语义。
	// ?limit=/?offset= 为分页续载参数：首屏与「加载更多」各取一页，不重拉前缀。
	res, err := d.getDbConn(rc).Metadata().SearchTables(rc.Query("like"), cast.ToInt(rc.Query("limit")), cast.ToInt(rc.Query("offset")))
	biz.ErrIsNilAppendErr(err, "get table error: %s")

	// 表级保护是「拒绝访问」：表名若照常出现在清单与补全里，用户会写出必然被拒的语句。
	// 与列级口径保持一致（c-metadata 已过滤无权列）。
	if tableAcl := sqlexec.GetTableAclApp(); tableAcl != nil && len(res) > 0 {
		names := make([]string, 0, len(res))
		for _, table := range res {
			names = append(names, table.TableName)
		}
		visible, aclErr := tableAcl.VisibleTables(rc.MetaCtx, getDbId(rc), names)
		biz.ErrIsNil(aclErr)
		if len(visible) != len(names) {
			allowed := make(map[string]bool, len(visible))
			for _, name := range visible {
				allowed[strings.ToLower(name)] = true
			}
			kept := make([]dbi.Table, 0, len(visible))
			for _, table := range res {
				if allowed[strings.ToLower(table.TableName)] {
					kept = append(kept, table)
				}
			}
			res = kept
		}
	}
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
	// 视图同样可能被加入保护名单：名字继续列出来，用户点开会拿到「无权访问」
	if tableAcl := sqlexec.GetTableAclApp(); tableAcl != nil && len(objects) > 0 {
		names := make([]string, 0, len(objects))
		for _, object := range objects {
			names = append(names, object.Name)
		}
		visible, aclErr := tableAcl.VisibleTables(rc.MetaCtx, getDbId(rc), names)
		biz.ErrIsNil(aclErr)
		if len(visible) != len(names) {
			allowed := make(map[string]bool, len(visible))
			for _, name := range visible {
				allowed[strings.ToLower(name)] = true
			}
			kept := make([]dbi.MetadataObject, 0, len(visible))
			for _, object := range objects {
				if allowed[strings.ToLower(object.Name)] {
					kept = append(kept, object)
				}
			}
			objects = kept
		}
	}
	rc.ResData = objects
}

// classifySQLForm 服务端分类请求体
type classifySQLForm struct {
	SQL string `json:"sql"`
}

// ClassifySQL 用服务端解析器判定单条 SQL 的语句类型（分类权威）。
//
// 存在意义：前端为了即时交互保留了一份 TS 解析（只读/写入判定），两份实现靠
// testdata/classify_cases.json 契约锁对齐；当业务需要更强判定（新语法、方言特性）时，
// 前端可调用本接口取服务端结论，避免再扩一份 TS 解析器（第三套实现）。本接口只解析不执行。
//
// @router /api/dbs/:dbId/classify [post]
func (d *Db) ClassifySQL(rc *req.Ctx) {
	form := rc.BindJson[classifySQLForm]()
	biz.NotEmpty(form.SQL, "sql cannot be empty")

	conn := d.getDbConn(rc)
	parser := conn.GetDialect().GetSQLParser()
	splitter := conn.GetDialect().GetSQLSplitter()
	stmt, parseErr := parser.Parse(form.SQL)
	stmtType := sqlparser.Classify(parser, splitter, form.SQL, stmt, parseErr)
	rc.ResData = map[string]any{"stmtType": string(stmtType)}
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

// TableKeys 返回表的主键/唯一键约束（含约束名与键内列序）。
//
// 为什么不复用 t-index：索引与键不是一回事——pg/mssql 撤销主键必须引用**约束名**
// （两家都没有 DROP PRIMARY KEY 语法），而索引列表刻意把主键索引排除在外。
// 表编辑据此生成 DROP CONSTRAINT 语句，缺了名字就只能靠猜。
//
// @router /api/dbs/:dbId/t-keys [get]
func (d *Db) TableKeys(rc *req.Ctx) {
	tn := rc.Query("tableName")
	biz.NotEmpty(tn, "tableName cannot be empty")
	res, err := d.getDbConn(rc).Metadata().GetKeys(rc.MetaCtx, rc.Query("schema"), tn)
	biz.ErrIsNilAppendErr(err, "get table keys error: %s")
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

	// 列级权限也要管元数据：列被保护时，结果集里已经没有它的值，若还从元数据把列名
	// 下发出去，界面上会留下一个永远空白的列、编辑器补全也会继续提示它——
	// 两者都会让人以为「只是暂时没数据」，而这层保护的本意是「这一列不存在」。
	denied, aclErr := sqlexec.GetTableAclApp().DeniedColumns(rc.MetaCtx, getDbId(rc), []string{tn})
	if aclErr != nil {
		biz.ErrIsNil(aclErr)
	}
	if len(denied) > 0 {
		kept := make([]dbi.Column, 0, len(res))
		for _, column := range res {
			if _, blocked := denied[strings.ToLower(column.ColumnName)]; blocked {
				continue
			}
			kept = append(kept, column)
		}
		res = kept
	}
	rc.ResData = res
}

// ColumnValues 返回某列出现过的去重值，供编辑器在「列 = 」上下文里补全值。
//
// 可达性校验在应用层的 GetDbConn 里做（与其余元数据读取同源），这里不再另取一次连接
// ScanSensitiveColumns 扫描敏感列候选
// @router /api/dbs/:dbId/sensitive-columns [get]
func (d *Db) ScanSensitiveColumns(rc *req.Ctx) {
	dbId := getDbId(rc)
	dbName := rc.Query("db")
	res, err := d.dbApp.ScanSensitiveColumns(rc.MetaCtx, dbId, dbName)
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (d *Db) ColumnValues(rc *req.Ctx) {
	table := rc.Query("tableName")
	biz.NotEmpty(table, "tableName cannot be empty")
	column := rc.Query("column")
	biz.NotEmpty(column, "column cannot be empty")

	values, err := d.dbApp.ColumnValues(rc.MetaCtx, getDbId(rc), getDbName(rc), table, column, cast.ToInt(rc.Query("limit")))
	biz.ErrIsNilAppendErr(err, "sample column values error: %s")
	rc.ResData = values
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

	// 带数据复制会把源表整表搬进一张不受保护的新表（INSERT ... SELECT *），
	// 是表级权限的旁路；只复制结构时不读行数据，不必拦
	if copy.CopyData {
		tableAcl := sqlexec.GetTableAclApp()
		biz.ErrIsNil(tableAcl.AssertTablesAllowed(rc.MetaCtx, form.Id, []string{copy.TableName}))
		// 复制的是整行：表上存在无权列时同样会绕过列保护
		biz.ErrIsNil(tableAcl.AssertWholeTableAllowed(rc.MetaCtx, form.Id, []string{copy.TableName}))
	}

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

// DdlPrecheck DDL 预检：读表规模与主键，给出「会不会丢数据/要不要重建表/能不能回滚」的提示。
// 只读、可重复调用，不做放行判定。
// @router /api/dbs/:dbId/ddl-precheck [post]
func (d *Db) DdlPrecheck(rc *req.Ctx) {
	param := rc.BindJson[form.DdlPrecheck]()
	result, err := application.GetDdlPrecheckApp().Precheck(rc.MetaCtx, getDbId(rc), param.Db, param.Sql)
	biz.ErrIsNil(err)
	rc.ResData = result
}

// DataDict 数据字典导出（CSV / Markdown）。
//
// 只导出结构事实，不含任何数据行；权限沿用 c-metadata 同档（GetDbConn 里的资源可达校验），
// 不新开权限码——开一个库里没建行的码，功能上线即除 admin 外全体 403。
// @router /api/dbs/:dbId/data-dict [get]
func (d *Db) DataDict(rc *req.Ctx) {
	dbId := getDbId(rc)
	dbName := rc.Query("db")
	format := strings.ToLower(rc.Query("format"))
	if format == "" {
		format = "csv"
	}
	var tables []string
	for _, table := range strings.Split(rc.Query("tables"), ",") {
		if trimmed := strings.TrimSpace(table); trimmed != "" {
			tables = append(tables, trimmed)
		}
	}

	app := application.GetDataDictApp()
	dict, err := app.Build(rc.MetaCtx, dbId, dbName, tables)
	biz.ErrIsNil(err)

	now := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("data-dict-%s-%s.%s", dbName, now, format)
	var payload []byte
	switch format {
	case "md", "markdown":
		rc.Header("Content-Type", "text/markdown; charset=utf-8")
		payload = app.RenderMarkdown(dict)
	default:
		filename = fmt.Sprintf("data-dict-%s-%s.csv", dbName, now)
		rc.Header("Content-Type", "text/csv; charset=utf-8")
		payload = app.RenderCSV(dict)
	}
	rc.Header("Content-Disposition", contentDisposition(filename))
	_, _ = rc.GetWriter().Write(payload)
}
