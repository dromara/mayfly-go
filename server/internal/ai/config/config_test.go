package config

import "testing"

// TestDisabledExtensionSet 禁用清单转集合（WithFilter 参数形态）
func TestDisabledExtensionSet(t *testing.T) {
	// 空清单返回 nil（WithFilter 原样返回注册中心）
	empty := &AgentConfig{}
	if got := empty.DisabledExtensionSet(); got != nil {
		t.Errorf("empty list should return nil, got %v", got)
	}

	conf := &AgentConfig{DisabledExtensions: []string{"db_tools", "memory", "db_tools"}}
	set := conf.DisabledExtensionSet()
	if len(set) != 2 {
		t.Errorf("expected 2 unique ids, got %v", set)
	}
	if _, ok := set["db_tools"]; !ok {
		t.Error("db_tools should be in set")
	}
	if _, ok := set["memory"]; !ok {
		t.Error("memory should be in set")
	}
}

func TestParseModelFailoverConfigInvalid(t *testing.T) {
	// 结构非法：返回 nil 不阻断装配
	if got := parseModelFailoverConfig("bad"); got != nil {
		t.Fatal("non-map raw should return nil")
	}
	if got := parseModelFailoverConfig(map[string]any{}); got != nil {
		t.Fatal("empty map should return nil (no fallbacks)")
	}
	// fallbacks 结构非法
	if got := parseModelFailoverConfig(map[string]any{"fallbacks": "bad"}); got != nil {
		t.Fatal("invalid fallbacks should return nil")
	}
	// 条目均缺 model 字段：全部跳过后返回 nil
	if got := parseModelFailoverConfig(map[string]any{
		"fallbacks": []any{map[string]any{"baseUrl": "https://no-model"}},
	}); got != nil {
		t.Fatal("all entries missing model should return nil")
	}
}

func TestParseModelFailoverConfig(t *testing.T) {
	fc := parseModelFailoverConfig(map[string]any{
		"maxFailovers": float64(2),
		"fallbacks": []any{
			map[string]any{
				"name":    "backup-1",
				"model":   "openai/gpt-4o-mini",
				"baseUrl": "https://api.example.com/v1",
				"apiKey":  "sk-test",
			},
			// 缺 model 字段：跳过
			map[string]any{"baseUrl": "https://invalid"},
			map[string]any{"model": "openai/deepseek-v4"},
		},
	})
	if fc == nil {
		t.Fatal("expected parsed failover config")
	}
	if fc.MaxFailovers != 2 {
		t.Fatalf("MaxFailovers = %d, want 2", fc.MaxFailovers)
	}
	if len(fc.Fallbacks) != 2 {
		t.Fatalf("len(Fallbacks) = %d, want 2 (invalid entry skipped)", len(fc.Fallbacks))
	}
	fb := fc.Fallbacks[0]
	if fb.Model != "openai/gpt-4o-mini" || fb.BaseUrl != "https://api.example.com/v1" || fb.ApiKey != "sk-test" {
		t.Fatalf("unexpected fallback fields: %+v", fb)
	}
	// 默认值对齐 GetModel 主配置语义
	if fb.TimeOut != 60 || fb.Temperature != 0.7 || fb.MaxTokens != DefaultMaxTokens {
		t.Fatalf("unexpected fallback defaults: %+v", fb)
	}
	if fc.Fallbacks[1].Model != "openai/deepseek-v4" {
		t.Fatalf("unexpected second fallback: %+v", fc.Fallbacks[1])
	}
}
