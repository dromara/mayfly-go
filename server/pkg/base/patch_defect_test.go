package base

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"

	"gorm.io/gorm/clause"
)

// 更新补丁的回归用例。
//
// 补丁存在的理由：实体形态更新沿用「只写非零字段」，因此「把某个字段清空」在基座里做不到且不报错。
// 实测有业务用「把字段置空来防止它被更新」反向绕开这个语义，说明这个缺口一直在被手工规避

func TestPatchSetWritesZeroValues(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()
	created := newLoadedRepo(t, repo)

	// 实体形态：零值字段被静默跳过（既有语义，作为对照钉住，避免有人把它当 bug「修反」）
	entity := created
	entity.Name = ""
	entity.Age = 0
	if err := repo.Save(context.Background(), entity); err != nil {
		t.Fatalf("entity save failed: %v", err)
	}
	var afterEntity ReviewEntity
	if err := db.Table("review_entity").First(&afterEntity, "id = ?", created.Id).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if afterEntity.Name != "origin" || afterEntity.Age != 10 {
		t.Fatalf("entity update unexpectedly wrote zero values: name=%q age=%d", afterEntity.Name, afterEntity.Age)
	}

	// 补丁形态：显式声明的列即使是零值也必须真实写入
	patch := NewPatch().Set("name", "").Set("age", 0)
	if _, err := repo.UpdateColsByCond(context.Background(), patch, model.NewCond().Eq(model.IdColumn, created.Id)); err != nil {
		t.Fatalf("patch update failed: %v", err)
	}
	var afterPatch ReviewEntity
	if err := db.Table("review_entity").First(&afterPatch, "id = ?", created.Id).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if afterPatch.Name != "" || afterPatch.Age != 0 {
		t.Errorf("patch did not clear the columns: name=%q age=%d", afterPatch.Name, afterPatch.Age)
	}
}

func TestPatchIncIsAtomicOnDatabaseSide(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()
	created := newLoadedRepo(t, repo)

	for range 5 {
		patch := NewPatch().Inc("age", 2)
		if _, err := repo.UpdateColsByCond(context.Background(), patch, model.NewCond().Eq(model.IdColumn, created.Id)); err != nil {
			t.Fatalf("increment failed: %v", err)
		}
	}
	var after ReviewEntity
	if err := db.Table("review_entity").First(&after, "id = ?", created.Id).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if after.Age != 20 {
		t.Errorf("age = %d, want 10 + 5*2 = 20", after.Age)
	}
}

// TestPatchRejectsReservedAndUnknownColumns 主键/软删列与不存在的列都必须被拦下。
//
// 列名写错如果放任到 SQL 层，最坏情况不是报错而是静默写进了另一个名字相近的合法列
func TestPatchRejectsReservedAndUnknownColumns(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()
	created := newLoadedRepo(t, repo)

	cases := map[string]struct {
		patch Patch
		want  error
	}{
		"primary key":    {NewPatch().Set(model.IdColumn, 999), ErrReservedPatchColumn},
		"deleted flag":   {NewPatch().Set(model.DeletedColumn, 1), ErrReservedPatchColumn},
		"delete time":    {NewPatch().Set(model.DeleteTimeColumn, nil), ErrReservedPatchColumn},
		"unknown column": {NewPatch().Set("no_such_column", 1), gormx.ErrUnknownColumn},
		"empty patch":    {NewPatch(), ErrEmptyPatch},
		// 注入形态的列名必须被白名单在构造期就拒掉（区别于「列不存在」的 schema 校验），
		// 否则两种故障在断言里无法区分，白名单被绕过也测不出来
		"injected column": {NewPatch().Set("age; DROP TABLE review_entity --", 1), ErrInvalidPatchColumn},
	}

	for name, tc := range cases {
		_, err := repo.UpdateColsByCond(context.Background(), tc.patch, model.NewCond().Eq(model.IdColumn, created.Id))
		if err == nil {
			t.Errorf("%s: expected an error, got nil", name)
			continue
		}
		if tc.want != nil && !errors.Is(err, tc.want) {
			t.Errorf("%s: error = %v, want %v", name, err, tc.want)
		}
	}

	// 表必须完好，且原行没有被任何一次失败写入改动
	var after ReviewEntity
	if err := db.Table("review_entity").First(&after, "id = ?", created.Id).Error; err != nil {
		t.Fatalf("table damaged or row lost: %v", err)
	}
	if after.Id != created.Id || after.Name != "origin" {
		t.Errorf("row changed by a rejected patch: id=%d name=%q", after.Id, after.Name)
	}
}

// TestPatchColumnOrderIsStable map 入参的遍历顺序随机，补丁必须先按列名排序，
// 否则同一份入参会生成多种 SQL 文本
func TestPatchColumnOrderIsStable(t *testing.T) {
	setupTestDb(t)
	patch := NewPatch().Sets(map[string]any{"zebra": 1, "alpha": 2, "middle": 3})
	columns := patch.Columns()
	for i := 1; i < len(columns); i++ {
		if columns[i-1] > columns[i] {
			t.Fatalf("patch columns are not sorted: %v", columns)
		}
	}
}

// TestPatchAuditColumnsAreFilled 补丁入口统一补审计列，调用方只需声明业务列
func TestPatchAuditColumnsAreFilled(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()
	created := newLoadedRepo(t, repo)

	ctx := contextWithAccount(11, "auditor")
	if _, err := repo.UpdateColsByCond(ctx, NewPatch().Set("age", 44), model.NewCond().Eq(model.IdColumn, created.Id)); err != nil {
		t.Fatalf("patch update failed: %v", err)
	}
	var after ReviewEntity
	if err := db.Table("review_entity").First(&after, "id = ?", created.Id).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if after.Modifier != "auditor" || after.ModifierId != 11 {
		t.Errorf("patch did not fill the modifier audit columns: modifier=%q id=%d", after.Modifier, after.ModifierId)
	}
	if after.UpdateTime == nil {
		t.Error("patch did not fill update_time")
	}
}

