package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

// localStorage 本地文件存储实现
type localStorage struct {
	basePath string
}

func newLocalStorage(basePath string) *localStorage {
	return &localStorage{basePath: basePath}
}

func (l *localStorage) OpenWriter(ctx context.Context, key string) (io.WriteCloser, error) {
	fp := filepath.Join(l.basePath, filepath.FromSlash(key))

	// 目录不存在则创建
	fileDir := filepath.Dir(fp)
	if _, err := os.Stat(fileDir); os.IsNotExist(err) {
		if err := os.MkdirAll(fileDir, 0755); err != nil {
			return nil, err
		}
	}

	// 使用O_TRUNC覆盖语义，与s3的PutObject覆盖行为保持一致，避免同key重复写入时旧文件残留导致数据损坏
	return os.OpenFile(fp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
}

func (l *localStorage) OpenReader(ctx context.Context, key string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(l.basePath, filepath.FromSlash(key)))
}

func (l *localStorage) Remove(ctx context.Context, key string) error {
	if err := os.Remove(filepath.Join(l.basePath, filepath.FromSlash(key))); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
