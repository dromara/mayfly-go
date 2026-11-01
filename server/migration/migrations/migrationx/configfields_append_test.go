package migrationx

// 追加配置字段：两种 params 形态都要能追加、且不破坏原有内容。
// 这条用例的直接动机：升级库上「新增的配置项没出现在配置页」时，
// 得先能排除「迁移没写进去」——否则会去前端找半天。

import (
	"encoding/json"
	"testing"
	"time"

	sysentity "mayfly-go/internal/sys/domain/entity"

	"mayfly-go/pkg/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newConfigDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&sysentity.Config{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	return db
}

func propNames(t *testing.T, params string) map[string]bool {
	t.Helper()
	var doc struct {
		Fields []map[string]any `json:"fields"`
	}
	if err := json.Unmarshal([]byte(params), &doc); err != nil {
		t.Fatalf("params is not v1 schema: %v", err)
	}
	names := make(map[string]bool, len(doc.Fields))
	for _, field := range doc.Fields {
		if prop, ok := field["prop"].(string); ok {
			names[prop] = true
		}
	}
	return names
}

func TestAppendConfigFieldsV1Schema(t *testing.T) {
	db := newConfigDB(t)
	// 真实形态：v1 + cols + fields，追加时不能丢 cols
	original := `{"version":1,"cols":2,"fields":[{"prop":"maxResultSet","label":"system.sysconf.maxResultSet"}]}`
	now := time.Now()
	seed := &sysentity.Config{Name: "DBMS", Key: "DbmsConfig", Params: original}
	seed.Model = model.Model{CreateTime: &now, UpdateTime: &now}
	if err := db.Create(seed).Error; err != nil {
		t.Fatalf("seed config failed: %v", err)
	}

	if err := AppendConfigFields(db, "DbmsConfig", []map[string]any{
		{"prop": "metricSampleMinutes", "label": "system.sysconf.metricSampleMinutes"},
		{"prop": "metricRetainDays", "label": "system.sysconf.metricRetainDays"},
	}, nil); err != nil {
		t.Fatalf("AppendConfigFields failed: %v", err)
	}

	config := &sysentity.Config{}
	if err := db.Where("`key` = ?", "DbmsConfig").First(config).Error; err != nil {
		t.Fatalf("reload config failed: %v", err)
	}
	names := propNames(t, config.Params)
	for _, prop := range []string{"maxResultSet", "metricSampleMinutes", "metricRetainDays"} {
		if !names[prop] {
			t.Fatalf("prop %q must exist after append, got %v", prop, names)
		}
	}
	if !json.Valid([]byte(config.Params)) {
		t.Fatal("params must stay valid json")
	}
	if !containsAll(config.Params, `"cols":2`) {
		t.Fatalf("existing layout keys must survive the append, got %s", config.Params)
	}

	// 再跑一次：幂等，不重复追加
	if err := AppendConfigFields(db, "DbmsConfig", []map[string]any{
		{"prop": "metricSampleMinutes", "label": "system.sysconf.metricSampleMinutes"},
	}, nil); err != nil {
		t.Fatalf("second append failed: %v", err)
	}
	config = &sysentity.Config{}
	_ = db.Where("`key` = ?", "DbmsConfig").First(config).Error
	count := 0
	var doc struct {
		Fields []map[string]any `json:"fields"`
	}
	_ = json.Unmarshal([]byte(config.Params), &doc)
	for _, field := range doc.Fields {
		if field["prop"] == "metricSampleMinutes" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("append must be idempotent, got %d entries", count)
	}
}

func containsAll(text, sub string) bool {
	return len(text) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(text); i++ {
			if text[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

// TestAppendConfigFieldsLegacyShape 旧版裸数组形态：去重键与字段定义形态无关，
// 且写回后仍是 legacy 键名（两种形态混在一个数组里，前端表单会读不出新条目）。
func TestAppendConfigFieldsLegacyShape(t *testing.T) {
	db := newConfigDB(t)
	now := time.Now()
	seed := &sysentity.Config{Name: "DBMS", Key: "DbmsConfig", Params: `[{"model":"maxResultSet","name":"最大结果集"}]`}
	seed.Model = model.Model{CreateTime: &now, UpdateTime: &now}
	if err := db.Create(seed).Error; err != nil {
		t.Fatalf("seed config failed: %v", err)
	}

	fields := []map[string]any{{"prop": "metricSampleMinutes", "label": "system.sysconf.metricSampleMinutes"}}
	if err := AppendConfigFields(db, "DbmsConfig", fields, nil); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	// 重跑一次：必须幂等（只比 model 的老实现会再追加一份）
	if err := AppendConfigFields(db, "DbmsConfig", fields, nil); err != nil {
		t.Fatalf("second append failed: %v", err)
	}

	config := &sysentity.Config{}
	_ = db.Where("`key` = ?", "DbmsConfig").First(config).Error
	var items []map[string]any
	if err := json.Unmarshal([]byte(config.Params), &items); err != nil {
		t.Fatalf("legacy params must stay a bare array: %v", err)
	}
	count := 0
	for _, item := range items {
		if item["model"] == "metricSampleMinutes" {
			count++
			if item["prop"] != nil {
				t.Fatalf("legacy entry must use model key, got %v", item)
			}
		}
	}
	if count != 1 {
		t.Fatalf("append must be idempotent in legacy shape, got %d entries: %s", count, config.Params)
	}
}
