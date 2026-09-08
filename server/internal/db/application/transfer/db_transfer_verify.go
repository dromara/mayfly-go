package transfer

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	sysapp "mayfly-go/internal/sys/application"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
)

// 校验抽样配置
const (
	// VerifySampleRows 单表抽样比对总行数预算
	VerifySampleRows = 1000
	// VerifySampleWindows 大表抽样窗口数（沿主键序均匀分布，覆盖头/中/尾）
	VerifySampleWindows = 5
	// VerifyPkInChunkSize 目标侧按主键回捞时单条IN查询的主键值上限（避免超长IN列表）
	VerifyPkInChunkSize = 200
	// VerifyMaxMismatchReport 单表最多记录的内容不一致主键数（防止报告过大）
	VerifyMaxMismatchReport = 20
)

// TableVerifyResult 单表校验结果
type TableVerifyResult struct {
	TableName   string   `json:"tableName"`
	SrcCount    int64    `json:"srcCount"`
	TargetCount int64    `json:"targetCount"`
	CountMatch  bool     `json:"countMatch"`
	Sampled     int      `json:"sampled"`             // 实际抽样比对的行数
	MismatchPk  []string `json:"mismatchPk"`          // 内容不一致/目标缺失行的主键规范化值（最多VerifyMaxMismatchReport个）
	SampleErr   string   `json:"sampleErr,omitempty"` // 抽样比对过程错误（如无主键列、分页语法不支持），不影响count比对
	Err         string   `json:"err,omitempty"`       // 整表校验错误（如查询失败）
}

// TransferVerifyReport 迁移任务数据校验报告
type TransferVerifyReport struct {
	TaskId   uint64              `json:"taskId"`
	Results  []TableVerifyResult `json:"results"`
	AllMatch bool                `json:"allMatch"` // 所有表count一致且抽样内容无差异
}

// Verify 校验迁移任务源库与目标库的数据一致性（复用Run的异步+syslog日志模式）。
// return logId, error
func (app *DbTransferAppImpl) Verify(ctx context.Context, taskId uint64) (uint64, error) {
	// 先原子占位再创建运行日志：顺序颠倒时并发触发的失败方会留下一条永不结束的Running日志
	if !app.runGuard.Acquire(taskId) {
		return 0, errorx.NewBizf("the db transfer task [%d] is running, please do not repeat the operation", taskId)
	}

	task, err := app.GetById(taskId)
	if err != nil {
		app.runGuard.Release(taskId)
		return 0, errorx.NewBizf("db transfer task [%d] not found", taskId)
	}

	logId, err := app.logApp.CreateLog(ctx, &sysapp.CreateLogReq{
		Description: "DBMS - Verify DB Transfer",
		ReqParam:    collx.Kvs("taskId", taskId),
		Type:        sysentity.SyslogTypeRunning,
		Resp:        "Data verification starts...",
	})
	if err != nil {
		app.runGuard.Release(taskId)
		return 0, err
	}
	task.LogId = logId
	task.RunningState = entity.DbTransferTaskRunStateRunning
	if err = app.UpdateById(ctx, task); err != nil {
		app.runGuard.Release(taskId)
		return logId, err
	}

	gox.Go(func() {
		// 后台异步执行，脱离请求ctx取消信号并保留链路信息
		ctx = context.WithoutCancel(ctx)
		defer app.runGuard.Release(taskId)
		defer app.logApp.Flush(logId, true)

		report := app.buildVerifyReport(ctx, logId, task)

		// 报告写入日志Resp（可查询）
		reportJson, err := json.Marshal(report)
		if err != nil {
			app.Log(ctx, logId, fmt.Sprintf("marshal verify report failed: %s", err.Error()))
			return
		}
		logType := sysentity.SyslogTypeSuccess
		if !report.AllMatch {
			logType = sysentity.SyslogTypeError
		}
		app.logApp.AppendLog(logId, &sysapp.AppendLogReq{
			AppendResp: string(reportJson),
			Type:       logType,
		})

		// 更新任务运行状态
		transferState := entity.DbTransferTaskRunStateSuccess
		if !report.AllMatch {
			transferState = entity.DbTransferTaskRunStateFail
		}
		ut := new(entity.DbTransferTask)
		ut.Id = taskId
		ut.RunningState = transferState
		if err := app.UpdateById(context.Background(), ut); err != nil {
			logx.Errorf("failed to update transfer task [%d] running state: %s", taskId, err.Error())
		}
	}, func(panicErr error) {
		// panic兜底：结束日志并重置任务运行态，否则任务永久停留在Running
		app.logApp.AppendLog(logId, &sysapp.AppendLogReq{
			AppendResp: fmt.Sprintf("db transfer verify panicked: %s", panicErr.Error()),
			Type:       sysentity.SyslogTypeError,
		})
		ut := new(entity.DbTransferTask)
		ut.Id = taskId
		ut.RunningState = entity.DbTransferTaskRunStateFail
		if err := app.UpdateById(context.Background(), ut); err != nil {
			logx.Errorf("failed to update transfer task [%d] running state: %s", taskId, err.Error())
		}
	})

	return logId, nil
}

