package writer

import "io"

type CustomWriter interface {
	io.Writer
	WriteString(data string)
	Close()
	TryFlush()
}
