package writerx

import (
	"compress/gzip"
	"io"
	"mayfly-go/pkg/biz"
)

type GzipWriter struct {
	tryFlushCount int
	writer        *gzip.Writer
	aborted       bool
}

func NewGzipWriter(writer io.Writer) *GzipWriter {
	return &GzipWriter{writer: gzip.NewWriter(writer)}
}

func (g *GzipWriter) WriteString(data string) {
	if g.aborted {
		return
	}
	if _, err := g.writer.Write([]byte(data)); err != nil {
		g.aborted = true
		biz.IsTrue(false, "数据库导出失败：%s", err)
	}
}

func (g *GzipWriter) Write(p []byte) (n int, err error) {
	if g.aborted {
		return 0, io.ErrClosedPipe
	}

	if _, err := g.writer.Write(p); err != nil {
		g.aborted = true
		biz.IsTrue(false, "数据库导出失败：%s", err)
	}
	// 必须返回真实字节数：标准库（如 encoding/csv）写后会校验 n == len(p)，
	// 拿 (0, nil) 会被判成 short write 并在 Close 时报错——底层写入其实全部成功，
	// 结果整份导出因这个返回值而失败
	return len(p), nil
}

func (g *GzipWriter) Close() error {
	return g.writer.Close()
}

func (g *GzipWriter) TryFlush() {
	if g.tryFlushCount%1000 == 0 {
		g.writer.Flush()
	}
	g.tryFlushCount += 1
}
