package protocol

import "testing"

// ImageUrlsOf 仅提取 image 段 extra.fileKey（文件服务引用），
// 其余类型段与无 fileKey 的 image 段均忽略，顺序保持原文档序
func TestImageUrlsOf(t *testing.T) {
	segments := []ContentSegment{
		NewInputTextSegment("看这两张图"),
		{Type: ContentSegmentImage, Text: "[图片]", Extra: map[string]any{"fileKey": "k1", "name": "a.png"}},
		{Type: ContentSegmentResource, Text: "[引用资源]", Extra: map[string]any{"fileKey": "should-ignore"}},
		{Type: ContentSegmentImage, Text: "[图片]", Extra: map[string]any{"name": "b.png"}}, // 无 fileKey
		{Type: ContentSegmentImage, Text: "[图片]", Extra: map[string]any{"fileKey": "k2"}},
	}
	got := ImageUrlsOf(segments)
	if len(got) != 2 || got[0] != "k1" || got[1] != "k2" {
		t.Errorf("应按序仅提取 image 段的 fileKey, got %v", got)
	}
}

func TestImageUrlsOf_Empty(t *testing.T) {
	if got := ImageUrlsOf(nil); len(got) != 0 {
		t.Errorf("空段应返回空列表, got %v", got)
	}
}
