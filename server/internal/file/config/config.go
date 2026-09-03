package config

import (
	"cmp"
	sysapp "mayfly-go/internal/sys/application"
	"mayfly-go/pkg/utils/collx"

	"github.com/spf13/cast"
)

const (
	ConfigKeyFile string = "FileConfig" // 文件配置key
)

type FileConfig struct {
	BasePath string    // 文件基础路径
	S3       *S3Config // s3对象存储配置，为nil则使用本地文件存储
}

// S3Config s3对象存储配置，对接任意符合s3协议的对象存储（minio、oss、cos、r2等）
type S3Config struct {
	Endpoint     string // 服务地址，如 http://127.0.0.1:9000
	Region       string // 区域，默认 us-east-1
	Bucket       string // 存储桶名
	AccessKey    string
	SecretKey    string
	UsePathStyle bool // 是否使用path-style访问（minio等自建存储一般为true）
}

func GetFileConfig() *FileConfig {
	c := sysapp.GetConfigApp().GetConfig(ConfigKeyFile)
	jm := c.GetJsonM()

	fc := new(FileConfig)
	fc.BasePath = cmp.Or(jm.GetStr("basePath"), "./file")
	fc.S3 = getS3Config(jm)
	return fc
}

// getS3Config 从配置中解析s3配置，若s3Endpoint、s3Bucket、s3AccessKey、s3SecretKey均配置则视为启用s3存储，否则返回nil
func getS3Config(jm collx.M) *S3Config {
	endpoint := jm.GetStr("s3Endpoint")
	bucket := jm.GetStr("s3Bucket")
	accessKey := jm.GetStr("s3AccessKey")
	secretKey := jm.GetStr("s3SecretKey")
	if endpoint == "" || bucket == "" || accessKey == "" || secretKey == "" {
		return nil
	}

	return &S3Config{
		Endpoint:     endpoint,
		Region:       cmp.Or(jm.GetStr("s3Region"), "us-east-1"),
		Bucket:       bucket,
		AccessKey:    accessKey,
		SecretKey:    secretKey,
		UsePathStyle: cast.ToBool(jm.GetStr("s3PathStyle")),
	}
}
