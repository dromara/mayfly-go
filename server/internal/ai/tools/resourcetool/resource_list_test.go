package resourcetool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"mayfly-go/internal/ai/application/resource"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/model"
)

// fakeProvider 测试用资源提供者
type fakeProvider struct {
	typ  string
	list []*resource.Resource
	err  error
}

func (p *fakeProvider) Type() string { return p.typ }

func (p *fakeProvider) List(ctx context.Context, accountId uint64) ([]*resource.Resource, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.list, nil
}

// newTestApp 构建含机器+数据库资源的测试 App
func newTestApp() resource.App {
	app := resource.NewApp()
	app.RegisterProvider(&fakeProvider{typ: resource.TypeMachine, list: []*resource.Resource{
		{
			Type:        resource.TypeMachine,
			Id:          "1",
			Code:        "ali",
			Name:        "aliyun",
			Description: "1.1.1.1:22",
			Detail:      map[string]string{"ip": "1.1.1.1", "port": "22"},
		},
		{
			Type:        resource.TypeMachine,
			Id:          "2",
			Code:        "web",
			Name:        "web-server",
			Description: "10.0.0.2:22",
		},
	}})
	app.RegisterProvider(&fakeProvider{typ: resource.TypeDb, list: []*resource.Resource{
		{
			Type:        resource.TypeDb,
			Id:          "3",
			Code:        "local",
			Name:        "本地库",
			Description: "local",
			Detail:      map[string]string{"databases": "mayfly_go sys"},
		},
	}})
	return app
}

func withTestApp(t *testing.T) {
	t.Helper()
	SetAppLoader(func() resource.App { return newTestApp() })
	t.Cleanup(func() { SetAppLoader(resource.GetApp) })
}

func invokeTool(t *testing.T, args string) (string, error) {
	t.Helper()
	tool, err := GetResourceList()
	if err != nil {
		t.Fatalf("create ListResources tool failed: %v", err)
	}
	ctx := contextx.WithLoginAccount(context.Background(), &model.LoginAccount{Id: 1})
	return tool.InvokableRun(ctx, args)
}

// TestListResources_AllTypes 空条件返回全部类型资源（含 Detail 透传）
func TestListResources_AllTypes(t *testing.T) {
	withTestApp(t)

	raw, err := invokeTool(t, `{}`)
	if err != nil {
		t.Fatalf("invoke failed: %v", err)
	}
	var out ListResourcesOutput
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("unmarshal output failed: %v, raw=%s", err, raw)
	}
	if out.Total != 3 || len(out.Resources) != 3 || out.Truncated {
		t.Fatalf("expected 3 resources no truncation, got %+v", out)
	}
	// 输出顺序取决于类型遍历，按类型定位断言
	machineItem := findResourceItem(t, out, "1")
	if machineItem.ResourceType != resource.TypeMachine || machineItem.Detail["ip"] != "1.1.1.1" {
		t.Fatalf("unexpected machine item: %+v", machineItem)
	}
}

// findResourceItem 按 id 定位输出条目
func findResourceItem(t *testing.T, out ListResourcesOutput, id string) ResourceItem {
	t.Helper()
	for _, item := range out.Resources {
		if item.Id == id {
			return item
		}
	}
	t.Fatalf("resource %s not found in output: %+v", id, out)
	return ResourceItem{}
}

// TestListResources_TypeFilter 类型过滤
func TestListResources_TypeFilter(t *testing.T) {
	withTestApp(t)

	raw, err := invokeTool(t, `{"resourceType":"db"}`)
	if err != nil {
		t.Fatalf("invoke failed: %v", err)
	}
	var out ListResourcesOutput
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if out.Total != 1 || len(out.Resources) != 1 || out.Resources[0].Id != "3" {
		t.Fatalf("expected only db resource id=3, got %+v", out)
	}
}

// TestListResources_UnknownType 未知类型返回空列表而非报错
func TestListResources_UnknownType(t *testing.T) {
	withTestApp(t)

	raw, err := invokeTool(t, `{"resourceType":"redis"}`)
	if err != nil {
		t.Fatalf("invoke failed: %v", err)
	}
	var out ListResourcesOutput
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if out.Total != 0 || len(out.Resources) != 0 {
		t.Fatalf("expected empty result for unknown type, got %+v", out)
	}
}

// TestListResources_Keyword 关键词过滤
func TestListResources_Keyword(t *testing.T) {
	withTestApp(t)

	raw, err := invokeTool(t, `{"keyword":"1.1.1.1"}`)
	if err != nil {
		t.Fatalf("invoke failed: %v", err)
	}
	var out ListResourcesOutput
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if out.Total != 1 || len(out.Resources) != 1 || out.Resources[0].Id != "1" {
		t.Fatalf("expected only machine id=1 by ip keyword, got %+v", out)
	}
}

// TestListResources_Truncate 超上限截断并保留完整 Total 标记
func TestListResources_Truncate(t *testing.T) {
	SetAppLoader(func() resource.App {
		app := resource.NewApp()
		list := make([]*resource.Resource, 0, maxOutputResources+10)
		for i := 0; i < maxOutputResources+10; i++ {
			list = append(list, &resource.Resource{
				Type: resource.TypeMachine,
				Id:   fmt.Sprintf("%d", i),
				Name: fmt.Sprintf("m-%d", i),
			})
		}
		app.RegisterProvider(&fakeProvider{typ: resource.TypeMachine, list: list})
		return app
	})
	t.Cleanup(func() { SetAppLoader(resource.GetApp) })

	raw, err := invokeTool(t, `{}`)
	if err != nil {
		t.Fatalf("invoke failed: %v", err)
	}
	var out ListResourcesOutput
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if !out.Truncated || out.Total != maxOutputResources+10 || len(out.Resources) != maxOutputResources {
		t.Fatalf("expected truncation (total=%d, items=%d, truncated=%v)",
			out.Total, len(out.Resources), out.Truncated)
	}
}

// TestListResources_ProviderError provider 失败转为工具错误
func TestListResources_ProviderError(t *testing.T) {
	SetAppLoader(func() resource.App {
		app := resource.NewApp()
		app.RegisterProvider(&fakeProvider{typ: resource.TypeMachine, err: errors.New("db down")})
		return app
	})
	t.Cleanup(func() { SetAppLoader(resource.GetApp) })

	_, err := invokeTool(t, `{}`)
	if err == nil || !strings.Contains(err.Error(), "db down") {
		t.Fatalf("expected tool error wrapping provider error, got %v", err)
	}
}

// TestListResources_NoLoginAccount 上下文无登录账号时报错
func TestListResources_NoLoginAccount(t *testing.T) {
	withTestApp(t)

	tool, err := GetResourceList()
	if err != nil {
		t.Fatalf("create ListResources tool failed: %v", err)
	}
	if _, err := tool.InvokableRun(context.Background(), `{}`); err == nil {
		t.Fatalf("expected error when no login account in context")
	}
}

// TestListResources_AppNotInitialized 资源服务未装配时返回可重试工具错误
func TestListResources_AppNotInitialized(t *testing.T) {
	SetAppLoader(func() resource.App { return nil })
	t.Cleanup(func() { SetAppLoader(resource.GetApp) })

	_, err := invokeTool(t, `{}`)
	if err == nil {
		t.Fatalf("expected error when resource app not initialized")
	}
}
