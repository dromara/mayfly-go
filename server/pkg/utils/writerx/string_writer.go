package writerx

import (
	"io"
)

type StringWriter struct {
	io.WriteCloser
}

func (sw *StringWriter) WriteString(s string) (n int, err error) {
	return sw.WriteCloser.Write([]byte(s))
}

func (sw *StringWriter) Close() error {
	return sw.WriteCloser.Close()
}

func NewStringWriter(writer io.WriteCloser) *StringWriter {
	return &StringWriter{WriteCloser: writer}
}