// buildVerifyReport 构建校验报告：逐表count比对 + 头/尾抽样内容比对
func (app *DbTransferAppImpl) buildVerifyReport(ctx context.Context, logId uint64, task *entity.DbTransferTask) *TransferVerifyReport {
	report := &TransferVerifyReport{TaskId: task.Id}

	srcConn, err := app.dbApp.GetDbConn(ctx, uint64(task.SrcDbId), task.SrcDbName)
	if err != nil {
		app.Log(ctx, logId, "failed to obtain source db connection: "+err.Error())
		report.Results = []TableVerifyResult{{Err: "failed to obtain source db connection: " + err.Error()}}
		return report
	}
	targetConn, err := app.dbApp.GetDbConn(ctx, uint64(task.TargetDbId), task.TargetDbName)
	if err != nil {
		app.Log(ctx, logId, "failed to obtain target db connection: "+err.Error())
		report.Results = []TableVerifyResult{{Err: "failed to obtain target db connection: " + err.Error()}}
		return report
	}

	// 待校验表（与Run取表逻辑一致）
	var tables []dbi.Table
	if task.CheckedKeys == "all" {
		tables, err = srcConn.GetMetadata().GetTables()
	} else {
		tables, err = srcConn.GetMetadata().GetTables(strings.Split(task.CheckedKeys, ",")...)
	}
	if err != nil {
		app.Log(ctx, logId, "failed to get source table information: "+err.Error())
		report.Results = []TableVerifyResult{{Err: "failed to get source table information: " + err.Error()}}
		return report
	}

	tableNames := collx.ArrayMap(tables, func(t dbi.Table) string { return t.TableName })
	sort.Strings(tableNames)

	results := make([]TableVerifyResult, 0, len(tableNames))
	for _, tableName := range tableNames {
		res := app.verifyTable(ctx, srcConn, targetConn, tableName)
		results = append(results, res)
		if res.Err != "" {
			app.Log(ctx, logId, fmt.Sprintf("verify table [%s] error: %s", tableName, res.Err))
		} else if !res.CountMatch || len(res.MismatchPk) > 0 {
			app.Log(ctx, logId, fmt.Sprintf("verify table [%s] mismatch: src=%d target=%d mismatchPk=%v",
				tableName, res.SrcCount, res.TargetCount, res.MismatchPk))
		} else {
			app.Log(ctx, logId, fmt.Sprintf("verify table [%s] match: count=%d, sampled=%d", tableName, res.SrcCount, res.Sampled))
		}
	}
	report.Results = results
	allMatch := true
	for _, res := range results {
		if res.Err != "" || !res.CountMatch || len(res.MismatchPk) > 0 {
			allMatch = false
			break
		}
	}
	report.AllMatch = allMatch
	return report
}

