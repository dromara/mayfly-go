package imsg

import (
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/i18n"
)

func init() {
	i18n.AppendLangMsg(i18n.Zh_CN, Zh_CN)
	i18n.AppendLangMsg(i18n.En, En)
}

const (
	ErrFileNotFound     = iota + consts.ImsgNumFile // 文件不存在
	ErrS3NotConfigured                              // 未配置s3存储
	ErrS3BucketMismatch                             // 文件所属s3桶与当前配置不一致
)
