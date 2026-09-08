package transfer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------- distinctRowCount ----------------

func TestDistinctRowCount(t *testing.T) {
	// 头尾抽样重叠（小表）同一行只计一次；NULL主键不计数；主键形态差异（int64/int32）归一后去重
	rows := []map[string]any{
		{"id": int64(1)}, {"id": int64(1)},
		{"id": int32(2)}, {"id": "2"},
		{"id": nil}, {"id": nil},
	}
	assert.Equal(t, 2, distinctRowCount(rows, "id"))
	assert.Equal(t, 0, distinctRowCount(nil, "id"))
}

func srcRow(id int64, val string) map[string]any {
	return map[string]any{"id": id, "val": val}
}

// ---------------- alignColumns ----------------

func TestAlignColumns(t *testing.T) {
	src := map[string]any{"id": 1, "val": "a", "only_src": 1}
	tgt := map[string]any{"ID": 1, "Val": "a", "only_tgt": 2} // 目标列名大小写不同（NameCase场景）

	common := alignColumns(src, tgt)
	require.Len(t, common, 2)
	assert.ElementsMatch(t, []string{"id", "val"}, common)
}

func TestAlignColumns_NoCommon(t *testing.T) {
	assert.Empty(t, alignColumns(map[string]any{"a": 1}, map[string]any{"b": 2}))
}

// ---------------- compareRowValues ----------------

func TestCompareRowValues(t *testing.T) {
	// 跨数值形态等值（mysql int64 vs pg int32等）
	assert.True(t, compareRowValues(
		map[string]any{"id": int64(1), "num": int64(42)},
		map[string]any{"id": int32(1), "num": int32(42)}, nil))

	// 值差异
	assert.False(t, compareRowValues(
		map[string]any{"id": int64(1), "val": "a"},
		map[string]any{"id": int64(1), "val": "b"}, nil))

	// 目标缺列：按公共列交集语义，缺列不由行比对检出（跨方言迁移目标可能增减列）
	assert.True(t, compareRowValues(
		map[string]any{"id": int64(1), "val": "a"},
		map[string]any{"id": int64(1)}, nil))

	// 空公共列 → false
	assert.False(t, compareRowValues(map[string]any{"a": 1}, map[string]any{"b": 2}, nil))

	// nil值等值（两侧均为NULL）
	assert.True(t, compareRowValues(
		map[string]any{"id": int64(1), "val": nil},
		map[string]any{"id": int64(1), "val": nil}, nil))
}

// TestCompareRowValues_NumericScaleTolerance 数值类列按数值语义判等，容忍同一数值的标度呈现差异
// （如mysql/pg numeric(20,6)返回"123.456000"，sqlite NUMERIC亲和后为"123.456"）
func TestCompareRowValues_NumericScaleTolerance(t *testing.T) {
	numericCols := map[string]bool{"amount": true}

	assert.True(t, compareRowValues(
		map[string]any{"amount": "123.456000"},
		map[string]any{"amount": "123.456"}, numericCols), "数值列尾零标度差异不应误报")

	assert.True(t, compareRowValues(
		map[string]any{"amount": float64(123.456)},
		map[string]any{"amount": "123.456000"}, numericCols), "浮点与十进制串表示同一数值应判等")

	assert.False(t, compareRowValues(
		map[string]any{"amount": "123.456"},
		map[string]any{"amount": "123.457"}, numericCols), "数值真实差异必须检出")

	assert.False(t, compareRowValues(
		map[string]any{"amount": "123.456000"},
		map[string]any{"amount": nil}, numericCols), "一侧NULL一侧有值必须检出")
}

// TestCompareRowValues_TextColumnStrict 非数值列（numericCols未含）必须保持严格形态比对，
// 否则"1"与"1.0"这类真实文本差异会被漏报
func TestCompareRowValues_TextColumnStrict(t *testing.T) {
	src := map[string]any{"code": "1", "amount": "123.456000"}
	tgt := map[string]any{"code": "1.0", "amount": "123.456"}

	// code非数值列 → 严格不等；即使amount可宽松判等，整行仍不一致
	assert.False(t, compareRowValues(src, tgt, map[string]bool{"amount": true}), "文本列标度差异不得宽松")

	// 无任何数值列时，amount也走严格比对
	assert.False(t, compareRowValues(src, tgt, nil))
	assert.False(t, compareRowValues(src, tgt, map[string]bool{}))
}