// verifyTable 校验单表：两侧count(*)比对 + 全量/多窗口抽样内容比对。
// 抽样失败（无单列主键/查询异常）不视为校验失败，记录至SampleErr，count比对照常。
func (app *DbTransferAppImpl) verifyTable(ctx context.Context, srcConn, targetConn *dbi.DbConn, tableName string) TableVerifyResult {
	res := TableVerifyResult{TableName: tableName}

	// count比对
	srcCount, err := countTableRows(ctx, srcConn, tableName)
	if err != nil {
		res.Err = fmt.Sprintf("query source count failed: %s", err.Error())
		return res
	}
	targetCount, err := countTableRows(ctx, targetConn, tableName)
	if err != nil {
		res.Err = fmt.Sprintf("query target count failed: %s", err.Error())
		return res
	}
	res.SrcCount = srcCount
	res.TargetCount = targetCount
	res.CountMatch = srcCount == targetCount

	// 抽样内容比对：需要两侧均有单列主键（任意类型）作对齐键
	srcPk := singlePkColumn(srcConn, tableName)
	tgtPk := singlePkColumn(targetConn, tableName)
	if srcPk == "" || tgtPk == "" {
		res.SampleErr = "no single primary key column, skip content sampling"
		return res
	}

	srcRows, sampleErr := sampleSourceRows(ctx, srcConn, tableName, srcPk, srcCount)
	if sampleErr != nil {
		// 抽样错误不可等同于“空表”：旧实现吞错后返回nil，两者无法区分，
		// 导致查询失败被当成无需比对而静默报“一致”，必须记录真实失败原因
		res.SampleErr = fmt.Sprintf("sample source rows failed: %s", sampleErr.Error())
		return res
	}
	if len(srcRows) == 0 {
		if srcCount > 0 {
			res.SampleErr = fmt.Sprintf("source table has %d rows but sampling got no rows", srcCount)
		}
		return res // 空表无需内容比对
	}

	// 目标侧按源侧抽到的主键值精确回捞，保证两侧比对的是同一批行（不能按目标侧offset抽样，
	// 两侧行数不一致时窗口会错位而产生假阳性差异）
	tgtRows, fetchErr := fetchRowsByPkValues(ctx, targetConn, tableName, tgtPk, srcRows, srcPk)
	if fetchErr != nil {
		res.SampleErr = fmt.Sprintf("fetch target rows by pk failed: %s", fetchErr.Error())
		return res
	}
	res.Sampled = distinctRowCount(srcRows, srcPk)
	res.MismatchPk = compareSampledRows(srcRows, tgtRows, srcPk, tgtPk, numericCommonColumns(srcConn, targetConn, tableName), VerifyMaxMismatchReport)
	return res
}

// sampleSourceRows 源侧抽样：行数不超预算时全量读取（小表100%校验），
// 否则沿主键序均匀切分为VerifySampleWindows个窗口（首个为头部、末个为尾部），
// 避免仅抽头/尾导致中间区段的差异永远检不出
func sampleSourceRows(ctx context.Context, conn *dbi.DbConn, tableName, pk string, total int64) ([]map[string]any, error) {
	if total <= 0 {
		return nil, nil
	}
	if total <= VerifySampleRows {
		return sampleOrderedRows(ctx, conn, tableName, pk, int(total), 0)
	}

	perWindow := VerifySampleRows / VerifySampleWindows
	span := total - int64(perWindow)
	rows := make([]map[string]any, 0, VerifySampleRows)
	for i := 0; i < VerifySampleWindows; i++ {
		offset := int64(i) * span / int64(VerifySampleWindows-1)
		windowRows, err := sampleOrderedRows(ctx, conn, tableName, pk, perWindow, offset)
		if err != nil {
			return nil, err
		}
		rows = append(rows, windowRows...)
	}
	return rows, nil
}

