package application

import (
	"context"
	"testing"

	"mayfly-go/internal/label/domain/entity"
	"mayfly-go/internal/label/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupLabelDb(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	// 必须让真唯一索引建出来：这条测试要证的就是「墓碑行仍占着 key+value」
	if err := db.AutoMigrate(&entity.Label{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	sqlDb, err := db.DB()
	if err != nil {
		t.Fatalf("underlying db failed: %v", err)
	}
	sqlDb.SetMaxOpenConns(1)
	sqlDb.SetMaxIdleConns(1)

	previous := global.Db
	global.Db = db
	t.Cleanup(func() { global.Db = previous })
	return db
}

// labelRepoStub 复用真实基座仓储，只补上 domain 接口里那几个按 key 取数的方法
type labelRepoStub struct {
	base.RepoImpl[*entity.Label]
}

func (r *labelRepoStub) GetByKey(labelKey, labelValue string) (*entity.Label, error) {
	var label entity.Label
	err := r.GetByCond(model.NewCond().Eq("label_key", labelKey).Eq("label_value", labelValue).Dest(&label))
	if err != nil {
		if model.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &label, nil
}

func (r *labelRepoStub) GetByKeyIncludingDeleted(labelKey, labelValue string) (*entity.Label, error) {
	var label entity.Label
	err := r.GetByCond(model.NewCond().IncludeDeleted().Eq("label_key", labelKey).Eq("label_value", labelValue).Dest(&label))
	if err != nil {
		if model.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &label, nil
}

func (r *labelRepoStub) ListKeys(context.Context) ([]string, error) { panic("not used") }

func (r *labelRepoStub) ListValuesByKey(context.Context, string) ([]string, error) { panic("not used") }

var _ repository.Label = (*labelRepoStub)(nil)

func newBindingAppWithLabelRepo() *labelBindingAppImpl {
	return &labelBindingAppImpl{
		AppImpl:   base.AppImpl[*entity.LabelBinding, repository.LabelBinding]{},
		labelRepo: &labelRepoStub{},
	}
}

// TestGetOrCreateLabelRevivesTombstone 删掉标签后重新添加同一 key+value：必须复用原行并撤销软删。
//
// 只按未删除数据判断「不存在」然后 INSERT，会撞 idx_label_kv（唯一索引不看 is_deleted），
// 用户看到的就是「刚删掉再加回去报系统错误」
func TestGetOrCreateLabelRevivesTombstone(t *testing.T) {
	db := setupLabelDb(t)
	app := newBindingAppWithLabelRepo()
	ctx := context.Background()

	label := &entity.Label{LabelKey: "env", LabelValue: "prod"}
	if err := app.labelRepo.Insert(ctx, label); err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	if err := app.labelRepo.DeleteById(ctx, label.Id); err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}

	// 用只看不含墓碑的口径确认它确实已不可见（否则下一条断言没有意义）
	if visible, err := app.labelRepo.GetByKey("env", "prod"); err != nil || visible != nil {
		t.Fatalf("precondition failed: visible label = %+v, err = %v", visible, err)
	}

	got, err := app.getOrCreateLabelIncludingTombstone(ctx, "env", "prod")
	if err != nil {
		t.Fatalf("get-or-create failed: %v", err)
	}
	if got.Id != label.Id {
		t.Errorf("label id = %d, want the tombstoned row %d reused", got.Id, label.Id)
	}
	if got.IsDeleted != model.ModelUndeleted {
		t.Errorf("is_deleted = %d, want undeleted after revival", got.IsDeleted)
	}

	var rows int64
	if err := db.Table("t_label").Count(&rows).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if rows != 1 {
		t.Errorf("label rows = %d, want 1: 复活而不是插新行", rows)
	}
}

// TestGetOrCreateLabelCreatesWhenAbsent 完全没登记过的 key+value 走新增
func TestGetOrCreateLabelCreatesWhenAbsent(t *testing.T) {
	setupLabelDb(t)
	app := newBindingAppWithLabelRepo()
	ctx := context.Background()

	got, err := app.getOrCreateLabelIncludingTombstone(ctx, "team", "sre")
	if err != nil {
		t.Fatalf("get-or-create failed: %v", err)
	}
	if got.Id == 0 {
		t.Fatal("created label has no id")
	}

	again, err := app.getOrCreateLabelIncludingTombstone(ctx, "team", "sre")
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if again.Id != got.Id {
		t.Errorf("second call created label %d, want the existing %d", again.Id, got.Id)
	}
}

// TestGetOrCreateLabelKeepsActiveRow 活跃行必须原样返回，绝不能被撤销软删或改写
func TestGetOrCreateLabelKeepsActiveRow(t *testing.T) {
	db := setupLabelDb(t)
	app := newBindingAppWithLabelRepo()
	ctx := context.Background()

	label := &entity.Label{LabelKey: "env", LabelValue: "prod"}
	label.SetExtraValue("color", "red")
	if err := app.labelRepo.Insert(ctx, label); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	got, err := app.getOrCreateLabelIncludingTombstone(ctx, "env", "prod")
	if err != nil {
		t.Fatalf("get-or-create failed: %v", err)
	}
	if got.Id != label.Id {
		t.Errorf("label id = %d, want %d", got.Id, label.Id)
	}

	var extra string
	if err := db.Table("t_label").Select("extra").Where("id = ?", label.Id).Row().Scan(&extra); err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	if extra != `{"color":"red"}` {
		t.Errorf("extra = %q, want %q: 复用活跃行不能改写它已有的配置", extra, `{"color":"red"}`)
	}
}
