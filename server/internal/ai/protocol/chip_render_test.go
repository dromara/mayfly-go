package protocol

import (
	"strings"
	"testing"
)

func TestRenderSegments_PlainText_NoFocusDirective(t *testing.T) {
	got := RenderSegments([]ContentSegment{NewInputTextSegment("你好")})
	if got != "你好" {
		t.Errorf("纯文本段应原样透出且无聚焦指令, got %q", got)
	}
}

func TestRenderSegments_ResourceChip_FullIdentity(t *testing.T) {
	got := RenderSegments([]ContentSegment{
		{Type: ContentSegmentResource, Text: "aliyun", Extra: map[string]any{
			"resourceType": "machine", "id": "1", "code": "c1", "ip": "10.0.0.1", "port": "22",
			"authCertName": "cert1", "username": "root",
		}},
		NewInputTextSegment(" 查进程"),
	})
	for _, want := range []string{"[引用资源] 机器: aliyun", "id=1, code=c1, ip=10.0.0.1, port=22", "authCertName=cert1, username=root", "查进程", "唯一目标"} {
		if !strings.Contains(got, want) {
			t.Errorf("渲染结果缺少 %q, got:\n%s", want, got)
		}
	}
}

func TestRenderSegments_DbChip_NoCert(t *testing.T) {
	got := RenderSegments([]ContentSegment{
		{Type: ContentSegmentResource, Text: "orders-db", Extra: map[string]any{
			"resourceType": "db", "id": "2", "code": "d1", "db": "orders",
		}},
	})
	for _, want := range []string{"[引用资源] 数据库: orders-db", "id=2, code=d1, db=orders"} {
		if !strings.Contains(got, want) {
			t.Errorf("渲染结果缺少 %q, got:\n%s", want, got)
		}
	}
	if strings.Contains(got, "authCertName") {
		t.Errorf("未选凭证时不应注入 authCertName, got:\n%s", got)
	}
}

func TestRenderSegments_SkillChip(t *testing.T) {
	got := RenderSegments([]ContentSegment{{Type: ContentSegmentSkill, Text: "db-ops-guide"}})
	if !strings.Contains(got, "[引用技能] db-ops-guide") {
		t.Errorf("技能段渲染缺失, got:\n%s", got)
	}
}

func TestRenderSegments_UnregisteredResourceType_FallbackId(t *testing.T) {
	got := RenderSegments([]ContentSegment{
		{Type: ContentSegmentResource, Text: "cache-1", Extra: map[string]any{"resourceType": "redis", "id": "9"}},
	})
	if !strings.Contains(got, "[引用资源] redis: cache-1 (id=9)") {
		t.Errorf("未注册资源类型应回落 id 标识, got:\n%s", got)
	}
}

// 开闭原则承诺：新增芯片/资源类型经注册接入，RenderSegments 零修改
func TestRegisterChipRenderer_ExtensionPoint(t *testing.T) {
	const segType = "test_doc"
	RegisterChipRenderer(segType, func(seg ContentSegment) string {
		return "[引用文档] " + seg.Text + "\n"
	})
	t.Cleanup(func() { delete(chipRenderers, segType) })

	got := RenderSegments([]ContentSegment{{Type: segType, Text: "spec.pdf"}})
	if !strings.Contains(got, "[引用文档] spec.pdf") {
		t.Errorf("注册渲染器未生效, got:\n%s", got)
	}
	if !strings.Contains(got, "唯一目标") {
		t.Errorf("自定义芯片段应触发聚焦指令, got:\n%s", got)
	}
}

func TestRegisterResourceIdentityRenderer_OverrideBuiltin(t *testing.T) {
	RegisterResourceIdentityRenderer("machine", func(map[string]any) (string, string) {
		return "主机", "id=custom"
	})
	t.Cleanup(func() {
		RegisterResourceIdentityRenderer("machine", renderMachineIdentity)
	})

	got := RenderSegments([]ContentSegment{
		{Type: ContentSegmentResource, Text: "h1", Extra: map[string]any{"resourceType": "machine", "id": "1"}},
	})
	if !strings.Contains(got, "[引用资源] 主机: h1 (id=custom)") {
		t.Errorf("后注册渲染器应覆盖内置, got:\n%s", got)
	}
}

func TestRenderSegments_Empty(t *testing.T) {
	if got := RenderSegments(nil); got != "" {
		t.Errorf("空段应返回空串, got %q", got)
	}
}
