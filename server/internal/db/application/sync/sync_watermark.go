package sync

import (
	"cmp"
	"context"
	"fmt"
	"strconv"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/domain/entity"

	"github.com/spf13/cast"
)

// 同步水位线推进与主键采集辅助。

// collectKeyStrings 执行查询 SQL 并将每行的多列值拼接为单个字符串切片。
// 返回底层查询错误：调用方据此区分「查询失败」与「结果为空」，避免跨方言等场景下
// SQL 执行失败被静默当成空结果、导致上层 checksum 校验假通过。
func collectKeyStrings(conn *dbi.DbConn, ctx context.Context, sql string) ([]string, error) {
	var keys []string
	if _, err := conn.WalkQueryRows(ctx, sql, func(row map[string]any, columns []*dbi.QueryColumn) error {
		keys = append(keys, encodeValidationRow(row, columns))
		return nil
	}); err != nil {
		return nil, err
	}
	return keys, nil
}

// encodeValidationRow 以长度前缀区分列边界，NULL 单独编码，避免分隔符和文本碰撞。
func encodeValidationRow(row map[string]any, columns []*dbi.QueryColumn) string {
	var encoded strings.Builder
	for _, column := range columns {
		value := row[column.Key]
		if value == nil {
			encoded.WriteString("N;")
			continue
		}
		text := fmt.Sprint(value)
		encoded.WriteString(strconv.Itoa(len(text)))
		encoded.WriteByte(':')
		encoded.WriteString(text)
	}
	return encoded.String()
}

// paginateTopN 用连接所属方言的分页改写能力，将不含分页子句的 baseSQL 改写为取前 n 行。
// 优先方言专用 rewriter（Oracle/DM/MSSQL 语法各异），无则回退默认 LIMIT/OFFSET（MySQL/PG/SQLite/ClickHouse 通用）。
// 源/目标可能异构，跨方言执行路径禁止直接拼 `LIMIT n`，否则不支持该语法的方言会整条 SQL 报错。
func paginateTopN(conn *dbi.DbConn, baseSQL string, n int64) (string, error) {
	if pw := sqlparser.GetPaginationRewriter(conn.GetDialect().GetSQLParser()); pw != nil {
		return pw.RewritePagination(baseSQL, 0, n)
	}
	return sqlparser.DefaultRewritePagination(baseSQL, 0, n), nil
}

// advanceSyncWatermark 推进同步水位
func advanceSyncWatermark(srcRes []map[string]any, task *entity.DataSyncTask, updFieldName string) error {
	if len(srcRes) == 0 {
		return nil
	}
	waterField := cmp.Or(task.UpdFieldSrc, updFieldName)
	if waterField == "" {
		return nil
	}
	val, ok := lookupRowValue(srcRes[len(srcRes)-1], waterField)
	if !ok {
		return fmt.Errorf("update field [%s] not found in the last synced row, keep watermark [%s] unchanged", waterField, task.UpdFieldVal)
	}
	if val == nil {
		return fmt.Errorf("update field [%s] of the last synced row is NULL, keep watermark [%s] unchanged", waterField, task.UpdFieldVal)
	}
	task.UpdFieldVal = cast.ToString(val)
	return nil
}

// lookupRowValue 从查询行中取列值
func lookupRowValue(row map[string]any, column string) (any, bool) {
	if column == "" {
		return nil, false
	}
	if val, ok := row[column]; ok {
		return val, true
	}
	for name, val := range row {
		if strings.EqualFold(name, column) {
			return val, true
		}
	}
	return nil, false
}

func (app *DataSyncAppImpl) persistUpdFieldVal(ctx context.Context, task *entity.DataSyncTask) error {
	if task.UpdField == "" {
		return nil
	}
	ut := &entity.DataSyncTask{Id: task.Id, UpdFieldVal: task.UpdFieldVal}
	if err := app.UpdateById(ctx, ut); err != nil {
		return fmt.Errorf("failed to persist the data sync watermark of task [%d]: %s", task.Id, err.Error())
	}
	return nil
}
