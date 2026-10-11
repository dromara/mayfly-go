package base

import (
	"context"
	"sync/atomic"
	"testing"

	"mayfly-go/pkg/model"
)

// 主键生成策略的回归用例。
//
// 钉住以下行为：实现了 model.IdGenerator 的实体新增时由策略发号；发号发生在 FillBaseInfo
// 之后（主键先非零会让新建被 IsCreate 判成更新，创建审计因此留空）；批量新增逐个发号
// 不重；已带主键的实体更新不被重新发号；未实现策略的实体保持数据库自增

// stubGenRepo 固定值发号桩：与自增的小整数区间明显区分，断言可以精确到值
type stubGenRepo struct {
	model.Model
	Name string `json:"name"`
}

func (*stubGenRepo) TableName() string { return "stub_gen_repo" }

const stubGeneratedId = 9000000000001

func (s *stubGenRepo) GenerateId() uint64 { return stubGeneratedId }

// seqGenRepo 递增发号桩：批量新增要求每个实体拿到不同主键
type seqGenRepo struct {
	model.Model
	Name string `json:"name"`
}

func (*seqGenRepo) TableName() string { return "seq_gen_repo" }

var seqGenCounter atomic.Uint64

func (s *seqGenRepo) GenerateId() uint64 { return 8000000000000 + seqGenCounter.Add(1) }

func TestIdGeneratorAppliesOnInsert(t *testing.T) {
	db := setupTestDb(t)
	if err := db.AutoMigrate(&stubGenRepo{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	repo := &RepoImpl[*stubGenRepo]{}

	entity := &stubGenRepo{Name: "gen"}
	if err := repo.Insert(context.Background(), entity); err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	if entity.Id != stubGeneratedId {
		t.Errorf("id = %d, want strategy-generated %d", entity.Id, stubGeneratedId)
	}
	// 顺序钉桩：创建审计必须按「新建」填。发号若挪到 FillBaseInfo 之前，主键已非零，
	// IsCreate() 判假会让创建人/创建时间整个留空且插入照常成功
	if entity.CreateTime == nil || entity.Creator == "" {
		t.Errorf("create audit not filled: create_time=%v creator=%q", entity.CreateTime, entity.Creator)
	}

	var fromDb stubGenRepo
	if err := db.First(&fromDb, "id = ?", stubGeneratedId).Error; err != nil {
		t.Fatalf("reload row by generated id failed: %v", err)
	}
	if fromDb.Name != "gen" {
		t.Errorf("row content mismatch after generated-id insert: %q", fromDb.Name)
	}
}

func TestIdGeneratorBatchInsertAssignsDistinctIds(t *testing.T) {
	db := setupTestDb(t)
	if err := db.AutoMigrate(&seqGenRepo{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	repo := &RepoImpl[*seqGenRepo]{}

	entities := []*seqGenRepo{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	if err := repo.BatchInsert(context.Background(), entities); err != nil {
		t.Fatalf("batch insert failed: %v", err)
	}
	seen := map[uint64]bool{}
	for _, e := range entities {
		if e.Id == 0 {
			t.Fatalf("batch entity %q got no generated id", e.Name)
		}
		if seen[e.Id] {
			t.Fatalf("batch entities share id %d", e.Id)
		}
		seen[e.Id] = true
	}
}

func TestIdGeneratorNotAppliedOnUpdate(t *testing.T) {
	db := setupTestDb(t)
	if err := db.AutoMigrate(&stubGenRepo{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	repo := &RepoImpl[*stubGenRepo]{}

	entity := &stubGenRepo{Name: "v1"}
	if err := repo.Insert(context.Background(), entity); err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	entity.Name = "v2"
	if err := repo.Save(context.Background(), entity); err != nil {
		t.Fatalf("save existing entity failed: %v", err)
	}
	if entity.Id != stubGeneratedId {
		t.Errorf("existing entity id was regenerated: %d, want %d", entity.Id, stubGeneratedId)
	}
	var count int64
	if err := db.Model(&stubGenRepo{}).Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("save regenerated id and inserted a second row: count = %d, want 1", count)
	}
}

func TestDefaultIdStaysDatabaseAutoIncrement(t *testing.T) {
	setupTestDb(t)
	repo := newTestRepo()

	created := newLoadedRepo(t, repo)
	// ReviewEntity 未实现 IdGenerator：主键必须由数据库自增给出，落在自增小值区间，
	// 不能被任何应用侧策略挪到时间戳量级
	if created.Id == 0 || created.Id > 1000000 {
		t.Errorf("default entity id = %d, want database auto-increment small value", created.Id)
	}
}
