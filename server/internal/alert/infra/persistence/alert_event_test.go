package persistence

import (
	"context"
	"testing"
	"time"

	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/pkg/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupAlertEventDb(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&entity.AlertEvent{}); err != nil {
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

func newFiringEvent(ruleId, resourceId uint64) *entity.AlertEvent {
	event := &entity.AlertEvent{
		RuleId:       ruleId,
		ResourceId:   resourceId,
		ResourceType: 1,
		Priority:     1,
		Status:       entity.AlertEventStatusFiring,
		ResourceName: "host-a",
		RuleName:     "rule",
	}
	event.FillBaseInfo(nil)
	return event
}

// TestCreateIfNotActiveInsertsOnceAndReportsExisting 首次创建返回「未已存在」，第二次必须报出已存在
func TestCreateIfNotActiveInsertsOnceAndReportsExisting(t *testing.T) {
	db := setupAlertEventDb(t)
	repo := &alertEventRepoImpl{}
	ctx := context.Background()

	created, existed, err := repo.CreateIfNotActive(ctx, newFiringEvent(1, 100))
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	if existed {
		t.Fatal("first create reported an existing active event")
	}
	if created.Id == 0 {
		t.Fatal("created event has no id")
	}

	second, existed, err := repo.CreateIfNotActive(ctx, newFiringEvent(1, 100))
	if err != nil {
		t.Fatalf("second create failed: %v", err)
	}
	if !existed {
		t.Fatal("second create did not report the existing active event")
	}
	if second.Id != created.Id {
		t.Errorf("second call returned event %d, want the existing %d", second.Id, created.Id)
	}

	var count int64
	if err := db.Model(&entity.AlertEvent{}).Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("active events = %d, want 1 (同一规则+资源只能有一条活跃事件)", count)
	}
}

// TestCreateIfNotActiveBlocksWhenGuardHeld 跨实例互斥真的在把关：另一个持有者占住
// (规则, 资源) 的守卫时，本次调用既不能插入也不能静默跳过，必须报错让调用方下一轮重试。
//
// 这条断言同时是「没有退回行锁」的证据：SQLite 会静默丢弃 FOR UPDATE，若实现仍依赖行锁，
// 这里会插入第二条活跃事件而不是报错
func TestCreateIfNotActiveBlocksWhenGuardHeld(t *testing.T) {
	db := setupAlertEventDb(t)
	repo := &alertEventRepoImpl{}
	ctx := context.Background()

	key := activeEventGuardKey(2, 200)
	if !activeEventGuard.Acquire(key) {
		t.Fatal("failed to acquire the guard in the test setup")
	}

	start := time.Now()
	_, _, err := repo.CreateIfNotActive(ctx, newFiringEvent(2, 200))
	waited := time.Since(start)

	activeEventGuard.Release(key)

	if err == nil {
		t.Fatal("CreateIfNotActive inserted an event although another holder owns the guard")
	}
	if waited < activeEventLockWait {
		t.Errorf("returned after %v, want it to wait up to %s before giving up", waited, activeEventLockWait)
	}

	var count int64
	if err := db.Model(&entity.AlertEvent{}).Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("active events = %d, want 0: 抢不到锁时绝不能写入", count)
	}
}

// TestCreateIfNotActiveReleasesGuard 临界区结束后守卫必须释放，否则同资源后续事件永久无法创建
func TestCreateIfNotActiveReleasesGuard(t *testing.T) {
	setupAlertEventDb(t)
	repo := &alertEventRepoImpl{}
	ctx := context.Background()

	if _, _, err := repo.CreateIfNotActive(ctx, newFiringEvent(3, 300)); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	key := activeEventGuardKey(3, 300)
	if !activeEventGuard.Acquire(key) {
		t.Fatal("the guard was not released after CreateIfNotActive returned")
	}
	activeEventGuard.Release(key)
}

// TestCreateIfNotActiveIsolatedPerResource 不同(规则, 资源)互不影响，锁不是全局串行的
func TestCreateIfNotActiveIsolatedPerResource(t *testing.T) {
	db := setupAlertEventDb(t)
	repo := &alertEventRepoImpl{}
	ctx := context.Background()

	if _, _, err := repo.CreateIfNotActive(ctx, newFiringEvent(4, 400)); err != nil {
		t.Fatalf("create for first resource failed: %v", err)
	}
	// 第一条资源仍被持有锁时，第二条资源必须照常创建
	if !activeEventGuard.Acquire(activeEventGuardKey(4, 400)) {
		t.Fatal("setup acquire failed")
	}
	defer activeEventGuard.Release(activeEventGuardKey(4, 400))

	if _, _, err := repo.CreateIfNotActive(ctx, newFiringEvent(4, 401)); err != nil {
		t.Fatalf("create for a different resource was blocked: %v", err)
	}
	var count int64
	if err := db.Model(&entity.AlertEvent{}).Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 2 {
		t.Errorf("events = %d, want 2 (锁按资源划分，不是全局串行)", count)
	}
}

// TestActiveEventLockTuning 守卫的 TTL 与等待窗口必须是显式设定的小值：
// 默认 2h 的锁 TTL 用于长任务，短临界区沿用它会因实例被杀而把该资源的告警阻塞两小时
func TestActiveEventLockTuning(t *testing.T) {
	if activeEventLockTTL > time.Minute {
		t.Errorf("activeEventLockTTL = %v, 短临界区不应使用分钟级以上的锁 TTL", activeEventLockTTL)
	}
	if activeEventGuard.LockTTL != activeEventLockTTL {
		t.Errorf("guard LockTTL = %v, want %v", activeEventGuard.LockTTL, activeEventLockTTL)
	}
	if activeEventLockWait <= activeEventLockPoll {
		t.Fatal("等待窗口必须大于轮询间隔")
	}
}
