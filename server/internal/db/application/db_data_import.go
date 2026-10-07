package application

import (
	"context"
	"strconv"
	"strings"
	"time"

	"mayfly-go/internal/db/application/dto"
	dbsync "mayfly-go/internal/db/application/sync"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/importer"
	msgdto "mayfly-go/internal/msg/application/dto"
	"mayfly-go/internal/pkg/event"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
)

// DbDataImport 表格文件（CSV/Excel）导入数据库表的应用能力：解析文件 → 按列映射与目标表列类型转换 →
// 复用方言 SQLGenerator 生成批量 INSERT → 事务内分批执行。与「SQL文件执行」不同，本入口消费的是表格数据而非 SQL 文本。
type DbDataImport interface {
	// Preview 解析文件表头与样本行，供前端构建列映射界面（不落库）
	Preview(ctx context.Context, req *dto.ImportPreviewReq) (*dto.ImportPreviewRes, error)

	// Import 按列映射把文件数据批量插入目标表，单事务提交，任一批失败即整体回滚
	Import(ctx context.Context, req *dto.DataImportReq) (*dto.DataImportRes, error)
}

type dbDataImportAppImpl struct {
}

var _ DbDataImport = (*dbDataImportAppImpl)(nil)

func (d *dbDataImportAppImpl) Preview(_ context.Context, req *dto.ImportPreviewReq) (*dto.ImportPreviewRes, error) {
	imp, err := importer.ForFilename(req.Filename)
	if err != nil {
		return nil, err
	}
	opts := req.Options
	if opts == nil {
		opts = &importer.Options{}
	}
	// 预览只取前若干行，避免为构建映射界面解析整份大文件
	previewOpts := *opts
	previewOpts.PreviewLimit = importPreviewRowCount

	table, err := imp.Parse(req.Reader, &previewOpts)
	if err != nil {
		return nil, err
	}
	// 无表头时 Headers 为 nil，序列化为 JSON null 会让前端读 .length 抛错，归一为空切片保证契约稳定
	headers := table.Headers
	if headers == nil {
		headers = []string{}
	}
	return &dto.ImportPreviewRes{
		Headers:    headers,
		SampleRows: table.Rows,
		TotalRows:  len(table.Rows),
		Sheets:     table.Sheets,
	}, nil
}

// importPreviewRowCount 预览返回的最大样本行数：够前端展示并推断列类型映射即可，无需整表
const importPreviewRowCount = 20

