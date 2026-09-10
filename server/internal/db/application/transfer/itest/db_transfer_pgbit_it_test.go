package itest

// pg bit类型族"备份恢复"roundtrip集成测试（bit(8)/bit(1)/bit varying(16)）：
//
//	pg的bit列经lib/pq读回为文本位串（如"11111111"），dbm曾未注册bit映射而落Default(string)通道，
//	依赖unknown literal隐式转换碰巧可用且从未测试；现已注册pg专用DTBitPg
//	（SQLValue输出'位串'字面量，不能输出裸数字——pg不允许integer到bit的隐式assignment）。
//	本测试走真实链路：GenInsert序列化写入 → DumpDbScript导出到文件 → DROP → ImportDumpStream恢复
//	→ 逐值比对（含边界：全1/全0/单个bit/变长位串/NULL）
//
// 运行：cd server && go test -tags it -count=1 -run TestITPgBitRoundtrip ./internal/db/application/transfer/

import (
	"mayfly-go/internal/db/application/transfer"
	"bufio"
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
)

/* bit列读回位串文本（string/[]byte/nil形态兑底） */
func bitText(t *testing.T, v any) string {
	t.Helper()
	switch x := v.(type) {
	case nil:
		return "<NIL>"
	case string:
		return x
	case []byte:
		return string(x)
	default:
		return fmt.Sprintf("%v", x)
	}
}

func TestITPgBitRoundtrip(t *testing.T) {
	conn := itPgNode(t)
	defer conn.Close()
	quote := conn.GetDialect().Quoter().QuoteIdent
	table := "it_pg_bit"

	_, err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
	require.NoError(t, err)
	_, err = conn.Exec(fmt.Sprintf(
		"CREATE TABLE %s (id int PRIMARY KEY, c_bit8 bit(8), c_bit1 bit(1), c_varbit bit varying(16))",
		quote(table)))
	require.NoError(t, err, "建bit表失败")

	// 写入边界值：bit列参数化需显式cast（$1::bit(8)），位串文本直接传
	type bitRow struct {
		id         string
		bit8, bit1 any // string或nil
		varbit     any
	}
	rows := []bitRow{
		{"1", "11111111", "1", "101011"}, // 全1/满位/变长位串
		{"2", "00000000", "0", ""},       // 全0/零bit/空位串
		{"3", "10101010", nil, nil},      // NULL混合
	}
	for _, r := range rows {
		sql := fmt.Sprintf(
			"INSERT INTO %s (id, c_bit8, c_bit1, c_varbit) VALUES ($1, $2::bit(8), $3::bit(1), $4::bit varying(16))",
			quote(table))
		_, err = conn.Exec(sql, r.id, r.bit8, r.bit1, r.varbit)
		require.NoError(t, err, "写入bit边界行失败: %+v", r)
	}

	// 快照：读回位串文本
	snap := func() map[int64]map[string]string {
		_, rs, err := conn.Query(fmt.Sprintf("SELECT * FROM %s ORDER BY id", quote(table)))
		require.NoError(t, err)
		m := make(map[int64]map[string]string, len(rs))
		for _, row := range rs {
			id, _ := dbi.ValToInt64(row["id"])
			m[id] = map[string]string{
				"c_bit8":   bitText(t, row["c_bit8"]),
				"c_bit1":   bitText(t, row["c_bit1"]),
				"c_varbit": bitText(t, row["c_varbit"]),
			}
		}
		require.Len(t, m, 3)
		return m
	}
	before := snap()
	// 位串文本基准校验：lib/pq读回应为原样位串（非数字），前导零/变长不得丢失
	require.Equal(t, "11111111", before[1]["c_bit8"], "bit(8)全1位串")
	require.Equal(t, "1", before[1]["c_bit1"], "bit(1)")
	require.Equal(t, "101011", before[1]["c_varbit"], "bit varying变长位串")
	require.Equal(t, "00000000", before[2]["c_bit8"], "全0位串前导零不得丢失")
	require.Equal(t, "10101010", before[3]["c_bit8"], "NULL混合行的c_bit8应有值")
	require.Equal(t, "<NIL>", before[3]["c_bit1"], "NULL bit(1)")
	require.Equal(t, "<NIL>", before[3]["c_varbit"], "NULL bit varying")

	// dump到文件
	p := t.TempDir() + "/bit_backup.sql"
	bf, err := os.Create(p)
	require.NoError(t, err)
	bw := bufio.NewWriter(bf)
	require.NoError(t, transfer.DumpDbScript(context.Background(), conn, &dto.DumpDb{
		DbName:   conn.Info.Database,
		Tables:   []string{table},
		DumpDDL:  true,
		DumpData: true,
		Writer:   bw,
	}), "dump失败")
	require.NoError(t, bw.Flush())
	require.NoError(t, bf.Close())

	// 毁库恢复
	_, err = conn.Exec(fmt.Sprintf("DROP TABLE %s", quote(table)))
	require.NoError(t, err)
	rf, err := os.Open(p)
	require.NoError(t, err)
	defer rf.Close()
	app := &transfer.DbTransferAppImpl{}
	require.NoError(t, app.ImportDumpStream(context.Background(), 0, conn, bufio.NewReaderSize(rf, 1<<20)),
		"从备份文件恢复失败")

	after := snap()
	for id, beforeRow := range before {
		afterRow, ok := after[id]
		require.True(t, ok, "恢复后缺少id=%d", id)
		for col, v := range beforeRow {
			require.Equal(t, v, afterRow[col], "id=%d 列[%s] 备份恢复后位串不一致", id, col)
		}
	}
}
