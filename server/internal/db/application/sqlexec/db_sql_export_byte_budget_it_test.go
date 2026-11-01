//go:build it

package sqlexec

// 查询导出的「字节预算」真实数据库验证。
//
// 历史缺陷：这条路径只有行数预算（500 行），没有字节预算 —— 与 dbm/export 的 8MB 双预算
// 不等价：500 行大 blob/text 会把整批原始值同时留在内存里（可能直接 OOM）。
// 本用例用 3 行 × 6MB 的大值：行数预算不会触发切批，只有字节预算会，因此
// 「批次回调次数 ≥ 2」即证明字节预算真的生效。

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/dto"
)

const exportBigTable = "it_query_export_big"

func TestITQueryExportByteBudgetSplitsLargeValues(t *testing.T) {
	for _, node := range appDialectNodes[:2] {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()

			quote := conn.GetDialect().Quoter().QuoteIdent
			if _, err := conn.Exec("DROP TABLE IF EXISTS " + quote(exportBigTable)); err != nil {
				t.Fatalf("drop failed: %s", err.Error())
			}
			// MySQL 的 TEXT 上限 64KB，装不下 6MB；PG 的 TEXT 无上限
			bigType := "TEXT"
			if conn.Info.Type == "mysql" {
				bigType = "LONGTEXT"
			}
			if _, err := conn.Exec(fmt.Sprintf("CREATE TABLE %s (%s INT PRIMARY KEY, %s %s)",
				quote(exportBigTable), quote("id"), quote("v"), bigType)); err != nil {
				t.Fatalf("create failed: %s", err.Error())
			}
			// 用 REPEAT 在服务端生成大值：避免客户端 6MB 绑定参数受 max_allowed_packet 限制
			for i := 1; i <= 3; i++ {
				if _, err := conn.Exec(fmt.Sprintf("INSERT INTO %s (%s, %s) VALUES (%d, REPEAT('x', %d))",
					quote(exportBigTable), quote("id"), quote("v"), i, 6*1024*1024)); err != nil {
					t.Fatalf("insert failed at %d: %s", i, err.Error())
				}
			}

			var buf bytes.Buffer
			batches := 0
			var seq []int64
			err := exportQuery(context.Background(), exportApp(), conn, &dto.QueryExportReq{
				DbId: 1, Db: conn.Info.Database, Format: "csv",
				SQL: fmt.Sprintf("SELECT %s, %s FROM %s", quote("id"), quote("v"), quote(exportBigTable)),
			}, writerTo(&buf), func(written int64) { batches++; seq = append(seq, written) })
			require.NoError(t, err)

			require.GreaterOrEqual(t, batches, 2,
				"3 行 × 6MB 必须发生多次 flush；只有行数预算时只会在尾批 flush 一次")
			// 关键判据：第一次 flush 必须发生在「读完全部行之前」。
			// 有字节预算：2 行 × 6MB = 12MB ≥ 8MB → 第 2 行后就 flush（seq 首元素 = 2）；
			// 无字节预算：3 行 < 500 行预算 → 直到尾批才 flush（seq 首元素 = 3）。
			require.Less(t, seq[0], int64(3),
				"字节预算必须在读完所有行之前触发一次 flush（首元素应为 2，而不是尾批的 3）")
			require.Contains(t, buf.String(), strings.Repeat("x", 100), "大值必须完整写入产物")
		})
	}
}