// fetchRowsByPkValues 按源侧抽样的主键值从目标表回捞对应行（IN分批查询）。
// 主键值以目标库该列的数据类型生成字面量，避免跨方言的日期/二进制呈现差异
func fetchRowsByPkValues(ctx context.Context, conn *dbi.DbConn, tableName, pkColumn string, srcRows []map[string]any, srcPk string) ([]map[string]any, error) {
	sqlValue, err := pkColumnSqlValue(conn, tableName, pkColumn)
	if err != nil {
		return nil, err
	}

	quote := conn.GetDialect().Quoter().QuoteIdent
	seen := make(map[string]struct{}, len(srcRows))
	values := make([]string, 0, len(srcRows))
	for _, row := range srcRows {
		pkVal := row[srcPk]
		if pkVal == nil {
			continue // NULL主键无法对齐，compareSampledRows同样跳过
		}
		literal := sqlValue(pkVal)
		if _, ok := seen[literal]; ok {
			continue
		}
		seen[literal] = struct{}{}
		values = append(values, literal)
	}

	res := make([]map[string]any, 0, len(values))
	for start := 0; start < len(values); start += VerifyPkInChunkSize {
		end := min(start+VerifyPkInChunkSize, len(values))
		sqlStr := fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)", quote(tableName), quote(pkColumn), strings.Join(values[start:end], ", "))
		_, rows, err := conn.QueryContext(ctx, sqlStr)
		if err != nil {
			return nil, err
		}
		res = append(res, rows...)
	}
	return res, nil
}

// pkColumnSqlValue 获取目标表主键列对应的SQL字面量生成函数
func pkColumnSqlValue(conn *dbi.DbConn, tableName, pkColumn string) (func(any) string, error) {
	columns, err := conn.GetMetadata().GetColumns(tableName)
	if err != nil {
		return nil, fmt.Errorf("get columns of table [%s] failed: %w", tableName, err)
	}
	for i := range columns {
		if columns[i].ColumnName == pkColumn {
			return dbi.GetDbDataType(conn.Info.Type, columns[i].DataType).DataType.SQLValue, nil
		}
	}
	return nil, fmt.Errorf("primary key column [%s] not found in table [%s]", pkColumn, tableName)
}

// distinctRowCount 统计实际参与比对的行数（按主键去重）：多窗口抽样区间重叠或同表多窗口
// 取到同一行时，直接取len(rows)会使报告行数翻倍
func distinctRowCount(rows []map[string]any, pk string) int {
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if row[pk] == nil { // 主键为NULL的行无法对齐，不参与比对也不计数
			continue
		}
		seen[dbi.CanonicalRowKey(row, pk)] = struct{}{}
	}
	return len(seen)
}

// numericCommonColumns 取两侧均为数值类的列名集合（小写）：此类列跨方言比对时按数值语义判等，
// 容忍同一数值的标度呈现差异（如numeric(20,6)的"1.500000"与sqlite NUMERIC亲和后的"1.5"）；
// 任一侧列元数据获取失败或类型非数值均不入集，保持严格文本比对（宁可误报不可漏报）
func numericCommonColumns(srcConn, tgtConn *dbi.DbConn, tableName string) map[string]bool {
	numeric := func(conn *dbi.DbConn) map[string]bool {
		cols, err := conn.GetMetadata().GetColumns(tableName)
		if err != nil {
			return nil
		}
		set := make(map[string]bool, len(cols))
		for i := range cols {
			col := &cols[i]
			if dbi.IsNumericCommonType(dbi.GetDbDataType(conn.Info.Type, col.DataType).CommonType) {
				set[strings.ToLower(col.ColumnName)] = true
			}
		}
		return set
	}

	srcNumeric := numeric(srcConn)
	tgtNumeric := numeric(tgtConn)
	if len(srcNumeric) == 0 || len(tgtNumeric) == 0 {
		return nil
	}
	result := make(map[string]bool, len(srcNumeric))
	for name := range srcNumeric {
		if tgtNumeric[name] {
			result[name] = true
		}
	}
	return result
}

// countTableRows 查询表行数
func countTableRows(ctx context.Context, conn *dbi.DbConn, tableName string) (int64, error) {
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.QueryContext(ctx, fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %s", quote(tableName)))
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, fmt.Errorf("empty count result of table [%s]", tableName)
	}
	cnt, ok := dbi.ValToInt64(rows[0]["cnt"])
	if !ok {
		return 0, fmt.Errorf("invalid count value of table [%s]: %#v", tableName, rows[0]["cnt"])
	}
	return cnt, nil
}

// singlePkColumn 获取表的单列主键列名（任意类型）；无主键/联合主键/查询失败返回""
func singlePkColumn(conn *dbi.DbConn, tableName string) string {
	columns, err := conn.GetMetadata().GetColumns(tableName)
	if err != nil {
		return ""
	}
	pk := ""
	count := 0
	for _, c := range columns {
		if c.IsPrimaryKey {
			count++
			pk = c.ColumnName
		}
	}
	if count != 1 {
		return ""
	}
	return pk
}

