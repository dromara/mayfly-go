package writer

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
		return
	}

	if _, err := g.writer.Write(p); err != nil {
		g.aborted = true
		biz.IsTrue(false, "数据库导出失败：%s", err)
	}
	return
}

func (g *GzipWriter) Close() {
	g.writer.Close()
}

func (g *GzipWriter) TryFlush() {
	if g.tryFlushCount%1000 == 0 {
		g.writer.Flush()
	}
	g.tryFlushCount += 1
}
