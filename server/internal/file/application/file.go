package application

import (
	"context"
	"io"
	"mayfly-go/internal/file/domain/entity"
	"mayfly-go/internal/file/domain/repository"
	"mayfly-go/internal/file/imsg"
	"mayfly-go/internal/file/infra/storage"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/utils/writerx"
	"path"
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
	//   - accessMode: 访问模式，必显式声明（参数而非默认值，避免新增调用方默默产出可匿名下载的文件）
	//
	// 返回值:
	//   - fileKey: 文件key
	//   - error: 错误信息
	//
	// 注意：此方法会在defer中自动调用saveFunc，无论成功或失败都会正确处理文件保存或清理工作
	Upload(ctx context.Context, fileKey string, filename string, r io.Reader, accessMode entity.AccessMode) (string, error)

	// NewWriter 创建文件writer
	//
	// 参数:
	//   - canEmptyFileKey: 文件key，若不为空则使用该文件key，否则生成新的文件key
	//   - filename: 文件名，带文件后缀
	//   - accessMode: 访问模式，必显式声明；同 key 覆盖写时也以本次声明为准（写入方决定该产物给谁读）
	//
	// 返回值:
	//   - fileKey: 文件key
	//   - writer: 文件writer，实现了计数功能的io.Write
	//   - closeFunc: 关闭回调。用于保存文件信息，关闭writer等操作
	//               必须在defer中调用才会入库保存该文件信息
	//               若传入的错误参数不为nil，则不会保存文件信息，并会删除已创建的文件
	//   - err: 错误信息
	NewWriter(ctx context.Context, canEmptyFileKey string, filename string, accessMode entity.AccessMode) (fileKey string, writer io.Writer, closeFunc func(*error) error, err error)

	// CanAnonymousRead 判定该文件是否允许经通用匿名文件路由（GET /sys/files/:key）读取。
	//
	// 不存在返回文件不存在错误；非 public（含遗留零值）返回访问受限错误。
	// 该判定只服务于匿名路由自身，业务模块的下载入口已在路由上校了权限码，
	// 因此内部读取（如恢复备份文件）直接用 GetReader，不走本方法。
	CanAnonymousRead(ctx context.Context, fileKey string) error

	// ListPublicByKeys 按 key 集合返回可匿名读取的文件信息（映射至调用方提供的接收结构）。
	//
	// 私有文件不在结果里：元数据端点不校验权限，若原样返回就成了探测他人数据文件名与大小的旁路。
	ListPublicByKeys(keys []string, res any) error

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

var _ File = (*fileAppImpl)(nil)

func (f *fileAppImpl) Upload(ctx context.Context, fileKey string, filename string, r io.Reader, accessMode entity.AccessMode) (string, error) {
	var err error
	fileKey, writer, closeFunc, err := f.NewWriter(ctx, fileKey, filename, accessMode)
	if err != nil {
		return fileKey, err
	}
	// closeFunc的返回值可能携带保存文件信息失败等错误，需传播给调用方
	defer func() {
		if cerr := closeFunc(&err); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if _, err = io.Copy(writer, r); err != nil {
		return fileKey, err
	}
	return fileKey, nil
}

func (f *fileAppImpl) NewWriter(ctx context.Context, canEmptyFileKey string, filename string, accessMode entity.AccessMode) (fileKey string, writer io.Writer, closeFunc func(*error) error, err error) {
	file := &entity.File{}

	// 覆盖写场景下旧文件的信息：新文件写入并保存成功后再清理旧物理文件，
	// 保证任一步失败时记录仍指向完整可用的旧文件（也需在覆盖介质信息前捕获）
	var oldPath, oldStorageType, oldBucket string

	if canEmptyFileKey == "" {
		canEmptyFileKey = stringx.RandUUID()
		file.FileKey = canEmptyFileKey
	} else {
		file.FileKey = canEmptyFileKey
		err := f.GetByCond(file)
		if err == nil {
			oldPath, oldStorageType, oldBucket = file.Path, file.StorageType, file.GetExtraString("bucket")
		} else if !model.IsNotFound(err) {
			// 查询异常不能当作新文件处理，否则同key覆盖语义会被破坏（旧文件不清理且可能产生重复记录）
			return "", nil, nil, err
		}
	}
	file.Filename = filename
	// 访问模式以本次写入声明为准：上面按 key 查旧记录时 GetByCond 会把旧行的 access_mode 一并载入，
	// 若不重新赋值，覆盖写会继承旧文件的公开度（例如先把产物当头像上传过、后被数据文件覆盖）
	file.AccessMode = accessMode

	// 记录文件所属存储介质及元信息（s3记录bucket），保证介质/桶切换后仍能按原介质访问或给出明确错误
	info := storage.CurrentInfo()
	file.StorageType = info.Type
	if info.Bucket != "" {
		file.SetExtraValue("bucket", info.Bucket)
	}

	// 生成新的文件名
	newFilename := canEmptyFileKey + path.Ext(filename)
	fp, w, err := f.newWriter(ctx, newFilename)
	if err != nil {
		return "", nil, nil, err
	}
	file.Path = fp

	fileKey = canEmptyFileKey
	countWriter := writerx.NewCountingWriteCloser(w)
	// 创建回调函数
	closeFunc = func(e *error) error {
		// 关闭失败（如s3上传失败）视为写入失败，删除已写入的新文件；此时旧文件尚未清理，记录仍指向完整的旧文件
		if err := countWriter.Close(); err != nil {
			logx.ErrorfContext(ctx, "failed to close file writer: %s", err.Error())
			_ = f.removeObject(ctx, file.StorageType, file.GetExtraString("bucket"), file.Path)
			return err
		}

		if e != nil {
			err := *e
			if err != nil {
				logx.ErrorfContext(ctx, "the write file business logic failed: %s", err.Error())
				// 删除已经创建的新文件
				_ = f.removeObject(ctx, file.StorageType, file.GetExtraString("bucket"), file.Path)
				return err
			}
		}

		// 获取已写入的字节数
		file.Size = countWriter.BytesWritten()
		// 保存文件信息，保存失败时同样清理新文件，此时记录仍指向完整的旧文件
		if err := f.Save(ctx, file); err != nil {
			_ = f.removeObject(ctx, file.StorageType, file.GetExtraString("bucket"), file.Path)
			return err
		}

		// 保存成功后再清理旧物理文件（同小时内覆盖写路径相同则无需重复删除）；
		// 清理失败不阻断本次操作（内部已记录错误日志）
		if oldPath != "" && oldPath != file.Path {
			_ = f.removeObject(ctx, oldStorageType, oldBucket, oldPath)
		}
		return nil
	}

	return fileKey, countWriter, closeFunc, nil
}

// CanAnonymousRead 只有显式声明为 public 的文件才能经通用匿名路由读取；其余（含遗留零值）一律拒读。
func (f *fileAppImpl) CanAnonymousRead(ctx context.Context, fileKey string) error {
	file, err := f.getByKey(ctx, fileKey)
	if err != nil {
		return err
	}
	if !file.IsPublic() {
		return errorx.NewBizI(ctx, imsg.ErrFileAccessDenied)
	}
	return nil
}

// ListPublicByKeys 只返回 public 文件信息，私有文件从结果中自然缺失（不告知存不存在）。
func (f *fileAppImpl) ListPublicByKeys(keys []string, res any) error {
	if len(keys) == 0 {
		return nil
	}
	return f.ListByCondToAny(model.NewCond().In("file_key", keys).Eq("access_mode", entity.FileAccessModePublic), res)
}

func (f *fileAppImpl) GetReader(ctx context.Context, fileKey string) (string, io.ReadCloser, error) {
	file, err := f.getByKey(ctx, fileKey)
	if err != nil {
		return "", nil, err
	}
	s, err := storage.GetByFileInfo(ctx, file.StorageType, file.GetExtraString("bucket"))
	if err != nil {
		return "", nil, err
	}
	r, err := s.OpenReader(ctx, file.Path)
	return file.Filename, r, err
}

// getByKey 按 fileKey 取文件记录，单点收敛「不存在即业务错误」的口径（GetReader 与访问判定共用）。
func (f *fileAppImpl) getByKey(ctx context.Context, fileKey string) (*entity.File, error) {
	file := &entity.File{FileKey: fileKey}
	if err := f.GetByCond(file); err != nil {
		if model.IsNotFound(err) {
			return nil, errorx.NewBizI(ctx, imsg.ErrFileNotFound)
		}
		return nil, err
	}
	return file, nil
}

func (f *fileAppImpl) Remove(ctx context.Context, fileKey string) error {
	file := &entity.File{FileKey: fileKey}
	if err := f.GetByCond(file); err != nil {
		if model.IsNotFound(err) {
			return errorx.NewBizI(ctx, imsg.ErrFileNotFound)
		}
		return err
	}
	// 先删除物理文件（按文件记录的介质），成功后再删记录；
	// 避免先删记录后物理删除失败时产生无记录、无法再通过系统清理的残留文件
	if err := f.remove(ctx, file); err != nil {
		return err
	}
	return f.DeleteById(ctx, file.Id)
}

func (f *fileAppImpl) newWriter(ctx context.Context, filename string) (string, io.WriteCloser, error) {
	now := time.Now()
	// 统一使用'/'分隔符，兼容s3 object key格式
	filePath := path.Join(cast.ToString(now.Year()), cast.ToString(int(now.Month())), cast.ToString(now.Day()), cast.ToString(now.Hour()), filename)

	s, err := storage.GetStorage(ctx)
	if err != nil {
		return "", nil, err
	}
	w, err := s.OpenWriter(ctx, filePath)
	if err != nil {
		return "", nil, err
	}

	return filePath, w, nil
}

func (f *fileAppImpl) remove(ctx context.Context, file *entity.File) error {
	return f.removeObject(ctx, file.StorageType, file.GetExtraString("bucket"), file.Path)
}

// removeObject 删除指定介质上的物理文件，介质按文件记录的元信息选择，失败时记录错误日志
func (f *fileAppImpl) removeObject(ctx context.Context, storageType string, bucket string, fp string) error {
	s, err := storage.GetByFileInfo(ctx, storageType, bucket)
	if err != nil {
		logx.ErrorfContext(ctx, "failed to get storage for file [%s]: %s", fp, err.Error())
		return err
	}
	// 清理操作使用不可取消的context，避免写入失败回滚时请求已结束导致清理被中断产生残留文件
	if err := s.Remove(context.WithoutCancel(ctx), fp); err != nil {
		logx.ErrorfContext(ctx, "failed to delete old file [%s]: %s", fp, err.Error())
		return err
	}
	return nil
}
