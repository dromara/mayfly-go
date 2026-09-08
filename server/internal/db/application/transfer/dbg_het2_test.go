//go:build it

package transfer

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
)

func TestDbgHetDump2(t *testing.T) {
	spec := &hetSpecs[0] // mysql->pg
	src := itMysqlNode(t)
	defer src.Close()
	tgt := itPgNode(t)
	defer tgt.Close()
	hetLoadSrc(t, src, spec)

	p := filepath.Join(t.TempDir(), "het2.sql")
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

	// 导入到pg
	rf, err := os.Open(p)
	require.NoError(t, err)
	app := &DbTransferAppImpl{}
	if err := app.importDumpStream(context.Background(), 0, tgt, bufio.NewReaderSize(rf, 1<<20)); err != nil {
		t.Log("import err:", err)
	}
	rf.Close()
	_, rows, err := tgt.Query(`SELECT COUNT(*) AS c FROM it_het_a`)
	require.NoError(t, err, "目标查表失败")
	t.Log("target count:", rows[0]["c"])

	content, err := os.ReadFile(p)
	require.NoError(t, err)
	s := string(content)
	re := regexp.MustCompile(`(?m)^(DROP|CREATE|INSERT|BEGIN|COMMIT)[^;]{0,80}`)
	for _, l := range re.FindAllString(s, 20) {
		t.Log("STMT:", strings.Split(l, "\n")[0])
	}
	t.Log("file size:", len(s))
	t.Log("has CREATE:", strings.Contains(s, "CREATE TABLE"))
}
