//go:build it

package transfer

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
)

func TestDbgHet3(t *testing.T) {
	spec := &hetSpecs[2] // mysql->sqlite
	src := itMysqlNode(t)
	defer src.Close()
	tgt := itSqliteNode(t)
	defer tgt.Close()
	hetLoadSrc(t, src, spec)

	p := filepath.Join(t.TempDir(), "het3.sql")
	bf, err := os.Create(p)
	require.NoError(t, err)
	bw := bufio.NewWriterSize(bf, 1<<20)
	err = DumpDbScript(context.Background(), src, &dto.DumpDb{
		DbName:       src.Info.Database,
		Tables:       []string{spec.table},
		DumpDDL:      true,
		DumpData:     true,
		TargetDbType: dbi.DbType(spec.tgtDialect),
		Writer:       bw,
	})
	require.NoError(t, err, "dump失败")
	require.NoError(t, bw.Flush())
	require.NoError(t, bf.Close())

	content, err := os.ReadFile(p)
	require.NoError(t, err)
	s := string(content)
	// 打印首个DDL段
	idx := strings.Index(s, "CREATE")
	if idx >= 0 {
		t.Log("DDL SNIPPET:", s[idx:idx+600])
	} else {
		t.Log("NO CREATE FOUND! file size:", len(s), "head:", s[:minInt2(len(s), 500)])
	}

	rf, err := os.Open(p)
	require.NoError(t, err)
	defer rf.Close()
	app := &DbTransferAppImpl{}
	if err := app.importDumpStream(context.Background(), 0, tgt, bufio.NewReaderSize(rf, 1<<20)); err != nil {
		t.Log("import err:", err)
	}
	_, rows, err := tgt.Query(`SELECT COUNT(*) AS c FROM it_het_b`)
	require.NoError(t, err, "目标查表失败")
	t.Log("target count:", rows[0]["c"])
}

func minInt2(a, b int) int {
	if a < b {
		return a
	}
	return b
}
