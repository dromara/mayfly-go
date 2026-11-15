package persistence

// 业务词典仓储：表名清单落在 json 列（table_names），这里盯两件事——
//  1. 清单能原样往返（serializer:json 在 sqlite 上也成立），空清单不变成脏值；
//  2. 按库资源过滤且按术语稳定排序（跨库串数据等于把 A 库的口径送给 B 库生成）。

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

func setupAiTermRepo(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&entity.DbAiTerm{}); err != nil {
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

func TestDbAiTermRepoRoundTrip(t *testing.T) {
	setupAiTermRepo(t)
	repo := newDbAiTermRepo()
	ctx := contextx.WithLoginAccount(context.Background(), &model.LoginAccount{Id: 1, Username: "tester"})

	entries := []*entity.DbAiTerm{
		{DbId: 7, Schema: "", Term: "金额单位", Tables: nil, Details: "分"},
		{DbId: 7, Schema: "demo", Term: "实例", Tables: []string{"t_db_instance", "t_es_instance"}, Details: "只算未删除的"},
		{DbId: 9, Schema: "other", Term: "订单", Tables: []string{"t_order"}},
	}
	for _, entry := range entries {
		if err := repo.Insert(ctx, entry); err != nil {
			t.Fatalf("insert failed: %v", err)
		}
	}

	got, err := repo.ListByDb(7)
	if err != nil {
		t.Fatalf("ListByDb failed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("only the entries of db 7 should be returned, got %d", len(got))
	}
	// 按术语升序：界面上的顺序不能随插入顺序漂移
	if got[0].Term != "实例" || got[1].Term != "金额单位" {
		t.Fatalf("unexpected order: %s, %s", got[0].Term, got[1].Term)
	}
	// schema_name 列名映射与「空串=全库共用」的语义都要能原样读回
	if got[0].Schema != "demo" || got[1].Schema != "" {
		t.Fatalf("schema round-trip broken, got %q / %q", got[0].Schema, got[1].Schema)
	}
	if len(got[0].Tables) != 2 || got[0].Tables[0] != "t_db_instance" {
		t.Fatalf("table list must survive the json column round-trip, got %v", got[0].Tables)
	}
	// 空清单读回来必须是空而不是一个脏元素
	if len(got[1].Tables) != 0 {
		t.Fatalf("an empty table list must stay empty, got %v", got[1].Tables)
	}
}
