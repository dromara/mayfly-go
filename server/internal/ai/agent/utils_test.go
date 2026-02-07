package agent

import (
	"testing"
)

// TestParseLLMJSON 测试 ParseLLMJSON 函数
func TestParseLLMJSON(t *testing.T) {
	// 定义测试用例结构体
	tests := []struct {
		name     string
		input    string
		expected any
		hasError bool
	}{
		{
			name: "Valid JSON Object",
			input: "```json\n{\n  \"name\": \"Alice\",\n  \"age\": \"30\"\n}\n```",
			expected: map[string]any{
				"name": "Alice",
				"age":  "30",
			},
			hasError: false,
		},
		{
			name: "Valid JSON Object",
			input: "```\n{\n  \"name\": \"Alice\",\n  \"age\": \"40\"\n}\n```",
			expected: map[string]any{
				"name": "Alice",
				"age":  "40",
			},
			hasError: false,
		},
		{
			name: "Valid JSON Object",
			input: "aaabbbccc```\n{\n  \"name\": \"Alice\",\n  \"age\": \"50\"\n}\n```dddd",
			expected: map[string]any{
				"name": "Alice",
				"age":  "50",
			},
			hasError: false,
		},
		{
			name: "Valid JSON Array",
			input: "```json\n[\n  {\"id\": \"1\", \"value\": \"foo\"},\n  {\"id\": \"2\", \"value\": \"bar\"}\n]\n```",
			expected: []map[string]any{
				{"id": "1", "value": "foo"},
				{"id": "2", "value": "bar"},
			},
			hasError: false,
		},
		{
			name: "Valid JSON Array",
			input: "aaaa```json\n[\n  {\"id\": \"11\", \"value\": \"foo\"},\n  {\"id\": \"22\", \"value\": \"bar\"}\n]\n```",
			expected: []map[string]any{
				{"id": "11", "value": "foo"},
				{"id": "22", "value": "bar"},
			},
			hasError: false,
		},
		{
			name:     "Invalid JSON Format",
			input:    "This is not a valid JSON",
			expected: nil,
			hasError: true,
		},
		{
			name:     "Empty Input",
			input:    "",
			expected: nil,
			hasError: true,
		},
	}

	// 执行测试用例
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result any
			var err error

			// 根据 expected 类型调用不同的 ParseLLMJSON 方法
			switch tt.expected.(type) {
			case map[string]any:
				result, err = ParseLLMJSON[map[string]any](tt.input)
			case []map[string]any:
				result, err = ParseLLMJSON[[]map[string]any](tt.input)
			default:
				result, err = ParseLLMJSON[any](tt.input)
			}

			// 验证错误情况
			if tt.hasError {
				if err == nil {
					t.Errorf("expected an error but got none")
				}
				return
			}

			// 验证无错误情况下的结果
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			t.Logf("%v", result)
		})
	}
}