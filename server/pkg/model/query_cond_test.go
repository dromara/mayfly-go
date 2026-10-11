package model

import (
	"strings"
	"testing"
)

// 条件构造器的回归用例。
//
// 对应曾实测复现的缺陷：条件放进 map 导致同列条件互相覆盖、SQL 文本随遍历顺序漂移、
// In/NotIn 对空集合的语义与调用方意图相反、In0 与 In 同体（死语义）、
// 排序片段可被塞进任意 SQL、复用同一条件对象会互相污染

func whereExprs(cond *QueryCond) []string {
	exprs := make([]string, 0, len(cond.GetWheres()))
	for _, w := range cond.GetWheres() {
		exprs = append(exprs, w.Expr)
	}
	return exprs
}

// TestSameColumnConditionsAreAllKept map 承载时后写同键条件会静默覆盖前者，
// 「age > 1 AND age > 5」这类区间只能剩下一条
func TestSameColumnConditionsAreAllKept(t *testing.T) {
	cond := NewCond().Gt("age", 1).Gt("age", 5)
	exprs := whereExprs(cond)
	if len(exprs) != 2 {
		t.Fatalf("kept %d conditions, want 2: %+v", len(exprs), exprs)
	}
	for i, want := range []string{"age > ?", "age > ?"} {
		if exprs[i] != want {
			t.Errorf("condition %d = %q, want %q", i, exprs[i], want)
		}
	}
}

// TestConditionOrderIsStable 同一串链式调用必须生成恒定顺序
func TestConditionOrderIsStable(t *testing.T) {
	build := func() string {
		cond := NewCond().
			Eq("instance_id", 1).
			Like("name", "a").
			Eq("status", 2).
			And("extra IS NULL").
			Eq("creator_id", 3)
		return strings.Join(whereExprs(cond), " | ")
	}
	first := build()
	for range 200 {
		if got := build(); got != first {
			t.Fatalf("condition order is not stable:\n %s\n %s", first, got)
		}
	}
}

// TestInDistinguishesMissingFromEmptySet 集合的「没传」与「传了空集合」必须走不同分支：
// 前者跳过条件，后者要恒不命中，绝不能放开全表
func TestInDistinguishesMissingFromEmptySet(t *testing.T) {
	var nilIds []uint64
	emptyIds := []uint64{}

	missing := NewCond().In("id", nilIds)
	if len(missing.GetWheres()) != 0 {
		t.Errorf("nil collection should skip the condition, got %+v", whereExprs(missing))
	}

	empty := NewCond().In("id", emptyIds)
	exprs := whereExprs(empty)
	if len(exprs) != 1 || exprs[0] != "1 = 0" {
		t.Errorf("empty collection should produce an always-false condition, got %+v", exprs)
	}

	// NotIn 的意图相反：没有排除项就等于不排除任何行
	for name, ids := range map[string][]uint64{"nil": nilIds, "empty": emptyIds} {
		cond := NewCond().NotIn("id", ids)
		if len(cond.GetWheres()) != 0 {
			t.Errorf("NotIn with %s collection should skip the condition, got %+v", name, whereExprs(cond))
		}
	}

	if got := NewCond().NotIn("id", []uint64{1}); len(got.GetWheres()) != 1 {
		t.Errorf("NotIn with values should add a condition, got %+v", whereExprs(got))
	}
}

// TestEqStrictKeepsZeroValues 零值过滤需要严格入口，默认的 Eq 是「传了才过滤」
func TestEqStrictKeepsZeroValues(t *testing.T) {
	skip := NewCond().Eq("is_deleted", 0)
	if len(skip.GetWheres()) != 0 {
		t.Errorf("Eq should skip zero values, got %+v", whereExprs(skip))
	}
	strict := NewCond().EqStrict("is_deleted", 0)
	exprs := whereExprs(strict)
	if len(exprs) != 1 || exprs[0] != "is_deleted = ?" {
		t.Errorf("EqStrict must keep the zero value, got %+v", exprs)
	}
}

// TestIsNullHasDedicatedEntry Eq(col, nil) 会被当成「未提供条件」，判空必须有独立入口
func TestIsNullHasDedicatedEntry(t *testing.T) {
	if len(NewCond().Eq("delete_time", nil).GetWheres()) != 0 {
		t.Error("Eq(nil) must stay a no-op, otherwise optional filters break")
	}
	exprs := whereExprs(NewCond().IsNull("deleteTime"))
	if len(exprs) != 1 || exprs[0] != "delete_time IS NULL" {
		t.Errorf("IsNull produced %+v, want a single \"delete_time IS NULL\"", exprs)
	}
	if got := len(whereExprs(NewCond().NotNull("deleteTime"))); got != 1 {
		t.Errorf("NotNull produced %d conditions, want 1", got)
	}
}

