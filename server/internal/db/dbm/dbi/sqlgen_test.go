package dbi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeGen 实现 SQLGenerator 必填方法（嵌入 DefaultSQLGenerator 复用 GenTruncate/GenBatchDelete 默认实现），
// 但【不】实现 ParamInserter，代表尚未接入参数化直插的方言。
type fakeGen struct {
	DefaultSQLGenerator
}

func (fakeGen) GenTableDDL(Table, []Column, bool) []string                          { return nil }
func (fakeGen) GenIndexDDL(Table, []Index) []string                                 { return nil }
func (fakeGen) GenInsert(string, []Column, [][]any, int, *TargetTableMeta) []string { return nil }

// fakeParamGen 额外实现 ParamInserter，代表已接入参数化直插的方言（如 mysql）。
type fakeParamGen struct {
	fakeGen
}

func (fakeParamGen) GenInsertParams(string, []Column, [][]any) (string, []any, error) {
	return "INSERT", []any{1}, nil
}

var (
	_ SQLGenerator  = (*fakeGen)(nil)
	_ SQLGenerator  = (*fakeParamGen)(nil)
	_ ParamInserter = (*fakeParamGen)(nil)
)

// GetParamInserter：具备能力返回可用实例，不具备返回 nil（迁移核心据此回退文本路径）。
func TestGetParamInserter(t *testing.T) {
	assert.NotNil(t, GetParamInserter(&fakeParamGen{}), "实现 ParamInserter 的方言应被探测到")
	assert.Nil(t, GetParamInserter(&fakeGen{}), "未实现的方言应返回 nil 以触发回退")
}
