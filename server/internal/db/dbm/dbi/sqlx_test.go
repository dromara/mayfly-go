package dbi

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 嵌套struct测试类型：Names映射必须以字段名为键（对齐上游sqlx），
// 若以Path（含父级前缀，如Page.PageNo）为键，嵌套字段将无法按名字命中
type sqlxPage struct {
	PageNo   int `db:"page_no"`
	PageSize int `db:"page_size"`
}

type sqlxQueryReq struct {
	sqlxPage        // 匿名嵌入
	Keyword  string `db:"keyword"`
}

type sqlxNamedEmbed struct {
	sqlxPage        // 具名嵌入
	Status   string `db:"status"`
}

// TraversalsByName按字段名查找嵌套struct的字段（原实现以Path为键导致查不到）
func TestMapper_TraversalsByName_NestedStructFields(t *testing.T) {
	m := mapper()

	// 匿名嵌入：顶层可直接按字段名访问
	anon := sqlxQueryReq{sqlxPage{PageNo: 1, PageSize: 10}, "kw"}
	traversals := m.TraversalsByName(reflect.TypeOf(anon), []string{"page_no", "page_size", "keyword"})
	assert.Len(t, traversals, 3)
	for i, tr := range traversals {
		assert.NotEmpty(t, tr, "nested field at index %d should be traversable", i)
	}

	// FieldsByName按名字取值，验证映射到正确字段（.Int()返回int64，用EqualValues比较）
	fields := m.FieldsByName(reflect.ValueOf(&anon).Elem(), []string{"page_no", "page_size", "keyword"})
	assert.Len(t, fields, 3)
	assert.EqualValues(t, 1, fields[0].Int())
	assert.EqualValues(t, 10, fields[1].Int())
	assert.Equal(t, "kw", fields[2].String())

	// 具名嵌入：TypeMap同样应包含嵌入struct的字段名映射
	named := sqlxNamedEmbed{sqlxPage{PageNo: 2, PageSize: 20}, "enabled"}
	namedFields := m.FieldsByName(reflect.ValueOf(&named).Elem(), []string{"page_no", "status"})
	assert.Len(t, namedFields, 2)
	assert.EqualValues(t, 2, namedFields[0].Int())
	assert.Equal(t, "enabled", namedFields[1].String())
}

// 不存在的字段名返回空traversal（调用方以missingFields判定缺列）
func TestMapper_TraversalsByName_MissingField(t *testing.T) {
	m := mapper()
	v := sqlxQueryReq{}
	traversals := m.TraversalsByName(reflect.TypeOf(v), []string{"page_no", "not_exist"})
	assert.Len(t, traversals, 2)
	assert.NotEmpty(t, traversals[0])
	assert.Empty(t, traversals[1])
}