// TestOrderByRejectsInjection 排序片段无法参数化，只能靠白名单拦在生成 SQL 之前
func TestOrderByRejectsInjection(t *testing.T) {
	valid := []string{"id", "id DESC", "create_time asc", "t.code_path DESC"}
	for _, fragment := range valid {
		cond := NewCond().OrderBy(fragment)
		if cond.GetErr() != nil {
			t.Errorf("valid order by %q rejected: %v", fragment, cond.GetErr())
		}
		if len(cond.GetOrderBy()) != 1 {
			t.Errorf("valid order by %q was not kept: %+v", fragment, cond.GetOrderBy())
		}
	}

	// 注入面：这些片段会原样出现在 ORDER BY 之后，必须全部拒绝
	injections := []string{
		"id; DROP TABLE t_sys_account",
		"(SELECT 1)",
		"id ASC, (SELECT password FROM t_sys_account)",
		"-- comment",
		"",
	}
	for _, fragment := range injections {
		cond := NewCond().OrderBy(fragment)
		if cond.GetErr() == nil {
			t.Errorf("order by %q was accepted", fragment)
		}
		if len(cond.GetOrderBy()) != 0 {
			t.Errorf("order by %q leaked into the clause list: %+v", fragment, cond.GetOrderBy())
		}
	}
}

// TestCloneIsolatesTemporaryConditions 游标/分页等临时条件必须加在副本上，不污染调用方条件
func TestCloneIsolatesTemporaryConditions(t *testing.T) {
	original := NewCond().Eq("status", 1)
	derived := original.Clone().Lt("id", 100)

	if len(original.GetWheres()) != 1 {
		t.Errorf("original condition was mutated: %+v", whereExprs(original))
	}
	if len(derived.GetWheres()) != 2 {
		t.Errorf("derived condition lost its base: %+v", whereExprs(derived))
	}
	// 再克隆一次，确认切片是独立副本而非共享底层数组
	other := original.Clone().Gt("age", 10)
	if len(original.GetWheres()) != 1 || len(other.GetWheres()) != 2 {
		t.Errorf("clones share the backing array: %+v / %+v", whereExprs(original), whereExprs(other))
	}
}

// TestOrderByAccumulatesAcrossCalls 多级排序靠追加而非覆盖
func TestOrderByAccumulatesAcrossCalls(t *testing.T) {
	cond := NewCond().OrderByAsc("pid").OrderByDesc("weight")
	got := cond.GetOrderBy()
	if len(got) != 2 || got[0] != "pid ASC" || got[1] != "weight DESC" {
		t.Fatalf("order by = %+v, want [pid ASC, weight DESC]", got)
	}
}

// TestDeletedColumnDetection 自动软删过滤依据「调用方是否已就该列表过态」，判断必须可靠
func TestDeletedColumnDetection(t *testing.T) {
	if !NewCond().Eq("is_deleted", 1).HasCondOnColumn(DeletedColumn) {
		t.Error("explicit Eq on the deleted column was not detected")
	}
	if !NewCond().Eq("isDeleted", 1).HasCondOnColumn(DeletedColumn) {
		t.Error("camelCase spelling of the deleted column was not detected")
	}
	if NewCond().Eq("status", 1).HasCondOnColumn(DeletedColumn) {
		t.Error("unrelated column reported as the deleted column")
	}
	if NewCond().IsDeletedIgnored() {
		t.Error("a fresh condition must not ignore the deleted filter")
	}
	if !NewCond().IncludeDeleted().IsDeletedIgnored() {
		t.Error("IncludeDeleted did not take effect")
	}
}

// TestBetweenDegradesGracefully 区间两端可缺省，两端都缺省则整条跳过
func TestBetweenDegradesGracefully(t *testing.T) {
	if got := len(whereExprs(NewCond().Between("collect_time", 0, 0))); got != 0 {
		t.Errorf("empty Between produced %d conditions, want 0", got)
	}
	oneSided := whereExprs(NewCond().Between("collect_time", 0, 10))
	if len(oneSided) != 1 || oneSided[0] != "collect_time <= ?" {
		t.Errorf("one-sided Between produced %+v", oneSided)
	}
	full := whereExprs(NewCond().Between("collect_time", 1, 10))
	if len(full) != 1 || full[0] != "collect_time BETWEEN ? AND ?" {
		t.Errorf("Between produced %+v, want a single range condition", full)
	}
}

// TestCondNilDestIsReportedNotPanicked 结果集缺失要能被检测（由执行层转成错误），不靠 panic
func TestCondNilDestIsReportedNotPanicked(t *testing.T) {
	if NewCond().HasDest() {
		t.Error("an empty condition must not claim to have a destination")
	}
	if got := NewCond().GetDest(); got != nil {
		t.Errorf("GetDest = %v, want nil so the caller can turn it into an error", got)
	}

	type query struct{ Name string }
	withModel := NewModelCond(&query{Name: "x"})
	if !withModel.HasDest() || withModel.GetDest() == nil {
		t.Error("the cond model should serve as the destination when no dest was bound")
	}
}

