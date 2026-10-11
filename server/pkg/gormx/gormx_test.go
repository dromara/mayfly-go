package gormx

import (
	"context"
	"errors"
	"testing"

	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type gxEntity struct {
	model.Model
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func (*gxEntity) TableName() string { return "gx_entity" }

// gxUniqueEntity 带唯一业务键的实体，用于验证「冲突即跳过」
type gxUniqueEntity struct {
	model.Model
	Code string `json:"code" gorm:"size:32;not null;uniqueIndex:uk_gx_code"`
}

func (*gxUniqueEntity) TableName() string { return "gx_unique_entity" }

// newEntity 构造一个已填充审计字段的实体。
//
// gormx 只负责 SQL，创建人/时间等由上层 base.RepoImpl 填充，因此这里的测试必须自己填，
// 否则非空列约束会拒绝插入——这个失败本身就是分层正确的证据
func newEntity(name string, age int) *gxEntity {
	entity := &gxEntity{Name: name, Age: age}
	entity.FillBaseInfo(nil)
	return entity
}

func setupGxDb(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&gxEntity{}, &gxUniqueEntity{}); err != nil {
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

// TestDBUsesTxBoundToCtx 连接选择只由 ctx 决定：绑了事务就用事务，不需要任何 WithDb 变体
func TestDBUsesTxBoundToCtx(t *testing.T) {
	setupGxDb(t)
	ctx := context.Background()

	txCtx, tx, err := Begin(ctx)
	if err != nil {
		t.Fatalf("begin failed: %v", err)
	}
	if !InTx(txCtx) {
		t.Fatal("ctx should report being inside a transaction")
	}
	if DB(txCtx).Statement.ConnPool != tx.Statement.ConnPool {
		t.Error("DB(ctx) must reuse the transaction connection pool bound in ctx")
	}
	if InTx(ctx) {
		t.Error("the original ctx must not be affected")
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback failed: %v", err)
	}
}

// TestTxWriteInvisibleAfterRollback 事务内写入在回滚后必须不存在（同一连接才谈得上原子性）
func TestTxWriteInvisibleAfterRollback(t *testing.T) {
	db := setupGxDb(t)
	ctx := context.Background()

	txCtx, tx, err := Begin(ctx)
	if err != nil {
		t.Fatalf("begin failed: %v", err)
	}
	if err := Insert(txCtx, newEntity("rolled-back", 1)); err != nil {
		tx.Rollback()
		t.Fatalf("insert in tx failed: %v", err)
	}
	seen, err := Exists(txCtx, &gxEntity{}, model.NewCond().Eq("name", "rolled-back"))
	if err != nil {
		t.Fatalf("exists failed: %v", err)
	}
	if !seen {
		t.Error("Exists must observe the row inserted earlier in the same transaction")
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	var count int64
	if err := db.Model(&gxEntity{}).Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("rows = %d, want 0 after rollback", count)
	}
}

// TestInsertIfAbsentAbsorbsConflict 幂等投递依赖唯一键：同键重复插入既不报错也不产生新行
func TestInsertIfAbsentAbsorbsConflict(t *testing.T) {
	db := setupGxDb(t)
	ctx := context.Background()

	for range 3 {
		entity := &gxUniqueEntity{Code: "once"}
		entity.FillBaseInfo(nil)
		if err := InsertIfAbsent(ctx, entity, "code"); err != nil {
			t.Fatalf("insert if absent failed: %v", err)
		}
	}
	var count int64
	if err := db.Model(&gxUniqueEntity{}).Where("code = ?", "once").Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("rows = %d, want 1", count)
	}
	if err := InsertIfAbsent(ctx, &gxEntity{}); !errors.Is(err, ErrConflictColumnsRequired) {
		t.Errorf("error = %v, want ErrConflictColumnsRequired when no conflict column given", err)
	}
}

// TestCountByCondReportsError 统计必须回报 SQL 错误：把错误当成 0 会让「为 0 则新建」写坏数据
func TestCountByCondReportsError(t *testing.T) {
	setupGxDb(t)

	type missing struct {
		model.Model
	}
	count, err := CountByCond(&missing{}, model.NewCond())
	if err == nil {
		t.Fatalf("count against an unmigrated table returned no error (count = %d)", count)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0 alongside the error", count)
	}
}

// TestCondWithoutDestIsError 条件没有输出去向时返回错误而不是 panic
func TestCondWithoutDestIsError(t *testing.T) {
	setupGxDb(t)

	if err := GetByCond(&gxEntity{}, model.NewCond()); !errors.Is(err, ErrCondDestMissing) {
		t.Errorf("error = %v, want ErrCondDestMissing", err)
	}
	if err := SelectByCond(&gxEntity{}, model.NewCond()); !errors.Is(err, ErrCondDestMissing) {
		t.Errorf("error = %v, want ErrCondDestMissing for select", err)
	}
}

// TestRestoreRequiresIncludeDeleted 未声明包含已删除数据时，自动软删过滤会让「恢复」永远匹配不到行，
// 必须报错而不是静默返回 0 行
func TestRestoreRequiresIncludeDeleted(t *testing.T) {
	db := setupGxDb(t)
	ctx := context.Background()

	entity := newEntity("gone", 1)
	if err := Insert(ctx, entity); err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	if _, err := DeleteById(ctx, &gxEntity{}, entity.Id); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if _, err := Restore(ctx, &gxEntity{}, model.NewCond().Eq(model.IdColumn, entity.Id)); !errors.Is(err, ErrRestoreNeedsIncludeDeleted) {
		t.Errorf("error = %v, want ErrRestoreNeedsIncludeDeleted", err)
	}
	rows, err := Restore(ctx, &gxEntity{}, model.NewCond().IncludeDeleted().Eq(model.IdColumn, entity.Id))
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if rows != 1 {
		t.Errorf("restored rows = %d, want 1", rows)
	}

	var stored gxEntity
	if err := db.First(&stored, entity.Id).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if stored.IsDeleted != model.ModelUndeleted || stored.DeleteTime != nil {
		t.Errorf("restore left is_deleted=%d delete_time=%v", stored.IsDeleted, stored.DeleteTime)
	}
	if stored.Modifier != model.SysAccount.Username {
		t.Errorf("restore attributed to %q, want the system account when no login account is present", stored.Modifier)
	}
}

// TestRestoreRejectsNonSoftDeletable 物理删除的表没有「已删除」可撤销
func TestRestoreRejectsNonSoftDeletable(t *testing.T) {
	setupGxDb(t)

	type plain struct {
		model.ModelNLD
	}
	if _, err := Restore(context.Background(), &plain{}, model.NewCond().IncludeDeleted()); !errors.Is(err, ErrNotSoftDeletable) {
		t.Errorf("error = %v, want ErrNotSoftDeletable", err)
	}
}

// TestValidateColumns 未知列必须被拒：列名写错若拼进 SET 子句，轻则运行期报错，重则写进相近的合法列
func TestValidateColumns(t *testing.T) {
	setupGxDb(t)

	if err := ValidateColumns(&gxEntity{}, []string{"name", "age", "id"}); err != nil {
		t.Errorf("valid columns rejected: %v", err)
	}
	err := ValidateColumns(&gxEntity{}, []string{"n0_such_column"})
	if !errors.Is(err, ErrUnknownColumn) {
		t.Errorf("error = %v, want ErrUnknownColumn", err)
	}
}

// TestDeletedFilterDoesNotOverrideCallerCondition 调用方显式就软删列表过态时不得被自动过滤覆盖，
// 否则「查已删除数据」永远返回空集
func TestDeletedFilterDoesNotOverrideCallerCondition(t *testing.T) {
	db := setupGxDb(t)
	ctx := context.Background()

	entity := newEntity("deleted-row", 1)
	if err := Insert(ctx, entity); err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	if _, err := DeleteById(ctx, &gxEntity{}, entity.Id); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// 物理行仍在
	var physical int64
	if err := db.Model(&gxEntity{}).Where("name = ?", "deleted-row").Count(&physical).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if physical != 1 {
		t.Fatalf("physical rows = %d, want 1", physical)
	}

	var live []*gxEntity
	if err := SelectByCond(&gxEntity{}, model.NewCond().Dest(&live)); err != nil {
		t.Fatalf("select live failed: %v", err)
	}
	if len(live) != 0 {
		t.Errorf("live query returned %d deleted rows", len(live))
	}

	var explicit []*gxEntity
	if err := SelectByCond(&gxEntity{}, model.NewCond().EqStrict(model.DeletedColumn, model.ModelDeleted).Dest(&explicit)); err != nil {
		t.Fatalf("select with explicit deleted condition failed: %v", err)
	}
	if len(explicit) != 1 {
		t.Errorf("explicit is_deleted=1 query returned %d rows, want 1 (auto filter overrode the caller)", len(explicit))
	}
}

// TestBatchInsertChunks 批量插入必须真的分批：整批一条巨型 SQL 会撞 max_allowed_packet
func TestBatchInsertChunks(t *testing.T) {
	db := setupGxDb(t)
	ctx := context.Background()

	bulk := make([]*gxEntity, DefaultBatchSize+7)
	for i := range bulk {
		bulk[i] = newEntity("b", i)
	}
	if err := BatchInsert(ctx, bulk); err != nil {
		t.Fatalf("batch insert failed: %v", err)
	}
	var count int64
	if err := db.Model(&gxEntity{}).Where("name = ?", "b").Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != int64(len(bulk)) {
		t.Errorf("rows = %d, want %d", count, len(bulk))
	}
}

// TestUpdateRejectsForeignValuesType 更新值形态非法必须报错：实体与 Model 不是同一对象时
// GORM 会把主键放进 SET 子句，等于允许搬动主键
func TestUpdateRejectsForeignValuesType(t *testing.T) {
	db := setupGxDb(t)
	ctx := context.Background()

	created := newEntity("origin", 1)
	if err := Insert(ctx, created); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	type other struct {
		Id   uint64
		Name string
	}
	if _, err := UpdateByCond(ctx, &gxEntity{}, &other{Id: 999, Name: "hacked"}, model.NewCond().Eq("name", "origin")); !errors.Is(err, ErrUpdateValues) {
		t.Errorf("error = %v, want ErrUpdateValues", err)
	}
	var stored gxEntity
	if err := db.First(&stored, created.Id).Error; err != nil {
		t.Fatalf("primary key was moved: %v", err)
	}
	if stored.Name != "origin" {
		t.Errorf("name = %q, want origin unchanged", stored.Name)
	}
}
