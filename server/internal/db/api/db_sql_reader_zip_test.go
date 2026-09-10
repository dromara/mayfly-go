package api

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// zipEntry 压缩包内一个条目：dir 以 "/" 结尾时写为目录；gz 为 true 时内容先 gzip 再写入
type zipEntry struct {
	name  string
	text  string
	dir   bool
	gz    bool
	bigIt int // >0 时写入该长度的 'A'（用于构造高压缩比内容）
}

// buildZip 构造内存zip，条目顺序即写入顺序（用于验证读取侧按名称重排）
func buildZip(t *testing.T, entries ...zipEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		if e.dir {
			_, err := zw.Create(e.name)
			require.NoError(t, err)
			continue
		}
		w, err := zw.Create(e.name)
		require.NoError(t, err)
		content := e.text
		if e.bigIt > 0 {
			content = strings.Repeat("A", e.bigIt)
		}
		if e.gz {
			var gzBuf bytes.Buffer
			gw := gzip.NewWriter(&gzBuf)
			_, err = gw.Write([]byte(content))
			require.NoError(t, err)
			require.NoError(t, gw.Close())
			content = gzBuf.String()
		}
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func readAll(t *testing.T, r io.Reader) string {
	t.Helper()
	content, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(content)
}

// TestNewSqlFileReader SQL文件读取解包：SQL文件执行与备份文件导入共用同一道关口
//
// 该处若静默丢内容/静默截断，导入会表现为「执行成功但没导全数据」，事后无法追溯，
// 故对每种输入都断言实际读到的字节，异常输入必须返回 error。
func TestNewSqlFileReader(t *testing.T) {
	sqlText := "INSERT INTO t VALUES (1);\n"

	t.Run("普通sql文件原样透传", func(t *testing.T) {
		got, err := newSqlFileReader("backup.sql", strings.NewReader(sqlText))
		require.NoError(t, err)
		assert.Equal(t, sqlText, readAll(t, got))
	})

	t.Run("无后缀文件名原样透传", func(t *testing.T) {
		got, err := newSqlFileReader("backup", strings.NewReader(sqlText))
		require.NoError(t, err)
		assert.Equal(t, sqlText, readAll(t, got))
	})

	t.Run("zip单文件且大小写后缀均生效", func(t *testing.T) {
		zipBytes := buildZip(t, zipEntry{name: "a.sql", text: sqlText})
		got, err := newSqlFileReader("BACKUP.ZIP", bytes.NewReader(zipBytes))
		require.NoError(t, err)
		// 尾部多出的一个换行来自条目分隔符（SQL中空行无意义，不影响语义）
		assert.Equal(t, sqlText+"\n", readAll(t, got))
	})

	t.Run("zip内全部sql按名称升序全部导入", func(t *testing.T) {
		// 刻意乱序写入：02 先于 01，验证按名称排序（顺序一致才可复现）
		zipBytes := buildZip(t,
			zipEntry{name: "db/02_data.sql", text: "INSERT INTO t VALUES (2);"}, // 末尾无换行，靠分隔补上
			zipEntry{name: "db/01_schema.sql", text: "CREATE TABLE t (id INT);\n"},
		)
		got, err := newSqlFileReader("backup.zip", bytes.NewReader(zipBytes))
		require.NoError(t, err)
		content := readAll(t, got)
		assert.Equal(t, "CREATE TABLE t (id INT);\n\nINSERT INTO t VALUES (2);\n", content, "多文件未全部导入或顺序/分隔不符")
		// 文件之间必须以换行分隔：否则前文件尾行与后文件首行粘连成语义错误的语句
		stmts := strings.Split(strings.TrimSpace(content), ";\n")
		require.Len(t, stmts, 2, "两条语句被粘连")
	})

	t.Run("跳过目录与打包垃圾条目", func(t *testing.T) {
		zipBytes := buildZip(t,
			zipEntry{name: "db/", dir: true},
			zipEntry{name: "__MACOSX/db/._01.sql", text: "binary-junk"},
			zipEntry{name: "db/.DS_Store", text: "binary-junk"},
			zipEntry{name: "db/._01.sql", text: "binary-junk"},
			zipEntry{name: "db/01.sql", text: sqlText},
		)
		got, err := newSqlFileReader("backup.zip", bytes.NewReader(zipBytes))
		require.NoError(t, err)
		assert.Equal(t, sqlText+"\n", readAll(t, got), "垃圾条目混入将被当作SQL执行")
	})

	t.Run("包内无sql后缀时回退全部文件", func(t *testing.T) {
		zipBytes := buildZip(t, zipEntry{name: "dump.txt", text: sqlText})
		got, err := newSqlFileReader("backup.zip", bytes.NewReader(zipBytes))
		require.NoError(t, err)
		assert.Equal(t, sqlText+"\n", readAll(t, got))
	})

	t.Run("包内gz条目自动解压", func(t *testing.T) {
		zipBytes := buildZip(t, zipEntry{name: "db/01.sql.gz", text: sqlText, gz: true})
		got, err := newSqlFileReader("backup.zip", bytes.NewReader(zipBytes))
		require.NoError(t, err)
		out := readAll(t, got)
		// 未解压时读到的是 gzip 魔数开头的二进制，会被当作SQL交给数据库
		assert.NotContains(t, out, "\x1f\x8b", "嵌套gz未解压，二进制乱码会被当作SQL执行")
		assert.Equal(t, sqlText+"\n", out)
	})

	t.Run("顶层gz解压", func(t *testing.T) {
		var gzBuf bytes.Buffer
		gw := gzip.NewWriter(&gzBuf)
		_, err := gw.Write([]byte(sqlText))
		require.NoError(t, err)
		require.NoError(t, gw.Close())

		got, err := newSqlFileReader("backup.sql.gz", bytes.NewReader(gzBuf.Bytes()))
		require.NoError(t, err)
		assert.Equal(t, sqlText, readAll(t, got))
	})

	t.Run("非法gz内容报错", func(t *testing.T) {
		_, err := newSqlFileReader("backup.sql.gz", strings.NewReader("not gzip at all"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid gz")
	})

	t.Run("空zip报错", func(t *testing.T) {
		_, err := newSqlFileReader("backup.zip", bytes.NewReader(buildZip(t, zipEntry{name: "db/", dir: true})))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("非zip内容报错", func(t *testing.T) {
		_, err := newSqlFileReader("backup.zip", strings.NewReader("not a zip at all"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid zip")
	})

	t.Run("压缩包超限报错而非按截断包继续", func(t *testing.T) {
		oversize := bytes.Repeat([]byte("A"), sqlZipMaxBytes+1)
		_, err := newSqlFileReader("backup.zip", bytes.NewReader(oversize))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds")
	})
}

// TestNewSqlFileReaderRejectsZipBomb 解压炸弹必须被上限终止，且**不能**表现为读完（EOF）
//
// 若上限实现成「读到上限即EOF」，截断后的半份SQL会被当作完整文件导入并提示成功——静默丢数据。
func TestNewSqlFileReaderRejectsZipBomb(t *testing.T) {
	const bombSize = sqlUnzipMaxBytes + 64<<20 // 解压后约 264MB，压缩后仅数百KB
	zipBytes := buildZip(t, zipEntry{name: "bomb.sql", bigIt: bombSize})

	reader, err := newSqlFileReader("bomb.zip", bytes.NewReader(zipBytes))
	require.NoError(t, err)
	_, copyErr := io.Copy(io.Discard, reader)
	require.Error(t, copyErr, "解压超限必须以错误终止")
	assert.Contains(t, copyErr.Error(), "exceeds")
}