// sampleOrderedRows 按主键升序从offset处抽样limit行。
//
// 使用LIMIT语法（mysql/pg/sqlite均支持）；mssql等不支持LIMIT的方言会返回错误，
// 由调用方记录SampleErr，count比对照常。错误不再吞掉，以便区分“空表”与“抽样失败”
func sampleOrderedRows(ctx context.Context, conn *dbi.DbConn, tableName, pkColumn string, limit int, offset int64) ([]map[string]any, error) {
	if limit <= 0 {
		return nil, nil
	}
	quote := conn.GetDialect().Quoter().QuoteIdent
	sqlStr := fmt.Sprintf("SELECT * FROM %s ORDER BY %s ASC LIMIT %d OFFSET %d", quote(tableName), quote(pkColumn), limit, offset)
	_, rows, err := conn.QueryContext(ctx, sqlStr)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// compareSampledRows 抽样内容比对：以源侧行为基准，按主键规范化值对齐目标行，
// 逐公共列（列名大小写不敏感对齐）做规范化比对，数值类列按数值语义判等。
// 返回内容不一致/目标缺失行的主键规范化值列表（最多maxReport个）。
// 注意：目标侧多出的行不在此检出（由count比对捕获）
func compareSampledRows(srcRows, tgtRows []map[string]any, srcPk, tgtPk string, numericCols map[string]bool, maxReport int) []string {
	tgtIndex := make(map[string]map[string]any, len(tgtRows))
	for _, row := range tgtRows {
		tgtIndex[dbi.CanonicalRowKey(row, tgtPk)] = row
	}

	mismatch := make([]string, 0)
	seen := make(map[string]bool)
	for _, srcRow := range srcRows {
		pk := dbi.CanonicalRowKey(srcRow, srcPk)
		if srcRow[srcPk] == nil {
			// 主键为NULL的行无法与目标对齐，跳过；显式判NULL而非比对规范化串，
			// 避免主键值恰为字符串"<nil>"的行被误跳过
			continue
		}
		if seen[pk] {
			continue
		}
		seen[pk] = true

		tgtRow, ok := tgtIndex[pk]
		if !ok {
			mismatch = append(mismatch, pk) // 目标缺失该行
		} else if !compareRowValues(srcRow, tgtRow, numericCols) {
			mismatch = append(mismatch, pk)
		}
		if len(mismatch) >= maxReport {
			break
		}
	}
	return mismatch
}

// compareRowValues 按公共列（列名大小写不敏感）比对两行数据；
// numericCols内的列按数值语义判等，其余列严格规范化形态判等
func compareRowValues(srcRow, tgtRow map[string]any, numericCols map[string]bool) bool {
	common := alignColumns(srcRow, tgtRow)
	if len(common) == 0 {
		return false
	}
	lowerTgtRow := make(map[string]any, len(tgtRow))
	for k, v := range tgtRow {
		lowerTgtRow[strings.ToLower(k)] = v
	}
	for _, col := range common {
		lowerCol := strings.ToLower(col)
		tgtVal := lowerTgtRow[lowerCol]
		if numericCols[lowerCol] {
			if !dbi.CanonicalNumericEqual(srcRow[col], tgtVal) {
				return false
			}
			continue
		}
		if !dbi.CanonicalEqual(srcRow[col], tgtVal) {
			return false
		}
	}
	return true
}

// alignColumns 取两行的公共列名（以源侧列名为准，大小写不敏感）
func alignColumns(srcRow, tgtRow map[string]any) []string {
	tgtLower := make(map[string]struct{}, len(tgtRow))
	for k := range tgtRow {
		tgtLower[strings.ToLower(k)] = struct{}{}
	}
	common := make([]string, 0, len(srcRow))
	for k := range srcRow {
		if _, ok := tgtLower[strings.ToLower(k)]; ok {
			common = append(common, k)
		}
	}
	sort.Strings(common)
	return common
}