// TestHasCondOnColumnIdentifiesFragmentShapes 片段识别不能依赖「列名后紧跟空白」的书写形态：
// "is_deleted=1"、表限定前缀与大小写差异都是合法写法，识别不了会让自动软删过滤
// 追加到调用方自己表达的条件上，「查已删除数据」静默查不到任何行
func TestHasCondOnColumnIdentifiesFragmentShapes(t *testing.T) {
	detected := map[string]string{
		"无空格":      "is_deleted=1",
		"表限定前缀":    "t.is_deleted = 1",
		"大写驼峰":     "IsDeleted = ?",
		"纯列名":      "is_deleted",
		"后随空白与制表符": " is_deleted\t= 1",
	}
	for name, expr := range detected {
		if !NewCond().And(expr).HasCondOnColumn(DeletedColumn) {
			t.Errorf("%s写法 %q 未被识别为软删列条件", name, expr)
		}
	}

	// 相近列名不得误认：识别错方向的后果是自动过滤被静默关闭
	for _, expr := range []string{"is_deleted_x = 1", "deleted = 1", "delete_time IS NULL"} {
		if NewCond().And(expr).HasCondOnColumn(DeletedColumn) {
			t.Errorf("无关片段 %q 被误认为软删列条件", expr)
		}
	}
}

// TestColumnsCopiesInputAndZeroArgsKeepProjection 投影声明不能与调用方切片共享底层数组，
// 也不能被零参调用清空：基座读取入口以零参展开表示「沿用条件里的投影」
func TestColumnsCopiesInputAndZeroArgsKeepProjection(t *testing.T) {
	cols := []string{"name"}
	cond := NewCond().Columns(cols...)
	cols[0] = "age"
	if got := cond.GetSelectColumns(); len(got) != 1 || got[0] != "name" {
		t.Errorf("调用方修改自己的切片后投影被连带改写: %v", got)
	}

	cond.Columns()
	if got := cond.GetSelectColumns(); len(got) != 1 || got[0] != "name" {
		t.Errorf("零参 Columns() 清空了既有投影: %v", got)
	}
}

// TestHasCondOnColumnAcceptsCamelCaseProbe 拿实体字段名（驼峰）提问与拿物理列名
// 提问同样成立：两种写法都是调用方的自然表达，识别不了会再次触发自动过滤覆盖调用方条件
func TestHasCondOnColumnAcceptsCamelCaseProbe(t *testing.T) {
	cond := NewCond().Eq("isDeleted", 1) // 片段实际为 "is_deleted = ?"
	if !cond.HasCondOnColumn("isDeleted") {
		t.Error("驼峰列名提问未被识别")
	}
	if !cond.HasCondOnColumn("is_deleted") {
		t.Error("蛇形列名提问未被识别")
	}
}

// embedProbeModel 软删列在嵌入的 Model 里，是实体条件模型的常规形态
type embedProbeModel struct {
	Model
	Name string
}

// TestHasCondOnColumnSeesEmbeddedModelValue 嵌入体的字段会被 GORM 展平成条件，
// 表态判定必须同样下钻：漏扫会让「用条件模型查已删除数据」被自动软删过滤覆盖而恒空
func TestHasCondOnColumnSeesEmbeddedModelValue(t *testing.T) {
	cond := NewModelCond(&embedProbeModel{Model: Model{IsDeleted: 1}})
	if !cond.HasCondOnColumn(DeletedColumn) {
		t.Error("嵌入体里的 is_deleted 取值未被识别为已表态")
	}
	if !cond.HasConds() {
		t.Error("嵌入体非零取值未被计为条件")
	}

	// 零值仍应视为未表态：自动过滤对未表态的列照常附加
	if NewModelCond(&embedProbeModel{}).HasCondOnColumn(DeletedColumn) {
		t.Error("零值嵌入体被误判为已表态")
	}
}

// embedTagProbeModel 覆盖非匿名嵌入与 ORM 忽略字段的口径
type embedTagProbeModel struct {
	Extra string `gorm:"-"`
	Info  struct {
		Status int
	} `gorm:"embedded"`
	Hidden struct {
		Status int
	} `gorm:"embedded:false"`
}

func TestModelCondColumnScanFollowsORMTag(t *testing.T) {
	cond := NewModelCond(&embedTagProbeModel{
		Extra: "ignored",
		Info:  struct{ Status int }{Status: 1},
	})
	// gorm:"-" 字段不参与条件，不得当成表态
	if NewModelCond(&embedTagProbeModel{Extra: "ignored"}).HasCondOnColumn("extra") {
		t.Error("gorm:- 字段被当成条件表态")
	}
	// gorm:"embedded" 展平参与条件
	if !cond.HasCondOnColumn("status") {
		t.Error("显式 embedded 字段内的取值未被识别")
	}
	// embedded:false 关闭展平，字段本身不映射任何列
	hidden := &embedTagProbeModel{Hidden: struct{ Status int }{Status: 1}}
	if NewModelCond(hidden).HasCondOnColumn("status") {
		t.Error("embedded:false 的字段被当作表态来源")
	}
}
