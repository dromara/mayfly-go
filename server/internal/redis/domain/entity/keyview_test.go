package entity

import (
	"encoding/json"
	"testing"
)

// TestMemberUnmarshalJSONNumberForms 成员行的数字字段必须同时接受数字字面量与字符串形态：
// 前端用 json-bigint(storeAsString) 解析响应，16 位以上整数（geo 的 geohash 分值）会被转成
// 字符串，成员行原样回传做改/删时若只认数字，整个请求会反序列化失败并报 500
func TestMemberUnmarshalJSONNumberForms(t *testing.T) {
	tests := []struct {
		name  string
		raw   string
		index int64
		score float64
	}{
		{"数字字面量", `{"value":"one","score":1.5}`, 0, 1.5},
		{"大分值字符串（geo/geohash）", `{"value":"hangzhou","score":"4054134072858107"}`, 0, 4.054134072858107e15},
		{"下标字符串", `{"index":"100","value":"bit"}`, 100, 0},
		{"负分值字符串", `{"score":"-2.5"}`, 0, -2.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			member := &Member{}
			if err := json.Unmarshal([]byte(tt.raw), member); err != nil {
				t.Fatalf("unmarshal %s: %v", tt.raw, err)
			}
			if member.Index != tt.index {
				t.Errorf("index = %d, want %d", member.Index, tt.index)
			}
			if member.Score != tt.score {
				t.Errorf("score = %v, want %v", member.Score, tt.score)
			}
		})
	}
}

// TestMemberJSONRoundTrip 序列化仍输出数字，且序列化-反序列化往返保持等价（描述符与读取接口共用该结构）
func TestMemberJSONRoundTrip(t *testing.T) {
	member := &Member{Index: 3, Field: "f", Value: "v", Score: 4054134072858107, Id: "1-0", Extra: map[string]string{"byte": "48"}}

	data, err := json.Marshal(member)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	restored := &Member{}
	if err := json.Unmarshal(data, restored); err != nil {
		t.Fatalf("unmarshal %s error: %v", data, err)
	}
	if restored.Score != member.Score || restored.Index != member.Index || restored.Field != member.Field ||
		restored.Value != member.Value || restored.Id != member.Id || restored.Extra["byte"] != "48" {
		t.Errorf("round trip mismatch: got %+v, want %+v", restored, member)
	}
}

// TestMemberUnmarshalInvalidJSON 非法 JSON 必须返回错误而不是静默吞掉（写入口靠它挡住畸形请求）
func TestMemberUnmarshalInvalidJSON(t *testing.T) {
	member := &Member{}
	if err := json.Unmarshal([]byte(`{"score":`), member); err == nil {
		t.Error("expected error for malformed json, got nil")
	}
}
