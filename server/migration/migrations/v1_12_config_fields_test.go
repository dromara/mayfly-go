package migrations

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// aiModelFields 取自 v1.12.0-ai-model-config-params 这一步的真实入参：
// enableThinking 刻意不带 placeholder（开关型字段只给了 options），这正是会把迁移跑崩的形态
func aiModelFields() []map[string]any {
	return []map[string]any{
		{"model": "maxTokens", "name": "system.sysconf.aiMaxTokens", "placeholder": "system.sysconf.aiMaxTokensPlaceholder", "required": false},
		{"model": "enableThinking", "name": "system.sysconf.aiEnableThinking", "options": "true,false", "required": false},
	}
}

func schemaFieldProps(t *testing.T, params string) []string {
	t.Helper()
	var doc map[string]any
	require.NoError(t, json.Unmarshal([]byte(params), &doc))
	fields, ok := doc["fields"].([]any)
	require.True(t, ok, "fields 不是数组：%s", params)
	props := make([]string, 0, len(fields))
	for _, entry := range fields {
		item, ok := entry.(map[string]any)
		require.True(t, ok, "字段条目不是对象：%v", entry)
		prop, _ := item["prop"].(string)
		props = append(props, prop)
	}
	return props
}

// TestAppendConfigFieldsKeepsLegacyShape 旧版裸数组形态：追加缺失字段、已存在则不重复追加（幂等）
func TestAppendConfigFieldsKeepsLegacyShape(t *testing.T) {
	legacy := `[{"model":"provider","name":"system.sysconf.aiProvider","placeholder":"p","required":true}]`

	updated, changed, err := appendConfigFields(legacy, aiModelFields())
	require.NoError(t, err)
	assert.True(t, changed, "缺失字段必须被追加")

	var items []map[string]any
	require.NoError(t, json.Unmarshal([]byte(updated), &items))
	assert.Len(t, items, 3, "原有 1 条 + 新增 2 条")
	assert.Equal(t, "provider", items[0]["model"], "原有条目按原键名原样保留，不改写成 v1 形态")

	// 重跑：一条都不该再加
	again, changed, err := appendConfigFields(updated, aiModelFields())
	require.NoError(t, err)
	assert.False(t, changed, "已存在的字段不得重复追加")
	assert.Equal(t, updated, again, "无需变更时原文返回")
}

// TestAppendConfigFieldsOnUpgradedSchema 配置已升级为 v1 JSON Schema 时的追加。
//
// 这条路径此前无任何用例：它一碰到不带 placeholder 的字段定义就 panic（迁移 panic 等于服务起不来），
// 且按结构体反序列化再写回会把 cols 这类布局键静默丢掉，用户改过的表单布局当场消失
func TestAppendConfigFieldsOnUpgradedSchema(t *testing.T) {
	upgraded := `{"version":1,"cols":2,"fields":[{"prop":"provider","label":"system.sysconf.aiProvider","placeholder":"p","rules":{"required":true}}]}`

	updated, changed, err := appendConfigFields(upgraded, aiModelFields())
	require.NoError(t, err, "字段定义缺 placeholder 不得让迁移 panic")
	assert.True(t, changed)

	// 追加顺序跟随入参顺序（不是按列名排序）：这一步迁移先补 maxTokens 再补 enableThinking
	assert.Equal(t, []string{"provider", "maxTokens", "enableThinking"}, schemaFieldProps(t, updated))

	var doc map[string]any
	require.NoError(t, json.Unmarshal([]byte(updated), &doc))
	assert.Equal(t, float64(2), doc["cols"], "布局键 cols 必须在写回后仍然存在")
	assert.Len(t, doc, 3, "除 fields 外的键一律原样保留，不得在写回时被丢键")

	// enableThinking 的 options 由逗号串展开成下拉项，且不写空的 placeholder
	fields := doc["fields"].([]any)
	var thinking map[string]any
	for _, entry := range fields {
		item := entry.(map[string]any)
		if item["prop"] == "enableThinking" {
			thinking = item
		}
	}
	require.NotNil(t, thinking)
	assert.NotContains(t, thinking, "placeholder", "空值不写键，与 convertLegacyFormParams 的 omitempty 口径一致")
	options, ok := thinking["options"].([]any)
	require.True(t, ok)
	require.Len(t, options, 2)
	assert.Equal(t, map[string]any{"value": "true", "label": "true"}, options[0])

	// 重跑同一份入参：幂等，原文返回
	again, changed, err := appendConfigFields(updated, aiModelFields())
	require.NoError(t, err)
	assert.False(t, changed, "已存在的 prop 不得重复追加")
	assert.Equal(t, updated, again)
}

// TestAppendConfigFieldsRejectsUnknownShape version 非 1 的对象形态必须报错而不是静默跳过：
// 静默跳过会让「这一步迁移加了字段」在界面上毫无痕迹，而配置其实一条没动
func TestAppendConfigFieldsRejectsUnknownShape(t *testing.T) {
	_, _, err := appendConfigFields(`{"version":2,"fields":[]}`, aiModelFields())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version")

	// 非 JSON 内容同样报错：不猜形态，也不把坏数据写回
	_, _, err = appendConfigFields("not json", aiModelFields())
	require.Error(t, err)
}