func (d *dbDataImportAppImpl) Import(ctx context.Context, req *dto.DataImportReq) (*dto.DataImportRes, error) {
	conn := req.DbConn
	if len(req.Columns) == 0 {
		return nil, errorx.NewBiz("no column mapping provided for import")
	}

	imp, err := importer.ForFilename(req.Filename)
	if err != nil {
		return nil, err
	}
	opts := req.Options
	if opts == nil {
		opts = &importer.Options{}
	}
	// 全量导入：不限预览行数
	fullOpts := *opts
	fullOpts.PreviewLimit = 0
	table, err := imp.Parse(req.Reader, &fullOpts)
	if err != nil {
		return nil, err
	}
	if len(table.Rows) == 0 {
		return &dto.DataImportRes{TotalRows: 0}, nil
	}

	// 目标表列元数据：按列名（不区分大小写）建立索引，供列映射解析与类型识别
	tableColumns, err := conn.Metadata().GetColumns(req.TableName)
	if err != nil {
		return nil, errorx.NewBizf("get columns of table [%s] failed: %s", req.TableName, err.Error())
	}
	colByName := make(map[string]dbi.Column, len(tableColumns))
	for _, c := range tableColumns {
		colByName[strings.ToLower(c.ColumnName)] = c
	}

	// 解析映射：为每条映射定位文件列下标（越界即报错，避免整列静默写 NULL），并解析目标列元数据。
	// 跳过（Target 为空）的映射不参与插入；重复目标列会生成 (a, a) 的非法 INSERT，前置拒绝。
	sourceIndex, err := resolveSourceColumns(table, req.Columns)
	if err != nil {
		return nil, err
	}
	insertColumns := make([]dbi.Column, 0, len(req.Columns))
	keptSources := make([]int, 0, len(req.Columns)) // 与 insertColumns 一一对应的文件列下标
	seenTarget := make(map[string]bool, len(req.Columns))
	for i, mapping := range req.Columns {
		if mapping.Target == "" {
			continue
		}
		key := strings.ToLower(mapping.Target)
		if seenTarget[key] {
			return nil, errorx.NewBizf("duplicate target column mapping for [%s]", mapping.Target)
		}
		seenTarget[key] = true
		col, ok := colByName[key]
		if !ok {
			return nil, errorx.NewBizf("target column [%s] does not exist in table [%s]", mapping.Target, req.TableName)
		}
		insertColumns = append(insertColumns, col)
		keptSources = append(keptSources, sourceIndex[i])
	}
	if len(insertColumns) == 0 {
		return nil, errorx.NewBiz("no valid target column mapped for import")
	}

	// 自增/标识列若被映射但整列在文件里全为空，则从插入列中剔除，交由数据库生成下一个值。
	// 写 NULL 到自增列只有 MySQL 碰巧接受，PG serial/identity、Oracle/MSSQL 标识列会直接报错。
	insertColumns, keptSources = dropAllEmptyAutoIncrement(table, insertColumns, keptSources)
	if len(insertColumns) == 0 {
		return nil, errorx.NewBiz("all mapped columns are auto-increment with empty values, nothing to insert")
	}

	// 逐列判定是否为二进制类列：文件单元格是文本，若原样交给迁移 codec 的 SQLValueBytes，
	// 形如 MD5/无横线 UUID 的「偶数长度全 hex」文本会被当作十六进制解码成字节而静默损坏。
	// 二进制列按字面字节写入（[]byte(text)），语义对导入文本而言唯一且可复现。
	colIsBinary := make([]bool, len(insertColumns))
	colIsTextual := make([]bool, len(insertColumns))
	for i, col := range insertColumns {
		colIsBinary[i] = isBinaryCategory(conn, col.DataType)
		colIsTextual[i] = isTextualCategory(conn, col.DataType)
	}

	// 装配待插入行值：按 keptSources 顺序取文件单元格（与 insertColumns 严格对齐），
	// 空单元格按选项与列可空性归一（见 cellValue）；二进制列转字面字节；其余交由方言 codec 归一
	values := make([][]any, 0, len(table.Rows))
	for _, row := range table.Rows {
		vals := make([]any, len(insertColumns))
		for i, srcIdx := range keptSources {
			vals[i] = cellValue(row, srcIdx, req.EmptyAsNull, colIsBinary[i], colIsTextual[i], insertColumns[i].Nullable)
		}
		values = append(values, vals)
	}

	strategy := req.DuplicateStrategy
	if strategy == 0 {
		strategy = dbi.DuplicateStrategyNone
	}
	// 冲突键仅在 upsert/merge(Update) 下才需要：MySQL insert ignore / PG on conflict do nothing 等
	// 忽略策略无需指定冲突列（可匹配任意唯一约束），故 Ignore 传 nil 即可，不能强求主键。
	var targetMeta *dbi.TargetTableMeta
	if strategy == dbi.DuplicateStrategyUpdate {
		targetMeta = dbsync.BuildTargetTableMeta(conn, req.TableName, tableColumns)
		if len(targetMeta.UniqueColumns) == 0 {
			return nil, errorx.NewBizf("cannot update on conflict: table [%s] has no primary key or single unique index to resolve conflicts", req.TableName)
		}
		// 冲突键列必须出现在映射列中：否则 on-conflict/merge 依据的列不在插入列里，
		// 数据库要么生成新值永不冲突（静默插重复，如 sqlite/PG/MySQL），要么报无效标识符（Oracle/达梦）——
		// 两种都不是用户要的「更新冲突行」，故前置拒绝并点名缺失列。
		mapped := make(map[string]bool, len(insertColumns))
		for _, c := range insertColumns {
			mapped[strings.ToLower(c.ColumnName)] = true
		}
		var missing []string
		for _, uk := range targetMeta.UniqueColumns {
			if !mapped[strings.ToLower(uk)] {
				missing = append(missing, uk)
			}
		}
		if len(missing) > 0 {
			return nil, errorx.NewBizf("update-on-conflict requires the conflict key column(s) %v to be mapped in the import file", missing)
		}
	}

	batchSize := req.BatchSize
	if batchSize <= 0 {
		batchSize = dto.DefaultImportBatchSize
	}
	sqlGen := conn.GetDialect().GetSQLGenerator()

	// 数据导入与 SQL 编辑器共用同一个权限码（db:sqlscript:run），也就必须共用同一套触发策略判定：
	// 否则「向该表插入数据需审批」在编辑器里拦得住，换个 Excel/CSV 文件导入就绕过去了。
	// 判定用「往该表插入一行」的代表语句而不是实际批次语句：批次大小是实现细节，
	// 拿它决定策略结论会出现「调小批量就绕过、调大批量就误拦」
	var warnings []string
	if probeStmts := sqlGen.GenInsert(req.TableName, insertColumns, values[:1], strategy, targetMeta); len(probeStmts) > 0 {
		notices, err := GetDbSQLExecApp().CheckSqlsWithoutTicket(ctx, conn, probeStmts)
		if err != nil {
			return nil, err
		}
		warnings = notices
	}

	tx, err := conn.Begin()
	if err != nil {
		return nil, errorx.NewBizf("begin import transaction failed: %s", err.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			dbi.RollbackTx(tx)
			panic(r)
		}
	}()

	res := &dto.DataImportRes{TotalRows: len(table.Rows), Warnings: warnings}

	// 进度回传：与「SQL文件执行」同构，仅在登录账号 + clientId + uploadId 齐全时发送 Ws 进度消息。
	// 事件订阅为异步（SubscribeAsync）且 Ws 发送持有 Params 引用，故每次发布必须 copy 一份新 map、
	// 造一个新事件，杜绝跨 goroutine 读写同一 map（并发 map 读写是不可 recover 的进程级 fatal）。
	la := contextx.GetLoginAccount(ctx)
	needSendMsg := la != nil && req.ClientId != "" && req.UploadId != ""
	dbInfo := conn.Info
	var receiverIds []uint64
	if la != nil {
		receiverIds = []uint64{la.Id}
	}
	baseParams := collx.M{
		"uploadId": req.UploadId,
		"clientId": req.ClientId,
		"dbCode":   dbInfo.DbCode,
		"dbName":   dbInfo.Name,
		"table":    req.TableName,
		"title":    stringx.Truncate(req.Filename, 20, 10, "..."),
		"total":    res.TotalRows,
	}
	startTime := time.Now()
	publishProgress := func(terminated bool, imported int, status string) {
		if !needSendMsg {
			return
		}
		params := collx.CopyM(baseParams)
		params["imported"] = imported
		params["terminated"] = terminated
		if status != "" {
			params["status"] = status
		}
		if terminated {
			params["costMs"] = time.Since(startTime).Milliseconds()
		}
		global.EventBus.Publish(ctx, event.EventTopicMsgTmplSend, &msgdto.MsgTmplSendEvent{
			TmplChannel: msgdto.MsgTmplDataImportProgress,
			Params:      params,
			ReceiverIds: receiverIds,
		})
	}
	// committed 为 false 时（含 panic/error 返回）终止消息回报 imported=0 与失败状态：
	// 单事务下任何未提交的计数都随回滚清零，报部分计数会误导用户以为已落库。
	committed := false
	defer func() {
		if committed {
			publishProgress(true, res.Imported, "success")
		} else {
			publishProgress(true, 0, "failed")
		}
	}()

	for start := 0; start < len(values); start += batchSize {
		end := min(start+batchSize, len(values))
		batch := values[start:end]
		stmts := sqlGen.GenInsert(req.TableName, insertColumns, batch, strategy, targetMeta)
		if len(stmts) == 0 {
			dbi.RollbackTx(tx)
			return nil, errorx.NewBiz("target dialect generated no insert statement for a non-empty batch")
		}
		for _, stmt := range stmts {
			if ctx.Err() != nil {
				dbi.RollbackTx(tx)
				return nil, errorx.NewBiz("import cancelled")
			}
			affected, execErr := conn.TxExecContext(ctx, tx, stmt)
			if execErr != nil {
				dbi.RollbackTx(tx)
				return nil, errorx.NewBizf("import failed at row batch %d-%d: %s", start+1, end, execErr.Error())
			}
			res.BatchCount++
			res.AffectedRow += affected
		}
		res.Imported += len(batch)
		publishProgress(false, res.Imported, "importing")
	}

	if err := tx.Commit(); err != nil {
		dbi.RollbackTx(tx)
		return nil, errorx.NewBizf("commit import transaction failed: %s", err.Error())
	}
	committed = true
	return res, nil
}