// TestPatchInvalidColumnErrorNamesTheColumn 非法列名的拒绝必须带出列名本身：
// 否则宽补丁里哪一列出了问题无从定位，只能人肉二分
func TestPatchInvalidColumnErrorNamesTheColumn(t *testing.T) {
	setupTestDb(t)
	repo := newTestRepo()
	created := newLoadedRepo(t, repo)

	_, err := repo.UpdateColsByCond(context.Background(),
		NewPatch().Set("age", 1).Set("name; DROP TABLE review_entity --", 2),
		model.NewCond().Eq(model.IdColumn, created.Id))
	if err == nil {
		t.Fatal("expected ErrInvalidPatchColumn, got nil")
	}
	if !errors.Is(err, ErrInvalidPatchColumn) {
		t.Fatalf("error = %v, want ErrInvalidPatchColumn", err)
	}
	// 断言带列名独有片段而非 "name"：哨兵文本本身就含 name 一词，那会恒真
	if !strings.Contains(err.Error(), "name;") {
		t.Errorf("拒绝错误必须指出非法列名，got: %v", err)
	}
}

// TestCursorUsesPrimaryKeyKeyset 游标遍历：按主键推进，且拒绝带自定义排序的条件
func TestCursorUsesPrimaryKeyKeyset(t *testing.T) {
	setupTestDb(t)
	repo := newTestRepo()
	app := newTestApp(repo)
	ctx := context.Background()

	for _, name := range []string{"c1", "c2", "c3", "c4"} {
		if err := repo.Insert(ctx, &ReviewEntity{Name: name, Age: 1}); err != nil {
			t.Fatalf("insert failed: %v", err)
		}
	}

	var visited []string
	if err := app.CursorByCond(model.NewCond(), func(entity *ReviewEntity) error {
		visited = append(visited, entity.Name)
		return nil
	}); err != nil {
		t.Fatalf("cursor failed: %v", err)
	}
	if len(visited) != 4 {
		t.Errorf("visited %d rows, want 4", len(visited))
	}

	// 自定义排序与主键游标语义冲突，必须报错而不是偷偷改写排序
	if err := app.CursorByCond(model.NewCond().OrderByAsc("name"), func(*ReviewEntity) error { return nil }); !errors.Is(err, ErrCursorOrder) {
		t.Errorf("error = %v, want ErrCursorOrder", err)
	}
}

// TestPatchDerivedFromSameBaseDoNotAlias 从同一份基补丁派生出的分支必须互不影响。
//
// Patch 是值接收者 + 链式返回自身，这个形态明确邀请「先攒一份公共列，再按分支各加几列」的写法。
// 若 append 复用了富余容量，分支之间会写进同一个底层数组并互相改写（实测：基补丁 3 列时
// p1=base.Set("x")、p2=base.Set("y")，p1 里的 x 会变成 y），而且不报任何错——直接更新到错的列上
func TestPatchDerivedFromSameBaseDoNotAlias(t *testing.T) {
	setupTestDb(t)

	basePatch := NewPatch().Sets(map[string]any{"a": 1, "b": 2, "c": 3})
	first := basePatch.Set("x", "one")
	second := basePatch.Set("y", "two")

	if got, want := first.Columns(), []string{"a", "b", "c", "x"}; !slices.Equal(got, want) {
		t.Errorf("first.Columns() = %v, want %v：派生分支被另一个分支改写了", got, want)
	}
	if got, want := second.Columns(), []string{"a", "b", "c", "y"}; !slices.Equal(got, want) {
		t.Errorf("second.Columns() = %v, want %v", got, want)
	}

	// 值也要跟着各自独立：编译成更新值后 x/y 不能互换
	firstValues, err := first.toValues()
	if err != nil {
		t.Fatalf("toValues failed: %v", err)
	}
	if firstValues["x"] != "one" {
		t.Errorf("first values[x] = %v, want %q", firstValues["x"], "one")
	}
	if _, leaked := firstValues["y"]; leaked {
		t.Error("first 里出现了 second 的列：两个补丁共用了底层数组")
	}
}

// TestPatchIncDerivedBranchKeepsOwnColumn Inc 与 Set 混用时同样不能串列：
// Inc 会把列名拼进 SQL 片段，串了就是「把另一列自增」
func TestPatchIncDerivedBranchKeepsOwnColumn(t *testing.T) {
	setupTestDb(t)

	basePatch := NewPatch().Sets(map[string]any{"a": 1, "b": 2, "c": 3})
	incAge := basePatch.Inc("age", 1)
	setName := basePatch.Set("name", "fixed")

	incValues, err := incAge.toValues()
	if err != nil {
		t.Fatalf("toValues failed: %v", err)
	}
	expr, ok := incValues["age"].(clause.Expr)
	if !ok {
		t.Fatalf("values[age] = %#v, want a clause.Expr：自增必须落到 age 列上", incValues["age"])
	}
	if expr.SQL != "age + ?" {
		t.Errorf("expr.SQL = %q, want %q", expr.SQL, "age + ?")
	}
	if _, leaked := incValues["name"]; leaked {
		t.Error("incAge 里出现了另一个分支的列")
	}

	setValues, err := setName.toValues()
	if err != nil {
		t.Fatalf("toValues failed: %v", err)
	}
	if setValues["name"] != "fixed" {
		t.Errorf("setValues[name] = %v, want %q", setValues["name"], "fixed")
	}
	if _, leaked := setValues["age"]; leaked {
		t.Error("setName 里出现了另一个分支的列")
	}
}
