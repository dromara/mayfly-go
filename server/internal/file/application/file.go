package application

import (
	"context"
	"io"

	"mayfly-go/internal/file/config"
	"mayfly-go/internal/file/domain/entity"
	"mayfly-go/internal/file/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/utils/writerx"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cast"
)

type File interface {
	base.App[*entity.File]

	// Upload 上传文件
	//
	// 参数:
	//   - fileKey: 文件key，若不为空则使用该文件key，否则生成新的文件key
	//   - filename: 文件名，带文件后缀
	//   - r: 文件内容读取流
	//
	// 返回值:
	//   - fileKey: 文件key
	//   - error: 错误信息
	//
	// 注意：此方法会在defer中自动调用saveFunc，无论成功或失败都会正确处理文件保存或清理工作
	Upload(ctx context.Context, fileKey string, filename string, r io.Reader) (string, error)

	// NewWriter 创建文件writer
	//
	// 参数:
	//   - canEmptyFileKey: 文件key，若不为空则使用该文件key，否则生成新的文件key
	//   - filename: 文件名，带文件后缀
	//
	// 返回值:
	//   - fileKey: 文件key
	//   - writer: 文件writer，实现了计数功能的io.Write
	//   - closeFunc: 关闭回调。用于保存文件信息，关闭writer等操作
	//               必须在defer中调用才会入库保存该文件信息
	//               若传入的错误参数不为nil，则不会保存文件信息，并会删除已创建的文件
	//   - err: 错误信息
	NewWriter(ctx context.Context, canEmptyFileKey string, filename string) (fileKey string, writer io.Writer, closeFunc func(*error) error, err error)

	// GetReader 获取文件读取器
	//
	// 参数:
	//   - fileKey: 文件唯一标识key
	//
	// 返回值:
	//   - filename: 文件名（带后缀）
	//   - reader: 文件读取流，调用方需负责关闭
	//   - err: 错误信息
	GetReader(ctx context.Context, fileKey string) (string, io.ReadCloser, error)

	// Remove 删除文件
	Remove(ctx context.Context, fileKey string) error
}

type fileAppImpl struct {
	base.AppImpl[*entity.File, repository.File]
}

func (f *fileAppImpl) Upload(ctx context.Context, fileKey string, filename string, r io.Reader) (string, error) {
	var err error
	fileKey, writer, closeFunc, err := f.NewWriter(ctx, fileKey, filename)
	if err != nil {
		return fileKey, err
	}
	defer closeFunc(&err)

	if _, err = io.Copy(writer, r); err != nil {
		return fileKey, err
	}
	return fileKey, nil
}

func (f *fileAppImpl) NewWriter(ctx context.Context, canEmptyFileKey string, filename string) (fileKey string, writer io.Writer, closeFunc func(*error) error, err error) {
	isNewFile := true
	file := &entity.File{}

	if canEmptyFileKey == "" {
		canEmptyFileKey = stringx.RandUUID()
		file.FileKey = canEmptyFileKey
	} else {
		file.FileKey = canEmptyFileKey
		if err := f.GetByCond(file); err == nil {
			isNewFile = false
		}
	}
	file.Filename = filename

	if !isNewFile {
		// 先删除旧文件
		f.remove(ctx, file)
	}

	// 生成新的文件名
	newFilename := canEmptyFileKey + filepath.Ext(filename)
	fp, w, err := f.newWriter(newFilename)
	if err != nil {
		return "", nil, nil, err
	}
	file.Path = fp

	fileKey = canEmptyFileKey
	countWriter := writerx.NewCountingWriteCloser(w)
	// 创建回调函数
	closeFunc = func(e *error) error {
		countWriter.Close()

		if e != nil {
			err := *e
			if err != nil {
				logx.Errorf("the write file business logic failed: %s", err.Error())
				// 删除已经创建的文件
				f.remove(ctx, file)
				return err
			}
		}

		// 获取已写入的字节数
		file.Size = countWriter.BytesWritten()
		// 保存文件信息
		return f.Save(ctx, file)
	}

	return fileKey, countWriter, closeFunc, nil
}

func (f *fileAppImpl) GetReader(ctx context.Context, fileKey string) (string, io.ReadCloser, error) {
	file := &entity.File{FileKey: fileKey}
	if err := f.GetByCond(file); err != nil {
		return "", nil, errorx.NewBiz("file not found")
	}
	r, err := os.Open(filepath.Join(config.GetFileConfig().BasePath, file.Path))
	return file.Filename, r, err
}

func (f *fileAppImpl) Remove(ctx context.Context, fileKey string) error {
	file := &entity.File{FileKey: fileKey}
	if err := f.GetByCond(file); err != nil {
		return errorx.NewBiz("file not found")
	}
	f.DeleteById(ctx, file.Id)
	return f.remove(ctx, file)
}

func (f *fileAppImpl) newWriter(filename string) (string, io.WriteCloser, error) {
	now := time.Now()
	filePath := filepath.Join(cast.ToString(now.Year()), cast.ToString(int(now.Month())), cast.ToString(now.Day()), cast.ToString(now.Hour()), filename)
	fileAbsPath := filepath.Join(config.GetFileConfig().BasePath, filePath)

	// 目录不存在则创建
	fileDir := filepath.Dir(fileAbsPath)
	if _, err := os.Stat(fileDir); os.IsNotExist(err) {
		err = os.MkdirAll(fileDir, os.ModePerm)
		if err != nil {
			return "", nil, err
		}
	}

	// 创建目标文件
	out, err := os.OpenFile(fileAbsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0766)
	if err != nil {
		return "", nil, err
	}

	return filePath, out, nil
}

func (f *fileAppImpl) remove(ctx context.Context, file *entity.File) error {
	if err := os.Remove(filepath.Join(config.GetFileConfig().BasePath, file.Path)); err != nil {
		logx.ErrorfContext(ctx, "failed to delete old file [%s]: %s", file.Path, err.Error())
		return err
	}
	return nil
}
