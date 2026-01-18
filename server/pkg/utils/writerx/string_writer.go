package writerx

import (
	"io"
)

type StringWriter struct {
	io.Writer
}

func (sw *StringWriter) WriteString(s string) (n int, err error) {
	return sw.Writer.Write([]byte(s))
}

func NewStringWriter(writer io.Writer) *StringWriter {
	return &StringWriter{Writer: writer}
}
