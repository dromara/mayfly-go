package storage

import (
	"context"
	"fmt"
	"io"
	"mayfly-go/internal/file/config"
	"mayfly-go/internal/file/imsg"
	"mayfly-go/pkg/errorx"
	"sync"
)

// 存储介质类型标识（与t_sys_file.storage_type取值一致）
const (
	TypeLocal = "local"
	TypeS3    = "s3"
)

// Info 存储介质信息
type Info struct {
	Type string
	// Bucket s3存储桶名，仅s3介质有值
	Bucket string
}

// Storage 文件存储抽象，屏蔽底层存储介质（本地磁盘、s3对象存储等）差异。
// key为全局唯一的文件路径标识，统一使用'/'作为分隔符（s3 object key格式）。
type Storage interface {
	// OpenWriter 创建指定key的文件写入器，调用方负责关闭。
	// 关闭后数据才会完整写入底层存储，关闭错误需视为写入失败
	OpenWriter(ctx context.Context, key string) (io.WriteCloser, error)

	// OpenReader 打开指定key的文件读取器，调用方负责关闭
	OpenReader(ctx context.Context, key string) (io.ReadCloser, error)

	// Remove 删除指定key的文件，文件不存在时不视为错误
	Remove(ctx context.Context, key string) error
}

// CurrentInfo 获取当前配置对应的存储介质信息
func CurrentInfo() Info {
	cfg := config.GetFileConfig()
	if cfg.S3 != nil {
		return Info{Type: TypeS3, Bucket: cfg.S3.Bucket}
	}
	return Info{Type: TypeLocal}
}

// GetStorage 获取当前配置对应的存储实现
func GetStorage(ctx context.Context) (Storage, error) {
	return GetStorageByType(ctx, CurrentInfo().Type)
}

// GetByFileInfo 根据文件记录的存储元信息获取对应的存储实现。
// storageType为空视为local（存量数据兼容）；
// s3文件会校验其记录的bucket与当前s3配置的一致性，避免bucket变更后对文件产生误操作
func GetByFileInfo(ctx context.Context, storageType string, extraBucket string) (Storage, error) {
	if storageType == "" {
		storageType = TypeLocal
	}
	s, err := GetStorageByType(ctx, storageType)
	if err != nil {
		return nil, err
	}

	if storageType == TypeS3 && extraBucket != "" {
		if info := CurrentInfo(); info.Type == TypeS3 && extraBucket != info.Bucket {
			return nil, errorx.NewBizI(ctx, imsg.ErrS3BucketMismatch, "bucket", extraBucket, "current", info.Bucket)
		}
	}
	return s, nil
}

type storageHolder struct {
	fingerprint string
	storage     Storage
}

var (
	mutex   sync.Mutex
	holders = map[string]*storageHolder{} // key: 存储介质类型
)

// GetStorageByType 获取指定类型的存储实现（按类型缓存实例，配置指纹变更后自动重建；
// 开闭原则：新增存储介质只需实现Storage接口并在此扩展选择逻辑）。
// 若指定s3但未配置s3则返回错误
func GetStorageByType(ctx context.Context, storageType string) (Storage, error) {
	cfg := config.GetFileConfig()

	var fp string
	var build func() (Storage, error)
	switch storageType {
	case TypeS3:
		if cfg.S3 == nil {
			return nil, errorx.NewBizI(ctx, imsg.ErrS3NotConfigured)
		}
		fp = configFingerprint(cfg.S3)
		build = func() (Storage, error) { return newS3Storage(cfg.S3) }
	case TypeLocal:
		fp = "local:" + cfg.BasePath
		build = func() (Storage, error) { return newLocalStorage(cfg.BasePath), nil }
	default:
		return nil, fmt.Errorf("unknown storage type: %s", storageType)
	}

	mutex.Lock()
	defer mutex.Unlock()
	if h := holders[storageType]; h != nil && h.fingerprint == fp {
		return h.storage, nil
	}

	s, err := build()
	if err != nil {
		return nil, err
	}
	holders[storageType] = &storageHolder{fingerprint: fp, storage: s}
	return s, nil
}

// configFingerprint 生成s3配置指纹，用于判断配置是否变更需要重建存储实现
func configFingerprint(cfg *config.S3Config) string {
	return fmt.Sprintf("s3:%s|%s|%s|%s|%s|%t", cfg.Endpoint, cfg.Region, cfg.Bucket, cfg.AccessKey, cfg.SecretKey, cfg.UsePathStyle)
}
