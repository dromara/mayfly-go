package storage

import (
	"context"
	"fmt"
	"io"
	"mayfly-go/internal/file/config"
	"mime"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// s3Storage 基于aws-sdk-go-v2的s3对象存储实现，可对接任意符合s3协议的对象存储
type s3Storage struct {
	client *s3.Client
	// Uploader并发安全可全局复用，默认5MB分片、5并发，大文件流式分片上传避免占用过多内存
	uploader *manager.Uploader
	bucket   string
}

func newS3Storage(cfg *config.S3Config) (*s3Storage, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load s3 config error: %w", err)
	}

	// 默认的flexible checksum会在流式上传时使用STREAMING-*-TRAILER流式签名，
	// 部分s3兼容存储（华为OBS、Cloudflare R2、B2等）不支持该协议，
	// 会报XAmzContentSHA256Mismatch，改为仅在操作显式要求时才计算/校验校验和
	awsCfg.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
	awsCfg.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(normalizeEndpoint(cfg.Endpoint))
		o.UsePathStyle = cfg.UsePathStyle
	})

	// Uploader自带同名配置且默认WhenSupported，不会继承上面aws.Config的设置，
	// 会给每个分片请求强加ChecksumAlgorithm（表现为x-amz-sdk-checksum-algorithm/x-amz-checksum-*头），
	// 导致s3兼容存储的分片上传失败（单请求PutObject正常但UploadPart报XAmzContentSHA256Mismatch），
	// 必须同步设置为WhenRequired
	uploader := manager.NewUploader(client, func(o *manager.Uploader) {
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
	})

	return &s3Storage{client: client, uploader: uploader, bucket: cfg.Bucket}, nil
}

// normalizeEndpoint 规范化endpoint，未携带协议时默认使用https
func normalizeEndpoint(endpoint string) string {
	if endpoint != "" && !strings.Contains(endpoint, "://") {
		return "https://" + endpoint
	}
	return endpoint
}

func (s *s3Storage) OpenWriter(ctx context.Context, key string) (io.WriteCloser, error) {
	pr, pw := io.Pipe()

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   pr,
	}
	// 根据扩展名设置Content-Type，保证浏览器访问/下载时的内容类型正确
	if ct := mime.TypeByExtension(path.Ext(key)); ct != "" {
		input.ContentType = aws.String(ct)
	}

	w := &s3UploadWriter{pw: pw, done: make(chan struct{})}
	// 使用WithoutCancel避免业务请求结束时上传被意外取消
	uploadCtx := context.WithoutCancel(ctx)
	go func() {
		defer close(w.done)
		_, err := s.uploader.Upload(uploadCtx, input)
		// 上传失败时通知写端，并通过writer将错误传递给Close调用方
		pr.CloseWithError(err)
		w.err = err
	}()

	return w, nil
}

func (s *s3Storage) OpenReader(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

func (s *s3Storage) Remove(ctx context.Context, key string) error {
	// s3删除不存在的object不会报错，无需额外处理
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

// s3UploadWriter 包装管道写端，将异步上传结果传递给Close调用方
type s3UploadWriter struct {
	pw   *io.PipeWriter
	done chan struct{}
	err  error
}

func (w *s3UploadWriter) Write(p []byte) (int, error) {
	return w.pw.Write(p)
}

// Close 关闭写端并等待上传完成，上传失败则返回对应错误。
// 注意：调用方必须调用Close，否则上传goroutine会因管道未关闭而永久阻塞
func (w *s3UploadWriter) Close() error {
	closeErr := w.pw.Close()
	// 等待上传goroutine结束，确保错误结果已写入
	<-w.done
	if w.err != nil {
		return w.err
	}
	return closeErr
}
