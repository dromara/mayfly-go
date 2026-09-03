package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	ErrFileNotFound:     "file not found",
	ErrS3NotConfigured:  "s3 storage is not currently configured, files stored on s3 are inaccessible",
	ErrS3BucketMismatch: "the file belongs to s3 bucket [{{.bucket}}], which does not match the currently configured bucket [{{.current}}]",
}
