package persistence

// 表访问控制仓储：规则落在 json 列（角色集合），这里盯两件事——
//   1. 角色集合能原样存取（serializer:json 在 sqlite 上也成立）；
//   2. 按库过滤且只返回未删除行（跨库串数据等于把 A 库的保护送给 B 库）。

import (
	"context"
	"testing"

	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTableAclRepo(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&entity.DbTableAcl{}); err != nil {
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

func TestDbTableAclRepoListByDb(t *testing.T) {
	setupTableAclRepo(t)
	repo := newDbTableAclRepo()
	ctx := contextx.WithLoginAccount(context.Background(), &model.LoginAccount{Id: 1, Username: "tester"})

	for _, acl := range []*entity.DbTableAcl{
		{DbId: 7, Table: "t_order", RoleIds: []uint64{3, 5}, Remark: "财务"},
		{DbId: 7, Table: "t_account"},
		{DbId: 9, Table: "t_order", RoleIds: []uint64{3}},
	} {
		if err := repo.Insert(ctx, acl); err != nil {
			t.Fatalf("insert failed: %v", err)
		}
	}

	rules, err := repo.ListByDb(7)
	if err != nil {
		t.Fatalf("ListByDb failed: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("only the rules of db 7 should be returned, got %d", len(rules))
	}
	// 按表名升序：界面上的顺序不能随插入顺序漂移
	if rules[0].Table != "t_account" || rules[1].Table != "t_order" {
		t.Fatalf("rules must be ordered by table name, got %s, %s", rules[0].Table, rules[1].Table)
	}
	if len(rules[1].RoleIds) != 2 || rules[1].RoleIds[0] != 3 || rules[1].RoleIds[1] != 5 {
		t.Fatalf("role ids must survive the json column round-trip, got %v", rules[1].RoleIds)
	}
	// 空角色集合要能区分「所有人」而不是读出脏数据
	if len(rules[0].RoleIds) != 0 {
		t.Fatalf("an empty role set must stay empty, got %v", rules[0].RoleIds)
	}

	// 删除后不再出现在列表里（逻辑删除由基座追加）
	if err := repo.DeleteById(ctx, rules[0].Id); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	after, err := repo.ListByDb(7)
	if err != nil {
		t.Fatalf("ListByDb failed: %v", err)
	}
	if len(after) != 1 || after[0].Table != "t_order" {
		t.Fatalf("a deleted rule must disappear from the list, got %+v", after)
	}
}
