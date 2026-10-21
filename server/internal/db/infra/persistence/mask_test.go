package persistence

import (
	"context"
	"sync"
	"testing"

	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func setupMaskDb(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&entity.DbMaskRule{}, &entity.DbMaskColumn{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	sqlDb, err := db.DB()
	if err != nil {
		t.Fatalf("underlying db failed: %v", err)
	}
	sqlDb.SetMaxOpenConns(1)
	sqlDb.SetMaxIdleConns(1)

	// 仓储经 gormx.DB(ctx) 取连接，无事务上下文时回落到 global.Db
	previous := global.Db
	global.Db = db
	t.Cleanup(func() { global.Db = previous })
	return db
}

func parseSchema(t *testing.T, dest any) *schema.Schema {
	t.Helper()
	s, err := schema.Parse(dest, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse %T schema failed: %v", dest, err)
	}
	return s
}

// TestMaskRuleStatusHasNoGormDefault 「停用」的枚举值就是 0，与 Go 零值同形。
// 一旦给 status 加回 gorm 的 default 标签，GORM 会把「值为零 + 带默认值」的字段从 INSERT 列里
// 整个剔除交给数据库默认值填充，用户建的停用规则就会被静默变成启用
func TestMaskRuleStatusHasNoGormDefault(t *testing.T) {
	field := parseSchema(t, &entity.DbMaskRule{}).FieldsByDBName["status"]
	if field == nil {
		t.Fatal("schema 里没有 status 列")
	}
	if field.HasDefaultValue {
		t.Fatalf("status 不得带 gorm default 标签（HasDefaultValue=true, DefaultValue=%q），否则停用状态永远存不进库", field.DefaultValue)
	}
}

// TestMaskRuleSaveAllColumnsCreatesDisabled 新建即停用必须原样落库为 0
func TestMaskRuleSaveAllColumnsCreatesDisabled(t *testing.T) {
	db := setupMaskDb(t)
	repo := &maskRuleRepoImpl{}
	ctx := context.Background()

	rule := &entity.DbMaskRule{
		Name: "disabled-rule", MatchType: entity.MaskMatchTypeExact, Pattern: "phone",
		Algorithm: "full", Status: entity.MaskRuleStatusDisabled, Weight: 3,
	}
	if err := repo.SaveAllColumns(ctx, rule); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	got := new(entity.DbMaskRule)
	if err := db.First(got, rule.Id).Error; err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	if got.Status != entity.MaskRuleStatusDisabled {
		t.Fatalf("停用状态未落库: got=%d want=0", got.Status)
	}
	if got.Weight != 3 {
		t.Fatalf("权重未落库: got=%d want=3", got.Weight)
	}
}

// TestMaskRuleSaveAllColumnsDisables 启用中的规则改为停用必须真实写入
func TestMaskRuleSaveAllColumnsDisables(t *testing.T) {
	db := setupMaskDb(t)
	repo := &maskRuleRepoImpl{}
	ctx := context.Background()

	rule := &entity.DbMaskRule{
		Name: "toggle-rule", MatchType: entity.MaskMatchTypeExact, Pattern: "email",
		Algorithm: "full", Status: entity.MaskRuleStatusEnabled,
	}
	if err := repo.SaveAllColumns(ctx, rule); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	rule.Status = entity.MaskRuleStatusDisabled
	rule.Remark = ""
	if err := repo.SaveAllColumns(ctx, rule); err != nil {
		t.Fatalf("disable failed: %v", err)
	}

	got := new(entity.DbMaskRule)
	if err := db.First(got, rule.Id).Error; err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	if got.Status != entity.MaskRuleStatusDisabled {
		t.Fatalf("启用→停用的修改未落库: got=%d want=0", got.Status)
	}

	// 对照：同一实体走隐式 Save 时零值 status 会被跳过，证明上面的成功来自显式选列
	other := &entity.DbMaskRule{
		Name: "control-rule", MatchType: entity.MaskMatchTypeExact, Pattern: "idcard",
		Algorithm: "full", Status: entity.MaskRuleStatusEnabled,
	}
	if err := repo.Save(ctx, other); err != nil {
		t.Fatalf("create control failed: %v", err)
	}
	other.Status = entity.MaskRuleStatusDisabled
	if err := repo.Save(ctx, other); err != nil {
		t.Fatalf("control disable failed: %v", err)
	}
	control := new(entity.DbMaskRule)
	if err := db.First(control, other.Id).Error; err != nil {
		t.Fatalf("read back control failed: %v", err)
	}
	if control.Status == entity.MaskRuleStatusDisabled {
		t.Fatal("隐式 Save 竟然写入了零值 status，判据已失效：GORM 的跳零值语义变了")
	}
}

// TestMaskColumnSaveAllColumnsUnbindsRule 列标签的零值同样是用户意图：
// 从「绑定规则」改回「直接指定算法」时 rule_id 要清 0，把限定范围放宽到全部库表时
// db_name/table_name 要清成空串
func TestMaskColumnSaveAllColumnsUnbindsRule(t *testing.T) {
	db := setupMaskDb(t)
	repo := &maskColumnRepoImpl{}
	ctx := context.Background()

	tag := &entity.DbMaskColumn{
		InstanceId: 9, DbName: "app", MatchTable: "t_user", ColumnName: "phone",
		Action: entity.MaskColumnActionBind, RuleId: 7,
	}
	if err := repo.SaveAllColumns(ctx, tag); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	tag.RuleId = 0
	tag.Algorithm = "full"
	tag.DbName = ""
	tag.MatchTable = ""
	if err := repo.SaveAllColumns(ctx, tag); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	got := new(entity.DbMaskColumn)
	if err := db.First(got, tag.Id).Error; err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	if got.RuleId != 0 {
		t.Fatalf("规则解绑未落库: ruleId=%d want=0", got.RuleId)
	}
	if got.Algorithm != "full" {
		t.Fatalf("算法未落库: got=%q", got.Algorithm)
	}
	if got.DbName != "" || got.MatchTable != "" {
		t.Fatalf("范围放宽未落库: dbName=%q tableName=%q", got.DbName, got.MatchTable)
	}
	if got.InstanceId != 9 || got.ColumnName != "phone" || got.Action != entity.MaskColumnActionBind {
		t.Fatalf("非零字段被误改: instanceId=%d columnName=%q action=%d", got.InstanceId, got.ColumnName, got.Action)
	}
}

// TestMaskColumnsCoverAllBusinessColumns 选列清单必须覆盖实体的全部业务列。
// 漏一列就等于该字段在编辑保存时永远改不动，且不会有任何报错
func TestMaskColumnsCoverAllBusinessColumns(t *testing.T) {
	// 非业务列：主键、逻辑删除位与创建审计（整表单覆盖不该动它们）
	skip := map[string]bool{
		"id": true, "is_deleted": true, "delete_time": true,
		"create_time": true, "creator_id": true, "creator": true,
	}
	cases := []struct {
		name     string
		entity   any
		selected []string
	}{
		{"DbMaskRule", &entity.DbMaskRule{}, maskRuleColumns},
		{"DbMaskColumn", &entity.DbMaskColumn{}, maskColumnColumns},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := parseSchema(t, tc.entity)
			seen := make(map[string]bool, len(tc.selected))
			for _, column := range tc.selected {
				if _, ok := s.FieldsByDBName[column]; !ok {
					t.Fatalf("选列清单里的 %q 不是 %s 的列，会导致更新直接报错", column, tc.name)
				}
				if seen[column] {
					t.Fatalf("选列清单里 %q 重复", column)
				}
				seen[column] = true
			}

			var missing []string
			for _, dbName := range s.DBNames {
				if skip[dbName] || seen[dbName] {
					continue
				}
				missing = append(missing, dbName)
			}
			if len(missing) > 0 {
				t.Fatalf("以下业务列未被选列清单覆盖，编辑保存时会静默不生效: %v", missing)
			}
			// 选列后 GORM 不再自动补修改审计列，漏掉会让「最后修改时间」永远停在创建时刻
			for _, column := range []string{"update_time", "modifier_id", "modifier"} {
				if !seen[column] {
					t.Fatalf("修改审计列 %q 未纳入选列清单，更新后不会再刷新", column)
				}
			}
		})
	}
}
