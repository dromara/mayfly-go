package sqlexec

// 查询结果集全量导出：重新执行同一条查询并把全部行流式写给下载响应。
//
// 为什么不在前端把「页面上看到的结果」导出：页面结果已被结果集行数上限截断，
// 拿它导出的是一份缺尾巴的 CSV，而文件看起来完全正常——用户不会知道少了行。
// 这里明确不走上限：一次读取整条语句的全部结果。
//
// 三条不能省的约束：
//  1. 只允许只读语句：导出入口不能变成一条绕过审批执行写语句的旁路；
//  2. 过与执行侧同一套触发策略：能拦「大表全表 SELECT」的策略，对导出同样有效；
//  3. 脱敏同样生效：导出走的是同一份数据，缺了脱敏就是开了一条把敏感数据整表带走的通道，
//     且脱敏计划构建失败时必须中止（fail-close），宁可不出文件。
//
// 内存由批次行数封顶（一次只在内存里保留一批），与结果集总行数无关。

import (
	"context"
	"io"
	"strings"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/export"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	masksvc "mayfly-go/internal/db/domain/mask"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
)

// exportBatchRows 单次写入格式层的行数：决定了导出过程的内存峰值，与结果集总大小无关
const exportBatchRows = 500

// ExportQuery 全量导出一次查询的结果集，产物写入 writer。
//
// 只支持非脚本类格式（CSV/JSON）：SQL 脚本形态的产物属于「表导出」的语义（带 DDL、事务钩子、
// 目标方言的 INSERT），把一次任意查询的列拼成 INSERT 需要知道目标表，这里不做该猜测
// openWriter 延迟创建产物写入器：校验（格式/语句类型/策略）全部通过后才第一次调用它。
//
// 下载入口在写第一字节之前都还有机会返回一个正常的 JSON 错误；一旦先设了下载响应头，
// 之后的失败就只能混进已开始的 gzip 流里，前端拿到的是一份坏文件而不是错误提示
type OpenWriter func() (io.Writer, error)

// ExportProgress 已写出行数的回调（每批一次，不是每行一次）。
//
// 异步导出任务靠它把进度写回库；同步下载入口不传，也就不会为一份已经
// 在流式下载的文件多写 N 次数据库更新
type ExportProgress func(rows int64)

func (d *DbSQLExecAppImpl) ExportQuery(ctx context.Context, req *dto.QueryExportReq, openWriter OpenWriter, onProgress ...ExportProgress) error {
	conn, err := d.dbApp.GetDbConn(ctx, req.DbId, req.Db)
	if err != nil {
		return err
	}
	return exportQuery(ctx, d, conn, req, openWriter, onProgress...)
}

// firstProgress 取可选的进度回调，没传则为 nil（调用方统一按 nil 判断）
func firstProgress(onProgress []ExportProgress) ExportProgress {
	if len(onProgress) > 0 {
		return onProgress[0]
	}
	return nil
}

