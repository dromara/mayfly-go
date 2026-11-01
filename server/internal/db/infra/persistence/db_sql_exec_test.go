package persistence

// 执行记录读取面的资源范围收敛：ScopeDbIds 是必填的可见库集合，
// 未注入时必须返回空页而不是退化成「全平台可见」。
//
// 这条用例盯的是数据外泄面而不是分页细节：执行记录里含 SQL 原文与行级旧/新值，
// 一旦「忘了注入」等于把全平台业务数据摊给任何已登录账号。

import (
	"context"
	"testing"
	"time"

	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupExecRepo(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&entity.DbSQLExec{}); err != nil {
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

func TestDbSQLExecScopeIsFailClosed(t *testing.T) {
	setupExecRepo(t)
	repo := newDbSQLExecRepo()

	rows := []*entity.DbSQLExec{
		{DbId: 1, Db: "shop", SQL: "update t set a = 1", Type: entity.DbSQLExecTypeUpdate},
		{DbId: 2, Db: "crm", SQL: "delete from u", Type: entity.DbSQLExecTypeDelete},
	}
	// 走仓储 Insert，让 fillBaseInfo 补齐 create_time/creator 等 NOT NULL 审计列
	ctx := contextx.WithLoginAccount(context.Background(), &model.LoginAccount{Id: 7, Username: "tester"})
	for _, row := range rows {
		if err := repo.Insert(ctx, row); err != nil {
			t.Fatalf("insert exec record: %v", err)
		}
	}
	page := model.PageParam{PageNum: 1, PageSize: 10}

	// 未注入范围：必须空页，不能退化成「全平台可见」
	unscoped, err := repo.GetPageList(&entity.DbSQLExecQuery{PageParam: page})
	if err != nil {
		t.Fatalf("query without scope: %v", err)
	}
	if unscoped.Total != 0 || len(unscoped.List) != 0 {
		t.Fatalf("missing scope must yield an empty page, got total=%d", unscoped.Total)
	}

	// 注入单个库：只返回该库的记录
	scoped, err := repo.GetPageList(&entity.DbSQLExecQuery{PageParam: page, ScopeDbIds: []uint64{1}})
	if err != nil {
		t.Fatalf("query with scope: %v", err)
	}
	if scoped.Total != 1 || len(scoped.List) != 1 || scoped.List[0].DbId != 1 {
		t.Fatalf("scope [1] must only return records of db 1, got total=%d", scoped.Total)
	}

	// 显式指定他人库：范围与条件取交集后为空，不因为「传了 dbId」而绕过范围
	denied, err := repo.GetPageList(&entity.DbSQLExecQuery{PageParam: page, DbId: 2, ScopeDbIds: []uint64{1}})
	if err != nil {
		t.Fatalf("query with foreign db id: %v", err)
	}
	if denied.Total != 0 {
		t.Fatalf("db id outside the scope must not be readable, got total=%d", denied.Total)
	}
}

// TestDbSQLExecCreatorFilterBoundFromQuery 操作人筛选必须能从查询参数绑定。
//
// 之前 CreatorId 没有 form tag，仓储层虽然写了 Eq("creator_id", ...)，但值永远是 0 —— 界面上
// 按人审计的能力等于不存在，而这恰恰是合规排查的第一个动作。
func TestDbSQLExecCreatorFilterBoundFromQuery(t *testing.T) {
	setupExecRepo(t)
	repo := newDbSQLExecRepo()

	rows := []*entity.DbSQLExec{
		{DbId: 1, Db: "shop", SQL: "update t set a = 1", Type: entity.DbSQLExecTypeUpdate, CreatorId: 7},
		{DbId: 1, Db: "shop", SQL: "update t set a = 2", Type: entity.DbSQLExecTypeUpdate, CreatorId: 8},
	}
	ctx := contextx.WithLoginAccount(context.Background(), &model.LoginAccount{Id: 7, Username: "tester"})
	for i, row := range rows {
		// Creator 由 fillBaseInfo 按上下文写入；这里需要两个不同操作人，故第二行改上下文
		rowCtx := ctx
		if i == 1 {
			rowCtx = contextx.WithLoginAccount(context.Background(), &model.LoginAccount{Id: 8, Username: "other"})
		}
		if err := repo.Insert(rowCtx, row); err != nil {
			t.Fatalf("insert exec record: %v", err)
		}
	}

	res, err := repo.GetPageList(&entity.DbSQLExecQuery{
		PageParam:  model.PageParam{PageNum: 1, PageSize: 10},
		ScopeDbIds: []uint64{1},
		CreatorId:  7,
	})
	if err != nil {
		t.Fatalf("query by creator: %v", err)
	}
	if res.Total != 1 || len(res.List) != 1 || res.List[0].CreatorId != 7 {
		t.Fatalf("creator filter must narrow to that operator, got total=%d", res.Total)
	}
}

// TestDbSQLExecDeleteBeforeRemovesOnlyExpired 保留期清理必须只删过期行，且是物理删除。
//
// 软删在这里是假清理：行只是被标记为不可见，表照样无限增长，保留期形同虚设。
func TestDbSQLExecDeleteBeforeRemovesOnlyExpired(t *testing.T) {
	db := setupExecRepo(t)
	repo := newDbSQLExecRepo()
	ctx := contextx.WithLoginAccount(context.Background(), &model.LoginAccount{Id: 7, Username: "tester"})

	oldRow := &entity.DbSQLExec{DbId: 1, Db: "shop", SQL: "update t set a = 1", Type: entity.DbSQLExecTypeUpdate}
	freshRow := &entity.DbSQLExec{DbId: 1, Db: "shop", SQL: "update t set a = 2", Type: entity.DbSQLExecTypeUpdate}
	for _, row := range []*entity.DbSQLExec{oldRow, freshRow} {
		if err := repo.Insert(ctx, row); err != nil {
			t.Fatalf("insert exec record: %v", err)
		}
	}
	// 把第一行改成 10 天前（Insert 会按当前时间填 create_time）
	tenDaysAgo := time.Now().AddDate(0, 0, -10)
	if err := db.Model(&entity.DbSQLExec{}).Where("id = ?", oldRow.Id).Update("create_time", tenDaysAgo).Error; err != nil {
		t.Fatalf("backdate create_time: %v", err)
	}

	removed, err := repo.DeleteBefore(ctx, time.Now().AddDate(0, 0, -5))
	if err != nil {
		t.Fatalf("delete before: %v", err)
	}
	if removed != 1 {
		t.Fatalf("only the expired row should be removed, got %d", removed)
	}

	var visible int64
	if err := db.Model(&entity.DbSQLExec{}).Count(&visible).Error; err != nil {
		t.Fatalf("count visible: %v", err)
	}
	var includingDeleted int64
	if err := db.Unscoped().Model(&entity.DbSQLExec{}).Count(&includingDeleted).Error; err != nil {
		t.Fatalf("count scoped: %v", err)
	}
	if visible != 1 || includingDeleted != 1 {
		t.Fatalf("cleanup must physically delete the expired row (visible=%d unscoped=%d)", visible, includingDeleted)
	}
}