// resolveSourceColumns 为每条映射解析对应的文件列下标：映射的 Source 为列位置下标的字符串
// （有表头/无表头一致，前端按解析后的列序稳定生成）。按位置而非列名匹配，杜绝同名表头被解析到
// 同一列导致数据错位；越界下标直接报错而非静默写 NULL。跳过（Target 为空）的映射返回 -1。
func resolveSourceColumns(table *importer.Table, mappings []dto.ImportColumn) ([]int, error) {
	fileCols := table.ColumnCount()
	idxs := make([]int, len(mappings))
	for i, m := range mappings {
		if m.Target == "" {
			idxs[i] = -1
			continue
		}
		colIdx, perr := strconv.Atoi(m.Source)
		if perr != nil || colIdx < 0 || colIdx >= fileCols {
			return nil, errorx.NewBizf("invalid file column index [%s], must be within [0, %d)", m.Source, fileCols)
		}
		idxs[i] = colIdx
	}
	return idxs, nil
}

// dropAllEmptyAutoIncrement 剔除「被映射为自增/标识列、且文件里整列所有行都为空」的列，
// 让数据库自行生成主键值。返回过滤后的列与对应文件下标（保持一一对应）。
func dropAllEmptyAutoIncrement(table *importer.Table, columns []dbi.Column, sources []int) ([]dbi.Column, []int) {
	drop := make([]bool, len(columns))
	anyDrop := false
	for i, col := range columns {
		if !col.AutoIncrement {
			continue
		}
		allEmpty := true
		for _, row := range table.Rows {
			if ci := sources[i]; ci >= 0 && ci < len(row) && strings.TrimSpace(row[ci]) != "" {
				allEmpty = false
				break
			}
		}
		if allEmpty {
			drop[i] = true
			anyDrop = true
		}
	}
	if !anyDrop {
		return columns, sources
	}
	keptCols := make([]dbi.Column, 0, len(columns))
	keptSrc := make([]int, 0, len(columns))
	for i := range columns {
		if drop[i] {
			continue
		}
		keptCols = append(keptCols, columns[i])
		keptSrc = append(keptSrc, sources[i])
	}
	return keptCols, keptSrc
}

