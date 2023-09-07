package api

import (
	"compress/gzip"
	"io"
	"mayfly-go/pkg/biz"
)

type gzipWriter struct {
	tryFlushCount int
	writer        *gzip.Writer
	aborted       bool
}

func newGzipWriter(writer io.Writer) *gzipWriter {
	return &gzipWriter{writer: gzip.NewWriter(writer)}
}

func (g *gzipWriter) WriteString(data string) {
	if g.aborted {
		return
	}
	if _, err := g.writer.Write([]byte(data)); err != nil {
		g.aborted = true
		biz.IsTrue(false, "数据库导出失败：%s", err)
	}
}

func (g *gzipWriter) Close() {
	g.writer.Close()
}

func (g *gzipWriter) TryFlush() {
	if g.tryFlushCount%1000 == 0 {
		g.writer.Flush()
	}
	g.tryFlushCount += 1
}
