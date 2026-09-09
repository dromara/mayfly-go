package resource

import (
	"context"
	"errors"
	"testing"
)

// fakeProvider 测试用资源提供者
type fakeProvider struct {
	typ       string
	list      []*Resource
	err       error
	listCalls int
}

func (p *fakeProvider) Type() string { return p.typ }

func (p *fakeProvider) List(ctx context.Context, accountId uint64) ([]*Resource, error) {
	p.listCalls++
	if p.err != nil {
		return nil, p.err
	}
	return p.list, nil
}

func newRes(typ, id, name, code, desc string) *Resource {
	return &Resource{Type: typ, Id: id, Name: name, Code: code, Description: desc}
}

// TestApp_ListAll 空条件查询返回全部已注册类型的资源
func TestApp_ListAll(t *testing.T) {
	app := NewApp()
	m := &fakeProvider{typ: TypeMachine, list: []*Resource{newRes(TypeMachine, "1", "aliyun", "ali", "1.1.1.1:22")}}
	d := &fakeProvider{typ: TypeDb, list: []*Resource{newRes(TypeDb, "2", "本地库", "local", "local")}}
	app.RegisterProvider(m)
	app.RegisterProvider(d)

	res, err := app.List(context.Background(), 1, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(res))
	}
	if m.listCalls != 1 || d.listCalls != 1 {
		t.Fatalf("each provider should be called once, machine=%d db=%d", m.listCalls, d.listCalls)
	}
}

// TestApp_TypesFilter 类型过滤：仅调用指定类型的 provider
func TestApp_TypesFilter(t *testing.T) {
	app := NewApp()
	m := &fakeProvider{typ: TypeMachine, list: []*Resource{newRes(TypeMachine, "1", "aliyun", "ali", "")}}
	d := &fakeProvider{typ: TypeDb, list: []*Resource{newRes(TypeDb, "2", "本地库", "local", "")}}
	app.RegisterProvider(m)
	app.RegisterProvider(d)

	res, err := app.List(context.Background(), 1, &Query{Types: []string{TypeDb}})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(res) != 1 || res[0].Type != TypeDb {
		t.Fatalf("expected only db resource, got %v", res)
	}
	if m.listCalls != 0 {
		t.Fatalf("machine provider should not be called, calls=%d", m.listCalls)
	}
}

// TestApp_UnknownTypeSkipped 未注册类型跳过且不报错
func TestApp_UnknownTypeSkipped(t *testing.T) {
	app := NewApp()
	app.RegisterProvider(&fakeProvider{typ: TypeMachine, list: []*Resource{newRes(TypeMachine, "1", "m", "m", "")}})

	res, err := app.List(context.Background(), 1, &Query{Types: []string{TypeMachine, "redis"}})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 machine resource, got %d", len(res))
	}
}

// TestApp_KeywordFilter 关键词对 Name/Code/Description 大小写不敏感匹配
func TestApp_KeywordFilter(t *testing.T) {
	app := NewApp()
	app.RegisterProvider(&fakeProvider{typ: TypeMachine, list: []*Resource{
		newRes(TypeMachine, "1", "web-server", "web", "10.0.0.1:22"),
		newRes(TypeMachine, "2", "db-server", "db", "10.0.0.2:22"),
	}})
	app.RegisterProvider(&fakeProvider{typ: TypeDb, list: []*Resource{
		newRes(TypeDb, "3", "sampledb", "sampledb", "sampledb"),
	}})

	cases := []struct {
		keyword string
		wantIds []string
	}{
		{"web", []string{"1"}},
		{"WEB", []string{"1"}},
		{"10.0.0.2", []string{"2"}}, // description（ip:port）命中
		{"sampledb", []string{"3"}}, // 名称与编码同时命中不去重（同一条资源）
		{"不存在的关键词", []string{}},
		{"  ", []string{"1", "2", "3"}}, // 空白关键词视为不过滤
	}
	for _, c := range cases {
		res, err := app.List(context.Background(), 1, &Query{Keyword: c.keyword})
		if err != nil {
			t.Fatalf("List(keyword=%q) failed: %v", c.keyword, err)
		}
		ids := make(map[string]struct{}, len(res))
		for _, r := range res {
			ids[r.Id] = struct{}{}
		}
		if len(ids) != len(c.wantIds) {
			t.Fatalf("keyword=%q expected ids %v, got %v", c.keyword, c.wantIds, ids)
		}
		for _, id := range c.wantIds {
			if _, ok := ids[id]; !ok {
				t.Fatalf("keyword=%q expected ids %v, got %v", c.keyword, c.wantIds, ids)
			}
		}
	}
}

// TestApp_RegisterProvider_Override 同类型后注册覆盖先注册
func TestApp_RegisterProvider_Override(t *testing.T) {
	app := NewApp()
	first := &fakeProvider{typ: TypeMachine, list: []*Resource{newRes(TypeMachine, "1", "old", "old", "")}}
	second := &fakeProvider{typ: TypeMachine, list: []*Resource{newRes(TypeMachine, "2", "new", "new", "")}}
	app.RegisterProvider(first)
	app.RegisterProvider(second)

	res, err := app.List(context.Background(), 1, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(res) != 1 || res[0].Id != "2" {
		t.Fatalf("second provider should override first, got %v", res)
	}
	if first.listCalls != 0 {
		t.Fatalf("first provider should not be called, calls=%d", first.listCalls)
	}
}

// TestApp_RegisterProvider_Invalid nil 或无类型标识的 provider 被忽略
func TestApp_RegisterProvider_Invalid(t *testing.T) {
	app := NewApp()
	app.RegisterProvider(nil)
	app.RegisterProvider(&fakeProvider{typ: ""})
	if types := app.Types(); len(types) != 0 {
		t.Fatalf("expected no registered types, got %v", types)
	}
}

// TestApp_ProviderError_Propagate provider 失败时整体失败（保证结果完整性）
func TestApp_ProviderError_Propagate(t *testing.T) {
	app := NewApp()
	app.RegisterProvider(&fakeProvider{typ: TypeMachine, list: []*Resource{newRes(TypeMachine, "1", "m", "m", "")}})
	wantErr := errors.New("db query failed")
	app.RegisterProvider(&fakeProvider{typ: TypeDb, err: wantErr})

	if _, err := app.List(context.Background(), 1, nil); !errors.Is(err, wantErr) {
		t.Fatalf("expected provider error to propagate, got %v", err)
	}
}

// TestApp_Types_StableSort Types 输出稳定排序
func TestApp_Types_StableSort(t *testing.T) {
	app := NewApp()
	app.RegisterProvider(&fakeProvider{typ: TypeDb})
	app.RegisterProvider(&fakeProvider{typ: TypeMachine})

	for i := 0; i < 3; i++ {
		types := app.Types()
		if len(types) != 2 || types[0] != TypeDb || types[1] != TypeMachine {
			t.Fatalf("expected [db machine], got %v", types)
		}
	}
}

// TestGetApp_DefaultNil 未装配时 GetApp 返回 nil（fail-open 判空约定）
func TestGetApp_DefaultNil(t *testing.T) {
	old := GetApp()
	defer SetDefault(old)

	SetDefault(nil)
	if GetApp() != nil {
		t.Fatalf("expected nil app after SetDefault(nil)")
	}
}
