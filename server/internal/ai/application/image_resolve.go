package application

import (
	"context"
	"encoding/base64"
	"io"
	"mime"
	"strings"
	"time"

	fileapp "mayfly-go/internal/file/application"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/logx"
)

// imageDataURLCache 图片解析结果进程内缓存（fileKey → base64 data URL）。
//
// GetHistory 在每轮 LLM 请求都会重建全部历史消息（agent/context.go、history 扩展），
// 无缓存时每轮都要对会话内全部历史图片重读 local/S3 并 base64 编码；
// 附件内容落库后不可变（fileKey 由上传新生成，无覆盖写场景），缓存安全。
// TTL 仅作内存回收手段：data URL 体积大（数百 KB/张），不适合放入 Redis 全局缓存
var imageDataURLCache = cache.NewLocalCache()

const imageDataURLCacheTTL = 30 * time.Minute

// ResolveImageUrls 将图片文件引用（文件服务 fileKey）解析为 LLM 可用的
// base64 data URL：经文件服务按介质路由（local/S3）读取内容后转 base64，
// 避免把内网文件 URL 交给 LLM 网关拉取。
// 单个引用解析失败仅告警并跳过该图（正文占位说明仍在，不阻断整轮对话）
func ResolveImageUrls(ctx context.Context, fileApp fileapp.File, refs []string) []string {
	if len(refs) == 0 {
		return refs
	}
	urls := make([]string, 0, len(refs))
	for _, ref := range refs {
		if cached, ok := imageDataURLCache.Get(ref); ok {
			if url, _ := cached.(string); url != "" {
				urls = append(urls, url)
				continue
			}
		}
		url, err := resolveImageDataUrl(ctx, fileApp, ref)
		if err != nil {
			// 解析失败仅告警并跳过该图（正文占位说明仍在，不阻断整轮对话）；
			// 失败结果不缓存，下次请求重试
			logx.ErrorfContext(ctx, "[ResolveImageUrls] resolve image failed, fileKey=%s, err=%v", ref, err)
			continue
		}
		_ = imageDataURLCache.Set(ref, url, imageDataURLCacheTTL)
		urls = append(urls, url)
	}
	return urls
}

// resolveImageDataUrl 读取单个图片文件并转 base64 data URL（mime 按扩展名推断）
func resolveImageDataUrl(ctx context.Context, fileApp fileapp.File, fileKey string) (string, error) {
	filename, reader, err := fileApp.GetReader(ctx, fileKey)
	if err != nil {
		return "", err
	}
	content, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil {
		return "", err
	}
	mimeType := "image/png"
	if idx := strings.LastIndex(filename, "."); idx >= 0 {
		if m := mime.TypeByExtension(strings.ToLower(filename[idx:])); m != "" {
			mimeType = m
		}
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(content), nil
}
