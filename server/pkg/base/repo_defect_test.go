package base

import (
	"context"
	"errors"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/gormx"
	"sync"
	"testing"
	"time"

	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 基座仓储/应用层的回归用例。
//
// 这批用例逐条对应曾实测复现的缺陷：调用方实体被就地破坏、指定列查询掉主键导致 Save 插重复行、
// 逻辑删除行永远查不出、软删不记操作人、统计吞错、按 id 便利入口把无条件查询放开全表、
// 更新值形态不校验导致主键被当普通列改写、单例仓储懒初始化的数据竞争

// ReviewEntity 带逻辑删除与完整审计字段的标准实体
type ReviewEntity struct {
	model.Model
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func (*ReviewEntity) TableName() string { return "review_entity" }

func setupTestDb(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&ReviewEntity{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	// SQLite 的 ":memory:" 在连接池下每条连接都是一个独立的空库，因此与生产配置一致地把连接数
	// 收到 1：否则并发用例拿到的是一条没有建表的连接，测的是池行为而不是被测逻辑
	sqlDb, err := db.DB()
	if err != nil {
		t.Fatalf("underlying db handle failed: %v", err)
	}
	sqlDb.SetMaxIdleConns(1)
	sqlDb.SetMaxOpenConns(1)

	previous := global.Db
	global.Db = db
	t.Cleanup(func() { global.Db = previous })
	return db
}

func newTestRepo() *RepoImpl[*ReviewEntity] {
	return &RepoImpl[*ReviewEntity]{}
}

// 每个用例都用独立的仓储实例，避免单例缓存互相干扰；并发用例除外
func newLoadedRepo(t *testing.T, repo *RepoImpl[*ReviewEntity]) *ReviewEntity {
	t.Helper()
	entity := &ReviewEntity{Name: "origin", Age: 10}
	if err := repo.Insert(context.Background(), entity); err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	return entity
}

func countRows(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	if err := db.Table("review_entity").Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	return count
}

// TestUpdateByCondKeepsCallerEntity 对应「基座把调用方实体的主键清成 0」：
// 实体被清 0 后再走 Save 会被 IsCreate 判成新建，插出一条重复数据
func TestUpdateByCondKeepsCallerEntity(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()
	ctx := context.Background()

	created := newLoadedRepo(t, repo)
	patchTarget := &ReviewEntity{Age: 18}
	if _, err := repo.UpdateByCond(ctx, patchTarget, model.NewCond().Eq(model.IdColumn, created.Id)); err != nil {
		t.Fatalf("update by cond failed: %v", err)
	}
	if got := countRows(t, db); got != 1 {
		t.Fatalf("rows after partial update = %d, want 1", got)
	}

	// 用同一个实体再更新一次：若基座把主键清 0，第二次调用会因为「实体主键为 0」而行为不同
	entity := created
	if _, err := repo.UpdateByCond(ctx, entity, model.NewCond().Eq(model.IdColumn, created.Id)); err != nil {
		t.Fatalf("second update failed: %v", err)
	}
	if entity.Id != created.Id {
		t.Errorf("caller entity id was mutated by the base: got %d, want %d", entity.Id, created.Id)
	}
	if got := countRows(t, db); got != 1 {
		t.Errorf("rows after second update = %d, want 1", got)
	}
}

// TestUpdateByCondDoesNotMutateCallerMap 对应「基座就地往调用方 map 里写审计字段」
func TestUpdateByCondDoesNotMutateCallerMap(t *testing.T) {
	setupTestDb(t)
	repo := newTestRepo()
	created := newLoadedRepo(t, repo)

	updates := collx.M{"age": 0}
	if _, err := repo.UpdateByCond(context.Background(), updates, model.NewCond().Eq(model.IdColumn, created.Id)); err != nil {
		t.Fatalf("update by cond failed: %v", err)
	}
	if _, existed := updates[model.UpdateTimeColumn]; existed {
		t.Error("base wrote audit columns into the caller's map")
	}
	if len(updates) != 1 {
		t.Errorf("caller map changed: %+v", updates)
	}

	// map 形态必须真的写出零值（这是它与实体形态的分工）
	var after ReviewEntity
	if err := global.Db.Table("review_entity").First(&after, "id = ?", created.Id).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if after.Age != 0 {
		t.Errorf("map update did not write zero value: age = %d, want 0", after.Age)
	}
}

// TestGetByIdWithColsKeepsPrimaryKey 对应「指定列查询掉主键 → Save 插重复行」
func TestGetByIdWithColsKeepsPrimaryKey(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()
	created := newLoadedRepo(t, repo)

	loaded, err := repo.GetById(created.Id, "name")
	if err != nil {
		t.Fatalf("get by id failed: %v", err)
	}
	if loaded.Id != created.Id {
		t.Fatalf("projection dropped the primary key: id = %d, want %d", loaded.Id, created.Id)
	}
	if loaded.IsCreate() {
		t.Fatal("loaded entity reports itself as new, so Save would insert a duplicate row")
	}

	loaded.Name = "changed"
	if err := repo.Save(context.Background(), loaded); err != nil {
		t.Fatalf("save after projected read failed: %v", err)
	}
	if got := countRows(t, db); got != 1 {
		t.Errorf("rows = %d, want 1 (Save turned into an INSERT because the primary key was missing)", got)
	}
}

// TestSelectByCondWithColsKeepsPrimaryKey 覆盖列表查询的同一处缺陷
func TestSelectByCondWithColsKeepsPrimaryKey(t *testing.T) {
	setupTestDb(t)
	repo := newTestRepo()
	newLoadedRepo(t, repo)

	list, err := repo.SelectByCond(model.NewCond(), "name")
	if err != nil {
		t.Fatalf("select by cond failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
	if list[0].Id == 0 {
		t.Error("projected list dropped the primary key")
	}
}

// TestQueryDeletedRowsRequiresIncludeDeleted 对应「自动软删过滤覆盖调用方条件」：
// 显式查已删除数据既不能静默返回空集，也不该需要调用方了解基座内部
func TestQueryDeletedRowsRequiresIncludeDeleted(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()
	created := newLoadedRepo(t, repo)

	if err := repo.DeleteById(context.Background(), created.Id); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	var live []*ReviewEntity
	if err := repo.SelectByCondToAny(model.NewCond().Dest(&live), &live); err != nil {
		t.Fatalf("select live failed: %v", err)
	}
	if len(live) != 0 {
		t.Fatalf("soft deleted row still visible: %+v", live)
	}

	var all []*ReviewEntity
	if err := repo.SelectByCondToAny(model.NewCond().IncludeDeleted().Dest(&all), &all); err != nil {
		t.Fatalf("select including deleted failed: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("IncludeDeleted did not return the deleted row: got %d, want 1", len(all))
	}

	// 物理行必须仍在（逻辑删除不是物删）
	if got := countRows(t, db); got != 1 {
		t.Errorf("rows = %d, want 1", got)
	}
}

// TestSoftDeleteRecordsAudit 对应「软删只写 is_deleted/delete_time，审计链断掉」
func TestSoftDeleteRecordsAudit(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()
	created := newLoadedRepo(t, repo)

	ctx := contextWithAccount(7, "ops")
	if err := repo.DeleteById(ctx, created.Id); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	var deleted ReviewEntity
	if err := db.Table("review_entity").First(&deleted, "id = ?", created.Id).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if deleted.IsDeleted != model.ModelDeleted {
		t.Fatalf("is_deleted = %d, want %d", deleted.IsDeleted, model.ModelDeleted)
	}
	if deleted.DeleteTime == nil {
		t.Fatal("delete_time was not written")
	}
	if deleted.Modifier != "ops" || deleted.ModifierId != 7 {
		t.Errorf("soft delete did not record the operator: modifier=%q modifierId=%d", deleted.Modifier, deleted.ModifierId)
	}
}

// TestCountByCondReportsError 对应「统计吞错」：SQL 出错不能与「统计为 0」混淆
func TestCountByCondReportsError(t *testing.T) {
	setupTestDb(t)

	type missingEntity struct {
		model.Model
	}
	// 表名取自未迁移的实体，统计必然失败
	missing := &RepoImpl[*missingEntity]{}
	count, err := missing.CountByCond(model.NewCond())
	if err == nil {
		t.Fatalf("count against a missing table returned no error, count = %d", count)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0 when the query failed", count)
	}
}

// TestGetByIdsEmptyIdsDoesNotScanTable 对应「In(nil) 被跳过 → 退化成无条件查询」
func TestGetByIdsEmptyIdsDoesNotScanTable(t *testing.T) {
	setupTestDb(t)
	repo := newTestRepo()
	for range 3 {
		newLoadedRepo(t, repo)
	}

	for name, ids := range map[string][]uint64{"nil": nil, "empty": {}} {
		list, err := repo.GetByIds(ids)
		if err != nil {
			t.Fatalf("GetByIds(%s) failed: %v", name, err)
		}
		if len(list) != 0 {
			t.Errorf("GetByIds(%s) returned %d rows; an empty id set must not fall back to a full table scan", name, len(list))
		}
	}
}

// TestDeleteByIdWithoutIdsFails 对应「无 where 删除」：必须显式拒绝，而不是靠驱动的兜底
func TestDeleteByIdWithoutIdsFails(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()
	newLoadedRepo(t, repo)

	if err := repo.DeleteById(context.Background()); !errors.Is(err, gorm.ErrInvalidData) && err == nil {
		t.Fatal("DeleteById without ids must fail instead of deleting everything")
	}
	if got := countRows(t, db); got != 1 {
		t.Errorf("rows = %d, want 1 (nothing should have been deleted)", got)
	}
}

// TestUpdateByCondRejectsForeignValuesType 对应「values 形态不校验 → 主键被当普通列改写」
func TestUpdateByCondRejectsForeignValuesType(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()
	created := newLoadedRepo(t, repo)

	// 结构体类型与仓储实体不一致：基座必须拒绝，而不是把它的 id 写进 SET 子句
	type foreignStruct struct {
		Id   uint64
		Name string
	}
	_, err := repo.UpdateByCond(context.Background(), &foreignStruct{Id: 999, Name: "hacked"}, model.NewCond().Eq("name", "origin"))
	if err == nil {
		t.Fatal("update accepted a values type unrelated to the model")
	}

	var survivor ReviewEntity
	if reloadErr := db.Table("review_entity").First(&survivor, "id = ?", created.Id).Error; reloadErr != nil {
		t.Fatalf("row %d disappeared, the primary key was overwritten: %v", created.Id, reloadErr)
	}
	if survivor.Name == "hacked" {
		t.Errorf("foreign values were written to the table: name = %q", survivor.Name)
	}
}

// TestConcurrentFirstUseIsRaceFree 对应「单例仓储跨请求共享，懒初始化无同步」的数据竞争
func TestConcurrentFirstUseIsRaceFree(t *testing.T) {
	setupTestDb(t)
	repo := newTestRepo()

	var wg sync.WaitGroup
	for range 64 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := repo.CountByCond(model.NewCond().Eq("name", "origin")); err != nil {
				t.Errorf("concurrent count failed: %v", err)
			}
			if model := repo.NewModel(); model == nil {
				t.Error("NewModel returned nil")
			}
		}()
	}
	wg.Wait()
}

// TestTypedNilCondDoesNotPanic 对应「typed-nil *QueryCond 直接 panic」
func TestTypedNilCondDoesNotPanic(t *testing.T) {
	setupTestDb(t)
	repo := newTestRepo()
	created := newLoadedRepo(t, repo)

	// 条件为 typed-nil 指针：不得 panic；归一成空条件后，无条件更新会被驱动护栏拒绝，
	// 这正是想要的结果——空条件绝不能变成一次全表更新
	var nilCond *model.QueryCond
	if _, err := repo.UpdateByCond(context.Background(), map[string]any{"age": 3}, nilCond); err == nil {
		t.Fatal("an empty condition must not be allowed to update every row")
	}
	var after ReviewEntity
	if err := global.Db.Table("review_entity").First(&after, "id = ?", created.Id).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if after.Age != 10 {
		t.Errorf("age = %d, want 10 (nothing should have been updated)", after.Age)
	}

	// 带条件时 typed-nil 的处理路径同样要正常 work
	if _, err := repo.UpdateByCond(context.Background(), map[string]any{"age": 3}, model.NewCond().Eq(model.IdColumn, created.Id)); err != nil {
		t.Fatalf("update with an explicit condition failed: %v", err)
	}
	if err := global.Db.Table("review_entity").First(&after, "id = ?", created.Id).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if after.Age != 3 {
		t.Errorf("age = %d, want 3", after.Age)
	}
}

// TestBatchInsertChunksAndToleratesEmpty 覆盖「名为批量实则一次拼完」与空集合语义
func TestBatchInsertChunksAndToleratesEmpty(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()

	if err := repo.BatchInsert(context.Background(), nil); err != nil {
		t.Fatalf("batch insert of an empty slice must be a no-op: %v", err)
	}
	if got := countRows(t, db); got != 0 {
		t.Errorf("rows = %d, want 0", got)
	}

	bulk := make([]*ReviewEntity, 0, DefaultTestBulkSize)
	for range DefaultTestBulkSize {
		bulk = append(bulk, &ReviewEntity{Name: "bulk", Age: 1})
	}
	if err := repo.BatchInsert(context.Background(), bulk); err != nil {
		t.Fatalf("batch insert failed: %v", err)
	}
	if got := countRows(t, db); got != int64(DefaultTestBulkSize) {
		t.Errorf("rows = %d, want %d", got, DefaultTestBulkSize)
	}
}

// DefaultTestBulkSize 用于验证真分批的条数，需大于基座的单批上限
const DefaultTestBulkSize = DefaultBatchSizeForTest*2 + 7

// DefaultBatchSizeForTest 与 gormx 的批次大小保持一致，集中声明避免两处字面量漂移
const DefaultBatchSizeForTest = 500

// TestGetByIdMissingRowIsNotFound 覆盖未找到语义：返回基座哨兵，业务不必依赖 ORM 错误值
func TestGetByIdMissingRowIsNotFound(t *testing.T) {
	setupTestDb(t)
	repo := newTestRepo()

	_, err := repo.GetById(4242)
	if !model.IsNotFound(err) {
		t.Fatalf("error = %v, want one wrapping model.ErrNotFound", err)
	}
}

// contextWithAccount 构造带登录账号的 ctx，用于校验审计字段是否记到了正确的人身上
func contextWithAccount(id uint64, username string) context.Context {
	return contextx.WithLoginAccount(context.Background(), &model.LoginAccount{Id: id, Username: username})
}

// TestRestoreByCond 撤销逻辑删除：必须显式声明包含已删除数据，否则基座自动附加的
// is_deleted = 0 过滤会让这条更新匹配不到任何行并静默返回 0 行
func TestRestoreByCond(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()
	ctx := context.Background()
	created := newLoadedRepo(t, repo)

	if err := repo.DeleteById(ctx, created.Id); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if _, err := repo.RestoreByCond(ctx, model.NewCond().Eq(model.IdColumn, created.Id)); !errors.Is(err, gormx.ErrRestoreNeedsIncludeDeleted) {
		t.Errorf("error = %v, want gormx.ErrRestoreNeedsIncludeDeleted", err)
	}

	rows, err := repo.RestoreByCond(contextWithAccount(21, "revivor"), model.NewCond().IncludeDeleted().Eq(model.IdColumn, created.Id))
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if rows != 1 {
		t.Errorf("restored rows = %d, want 1", rows)
	}

	var after ReviewEntity
	if err := db.Table("review_entity").First(&after, "id = ?", created.Id).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if after.IsDeleted != model.ModelUndeleted {
		t.Errorf("is_deleted = %d, want %d", after.IsDeleted, model.ModelUndeleted)
	}
	if after.DeleteTime != nil {
		t.Errorf("delete_time = %v, want NULL after restore", after.DeleteTime)
	}
	if after.Modifier != "revivor" || after.ModifierId != 21 {
		t.Errorf("restore attributed the row to %q/%d, want revivor/21", after.Modifier, after.ModifierId)
	}
}

// TestUpdateByCondEntityKeepsCreateAudit 实体形态的条件更新只应刷新修改审计。
//
// 填充顺序决定这件事：先清主键会让 IsCreate() 为真，创建审计被填成新值后随非零字段写进 SET，
// 于是一次「改标题」就把该行的创建人与创建时间改成当前账号与当前时间
func TestUpdateByCondEntityKeepsCreateAudit(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()
	ctx := contextWithAccount(7, "alice")

	created := &ReviewEntity{Name: "orig", Age: 10}
	if err := repo.Insert(ctx, created); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	var origCreatorId uint64
	var origCreateTime time.Time
	if err := db.Table("review_entity").Select("creator_id, create_time").Where("id = ?", created.Id).
		Row().Scan(&origCreatorId, &origCreateTime); err != nil {
		t.Fatalf("read back insert audit failed: %v", err)
	}
	if origCreatorId != 7 {
		t.Fatalf("test precondition failed: creator_id = %d, want 7", origCreatorId)
	}

	// 换一个人来改标题：这正是「改会话标题」这类操作的真实形态——操作者不是创建者
	updaterCtx := contextWithAccount(9, "bob")
	if _, err := repo.UpdateByCond(updaterCtx, &ReviewEntity{Id: created.Id, Name: "renamed"},
		model.NewCond().Eq(model.IdColumn, created.Id)); err != nil {
		t.Fatalf("update by cond failed: %v", err)
	}

	var storedCreatorId uint64
	var storedCreateTime time.Time
	var storedName string
	if err := db.Table("review_entity").Select("name, creator_id, create_time").Where("id = ?", created.Id).
		Row().Scan(&storedName, &storedCreatorId, &storedCreateTime); err != nil {
		t.Fatalf("read back update result failed: %v", err)
	}
	if storedName != "renamed" {
		t.Errorf("name = %q, want %q", storedName, "renamed")
	}
	if storedCreatorId != origCreatorId {
		t.Errorf("creator_id = %d, want %d：条件更新不该改写创建人", storedCreatorId, origCreatorId)
	}
	if !storedCreateTime.Equal(origCreateTime) {
		t.Errorf("create_time = %v, want %v：条件更新不该改写创建时间", storedCreateTime, origCreateTime)
	}
}

// TestDestructiveOpsRequireCallerCondition 删除/按条件更新/恢复必须带调用方声明的条件。
//
// 不能把这层拦截交给驱动：GORM 确实也会报 ErrMissingWhereClause，但 AllowGlobalUpdate 一开就失效，
// 而且报出来的文案说不清是调用方漏传了条件。「能不能整表操作」必须是基座自己的显式策略
func TestDestructiveOpsRequireCallerCondition(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()
	ctx := context.Background()
	newLoadedRepo(t, repo)

	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{"DeleteByCond(nil)", func() error {
			return repo.DeleteByCond(ctx, nil)
		}},
		{"DeleteByCond(空条件)", func() error {
			return repo.DeleteByCond(ctx, model.NewCond())
		}},
		{"DeleteByCond(全零实体)", func() error {
			return repo.DeleteByCond(ctx, &ReviewEntity{})
		}},
		{"UpdateByCond(nil)", func() error {
			_, err := repo.UpdateByCond(ctx, map[string]any{"age": 1}, nil)
			return err
		}},
		{"UpdateColsByCond(空条件)", func() error {
			_, err := repo.UpdateColsByCond(ctx, NewPatch().Set("age", 1), model.NewCond())
			return err
		}},
		{"RestoreByCond(只声明 IncludeDeleted)", func() error {
			_, err := repo.RestoreByCond(ctx, model.NewCond().IncludeDeleted())
			return err
		}},
	} {
		if err := tc.run(); !errors.Is(err, gormx.ErrCondRequired) {
			t.Errorf("%s err = %v, want ErrCondRequired", tc.name, err)
		}
	}

	// 一行都不能动：拦截必须发生在生成 SQL 之前
	if got := countRows(t, db); got != 1 {
		t.Errorf("rows = %d, want 1：无条件破坏性操作真的动了数据", got)
	}
	var untouched int64
	if err := db.Table("review_entity").Where("is_deleted = ?", model.ModelDeleted).Count(&untouched).Error; err != nil {
		t.Fatalf("count deleted failed: %v", err)
	}
	if untouched != 0 {
		t.Errorf("%d 行被软删，want 0", untouched)
	}

	// 带条件的删除照常工作：卡点不能把正常调用一起拦下
	if err := repo.DeleteByCond(ctx, model.NewCond().Eq("name", "origin")); err != nil {
		t.Errorf("有条件的删除被误拦: %v", err)
	}
}

// TestUpdateByCondMapKeepsCallerAudit map 形态的更新不得改写调用方显式写明的审计列。
//
// 回填历史操作的执行时间这类场景需要调用方自己指定 update_time；基座只在缺失时补齐。
// 这条契约必须只有一个实现点：曾经在仓储层也补一遍（无条件覆盖），于是同一条契约在
// 实体路径成立、在 map 路径失效
func TestUpdateByCondMapKeepsCallerAudit(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()
	created := newLoadedRepo(t, repo)

	backdated := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	updates := map[string]any{
		"age":                  33,
		model.UpdateTimeColumn: backdated,
		model.ModifierColumn:   "history-import",
		model.ModifierIdColumn: uint64(999),
	}
	if _, err := repo.UpdateByCond(contextWithAccount(7, "alice"), updates,
		model.NewCond().Eq(model.IdColumn, created.Id)); err != nil {
		t.Fatalf("update by cond failed: %v", err)
	}

	var after ReviewEntity
	if err := db.Table("review_entity").First(&after, "id = ?", created.Id).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if after.Age != 33 {
		t.Errorf("age = %d, want 33", after.Age)
	}
	if after.UpdateTime == nil || !after.UpdateTime.Equal(backdated) {
		t.Errorf("update_time = %v, want %v：调用方显式写明的审计列被基座改写了", after.UpdateTime, backdated)
	}
	if after.Modifier != "history-import" || after.ModifierId != 999 {
		t.Errorf("modifier = %q/%d, want history-import/999", after.Modifier, after.ModifierId)
	}
	if len(updates) != 4 {
		t.Errorf("调用方的 map 被就地改了：%+v", updates)
	}
}

// TestSelectByCondKeepsCondProjection 读取入口的零参列展开不得清空条件里声明的投影：
// 投影被静默丢弃会退化成全列查询，宽表上白拉一大堆列且无人发现
func TestSelectByCondKeepsCondProjection(t *testing.T) {
	setupTestDb(t)
	repo := newTestRepo()
	created := newLoadedRepo(t, repo)
	if _, err := repo.UpdateByCond(context.Background(), collx.M{"age": 42},
		model.NewCond().Eq(model.IdColumn, created.Id)); err != nil {
		t.Fatalf("update age: %v", err)
	}

	// 投影生效时未选中的列应保持零值：Age 没躲过 42 就说明投影被丢掉了
	list, err := repo.SelectByCond(model.NewCond().Columns("name"))
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(list) != 1 || list[0].Age != 0 {
		t.Fatalf("cond 内投影被丢弃：Age=%d（非 0 即全列查询）", list[0].Age)
	}

	// 分页入口同一契约
	page, err := repo.PageByCond(model.NewCond().Columns("name"), model.PageParam{PageNum: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("page: %v", err)
	}
	if len(page.List) != 1 || page.List[0].Age != 0 {
		t.Fatalf("分页投影被丢弃：Age=%d", page.List[0].Age)
	}

	// 显式列参数优先于 cond 内投影，且自动补主键
	explicit, err := repo.SelectByCond(model.NewCond().Columns("name"), "age")
	if err != nil {
		t.Fatalf("select with cols: %v", err)
	}
	if len(explicit) != 1 || explicit[0].Age != 42 || explicit[0].Id == 0 {
		t.Fatalf("显式 cols 应覆盖 cond 投影并补主键：Age=%d Id=%d", explicit[0].Age, explicit[0].Id)
	}

	// GetByIds 不传 cols 时沿用 cond 的空投影即全列查询，显式 cols 时补主键
	byIds, err := repo.GetByIds([]uint64{created.Id})
	if err != nil {
		t.Fatalf("get by ids: %v", err)
	}
	if len(byIds) != 1 || byIds[0].Age != 42 {
		t.Fatalf("GetByIds 无 cols 时应为全列查询：Age=%d", byIds[0].Age)
	}
	projected, err := repo.GetByIds([]uint64{created.Id}, "name")
	if err != nil {
		t.Fatalf("get by ids with cols: %v", err)
	}
	if len(projected) != 1 || projected[0].Age != 0 || projected[0].Id == 0 {
		t.Fatalf("GetByIds 显式 cols 应生效并补主键：Age=%d Id=%d", projected[0].Age, projected[0].Id)
	}
}
