package migrations

import (
	"encoding/json"
	"testing"
)

func mustUnmarshalSchema(t *testing.T, params string) jsonFormSchema {
	t.Helper()
	var schema jsonFormSchema
	if err := json.Unmarshal([]byte(params), &schema); err != nil {
		t.Fatalf("转换结果非法 JSON Schema: %v, params: %s", err, params)
	}
	return schema
}

func TestConvertLegacyFormParams(t *testing.T) {
	t.Run("基础字段映射", func(t *testing.T) {
		legacy := `[{"model":"host","name":"machine.host","placeholder":"machine.hostPlaceholder","required":true}]`
		converted, err := convertLegacyFormParams(legacy)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		schema := mustUnmarshalSchema(t, converted)
		if schema.Version != 1 {
			t.Fatalf("version should be 1, got %d", schema.Version)
		}
		if len(schema.Fields) != 1 {
			t.Fatalf("fields length should be 1, got %d", len(schema.Fields))
		}
		field := schema.Fields[0]
		if field.Prop != "host" || field.Label != "machine.host" || field.Placeholder != "machine.hostPlaceholder" {
			t.Fatalf("field mapping error: %+v", field)
		}
		if field.Rules == nil || !field.Rules.Required {
			t.Fatalf("required should be lifted to rules.required: %+v", field)
		}
	})

	t.Run("options 逗号分隔转静态选项", func(t *testing.T) {
		legacy := `[{"model":"env","name":"env","options":"dev, prod ,"}]`
		converted, err := convertLegacyFormParams(legacy)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		schema := mustUnmarshalSchema(t, converted)
		if len(schema.Fields[0].Options) != 2 {
			t.Fatalf("options length should be 2, got %d", len(schema.Fields[0].Options))
		}
		if schema.Fields[0].Options[0].Value != "dev" || schema.Fields[0].Options[1].Label != "prod" {
			t.Fatalf("options mapping error: %+v", schema.Fields[0].Options)
		}
	})

	t.Run("空 params 与空数组", func(t *testing.T) {
		if converted, err := convertLegacyFormParams(""); err != nil || converted != "" {
			t.Fatalf("empty params should skip, got %q, err %v", converted, err)
		}
		// 空数组升级为空 v1 Schema，避免前端再兼容旧数组格式
		converted, err := convertLegacyFormParams("[]")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		schema := mustUnmarshalSchema(t, converted)
		if schema.Version != 1 || len(schema.Fields) != 0 {
			t.Fatalf("empty array should convert to empty v1 schema, got %+v", schema)
		}
	})

	t.Run("非 JSON 与非数组数据不处理", func(t *testing.T) {
		for _, params := range []string{"not-json", `{"key":"value"}`, "123", `"str"`} {
			converted, err := convertLegacyFormParams(params)
			if err != nil {
				t.Fatalf("params %q unexpected err: %v", params, err)
			}
			if converted != "" {
				t.Fatalf("params %q should skip, got %q", params, converted)
			}
		}
	})

	t.Run("已是 v1 Schema 时幂等跳过", func(t *testing.T) {
		v1 := `{"version":1,"fields":[{"prop":"host"}]}`
		converted, err := convertLegacyFormParams(v1)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if converted != "" {
			t.Fatalf("v1 schema should skip, got %q", converted)
		}
	})

	t.Run("多余字段不进入 v1 结果", func(t *testing.T) {
		legacy := `[{"model":"host","name":"主机","unknownField":"x"}]`
		converted, err := convertLegacyFormParams(legacy)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		schema := mustUnmarshalSchema(t, converted)
		if len(schema.Fields) != 1 || schema.Fields[0].Prop != "host" || schema.Fields[0].Label != "主机" {
			t.Fatalf("field mapping error: %+v", schema.Fields)
		}
	})
}