// exportQuery 导出本体：连接与脱敏引擎都由参数给出（取连接是 I/O，判定与写出要能被真库用例直接驱动），
// 与本包 checkSqlsWithoutTicket 的拆分手法一致
func exportQuery(ctx context.Context, app *DbSQLExecAppImpl, conn *dbi.DbConn, req *dto.QueryExportReq, openWriter OpenWriter, onProgress ...ExportProgress) error {
	progress := firstProgress(onProgress)
	sql := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(req.SQL), ";"))
	if sql == "" {
		return errorx.NewBizI(ctx, imsg.ErrQueryExportEmptySql)
	}

	// 格式能力单点判定：未知格式与脚本类格式都是「不能用于结果集导出」，给同一句提示与同一份可用清单，
	// 不会出现「这里能用、那里报不支援」。脚本类产物属于「表导出」语义（带 DDL、事务钩子、目标方言的
	// INSERT），把一次任意查询的列拼成 INSERT 需要知道目标表，这里不做该猜测
	if !isReadOnlyExportFormat(req.Format) {
		return errorx.NewBizI(ctx, imsg.ErrQueryExportFormatUnsupported, "format", req.Format, "supported", strings.Join(readOnlyExportFormats(), ", "))
	}
	consumer := export.Get(req.Format)

	parser, splitter := conn.GetDialect().GetSQLParser(), conn.GetDialect().GetSQLSplitter()
	stmt, parseErr := parser.Parse(sql)
	if parseErr != nil {
		logx.WarnfContext(ctx, "query export: failed to parse the statement, classified by keyword: %s", parseErr.Error())
	}
	stmtType := sqlparser.Classify(parser, splitter, sql, stmt, parseErr)
	switch stmtType {
	case sqlstmt.StmtTypeSelect, sqlstmt.StmtTypeOther:
		// 只读语句允许导出；其余（写语句、DDL）在下面被拒
	default:
		return errorx.NewBizI(ctx, imsg.ErrQueryExportReadOnlyOnly, "type", string(stmtType))
	}

	// 与交互执行同一道策略闸门：导出用的也是这条 SELECT，策略既然管住了执行就必须管住导出
	if _, err := app.checkSqlsWithoutTicket(ctx, conn, conn.Info.CodePath, conn.GetDialect(), []string{sql}); err != nil {
		return err
	}

	// 到这里才认为这次导出真的会写产物：校验失败一律在上面返回
	writer, err := openWriter()
	if err != nil {
		return err
	}

	settings := export.DefaultSettings(consumer.Format())
	settings.IncludeHeader = true

	// 脱敏器在拿到结果列后构建；未启用脱敏或引擎不可用时与查询侧保持同一判定
	maskEnabled := app.maskEngine != nil
	var rowMasker *masksvc.RowMasker

	tableName := conn.Info.Database
	batch := make([][]any, 0, exportBatchRows)
	columns := make([]dbi.Column, 0, 16)
	keyNames := make([]string, 0, 16)
	// begun 标记格式头部是否已写出：Begin 需要列信息，而列信息要等第一行
	begun := false
	// written 已交付格式层的行数（含尾批），进度与任务收尾的行数均以此为准
	var written int64

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := consumer.ConsumeBatch(writer, tableName, columns, batch, nil, nil, settings); err != nil {
			return err
		}
		written += int64(len(batch))
		batch = batch[:0]
		if progress != nil {
			progress(written)
		}
		return nil
	}

	_, walkErr := conn.WalkQueryRows(ctx, sql, func(row map[string]any, queryColumns []*dbi.QueryColumn) error {
		if len(columns) == 0 {
			for _, queryColumn := range queryColumns {
				columns = append(columns, dbi.Column{ColumnName: queryColumn.Name, DataType: queryColumn.Type})
				keyNames = append(keyNames, queryColumnKey(queryColumn))
			}
			if maskEnabled {
				masser, maskErr := app.maskEngine.BuildStmtRowMasker(ctx, conn, stmt, queryColumns)
				if maskErr != nil {
					// fail-close：脱敏计划不可用时不出文件，避免导出成为绕过脱敏的通道
					return maskErr
				}
				rowMasker = masser
			}
			if err := consumer.Begin(writer, tableName, columns, settings); err != nil {
				return err
			}
			begun = true
		}
		if rowMasker != nil {
			rowMasker.MaskRow(row)
		}

		values := make([]any, 0, len(keyNames))
		for _, key := range keyNames {
			values = append(values, row[key])
		}
		batch = append(batch, values)
		if len(batch) >= exportBatchRows {
			return flush()
		}
		return nil
	})
	if walkErr != nil {
		return walkErr
	}

	if begun {
		if err := flush(); err != nil {
			return err
		}
		// 尾批不足一个批次时 flush 不会回报进度，这里再补一次，否则任务进度永远少于实际行数
		if progress != nil {
			progress(written)
		}
	} else {
		// 零行结果：列信息只能从第一行拿到，所以表头拿不出（已知限制）。但仍要写完格式骨架，
		// 让产物是「一个合法的空结果」（JSON 为 []）而不是 0 字节文件——
		// 空文件与「导出坏了」下载后长得一样，用户只能猜
		if err := consumer.Begin(writer, tableName, columns, settings); err != nil {
			return err
		}
	}
	if err := consumer.End(writer, tableName, settings); err != nil {
		return err
	}
	return consumer.Finish(writer, settings)
}

// isReadOnlyExportFormat 该格式能否用于「一次查询的结果集」导出。
//
// 判据与报错提示同源（都走 readOnlyExportFormats），不会出现「这里说能用、那里报不支援」
func isReadOnlyExportFormat(format string) bool {
	for _, available := range readOnlyExportFormats() {
		if available == format {
			return true
		}
	}
	return false
}

// readOnlyExportFormats 查询结果导出可用的格式清单（非脚本类），报错时也用它提示用户
func readOnlyExportFormats() []string {
	formats := make([]string, 0, 2)
	for _, format := range export.Formats() {
		consumer := export.Get(format)
		if consumer != nil && !consumer.SupportsScript() {
			formats = append(formats, format)
		}
	}
	return formats
}

// queryColumnKey 结果行 map 的键：库侧返回重复列名时驱动会给出 id/id1 这样的去重键，
// 取值必须用 Key 而不是 Name，否则重复列会整列变 NULL
func queryColumnKey(column *dbi.QueryColumn) string {
	if column.Key != "" {
		return column.Key
	}
	return column.Name
}
