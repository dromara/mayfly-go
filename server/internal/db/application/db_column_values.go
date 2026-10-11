package application

// 列值采样：SQL 编辑器在「列 = 」这类上下文里给出该列出现过的值作为补全候选。
//
// 两条约束决定了这里的写法：
//   - 表名/列名一律先过库侧元数据校验再用方言 Quoter 引用：把请求里的字符串直接拼进 SQL
//     会让这个补全接口变成注入入口，而列不存在时报出的方言原始错误对用户也毫无意义；
//   - 限制行数必须走方言的分页改写能力。跨方言硬拼 `LIMIT n` 在 Oracle/DM/MSSQL 上整条报错，
//     而这里拿不到 rewriter 时选择「不采样」而不是回退默认语法：补全失败应当是静默少一批候选，
//     不该在编辑器里冒出一条红色错误（与同步链路「拿不到就降级并记录」的取舍不同，那边有可观测出口）。

import (
	"context"
	"fmt"
	"strings"

	"mayfly-go/internal/db/application/sqlexec"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/errorx"

	"github.com/spf13/cast"
)

// columnValuesMax 单次采样的值数上限。
// DISTINCT 在大表上是全表扫描，上限只约束返回体量、不约束成本，因此值补全必须由用户显式触发并有缓存兜着
const columnValuesMax = 200

// ColumnValues 返回某列出现过的去重值（不含 NULL），最多 columnValuesMax 个。
func (d *dbAppImpl) ColumnValues(ctx context.Context, dbId uint64, dbName, table, column string, limit int) ([]string, error) {
	conn, err := d.GetDbConn(ctx, dbId, dbName)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > columnValuesMax {
		limit = columnValuesMax
	}

	// 标识符以元数据为准：大小写按库侧真实列名取回，请求里写的名字只用于匹配
	realColumn, err := findColumnOf(ctx, conn, table, column)
	if err != nil {
		return nil, err
	}

	sampleSQL, err := buildColumnValuesSQL(ctx, conn, table, realColumn, limit)
	if err != nil {
		return nil, err
	}

	_, rows, err := conn.QueryContext(ctx, sampleSQL)
	if err != nil {
		return nil, err
	}

	values := make([]string, 0, len(rows))
	for _, row := range rows {
		// 取列复用回滚链路那份 lookupRowValue：它同样要兼容库侧把列名回成 alias.col 的形态
		raw, ok := sqlexec.LookupRowValue(row, realColumn)
		if !ok || raw == nil {
			continue
		}
		values = append(values, cast.ToString(raw))
	}
	return values, nil
}

// buildColumnValuesSQL 组装该列的采样语句（真实列名已由元数据校验过）。
//
// 单独成函数是为了让「标识符如何引用、限行子句从哪来」可被用例直接钉住：
// 这两点分别是注入面与跨方言报错面，只看最终返回的值列表看不出来。
//
// ctx 只为「方言不支持」这条错误渲染语言：它拿不到请求上下文时，中文界面的用户会收到
// 一截英文提示，而这条错误恰恰是界面上会弹出来的
func buildColumnValuesSQL(ctx context.Context, conn *dbi.DbConn, table, column string, limit int) (string, error) {
	rewriter := sqlparser.GetPaginationRewriter(conn.GetDialect().GetSQLParser())
	if rewriter == nil {
		return "", errorx.NewBizI(ctx, imsg.ErrColumnValuesUnsupported)
	}
	quoter := conn.GetDialect().Quoter()
	quotedColumn, quotedTable := quoter.QuoteIdent(column), quoter.QuoteIdent(table)
	baseSQL := fmt.Sprintf("SELECT DISTINCT %s FROM %s WHERE %s IS NOT NULL", quotedColumn, quotedTable, quotedColumn)
	return rewriter.RewritePagination(baseSQL, 0, int64(limit))
}

// findColumnOf 校验列存在于该表的库侧元数据中，并返回库侧真实列名。
func findColumnOf(ctx context.Context, conn *dbi.DbConn, table, column string) (string, error) {
	columns, err := conn.Metadata().GetColumns(table)
	if err != nil {
		return "", err
	}
	for _, col := range columns {
		if strings.EqualFold(col.ColumnName, column) {
			return col.ColumnName, nil
		}
	}
	return "", errorx.NewBizI(ctx, imsg.ErrColumnValuesNoColumn, "table", table, "column", column)
}