// TestCompareRowValues_CaseInsensitiveNumeric 数值列集合的列名匹配需大小写不敏感（方言返回列名形态不一）
func TestCompareRowValues_CaseInsensitiveNumeric(t *testing.T) {
	assert.True(t, compareRowValues(
		map[string]any{"Amount": "123.456000"},
		map[string]any{"AMOUNT": "123.456"}, map[string]bool{"amount": true}))
}

// ---------------- compareSampledRows ----------------

func TestCompareSampledRows_AllMatch(t *testing.T) {
	src := []map[string]any{srcRow(1, "a"), srcRow(2, "b"), srcRow(3, "c")}
	tgt := []map[string]any{srcRow(3, "c"), srcRow(1, "a"), srcRow(2, "b")} // 顺序无关

	assert.Empty(t, compareSampledRows(src, tgt, "id", "id", nil, VerifyMaxMismatchReport))
}

func TestCompareSampledRows_MismatchAndMissing(t *testing.T) {
	src := []map[string]any{srcRow(1, "a"), srcRow(2, "b"), srcRow(3, "c"), srcRow(4, "d")}
	// id=2值被改；id=4被删
	tgt := []map[string]any{srcRow(1, "a"), srcRow(2, "B"), srcRow(3, "c")}

	mismatch := compareSampledRows(src, tgt, "id", "id", nil, VerifyMaxMismatchReport)
	assert.ElementsMatch(t, []string{"2", "4"}, mismatch)
}

func TestCompareSampledRows_CapAndOrder(t *testing.T) {
	src := make([]map[string]any, 0, 30)
	tgt := make([]map[string]any, 0, 30)
	for i := 1; i <= 30; i++ {
		src = append(src, srcRow(int64(i), "v"))
		if i > 25 { // 前25行不一致（超过上限20）
			tgt = append(tgt, srcRow(int64(i), "v"))
		} else {
			tgt = append(tgt, srcRow(int64(i), "wrong"))
		}
	}

	mismatch := compareSampledRows(src, tgt, "id", "id", nil, VerifyMaxMismatchReport)
	require.Len(t, mismatch, VerifyMaxMismatchReport, "超过上限应截断")
	// 按源侧顺序，前20个不一致主键为1~20
	assert.Equal(t, "1", mismatch[0])
	assert.Equal(t, "20", mismatch[19])
}

func TestCompareSampledRows_PkTypeDifference(t *testing.T) {
	// 主键类型形态不同（int64 vs int32）应通过规范化对齐
	src := []map[string]any{srcRow(int64(7), "x")}
	tgt := []map[string]any{map[string]any{"id": int32(7), "val": "x"}}

	assert.Empty(t, compareSampledRows(src, tgt, "id", "id", nil, VerifyMaxMismatchReport))
}

func TestCompareSampledRows_NilPkSkipped(t *testing.T) {
	// 主键为NULL的异常行跳过（不误报为missing）
	src := []map[string]any{{"id": nil, "val": "a"}, srcRow(1, "a")}
	tgt := []map[string]any{srcRow(1, "a")}

	assert.Empty(t, compareSampledRows(src, tgt, "id", "id", nil, VerifyMaxMismatchReport))
}

func TestCompareSampledRows_NumericScaleNotMismatch(t *testing.T) {
	// 数值列标度呈现差异在启用numericCols后不误报；未启用则报不一致
	src := []map[string]any{{"id": int64(1), "amount": "10.500000"}}
	tgt := []map[string]any{{"id": int64(1), "amount": "10.5"}}

	assert.Empty(t, compareSampledRows(src, tgt, "id", "id", map[string]bool{"amount": true}, VerifyMaxMismatchReport))
	assert.Equal(t, []string{"1"}, compareSampledRows(src, tgt, "id", "id", nil, VerifyMaxMismatchReport))
}

func TestCompareSampledRows_TargetExtraNotReported(t *testing.T) {
	// 目标多出的行不由内容比对检出（由count比对捕获）
	src := []map[string]any{srcRow(1, "a")}
	tgt := []map[string]any{srcRow(1, "a"), srcRow(2, "b")}

	assert.Empty(t, compareSampledRows(src, tgt, "id", "id", nil, VerifyMaxMismatchReport))
}
