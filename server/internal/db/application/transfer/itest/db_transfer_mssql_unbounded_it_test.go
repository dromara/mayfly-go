package itest

// mssql「无限长列」的同方言备份恢复真实链路集成测试（本机 SQL Server 容器，未启动自动跳过）。
//
// 为何必须补这组用例：SQL Server 的 sys.columns 对 varchar(max)/nvarchar(max)/varbinary(max)
// 回报 max_length = -1，而 nchar/nvarchar 还要把字节数换算成字符数；Go 整型除法向零截断使
// -1/2 == 0，一旦换算先于 -1 判定，(max) 形态就永久丢失，退化为“长度 0 的 nvarchar”：
//   - 同方言备份恢复：结构 DDL 被拼成裸 nvarchar，SQL Server 解释为 nvarchar(1)，
//     导入超长值直接报截断错误（数据无法恢复）；
//   - 异构迁移：目标方言取不到列长，mysql 兜底为 varchar(255)（Data too long/静默截断）。
//
// 断言两翼：dump 产物必须逐列保留 (max)/ntext 形态（结构性断言，定位归因快）；
// 恢复后回读必须逐值一致（端到端断言，覆盖中文/emoji/超长/二进制/NULL）。
//
// 运行：cd server && go test -tags it -count=1 -run TestITMssqlUnbounded ./internal/db/application/transfer/

import (
	"mayfly-go/internal/db/application/transfer"
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
)

const itMsUnboundedTable = "it_ms_unbounded"

// itMsUnboundedRows 行数据：7500字符中文+emoji、8k ASCII（含引号反斜杠）、12k二进制、NULL混合
// 注：c_vmax（varchar(max)）只存ASCII——本机库排序规则为Latin1代码页，非ASCII在**写入阶段**即会变'?'，与dump无关；
// nvarchar(max)/ntext属国家Unicode类型，可存中文与emoji
func itMsUnboundedRows() []map[string]any {
	longText := strings.Repeat("中文😀", 2500)                 // 7500字符/25000字节（nvarchar的n与(max)均以字符计）
	longAscii := strings.Repeat("a'b\\;", 2000)             // 8000字符，含引号与反斜杠
	longBlob := bytes.Repeat([]byte{0x00, 0xFF, 'x'}, 4000) // 12000字节
	normal := "短文本中文"                                       // 有界列（NVARCHAR(50)），验证不被误判为(max)
	return []map[string]any{
		{
			"id": int64(1), "c_nmax": longText, "c_vmax": longAscii, "c_bmax": longBlob,
			"c_ntext": longText, "c_bounded": normal,
		},
		{
			// c_bmax必须用**typed nil**（[]byte(nil)）：无类型nil会被驱动当作nvarchar参数下发，
			// SQL Server报“Implicit conversion from nvarchar to varbinary(max) is not allowed”
			"id": int64(2), "c_nmax": "n2", "c_vmax": nil, "c_bmax": []byte(nil),
			"c_ntext": nil, "c_bounded": nil,
		},
	}
}

// itMsPrepareUnbounded 建含无限长列的表并参数化写入（不经字面量拼接，确保源库存的是原始值）
func itMsPrepareUnbounded(t *testing.T, conn *dbi.DbConn) []map[string]any {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, err := conn.Exec("DROP TABLE IF EXISTS " + quote(itMsUnboundedTable))
	require.NoError(t, err)
	_, err = conn.Exec(fmt.Sprintf("CREATE TABLE %s ("+
		"id INT PRIMARY KEY, c_nmax NVARCHAR(MAX), c_vmax VARCHAR(MAX), c_bmax VARBINARY(MAX), "+
		"c_ntext NTEXT, c_bounded NVARCHAR(50))", quote(itMsUnboundedTable)))
	require.NoError(t, err, "建表失败")

	rows := itMsUnboundedRows()
	for _, row := range rows {
		_, err = conn.Exec(fmt.Sprintf("INSERT INTO %s (id, c_nmax, c_vmax, c_bmax, c_ntext, c_bounded) VALUES (?, ?, ?, ?, ?, ?)",
			quote(itMsUnboundedTable)),
			row["id"], row["c_nmax"], row["c_vmax"], row["c_bmax"], row["c_ntext"], row["c_bounded"])
		require.NoError(t, err, "写入id=%d失败（超长值在源库即被截断？）", row["id"])
	}
	return rows
}

