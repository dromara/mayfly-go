package migrationx

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	sysentity "mayfly-go/internal/sys/domain/entity"

	"gorm.io/gorm"
)

// AppendConfigFields 向系统配置 params 追加新字段定义（按 model/prop 去重）。
// params 有两种形态：旧版裸数组与 v1 JSON Schema（form-params 迁移升级后的形态），
// 追加后按原形态写回，不能把已升级的配置悄悄降级回裸数组。
// afterUpdate 在 params 更新后执行（如清缓存、补默认值），无需后置操作时传 nil。
func AppendConfigFields(tx *gorm.DB, configKey string, fields []map[string]any, afterUpdate func(*gorm.DB, *sysentity.Config) error) error {
	config := &sysentity.Config{}
	if err := tx.Where("`key` = ?", configKey).First(config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // 无该配置行则跳过
		}
		return err
	}
	params, changed, err := appendConfigFields(config.Params, fields)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	if err := tx.Model(config).Update("params", params).Error; err != nil {
		return err
	}
	if afterUpdate != nil {
		return afterUpdate(tx, config)
	}
	return nil
}

// appendConfigFields 按原有形态向 params 追加字段定义，返回追加后的 JSON 与「是否真的追加过」。
//
// v1 schema 形态按 map 处理而不是反序列化成结构体再写回：结构体只认识 version/fields 两个键，
// 配置里其余的键（如表单布局 cols）会在写回时被静默丢掉，用户调过的布局当场消失
func appendConfigFields(params string, fields []map[string]any) (string, bool, error) {
	// v1 JSON Schema 形态：裸数组 unmarshal 到 map 必报错，据此区分两种形态
	var doc map[string]any
	if err := json.Unmarshal([]byte(params), &doc); err == nil {
		if version, _ := doc["version"].(float64); version != 1 {
			return "", false, fmt.Errorf("unknown config form params shape, expect version 1 but got %v", doc["version"])
		}
		merged, changed := mergeSchemaFields(doc["fields"], fields)
		if !changed {
			return params, false, nil
		}
		doc["fields"] = merged
		updated, err := json.Marshal(doc)
		if err != nil {
			return "", false, err
		}
		return string(updated), true, nil
	}
	// 旧版裸数组形态
	var items []map[string]any
	if err := json.Unmarshal([]byte(params), &items); err != nil {
		return "", false, err
	}
	changed := false
	for _, add := range fields {
		exists := false
		for _, it := range items {
			if it["model"] == add["model"] {
				exists = true
				break
			}
		}
		if !exists {
			items = append(items, add)
			changed = true
		}
	}
	if !changed {
		return params, false, nil
	}
	updated, err := json.Marshal(items)
	if err != nil {
		return "", false, err
	}
	return string(updated), true, nil
}

// mergeSchemaFields 把 legacy 字段定义追加进 v1 schema 的 fields 数组，已有同 prop 的字段则跳过
func mergeSchemaFields(raw any, fields []map[string]any) ([]any, bool) {
	existing, _ := raw.([]any)
	changed := false
	for _, add := range fields {
		prop, _ := add["model"].(string)
		if prop == "" || schemaHasField(existing, prop) {
			continue
		}
		existing = append(existing, legacyFieldToFormField(add))
		changed = true
	}
	return existing, changed
}

// schemaHasField 判断 v1 schema 里是否已有该 prop 的字段（条目形态按 JSON 读，不依赖结构体）
func schemaHasField(fields []any, prop string) bool {
	for _, entry := range fields {
		item, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if itemProp, _ := item["prop"].(string); itemProp == prop {
			return true
		}
	}
	return false
}

// legacyFieldToFormField 把 legacy 字段定义 map（model/name/placeholder/options/required）
// 转成 v1 schema 字段的 JSON 形态，映射规则与 convertLegacyFormParams 一致（空值一律不写键，
// 对应结构体上的 omitempty）。
//
// 取键统一用 comma-ok：字段定义里 placeholder/required 都是可选的（如开关型字段只带 options），
// 直接断言会 panic，而迁移 panic 等于服务启动即退出
func legacyFieldToFormField(add map[string]any) map[string]any {
	field := map[string]any{}
	if prop, _ := add["model"].(string); prop != "" {
		field["prop"] = prop
	}
	if label, _ := add["name"].(string); label != "" {
		field["label"] = label
	}
	if placeholder, _ := add["placeholder"].(string); placeholder != "" {
		field["placeholder"] = placeholder
	}
	if required, _ := add["required"].(bool); required {
		field["rules"] = map[string]any{"required": true}
	}
	if options, _ := add["options"].(string); options != "" {
		formOptions := make([]any, 0, strings.Count(options, ",")+1)
		for _, option := range strings.Split(options, ",") {
			option = strings.TrimSpace(option)
			if option == "" {
				continue
			}
			formOptions = append(formOptions, map[string]any{"value": option, "label": option})
		}
		if len(formOptions) > 0 {
			field["options"] = formOptions
		}
	}
	return field
}
