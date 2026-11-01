//go:build it

package itest

// Settings 的批预算必须真正改变导出产物形态（驱动生产 DumpDbScript + 真实库）。
//
// 历史缺陷：exportData 只读常量 DefaultBatchRows/DefaultBatchBytes，Settings 里声明的
// BatchRows/BatchBytes 从未被读取 —— 调用方"为大字段调小批量"的诉求被静默忽略，
// 配置项看起来生效、实际没有任何作用。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/application/transfer"
	"mayfly-go/internal/db/dbm/export"
)

func TestITDumpSettingsBatchRowsTakeEffect(t *testing.T) {
	src := itMysqlNode(t)
	t.Cleanup(func() { src.Close() })

	table := "it_dump_settings_batch"
	q := src.GetDialect().Quoter().QuoteIdent(table)
	mustItExec(t, src, "DROP TABLE IF EXISTS "+q)
	mustItExec(t, src, fmt.Sprintf("CREATE TABLE %s (id int primary key, name varchar(50))", q))
	for i := 1; i <= 5; i++ {
		mustItExec(t, src, fmt.Sprintf("INSERT INTO %s VALUES (%d,'n%d')", q, i, i))
	}

	path := filepath.Join(t.TempDir(), "batch.sql")
	f, err := os.Create(path)
	require.NoError(t, err)

	// BatchRows=2：5 行应切成 3 条 INSERT（2 + 2 + 1）
	settings := export.DefaultSettings("sql")
	settings.BatchRows = 2
	require.NoError(t, transfer.DumpDbScript(context.Background(), src, &dto.DumpDb{
		DbName:   src.Info.Database,
		Tables:   []string{table},
		DumpDDL:  false,
		DumpData: true,
		Settings: settings,
		Writer:   f,
	}))
	require.NoError(t, f.Close())

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	sqlText := string(content)

	require.Equal(t, 3, strings.Count(sqlText, "INSERT INTO"),
		"BatchRows=2 时 5 行应产生 3 条 INSERT（配置必须真正生效，而不是恒用默认 100）")

	// 每条 INSERT 的 VALUES 元组数不得超过 2
	for _, stmt := range strings.Split(sqlText, "INSERT INTO")[1:] {
		tuples := strings.Count(stmt, "),(") + 1
		require.LessOrEqual(t, tuples, 2, "单条 INSERT 的元组数不得超过 BatchRows=2")
	}

	// 对照组：默认设置（100/批）下 5 行应为 1 条
	path2 := filepath.Join(t.TempDir(), "default.sql")
	f2, err := os.Create(path2)
	require.NoError(t, err)
	require.NoError(t, transfer.DumpDbScript(context.Background(), src, &dto.DumpDb{
		DbName:   src.Info.Database,
		Tables:   []string{table},
		DumpData: true,
		Writer:   f2,
	}))
	require.NoError(t, f2.Close())
	content2, err := os.ReadFile(path2)
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(string(content2), "INSERT INTO"),
		"默认设置应把 5 行放进同一条批量 INSERT")
}
