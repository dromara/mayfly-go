package persistence

import (
	"context"
	"sync"
	"testing"
	"time"

	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/pkg/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func setupAlertRuleDb(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&entity.AlertRule{}); err != nil {
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

func parseAlertRuleSchema(t *testing.T) *schema.Schema {
	t.Helper()
	s, err := schema.Parse(&entity.AlertRule{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse AlertRule schema failed: %v", err)
	}
	return s
}

// newRule 构造一条通过校验的最小规则：条件必填，故 alert_condition 始终非空
func newRule(name string, priority entity.AlertPriority) *entity.AlertRule {
	return &entity.AlertRule{
		Name:         name,
		Status:       entity.AlertRuleStatusEnable,
		Priority:     priority,
		Remark:       "备注-" + name,
		ResourceType: 1,
		ScopeType:    entity.AlertScopeResource,
		ScopeValue:   "100",
		Condition: &entity.AlertCondition{
			Operator: entity.ConditionOperatorAnd,
			Items:    []entity.ConditionItem{{Metric: "cpu_rate", Compare: "gt", Value: 1}},
		},
		NotifyConfig:  &entity.AlertNotifyConfig{GroupWait: 30},
		EvalInterval:  60,
		TriggerCount:  1,
		RecoveryCount: 1,
		Labels:        `{"env":"prod"}`,
	}
}

// TestPriorityFieldHasNoGormDefault P0 的枚举值就是 0，与 Go 零值同形。
// 一旦给 priority 加回 gorm 的 default 标签，GORM 会把「值为零 + 带默认值」的字段从 INSERT 列里
// 整个剔除交给数据库默认值填充，用户选的 P0 就会被静默改写成 P2，且接口照样返回成功。
func TestPriorityFieldHasNoGormDefault(t *testing.T) {
	field := parseAlertRuleSchema(t).FieldsByDBName["priority"]
	if field == nil {
		t.Fatal("schema 里没有 priority 列")
	}
	if field.HasDefaultValue {
		t.Fatalf("priority 不得带 gorm default 标签（HasDefaultValue=true, DefaultValue=%q），否则 P0 永远存不进库", field.DefaultValue)
	}
}

// TestCreateRulePersistsCriticalPriority 新建 P0 规则必须原样落库为 0
func TestCreateRulePersistsCriticalPriority(t *testing.T) {
	db := setupAlertRuleDb(t)
	repo := &alertRuleRepoImpl{}
	ctx := context.Background()

	rule := newRule("p0-rule", entity.AlertPriorityCritical)
	if err := repo.Insert(ctx, rule); err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	if rule.Id == 0 {
		t.Fatal("insert 后未回填主键")
	}

	got := new(entity.AlertRule)
	if err := db.First(got, rule.Id).Error; err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	if got.Priority != entity.AlertPriorityCritical {
		t.Fatalf("P0 未落库: got=%d(%s) want=0(P0)", got.Priority, got.Priority.Code())
	}
}

// TestUpdateAllColumnsWritesZeroPriority 规则从 P2 降级为 P0 时必须真实写入。
// UpdateById（隐式 Updates(struct)）会跳过零值字段，只有显式选列才写
func TestUpdateAllColumnsWritesZeroPriority(t *testing.T) {
	db := setupAlertRuleDb(t)
	repo := &alertRuleRepoImpl{}
	ctx := context.Background()

	rule := newRule("p2-rule", entity.AlertPriorityMedium)
	if err := repo.Insert(ctx, rule); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	rule.Priority = entity.AlertPriorityCritical
	if err := repo.UpdateAllColumns(ctx, rule); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	got := new(entity.AlertRule)
	if err := db.First(got, rule.Id).Error; err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	if got.Priority != entity.AlertPriorityCritical {
		t.Fatalf("P2→P0 的修改未落库: got=%d want=0", got.Priority)
	}

	// 对照：同一实体走隐式 UpdateById 时零值 priority 会被跳过，证明上面的成功来自显式选列
	other := newRule("p2-other", entity.AlertPriorityMedium)
	if err := repo.Insert(ctx, other); err != nil {
		t.Fatalf("insert control rule failed: %v", err)
	}
	other.Priority = entity.AlertPriorityCritical
	if err := repo.UpdateById(ctx, other); err != nil {
		t.Fatalf("control update failed: %v", err)
	}
	control := new(entity.AlertRule)
	if err := db.First(control, other.Id).Error; err != nil {
		t.Fatalf("read back control failed: %v", err)
	}
	if control.Priority == entity.AlertPriorityCritical {
		t.Fatal("隐式 UpdateById 竟然写入了零值 priority，判据已失效：GORM 的跳零值语义变了")
	}
}

// TestUpdateAllColumnsClearsRemarkAndLabels 清空备注与标签同样是「零值即用户意图」，必须真实写入
func TestUpdateAllColumnsClearsRemarkAndLabels(t *testing.T) {
	db := setupAlertRuleDb(t)
	repo := &alertRuleRepoImpl{}
	ctx := context.Background()

	rule := newRule("clear-rule", entity.AlertPriorityHigh)
	if err := repo.Insert(ctx, rule); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	rule.Remark = ""
	rule.Labels = ""
	rule.NotifyConfig = nil
	if err := repo.UpdateAllColumns(ctx, rule); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	got := new(entity.AlertRule)
	if err := db.First(got, rule.Id).Error; err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	if got.Remark != "" {
		t.Fatalf("备注清空未落库: got=%q", got.Remark)
	}
	if got.Labels != "" {
		t.Fatalf("标签清空未落库: got=%q", got.Labels)
	}
	if got.NotifyConfig != nil {
		t.Fatalf("通知配置移除未落库: got=%+v", got.NotifyConfig)
	}
}

// TestUpdateAllColumnsKeepsCreateAudit 覆盖式更新只该动业务列与修改审计列，
// 创建审计列（含逻辑删除位）必须原样保留，否则等于篡改历史
func TestUpdateAllColumnsKeepsCreateAudit(t *testing.T) {
	db := setupAlertRuleDb(t)
	repo := &alertRuleRepoImpl{}
	ctx := context.Background()

	rule := newRule("audit-rule", entity.AlertPriorityHigh)
	if err := repo.Insert(ctx, rule); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	// 创建审计由 Insert 内的 FillBaseInfo 填充，以落库结果为准再比对
	before := new(entity.AlertRule)
	if err := db.First(before, rule.Id).Error; err != nil {
		t.Fatalf("read back before failed: %v", err)
	}

	rule.Name = "renamed"
	// 稍作等待：修改时间靠 time.Now() 先后比较，部分平台时钟粒度较粗会误判为「未刷新」
	time.Sleep(2 * time.Millisecond)
	if err := repo.UpdateAllColumns(ctx, rule); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	got := new(entity.AlertRule)
	if err := db.First(got, rule.Id).Error; err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	if got.Name != "renamed" {
		t.Fatalf("业务列未更新: got=%q", got.Name)
	}
	if got.CreatorId != before.CreatorId || got.Creator != before.Creator {
		t.Fatalf("创建人被覆盖: got=(%d,%q) before=(%d,%q)", got.CreatorId, got.Creator, before.CreatorId, before.Creator)
	}
	if before.CreateTime == nil || got.CreateTime == nil || !got.CreateTime.Equal(*before.CreateTime) {
		t.Fatalf("创建时间被覆盖: got=%v before=%v", got.CreateTime, before.CreateTime)
	}
	if got.IsDeleted != before.IsDeleted {
		t.Fatalf("逻辑删除位被改动: got=%d before=%d", got.IsDeleted, before.IsDeleted)
	}
	// 选列后 GORM 不再自动补 update_time，必须靠清单点名才会刷新
	if before.UpdateTime == nil || got.UpdateTime == nil || !got.UpdateTime.After(*before.UpdateTime) {
		t.Fatalf("修改时间未刷新: got=%v before=%v", got.UpdateTime, before.UpdateTime)
	}
}

// TestAlertRuleColumnsCoverAllBusinessColumns 选列清单必须覆盖实体的全部业务列。
// 漏一列就等于该字段在编辑保存时永远改不动，且不会有任何报错
func TestAlertRuleColumnsCoverAllBusinessColumns(t *testing.T) {
	s := parseAlertRuleSchema(t)

	// 非业务列：主键、逻辑删除位与创建审计（整表单覆盖不该动它们）
	skip := map[string]bool{
		"id": true, "is_deleted": true, "delete_time": true,
		"create_time": true, "creator_id": true, "creator": true,
	}
	selected := make(map[string]bool, len(alertRuleColumns))
	for _, column := range alertRuleColumns {
		if _, ok := s.FieldsByDBName[column]; !ok {
			t.Fatalf("选列清单里的 %q 不是 AlertRule 的列，会导致更新直接报错", column)
		}
		if selected[column] {
			t.Fatalf("选列清单里 %q 重复", column)
		}
		selected[column] = true
	}

	var missing []string
	for _, dbName := range s.DBNames {
		if skip[dbName] || selected[dbName] {
			continue
		}
		missing = append(missing, dbName)
	}
	if len(missing) > 0 {
		t.Fatalf("以下业务列未被 alertRuleColumns 覆盖，编辑保存时会静默不生效: %v", missing)
	}

	// 修改审计列必须点名：选列后 GORM 不再自动补 update_time，漏掉会让「最后修改时间」永远停在创建时刻
	for _, column := range []string{"update_time", "modifier_id", "modifier"} {
		if !selected[column] {
			t.Fatalf("修改审计列 %q 未纳入选列清单，更新后不会再刷新", column)
		}
	}
}
