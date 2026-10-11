package itest

// 导出链路行级钩子（查询结果脱敏的导出侧）的真实数据库集成测试。
//
// 钉死三件事，缺一不可：
//  1. WithRowMask 确实作用在真实 dump 的每一行上——否则它只是单测里被调用到的死代码，
//     线上导出依旧明文透出；
//  2. 钩子返回错误必须中止整次导出（fail-close）：半途透出未脱敏数据比不出文件更糟；
//  3. 声明了要脱敏却没装配钩子时直接拒绝，不能静默按不脱敏导出。
//
// 运行：cd server && go test -tags it -count=1 -run TestITExportRowMask ./internal/db/application/transfer/itest/

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/application/transfer"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/export"
)

const itMaskTable = "it_export_mask"

// itMaskPlain 敏感列原文：导出产物里只要出现它，就说明钩子没生效
const itMaskPlain = "13800001234"

// itMaskCreate 建表并写入 3 行（行数>1 才能证明钩子是逐行调用而非每表一次）
func itMaskCreate(t *testing.T, conn *dbi.DbConn) {
	t.Helper()
	q := conn.GetDialect().Quoter().QuoteIdent
	mustItExec(t, conn, "DROP TABLE IF EXISTS "+q(itMaskTable))
	mustItExec(t, conn, fmt.Sprintf("CREATE TABLE %s (id INT PRIMARY KEY, phone VARCHAR(32))", q(itMaskTable)))
	for i := 1; i <= 3; i++ {
		mustItExec(t, conn, fmt.Sprintf("INSERT INTO %s (id, phone) VALUES (%d, '%s')", q(itMaskTable), i, itMaskPlain))
	}
}

// itMaskDump 走真实 dump 链路导出 CSV（数据段逐行经过钩子）
func itMaskDump(t *testing.T, conn *dbi.DbConn, mask export.RowMaskFn, wantErr bool) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	err := transfer.DumpDbScript(context.Background(), conn, &dto.DumpDb{
		DbName:   conn.Info.Database,
		Tables:   []string{itMaskTable},
		DumpData: true,
		Mask:     mask != nil,
		RowMask:  mask,
		Writer:   buf,
	})
	if wantErr {
		require.Error(t, err, "导出必须失败")
	} else {
		require.NoError(t, err)
	}
	return buf
}

func TestITExportRowMaskApplied(t *testing.T) {
	for _, node := range []itDialectNode{itMysql, itPg, itSQLite} {
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			itMaskCreate(t, conn)

			var rowCount int
			var seenColumns []string
			buf := itMaskDump(t, conn, func(_ context.Context, tableName string, columns []string, row map[string]any) error {
				rowCount++
				// 表名与列清单必须由导出器如实给出：钩子要靠它们解析「这列属于哪张表」
				assert.Equal(t, itMaskTable, tableName)
				seenColumns = columns
				for key, val := range row {
					if fmt.Sprint(val) == itMaskPlain {
						row[key] = "MASKED"
					}
				}
				return nil
			}, false)

			assert.Equal(t, 3, rowCount, "钩子必须逐行调用（3行数据）")
			assertContainsPhoneColumn(t, seenColumns)
			out := buf.String()
			assert.NotContains(t, out, itMaskPlain, "导出产物不得含敏感列原文")
			assert.Contains(t, out, "MASKED", "导出产物必须是钩子改写后的值")
		})
	}
}

// TestITExportRowMaskFailClose 钩子报错时必须中止整次导出：
// 若实现把错误吞掉继续写，产物会同时包含已脱敏与未脱敏的行，比不出文件更危险
func TestITExportRowMaskFailClose(t *testing.T) {
	for _, node := range []itDialectNode{itMysql, itPg, itSQLite} {
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			itMaskCreate(t, conn)

			var rowCount int
			buf := itMaskDump(t, conn, func(_ context.Context, _ string, _ []string, row map[string]any) error {
				rowCount++
				if rowCount == 2 {
					return fmt.Errorf("mock mask plan failure")
				}
				for key, val := range row {
					if fmt.Sprint(val) == itMaskPlain {
						row[key] = "MASKED"
					}
				}
				return nil
			}, true)

			assert.NotContains(t, buf.String(), itMaskPlain, "失败后不得留下含明文的数据段")
		})
	}
}

// TestITExportMaskRequestedWithoutHook 只声明脱敏意图却没装配钩子：必须拒绝而不是静默导出明文
func TestITExportMaskRequestedWithoutHook(t *testing.T) {
	conn := itMysql.conn(t)
	defer conn.Close()
	itMaskCreate(t, conn)

	buf := &bytes.Buffer{}
	err := transfer.DumpDbScript(context.Background(), conn, &dto.DumpDb{
		DbName:   conn.Info.Database,
		Tables:   []string{itMaskTable},
		DumpData: true,
		Mask:     true,
		Writer:   buf,
	})
	require.Error(t, err)
	assert.Empty(t, buf.String(), "拒绝时不应产出任何内容")
}

// assertContainsPhoneColumn 列清单里的敏感列名（大小写按各方言返回形态不同，按不区分大小写判定）
func assertContainsPhoneColumn(t *testing.T, columns []string) {
	t.Helper()
	for _, c := range columns {
		if strings.EqualFold(c, "phone") {
			return
		}
	}
	t.Fatalf("导出器传给钩子的列清单缺少 phone 列，实际为 %v", columns)
}