// itMsReadUnbounded 回读全表（按id升序），列名统一小写
func itMsReadUnbounded(t *testing.T, conn *dbi.DbConn) []map[string]any {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, res, err := conn.Query(fmt.Sprintf("SELECT * FROM %s ORDER BY id", quote(itMsUnboundedTable)))
	require.NoError(t, err, "回读失败")
	out := make([]map[string]any, 0, len(res))
	for _, r := range res {
		m := make(map[string]any, len(r))
		for k, v := range r {
			m[strings.ToLower(k)] = v
		}
		out = append(out, m)
	}
	return out
}

// TestITMssqlUnboundedDumpProductDdl 同方言dump产物必须逐列保留(max)/ntext形态
func TestITMssqlUnboundedDumpProductDdl(t *testing.T) {
	conn := itMssqlNode(t)
	defer conn.Close()

	itMsPrepareUnbounded(t, conn)
	var buf bytes.Buffer
	require.NoError(t, transfer.DumpDbScript(context.Background(), conn, &dto.DumpDb{
		DbName: conn.Info.Database, Tables: []string{itMsUnboundedTable},
		DumpDDL: true, DumpData: true, TargetDbType: "mssql", Writer: &buf,
	}), "导出失败")
	script := buf.String()

	// (max)与ntext形态必须原样出现在DDL中：裸nvarchar/varchar会被SQL Server解释为长度1
	for _, want := range []string{"[c_nmax] nvarchar(max)", "[c_vmax] varchar(max)", "[c_bmax] varbinary(max)", "[c_ntext] ntext"} {
		assert.Contains(t, script, strings.ToLower(want), "dump产物未保留列形态，备份无法恢复:\n%s", itTruncate(script, 3000))
	}
	// 有界列不得被一并放大为(max)（否则目标库丢失列长约束）
	assert.NotContains(t, script, "[c_bounded] nvarchar(max)", "有界列被误判为无限长:\n%s", itTruncate(script, 3000))
}

// TestITMssqlUnboundedSameDialectRoundtrip 同方言备份→删除→恢复：超长中文/二进制必须逐值一致
func TestITMssqlUnboundedSameDialectRoundtrip(t *testing.T) {
	conn := itMssqlNode(t)
	defer conn.Close()

	wantRows := itMsPrepareUnbounded(t, conn)
	srcRows := itMsReadUnbounded(t, conn)
	require.Len(t, srcRows, len(wantRows))

	var buf bytes.Buffer
	require.NoError(t, transfer.DumpDbScript(context.Background(), conn, &dto.DumpDb{
		DbName: conn.Info.Database, Tables: []string{itMsUnboundedTable},
		DumpDDL: true, DumpData: true, TargetDbType: "mssql", Writer: &buf,
	}), "导出失败")
	script := buf.String()

	itMsDropTable(t, conn, itMsUnboundedTable)

	app := &transfer.DbTransferAppImpl{}
	require.NoError(t, app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(script)),
		"无限长列同方言恢复失败, dump脚本:\n%s", itTruncate(script, 3000))

	gotRows := itMsReadUnbounded(t, conn)
	require.Len(t, gotRows, len(srcRows), "恢复后行数不一致")
	for i := range srcRows {
		for _, col := range []string{"id", "c_nmax", "c_vmax", "c_bmax", "c_ntext", "c_bounded"} {
			assert.Equal(t, itTextAt(srcRows[i], col), itTextAt(gotRows[i], col),
				"恢复后第%d行列[%s]不一致（长度%q vs %q）", i+1, col,
				itTextAt(srcRows[i], col)[:min(20, len(itTextAt(srcRows[i], col)))],
				itTextAt(gotRows[i], col)[:min(20, len(itTextAt(gotRows[i], col)))])
		}
	}
	// 显式钉死超长文本未被截断为1字符（截断时上方比对已红，此处给出可读归因）：
	// 字符数而非字节数，因SQL Server的nvarchar长度语义以字符计
	assert.Equal(t, 7500, utf8.RuneCountInString(itTextAt(gotRows[0], "c_nmax")), "nvarchar(max)内容未原样恢复")
}
