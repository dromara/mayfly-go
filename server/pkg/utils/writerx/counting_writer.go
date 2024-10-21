package writerx

import "io"

type CountingWriteCloser struct {
	w io.WriteCloser
	n int64 // 已写入的字节数
}

func (c *CountingWriteCloser) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

func (c *CountingWriteCloser) Close() error {
	return c.w.Close()
}

func (c *CountingWriteCloser) BytesWritten() int64 {
	return c.n
}

func NewCountingWriteCloser(writer io.WriteCloser) *CountingWriteCloser {
	return &CountingWriteCloser{w: writer}
}
