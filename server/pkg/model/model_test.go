package model

import (
	"testing"
	"time"
)

// 基础模型审计填充的策略用例。
//
// 六个基础模型原来各写一份 FillBaseInfo，规则有细微分叉；现在填充策略收敛成
// applyCreate/applyUpdate 两份实现，这组用例把收敛后的每一条语义钉住，尤其是
// 「更新且无操作者上下文时不得覆盖修改人」这类容易被顺手改错的行为

func acc() *LoginAccount {
	return &LoginAccount{Id: 42, Username: "ops"}
}

func TestFillBaseInfoOnCreateWithAccount(t *testing.T) {
	m := &Model{}
	m.FillBaseInfo(acc())

	if m.CreateTime == nil || m.UpdateTime == nil {
		t.Fatalf("create must set both timestamps: create=%v update=%v", m.CreateTime, m.UpdateTime)
	}
	if m.CreatorId != 42 || m.Creator != "ops" {
		t.Errorf("creator = %d/%q, want 42/ops", m.CreatorId, m.Creator)
	}
	// 新建时修改人即创建人
	if m.ModifierId != 42 || m.Modifier != "ops" {
		t.Errorf("modifier = %d/%q, want 42/ops", m.ModifierId, m.Modifier)
	}
	if m.IsDeleted != ModelUndeleted {
		t.Errorf("is_deleted = %d, want %d", m.IsDeleted, ModelUndeleted)
	}
}

// TestFillBaseInfoCreateWithoutAccountUsesSysAccount 无操作者上下文（启动兜底、后台任务）时
// 创建人必须显式记为系统账号，而不是留下空值让界面显示成未知
func TestFillBaseInfoCreateWithoutAccountUsesSysAccount(t *testing.T) {
	m := &Model{}
	m.FillBaseInfo(nil)

	if m.Creator != SysAccount.Username || m.CreatorId != SysAccount.Id {
		t.Errorf("creator = %d/%q, want system account %q", m.CreatorId, m.Creator, SysAccount.Username)
	}
	if m.Modifier != SysAccount.Username {
		t.Errorf("modifier = %q, want system account on insert", m.Modifier)
	}
}

// TestFillBaseInfoUpdateKeepsModifierWithoutAccount 钉住既有语义：更新且无操作者时
// 只刷新更新时间，修改人保留实体自带值——覆盖成空会把「最后改这条的人」抹掉
func TestFillBaseInfoUpdateKeepsModifierWithoutAccount(t *testing.T) {
	m := &Model{}
	m.FillBaseInfo(acc())
	before := m.Modifier
	beforeId := m.ModifierId
	// 模拟「查出来再改」：主键非零才会走更新路径
	m.Id = 5
	earlier := m.UpdateTime
	time.Sleep(time.Millisecond)

	m.FillBaseInfo(nil)

	if m.UpdateTime.Equal(*earlier) {
		t.Error("update_time must always be refreshed")
	}
	if m.Modifier != before || m.ModifierId != beforeId {
		t.Errorf("modifier changed to %q/%d, want it preserved as %q/%d", m.Modifier, m.ModifierId, before, beforeId)
	}
}

func TestFillBaseInfoUpdateWithAccount(t *testing.T) {
	m := &Model{Modifier: "old", ModifierId: 1}
	m.Id = 7
	newAccount := &LoginAccount{Id: 99, Username: "approver"}
	m.FillBaseInfo(newAccount)

	if m.Modifier != "approver" || m.ModifierId != 99 {
		t.Errorf("modifier = %q/%d, want approver/99", m.Modifier, m.ModifierId)
	}
	// 更新不得改写创建信息
	if m.CreateTime != nil {
		t.Error("create_time must not be set when updating an existing row")
	}
	if m.Id != 7 {
		t.Errorf("id changed to %d, must stay 7", m.Id)
	}
}

// TestLogicDeleteFlagByComposition 是否逻辑删除完全由嵌入了哪个基础模型决定
func TestLogicDeleteFlagByComposition(t *testing.T) {
	cases := map[string]struct {
		entity ModelI
		want   bool
	}{
		"IdModel":        {&IdModel{}, false},
		"CreateModelNLD": {&CreateModelNLD{}, false},
		"ModelNLD":       {&ModelNLD{}, false},
		"DeletedModel":   {&DeletedModel{}, true},
		"CreateModel":    {&CreateModel{}, true},
		"Model":          {&Model{}, true},
	}
	for name, tc := range cases {
		if got := tc.entity.LogicDelete(); got != tc.want {
			t.Errorf("%s.LogicDelete() = %v, want %v", name, got, tc.want)
		}
	}
}

// TestModelNLDFillBaseInfo NLD（无逻辑删除列）只填创建/修改信息，不碰删除标记
func TestModelNLDFillBaseInfo(t *testing.T) {
	m := &ModelNLD{}
	m.FillBaseInfo(acc())
	if m.CreateTime == nil || m.UpdateTime == nil {
		t.Fatal("NLD create must set both timestamps")
	}
	if m.Creator != "ops" || m.Modifier != "ops" {
		t.Errorf("creator/modifier = %q/%q, want ops/ops", m.Creator, m.Modifier)
	}
}

// TestCreateModelHasNoUpdateColumns CreateModel 不含审计修改列，更新时必须整体不动
func TestCreateModelHasNoUpdateColumns(t *testing.T) {
	m := &CreateModel{}
	m.Id = 3
	m.FillBaseInfo(acc())
	// 已有主键即更新：CreateModel 没有修改列，整次填充必须是空操作
	if m.CreateTime != nil || m.Creator != "" {
		t.Errorf("CreateModel filled create fields on update: create=%v creator=%q", m.CreateTime, m.Creator)
	}
}

// TestPhysicalColumnContract 列名与逻辑删除语义是基座与迁移的共同契约，改名必须是有意的
func TestPhysicalColumnContract(t *testing.T) {
	if IdColumn != "id" || DeletedColumn != "is_deleted" || DeleteTimeColumn != "delete_time" {
		t.Fatalf("column constants drifted: %q %q %q", IdColumn, DeletedColumn, DeleteTimeColumn)
	}
	if ModifierColumn != "modifier" || ModifierIdColumn != "modifier_id" || UpdateTimeColumn != "update_time" {
		t.Fatal("audit column constants drifted")
	}
	if ModelDeleted == ModelUndeleted {
		t.Fatal("deleted flag values must differ")
	}
}
