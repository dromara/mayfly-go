package writerx

import (
	"bytes"
	"compress/gzip"
	"encoding/csv"
	"io"
	"testing"
)

// TestGzipWriterReturnsWrittenCount GzipWriter.Write 必须返回真实字节数。
//
// 标准库会在每次 Write 后校验 n == len(p)，不等就记 io.ErrShortWrite 并在 Flush/Close 时抛出。
// 返回值一旦被写成 (0, nil)，所有走 csv/json 等标准写入器的 gzip 产物都会报
// 「short write」而底层写入其实全部成功——这是一类「代码看着没错的错」，只能被断言钉住。
func TestGzipWriterReturnsWrittenCount(t *testing.T) {
	var buf bytes.Buffer
	gz := NewGzipWriter(&buf)

	writer := csv.NewWriter(gz)
	if err := writer.Write([]string{"id", "name"}); err != nil {
		t.Fatalf("csv write failed: %s", err.Error())
	}
	if err := writer.Write([]string{"1", "n1"}); err != nil {
		t.Fatalf("csv write failed: %s", err.Error())
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		t.Fatalf("csv flush failed: %s", err.Error())
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close failed: %s", err.Error())
	}

	reader, err := gzip.NewReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("the payload must be valid gzip: %s", err.Error())
	}
	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read failed: %s", err.Error())
	}
	if got := string(raw); got != "id,name\n1,n1\n" {
		t.Fatalf("uncompressed payload = %q", got)
	}
}

// TestGzipWriterReportsAfterAbort 中止后的写入必须报错而不是静默返回 0,nil：
// 静默成功会让上层以为数据已写出，产出一份缺内容的文件
func TestGzipWriterReportsAfterAbort(t *testing.T) {
	var buf bytes.Buffer
	gz := NewGzipWriter(&buf)
	gz.aborted = true

	if _, err := gz.Write([]byte("x")); err == nil {
		t.Fatal("an aborted writer must report an error instead of pretending success")
	}
}
