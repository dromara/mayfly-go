package writer

import (
	"io"
	"os"
	"path/filepath"
)

type FileWriter struct {
	tryFlushCount int
	writer        *os.File
	aborted       bool
}

func NewFileWriter(filePath string) *FileWriter {
	if filePath == "" {
		panic("filePath is empty")
	}

	// 使用filepath.Dir函数提取文件夹路径
	dir := filepath.Dir(filePath)
	if dir != "" {
		// 检查文件夹路径，不存在则创建
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			err = os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				panic(err)
			}
		}
	}

	fw, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}
	return &FileWriter{writer: fw}
}

func (f *FileWriter) Close() {
	f.writer.Close()
}

func (f *FileWriter) TryFlush() {
}
func (f *FileWriter) Write(b []byte) (n int, err error) {
	return f.writer.Write(b)
}

func (f *FileWriter) WriteString(data string) {
	io.WriteString(f.writer, data)
}