// cellValue 取行值（含空单元格/越界的归一）。空值的落库形态按「列是否可空 + 列类型」决定，而非只看
// EmptyAsNull 开关——因为 ” 只对文本列是合法值，对日期/时间、数值等列写 ” 会被数据库判为非法值（MySQL 1292）：
//   - 二进制列：空 → 空字节（合法）。
//   - NOT NULL 文本列：空 → 空串（写 NULL 必 1048）。
//   - NOT NULL 非文本列：空 → NULL，交由数据库给清晰的「不能为空」错误（数据确实缺必填值，不静默塞非法 ”）。
//   - 可空列：勾选「空→NULL」或非文本列 → NULL；仅「可空文本列且未勾选」写空串。
//
// 非空二进制列文本按字面字节写入（避免被 SQLValueBytes 当 hex 解码）；其余原样透传给方言列类型 codec 归一。
func cellValue(row []string, colIdx int, emptyAsNull, isBinary, isText, nullable bool) any {
	v := ""
	if colIdx >= 0 && colIdx < len(row) {
		v = row[colIdx]
	}
	if v == "" {
		switch {
		case isBinary:
			return []byte{}
		case !nullable:
			if isText {
				return ""
			}
			return nil
		case emptyAsNull || !isText:
			return nil
		default:
			return ""
		}
	}
	if isBinary {
		return []byte(v)
	}
	return v
}

// isBinaryCategory 判定目标列类型是否二进制族（blob/binary/varbinary），其值不得走文本→hex 猜测路径
func isBinaryCategory(conn *dbi.DbConn, dataType string) bool {
	switch dbi.GetDbDataType(conn.Info.Type, dataType).Category() {
	case dbi.TCBinary, dbi.TCVarbinary, dbi.TCBlob, dbi.TCMediumblob, dbi.TCLongblob:
		return true
	default:
		return false
	}
}

// isTextualCategory 判定目标列是否为「空串是合法值」的文本族（varchar/char/text 系列）。
// 日期/时间、数值、枚举、json 等类型即使可空，空单元格也不能写 ”（会被判非法值），只能写 NULL。
func isTextualCategory(conn *dbi.DbConn, dataType string) bool {
	switch dbi.GetDbDataType(conn.Info.Type, dataType).Category() {
	case dbi.TCVarchar, dbi.TCChar, dbi.TCText, dbi.TCMediumtext, dbi.TCLongtext:
		return true
	default:
		return false
	}
}
