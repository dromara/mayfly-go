package application

import (
	"context"
	"io"
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

// maxImportBatchSize 单条批量 INSERT 的服务端硬上限（与前端输入上限一致）：
// batchSize 直接来自请求参数，若不收敛，make([][]any, 0, batchSize) 会在读数据前按用户值
// 预分配（1e9 ≈ 24GB），触发不可 recover 的 OOM 使整个进程退出
const maxImportBatchSize = 5000

// importBatchCapHint 批次切片预分配上限：实际批次仍可增长到 batchSize，但初始容量不随请求值放大
const importBatchCapHint = 1024

// clampImportBatchSize 把请求的 batchSize 收敛到 [默认值, 硬上限] 区间
func clampImportBatchSize(batchSize int) int {
	if batchSize <= 0 {
		return dto.DefaultImportBatchSize
	}
	if batchSize > maxImportBatchSize {
		return maxImportBatchSize
	}
	return batchSize
}

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
	// 全量导入：预览限制关闭；行数据端到端流式（Excel 由 excelize 整体读文件，但行不聚合）
	fullOpts := *opts
	fullOpts.PreviewLimit = 0

	// 目标表列元数据：按列名（不区分大小写）建立索引，供列映射解析与类型识别
	tableColumns, err := conn.Metadata().GetColumns(req.TableName)
	if err != nil {
		return nil, errorx.NewBizf("get columns of table [%s] failed: %s", req.TableName, err.Error())
	}
	colByName := make(map[string]dbi.Column, len(tableColumns))
	for _, c := range tableColumns {
		colByName[strings.ToLower(c.ColumnName)] = c
	}

	// ===== 第一遍（流式）：表头 + 总行数 + 每列是否出现过非空值 =====
	// 自增列「整列为空」的判定需要知道整列是否出现过非空，故先扫一遍；该遍不驻留行数据。
	reader := req.Reader
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, errorx.NewBizf("import requires a seekable file: %s", err.Error())
	}
	var headers []string
	var firstRow []string
	maxCols := 0
	totalRows := 0
	anyNonEmpty := make([]bool, 0, 16)
	resizeAny := func(n int) {
		if n > len(anyNonEmpty) {
			anyNonEmpty = append(anyNonEmpty, make([]bool, n-len(anyNonEmpty))...)
		}
	}
	if err := imp.Stream(reader, &fullOpts, &importer.StreamHandler{
		OnHeader: func(hs []string, _ []string) error {
			headers = hs
			if len(hs) > maxCols {
				maxCols = len(hs)
			}
			resizeAny(len(hs))
			return nil
		},
		OnRow: func(row []string) error {
			if ctx.Err() != nil {
				return errorx.NewBiz("import cancelled")
			}
			totalRows++
			if firstRow == nil {
				firstRow = append([]string(nil), row...)
			}
			if len(row) > maxCols {
				maxCols = len(row)
			}
			resizeAny(len(row))
			for i, cell := range row {
				if !anyNonEmpty[i] && strings.TrimSpace(cell) != "" {
					anyNonEmpty[i] = true
				}
			}
			return nil
		},
	}); err != nil {
		return nil, err
	}
	if totalRows == 0 {
		return &dto.DataImportRes{TotalRows: 0}, nil
	}
	// 列数口径与原解析一致：有表头取表头宽度，否则取数据行最大宽度
	fileCols := len(headers)
	if fileCols == 0 {
		fileCols = maxCols
	}

	// 解析映射：为每条映射定位文件列下标（越界即报错，避免整列静默写 NULL），并解析目标列元数据。
	// 跳过（Target 为空）的映射不参与插入；重复目标列会生成 (a, a) 的非法 INSERT，前置拒绝。
	sourceIndex, err := resolveSourceColumns(fileCols, req.Columns)
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
	// 用第一遍的 anyNonEmpty 判定，不再回扫行数据（流式约束）
	insertColumns, keptSources = dropAllEmptyAutoIncrement(anyNonEmpty, insertColumns, keptSources)
	if len(insertColumns) == 0 {
		return nil, errorx.NewBiz("all mapped columns are auto-increment with empty values, nothing to insert")
	}

	// 逐列判定是否为二进制/文本类列（文件单元格是文本，二进制列按字面字节写入）
	colIsBinary := make([]bool, len(insertColumns))
	colIsTextual := make([]bool, len(insertColumns))
	for i, col := range insertColumns {
		colIsBinary[i] = isBinaryCategory(conn, col.DataType)
		colIsTextual[i] = isTextualCategory(conn, col.DataType)
	}

	strategy := req.DuplicateStrategy
	if strategy == 0 {
		strategy = dbi.DuplicateStrategyNone
	}
	// 冲突键仅在 upsert/merge(Update) 下才需要：Ignore 策略无需指定冲突列
	var targetMeta *dbi.TargetTableMeta
	if strategy == dbi.DuplicateStrategyUpdate {
		targetMeta = dbsync.BuildTargetTableMeta(conn, req.TableName, tableColumns)
		if len(targetMeta.UniqueColumns) == 0 {
			return nil, errorx.NewBizf("cannot update on conflict: table [%s] has no primary key or single unique index to resolve conflicts", req.TableName)
		}
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

	batchSize := clampImportBatchSize(req.BatchSize)
	sqlGen := conn.GetDialect().GetSQLGenerator()

	// 数据导入与 SQL 编辑器共用同一个权限码（db:sqlscript:run），必须共用同一套触发策略判定。
	// 判定必须在开事务前完成：被策略拒绝时不应开启事务，更不能「写完再问」。
	// 代表语句用「往该表插入一行」而不是实际批次语句（批次大小是实现细节）。
	var warnings []string
	if firstRow != nil {
		values := make([][]any, 1)
		values[0] = make([]any, len(insertColumns))
		for i, srcIdx := range keptSources {
			values[0][i] = cellValue(firstRow, srcIdx, req.EmptyAsNull, colIsBinary[i], colIsTextual[i], insertColumns[i].Nullable)
		}
		// 代表语句固定用「插入一行」：批次大小是实现细节，不应影响策略结论
		if probeStmts := sqlGen.GenInsert(req.TableName, insertColumns, values[:1], strategy, targetMeta); len(probeStmts) > 0 {
			notices, checkErr := GetDbSQLExecApp().CheckSqlsWithoutTicket(ctx, conn, probeStmts)
			if checkErr != nil {
				return nil, checkErr
			}
			warnings = notices
		}
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

	res := &dto.DataImportRes{TotalRows: totalRows, Warnings: warnings}

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

	// ===== 第二遍（流式）：逐行转换 → 按批生成 INSERT → 事务内执行 =====
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		dbi.RollbackTx(tx)
		return nil, errorx.NewBizf("import requires a seekable file: %s", err.Error())
	}

	capHint := batchSize
	if capHint > importBatchCapHint {
		capHint = importBatchCapHint
	}
	batch := make([][]any, 0, capHint)

	flushBatch := func() error {
		if len(batch) == 0 {
			return nil
		}
		stmts := sqlGen.GenInsert(req.TableName, insertColumns, batch, strategy, targetMeta)
		if len(stmts) == 0 {
			return errorx.NewBiz("target dialect generated no insert statement for a non-empty batch")
		}
		batchStart := res.Imported + 1
		for _, stmt := range stmts {
			if ctx.Err() != nil {
				return errorx.NewBiz("import cancelled")
			}
			affected, execErr := conn.TxExecContext(ctx, tx, stmt)
			if execErr != nil {
				return errorx.NewBizf("import failed at row batch starting at %d: %s", batchStart, execErr.Error())
			}
			res.BatchCount++
			res.AffectedRow += affected
		}
		res.Imported += len(batch)
		batch = batch[:0]
		publishProgress(false, res.Imported, "importing")
		return nil
	}

	if err := imp.Stream(reader, &fullOpts, &importer.StreamHandler{
		OnRow: func(row []string) error {
			if ctx.Err() != nil {
				return errorx.NewBiz("import cancelled")
			}
			vals := make([]any, len(insertColumns))
			for i, srcIdx := range keptSources {
				vals[i] = cellValue(row, srcIdx, req.EmptyAsNull, colIsBinary[i], colIsTextual[i], insertColumns[i].Nullable)
			}
			batch = append(batch, vals)
			if len(batch) >= batchSize {
				return flushBatch()
			}
			return nil
		},
	}); err != nil {
		dbi.RollbackTx(tx)
		return nil, err
	}
	if err := flushBatch(); err != nil {
		dbi.RollbackTx(tx)
		return nil, err
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
func resolveSourceColumns(fileCols int, mappings []dto.ImportColumn) ([]int, error) {
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
// 让数据库自行生成主键值。anyNonEmpty[i] 表示文件第 i 列是否出现过非空单元格（第一遍流式扫描结果）。
// 返回过滤后的列与对应文件下标（保持一一对应）。
func dropAllEmptyAutoIncrement(anyNonEmpty []bool, columns []dbi.Column, sources []int) ([]dbi.Column, []int) {
	drop := make([]bool, len(columns))
	anyDrop := false
	for i, col := range columns {
		if !col.AutoIncrement {
			continue
		}
		ci := sources[i]
		allEmpty := ci < 0 || ci >= len(anyNonEmpty) || !anyNonEmpty[ci]
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
