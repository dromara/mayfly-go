//go:build it

package application

// 列值采样（编辑器值补全）的真实库用例。
//
// 钉三件只有真库能证伪的事：
//  1. 标识符必须被方言自己的引用符包裹——列名撞上保留字（order）时不引用就是语法错误，
//     而这条路径的表/列名来自用户输入，引用缺失同时也是一处注入面；
//  2. 限行子句必须来自方言的分页改写能力，硬拼 `LIMIT n` 在 Oracle/DM/MSSQL 上整条报错；
//  3. NULL 不参与采样（补全给的是「填什么值」，NULL 由用户自己写），且上限真的生效。
//
// 运行：cd server && go test -tags it -count=1 -run TestITColumnValues ./internal/db/application/

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/sqlexec"
	"mayfly-go/internal/db/dbm/dbi"
)

const colValuesTable = "it_col_values"

// colValuesDialect 各方言的建表语句：status 有重复值与 NULL，order 是保留字列名
type colValuesDialect struct {
	name     string
	conn     func(t *testing.T) *dbi.DbConn
	create   string
	statuses []string
}

func colValuesDialects() []colValuesDialect {
	return []colValuesDialect{
		{
			name:     "mysql",
			conn:     appMysqlConn,
			create:   "CREATE TABLE %s (id INT PRIMARY KEY, status VARCHAR(20), `order` INT)",
			statuses: []string{"active", "active", "closed"},
		},
		{
			name:     "pg",
			conn:     appPgConn,
			create:   `CREATE TABLE %s (id INT PRIMARY KEY, status VARCHAR(20), "order" INT)`,
			statuses: []string{"active", "active", "closed"},
		},
	}
}

func appExec(t *testing.T, conn *dbi.DbConn, sql string) {
	t.Helper()
	if _, err := conn.Exec(sql); err != nil {
		t.Fatalf("exec failed [%s]: %s", sql, err.Error())
	}
}

// setupColValues 建表并写入含重复值、NULL 与保留字列名的数据
func setupColValues(t *testing.T, conn *dbi.DbConn, node colValuesDialect) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	appExec(t, conn, "DROP TABLE IF EXISTS "+quote(colValuesTable))
	appExec(t, conn, fmt.Sprintf(node.create, quote(colValuesTable)))
	for i, status := range node.statuses {
		appExec(t, conn, fmt.Sprintf("INSERT INTO %s (id, status, %s) VALUES (%d, '%s', %d)",
			quote(colValuesTable), quote("order"), i+1, status, i))
	}
	// NULL 行：id=99 的 status 为空
	appExec(t, conn, fmt.Sprintf("INSERT INTO %s (id, status, %s) VALUES (99, NULL, 99)",
		quote(colValuesTable), quote("order")))
}

// sampleValues 走与生产完全同一条路径：组装采样语句 → 执行 → 取列值
func sampleValues(t *testing.T, conn *dbi.DbConn, table, column string, limit int) []string {
	t.Helper()
	sampleSQL, err := buildColumnValuesSQL(context.Background(), conn, table, column, limit)
	require.NoError(t, err)

	_, rows, err := conn.QueryContext(context.Background(), sampleSQL)
	require.NoError(t, err, "采样语句必须能被该方言执行")

	values := make([]string, 0, len(rows))
	for _, row := range rows {
		raw, ok := sqlexec.LookupRowValue(row, column)
		require.True(t, ok, "结果行里应能按列名取到值，实际列：%v", row)
		if raw == nil {
			continue
		}
		values = append(values, fmt.Sprint(raw))
	}
	return values
}

func TestITColumnValuesSampling(t *testing.T) {
	for _, node := range colValuesDialects() {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			setupColValues(t, conn, node)
			quote := conn.GetDialect().Quoter().QuoteIdent
			defer func() {
				_, _ = conn.Exec("DROP TABLE IF EXISTS " + quote(colValuesTable))
			}()

			// 保留字列名能被引用：不引用就是语法错误，也是注入面的证据
			sampleSQL, err := buildColumnValuesSQL(context.Background(), conn, colValuesTable, "order", 10)
			require.NoError(t, err)
			assert.Contains(t, sampleSQL, quote("order"), "列名必须经方言引用符包裹，实际语句：%s", sampleSQL)
			assert.Contains(t, sampleSQL, quote(colValuesTable), "表名必须经方言引用符包裹")
			assert.Contains(t, sampleSQL, "IS NOT NULL", "NULL 不参与采样")

			statuses := sampleValues(t, conn, colValuesTable, "status", 200)
			assert.ElementsMatch(t, []string{"active", "closed"}, statuses, "必须去重且不含 NULL")

			// 上限生效：3 行数据里有 2 个去重值，limit=1 只能拿到 1 个
			assert.Len(t, sampleValues(t, conn, colValuesTable, "status", 1), 1, "限行上限必须生效")
		})
	}
}

// TestITColumnValuesRejectsUnknownColumn 列不存在时给业务错误而不是让方言把用户输入回显成 SQL 报错
func TestITColumnValuesRejectsUnknownColumn(t *testing.T) {
	for _, node := range colValuesDialects() {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			setupColValues(t, conn, node)
			quote := conn.GetDialect().Quoter().QuoteIdent
			defer func() {
				_, _ = conn.Exec("DROP TABLE IF EXISTS " + quote(colValuesTable))
			}()

			ctx := context.Background()
			_, err := findColumnOf(ctx, conn, colValuesTable, "no_such_col")
			require.Error(t, err, "不存在的列必须报错")
			assert.NotContains(t, err.Error(), "syntax", "应给业务错误而不是方言语法错误: %v", err)

			// 注入串走的是同一条校验：它不是该表的列，根本到不了拼 SQL 那一步
			_, err = findColumnOf(ctx, conn, colValuesTable, "status; DROP TABLE it_col_values--")
			require.Error(t, err, "注入串必须被列名校验挡下")
			_, rows, queryErr := conn.QueryContext(ctx, "SELECT id FROM "+quote(colValuesTable))
			require.NoError(t, queryErr, "表必须还在（注入未生效）")
			assert.NotEmpty(t, rows)
		})
	}
}
