package itest

// 备份落盘加密的真实产物集成测试：驱动生产 DumpDbScript，按生产同样的顺序
// 把「加密器 → 压缩器 → 导出内核」套在一起，然后从磁盘读回来验三件事：
//   1. 磁盘上的字节以 magic 开头，且**不含明文表名/SQL 关键字**；
//   2. 解密后是合法 SQL，且数据行完整；
//   3. 没有 magic 的历史明文文件仍能被读取（旧备份必须可恢复）。
//
// 运行：cd server && go test -tags it -count=1 -run TestITBackupEncryptAtRest ./internal/db/application/transfer/itest/

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/application/transfer"
	"mayfly-go/internal/db/application/transfer/backupenc"
	"mayfly-go/internal/db/dbm/dbi"
)

func TestITBackupEncryptAtRest(t *testing.T) {
	// 先建连接再改配置：连接建立会读运行时配置（连接池/超时），
	// 用一个临时最小配置去建连会把 di.db 留成 nil，随后 Exec 直接空指针
	conn := itMysqlNode(t)
	// 用 Cleanup 而不是 defer：defer 先于 Cleanup 执行，会把「清理时 DROP 表」留在已关闭的连接上
	t.Cleanup(func() { _ = conn.Close() })
	// 连接是懒打开的：先把底层 *sql.DB 真正建立起来，再覆写配置，
	// 否则首次 Exec 会在最小配置下走一次建连并留下 nil 的 db
	require.NoError(t, conn.Ping())

	const testKey = "0123456789abcdef0123456789abcdef"

	table := "it_bk_enc_probe"
	mustItExec(t, conn, fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
	// 表名与行值里都放可搜索的标记：密文里出现任何一个都说明没加密
	mustItExec(t, conn, fmt.Sprintf("CREATE TABLE %s (id INT PRIMARY KEY, note VARCHAR(64))", table))
	mustItExec(t, conn, fmt.Sprintf("INSERT INTO %s (id, note) VALUES (1, 'plaintext-marker-alpha'), (2, 'plaintext-marker-beta')", table))
	t.Cleanup(func() { mustItExec(t, conn, fmt.Sprintf("DROP TABLE IF EXISTS %s", table)) })

	path := filepath.Join(t.TempDir(), "backup.sql.gz")
	file, err := os.Create(path)
	require.NoError(t, err)

	// 与生产同序：底层文件 → 加密器 → gzip → 导出内核
	encWriter, err := backupenc.NewWriterWithKey(file, testKey)
	require.NoError(t, err)
	gz := gzip.NewWriter(encWriter)
	dumpErr := transfer.DumpDbScript(context.Background(), conn, &dto.DumpDb{
		DbName:       conn.Info.Database,
		Tables:       []string{table},
		DumpDDL:      true,
		DumpData:     true,
		TargetDbType: dbi.ToDbType("mysql"),
		Writer:       gz,
	})
	require.NoError(t, gz.Close())
	require.NoError(t, encWriter.Close())
	require.NoError(t, file.Close())
	require.NoError(t, dumpErr)

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(raw, []byte("MAYFLYDB1")), "盘上文件必须以加密 magic 开头")
	for _, marker := range []string{"plaintext-marker-alpha", "CREATE TABLE", table} {
		require.False(t, bytes.Contains(raw, []byte(marker)), "密文里不应出现明文片段：%s", marker)
	}

	// 读回：解密 → 解压 → 断言内容完整
	readFile, err := os.Open(path)
	require.NoError(t, err)
	defer readFile.Close()
	decrypted, err := backupenc.WrapReaderWithKey(readFile, testKey)
	require.NoError(t, err)
	plain, err := gzip.NewReader(decrypted)
	require.NoError(t, err)
	script, err := io.ReadAll(plain)
	require.NoError(t, err)
	text := string(script)
	require.Contains(t, text, "CREATE TABLE")
	require.Contains(t, text, "plaintext-marker-alpha")
	require.Contains(t, text, "plaintext-marker-beta")

	// 历史明文备份：没有 magic，必须原样返回
	legacy := []byte("-- legacy plain backup\nSELECT 1;\n")
	legacyPath := filepath.Join(t.TempDir(), "legacy.sql")
	require.NoError(t, os.WriteFile(legacyPath, legacy, 0o600))
	lf, err := os.Open(legacyPath)
	require.NoError(t, err)
	defer lf.Close()
	plainReader, err := backupenc.WrapReaderWithKey(lf, testKey)
	require.NoError(t, err)
	got, err := io.ReadAll(plainReader)
	require.NoError(t, err)
	require.Equal(t, string(legacy), string(got), "历史明文备份必须逐字节读出")
	require.True(t, strings.HasPrefix(string(got), "-- legacy"), "开头不能被吃掉")
}
