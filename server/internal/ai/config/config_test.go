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
