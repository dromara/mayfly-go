package imsg

import "mayfly-go/pkg/i18n"

var Zh_CN = map[i18n.MsgId]string{
	ErrFileNotFound:     "文件不存在",
	ErrS3NotConfigured:  "当前未配置s3存储，无法访问存储在s3上的文件",
	ErrS3BucketMismatch: "该文件所属s3存储桶【{{.bucket}}】与当前配置的存储桶【{{.current}}】不一致，请调整s3配置或联系管理员",
}
