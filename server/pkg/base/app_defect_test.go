package base

import (
	"context"
	"errors"
	"testing"

	"mayfly-go/pkg/model"
)

// 条件对象复用维度的回归用例。
//
// QueryCond 是引用语义的构造器，基座入口在消费调用方条件时必须只读：
// 任何就地改写（列投影、追加条件）都会顺着复用泄漏到后续查询，且故障形态是
// 「后面某次查询结果不对」，与肇事代码相隔很远

// TestSelectByCondDoesNotMutateCallerCond 显式列参数不得写进调用方的条件对象：
//
// 条件常被构建一次、多处复用（先查明细再统计），SelectByCond 若把 cols 投影就地
// 烧进 cond，后续不带 cols 的查询与 CountByCond 都会沿用这份意外投影
func TestSelectByCondDoesNotMutateCallerCond(t *testing.T) {
	setupTestDb(t)
	repo := newTestRepo()
	newLoadedRepo(t, repo)

	cond := model.NewCond().Columns("name")
	if _, err := repo.SelectByCond(cond, "age"); err != nil {
		t.Fatalf("select with cols failed: %v", err)
	}
	if got := cond.GetSelectColumns(); len(got) != 1 || got[0] != "name" {
		t.Errorf("caller cond projection mutated by SelectByCond: %v, want [name]", got)
	}

	// 复用同一条件做不带 cols 的查询：必须沿用调用方声明的投影而不是上一次调用的残留
	if _, err := repo.SelectByCond(cond); err != nil {
		t.Fatalf("select without cols failed: %v", err)
	}
	if got := cond.GetSelectColumns(); len(got) != 1 || got[0] != "name" {
		t.Errorf("caller cond projection mutated across calls: %v, want [name]", got)
	}
}

// TestPageByCondDoesNotMutateCallerCond 分页入口与列表入口同口径
func TestPageByCondDoesNotMutateCallerCond(t *testing.T) {
	setupTestDb(t)
	repo := newTestRepo()
	newLoadedRepo(t, repo)

	cond := model.NewCond().Columns("name")
	if _, err := repo.PageByCond(cond, model.PageParam{PageNum: 1, PageSize: 10}, "age"); err != nil {
		t.Fatalf("page with cols failed: %v", err)
	}
	if got := cond.GetSelectColumns(); len(got) != 1 || got[0] != "name" {
		t.Errorf("caller cond projection mutated by PageByCond: %v, want [name]", got)
	}
}

// TestCursorByCondTerminatesWithProjection 游标遍历必须给投影补主键：
//
// 游标按主键推进，条件声明的投影若不含主键，每批回来的实体主键恒为 0，
// 游标边界永远不动，同一批数据被无限重复处理
func TestCursorByCondTerminatesWithProjection(t *testing.T) {
	setupTestDb(t)
	repo := newTestRepo()
	for i := 0; i < 3; i++ {
		if err := repo.Insert(context.Background(), &ReviewEntity{Name: "cursor", Age: i}); err != nil {
			t.Fatalf("insert failed: %v", err)
		}
	}

	cond := model.NewCond().Columns("name")
	var names []string
	err := CursorByCond(repo, cond, 2, func(e *ReviewEntity) error {
		names = append(names, e.Name)
		// 死循环的止损哨兵：正确实现只应恰好处理每个实体一次
		if len(names) > 3 {
			return errors.New("cursor iteration did not terminate")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("cursor failed: %v", err)
	}
	if len(names) != 3 {
		t.Errorf("cursor visited %d entities, want 3 (names=%v)", len(names), names)
	}
}

// TestModelCondEmbeddedSoftDeleteValueIsRespected 条件模型里嵌入体的软删取值必须被识别：
//
// 软删字段（is_deleted）位于嵌入的 model.Model 里，调用方把已删除取值装进条件模型
// 查询回收站数据时，基座若只扫顶层字段就会误判「调用方没就软列表过态」，自动附加的
// is_deleted = 0 与条件模型里的 is_deleted = 1 互相矛盾，结果恒为空
func TestModelCondEmbeddedSoftDeleteValueIsRespected(t *testing.T) {
	setupTestDb(t)
	repo := newTestRepo()
	created := newLoadedRepo(t, repo)

	if err := repo.DeleteByCond(context.Background(), model.NewCond().Eq(model.IdColumn, created.Id)); err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}

	condEntity := repo.NewModel()
	condEntity.IsDeleted = model.ModelDeleted
	var entities []*ReviewEntity
	if err := repo.SelectByCondToAny(condEntity, &entities); err != nil {
		t.Fatalf("select deleted rows failed: %v", err)
	}
	if len(entities) != 1 {
		t.Errorf("got %d deleted rows, want 1: model condition on embedded is_deleted was overridden", len(entities))
	}
}

// TestPageByCondCountsRowsNotNonNullProjection 分页总数必须是行数而不是投影列的非空值数：
// GORM 对单列投影的 Count 会生成 COUNT(该列)，列里出现 NULL 时总数偏小、
// 页尾数据被静默截断
func TestPageByCondCountsRowsNotNonNullProjection(t *testing.T) {
	setupTestDb(t)
	repo := newTestRepo()
	created := newLoadedRepo(t, repo)
	if err := repo.Insert(context.Background(), &ReviewEntity{Name: "second"}); err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	// 补丁的零值写入能力：把一行的 name 置 NULL
	if _, err := repo.UpdateColsByCond(context.Background(),
		NewPatch().Set("name", nil), model.NewCond().Eq(model.IdColumn, created.Id)); err != nil {
		t.Fatalf("null out name failed: %v", err)
	}

	result, err := repo.PageByCond(model.NewCond().Columns("name"), model.PageParam{PageNum: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("page failed: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("page total = %d, want 2: NULL projection column was excluded from count", result.Total)
	}
	if len(result.List) != 2 {
		t.Errorf("page list = %d rows, want 2", len(result.List))
	}
}
