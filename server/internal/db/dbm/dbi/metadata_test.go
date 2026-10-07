package dbi

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeMetaProvider 实现 MetadataProvider 全部必填方法，代表「未声明任何新对象能力」的方言
type fakeMetaProvider struct{}

func (fakeMetaProvider) GetSchemas() ([]string, error)            { return nil, nil }
func (fakeMetaProvider) GetDbNames() ([]string, error)            { return nil, nil }
func (fakeMetaProvider) GetTables(...string) ([]Table, error)     { return nil, nil }
func (fakeMetaProvider) GetColumns(...string) ([]Column, error)   { return nil, nil }
func (fakeMetaProvider) GetPrimaryKeys(string) ([]string, error)  { return nil, nil }
func (fakeMetaProvider) GetTableIndex(string) ([]Index, error)    { return nil, nil }
func (fakeMetaProvider) GetTableDDL(string, bool) (string, error) { return "", nil }

// fakeNavProvider 额外实现 MetadataNavigator + ForeignKeyProvider，代表「按需支持新对象」的方言
type fakeNavProvider struct {
	fakeMetaProvider
}

func (fakeNavProvider) SupportedKinds() []ObjectKind { return []ObjectKind{KindView} }

func (fakeNavProvider) ListObjects(_ context.Context, schema string, kind ObjectKind) ([]MetadataObject, error) {
	if kind != KindView {
		return nil, ErrUnsupportedKind
	}
	return []MetadataObject{{Name: "v_active_users", Kind: KindView, Schema: schema}}, nil
}

func (fakeNavProvider) ObjectDDL(_ context.Context, _ string, kind ObjectKind, name string) (string, error) {
	if kind != KindView {
		return "", ErrUnsupportedKind
	}
	return "CREATE VIEW " + name + " AS SELECT 1", nil
}

func (fakeNavProvider) GetForeignKeys(_ context.Context, _, table string) ([]ForeignKey, error) {
	return []ForeignKey{{Table: table, Column: "uid", RefTable: "users", RefColumn: "id"}}, nil
}

var (
	_ MetadataProvider   = fakeMetaProvider{}
	_ MetadataProvider   = fakeNavProvider{}
	_ MetadataNavigator  = fakeNavProvider{}
	_ ForeignKeyProvider = fakeNavProvider{}
)

// 未实现 MetadataNavigator 的方言：代理方法必须返回 ErrUnsupportedKind，而非静默返回空切片
func TestMetadata_CapabilityProbe_Unsupported(t *testing.T) {
	m := NewMetadataReader(nil, fakeMetaProvider{}, nil, nil)

	_, err := m.ListObjects(context.Background(), "", KindView)
	assert.ErrorIs(t, err, ErrUnsupportedKind)

	_, err = m.ObjectDDL(context.Background(), "", KindView, "v")
	assert.ErrorIs(t, err, ErrUnsupportedKind)

	_, err = m.GetForeignKeys(context.Background(), "", "t")
	assert.ErrorIs(t, err, ErrUnsupportedKind)
}

// 实现了能力的方言：代理透传到 provider；provider 对不支持的具体 kind 仍回传 ErrUnsupportedKind
func TestMetadata_CapabilityProbe_Supported(t *testing.T) {
	m := NewMetadataReader(nil, fakeNavProvider{}, nil, nil)

	nodes, err := m.ListObjects(context.Background(), "public", KindView)
	assert.NoError(t, err)
	assert.Len(t, nodes, 1)
	assert.Equal(t, KindView, nodes[0].Kind)
	assert.Equal(t, "public", nodes[0].Schema)

	ddl, err := m.ObjectDDL(context.Background(), "public", KindView, "v_active_users")
	assert.NoError(t, err)
	assert.Contains(t, ddl, "v_active_users")

	rels, err := m.GetForeignKeys(context.Background(), "public", "orders")
	assert.NoError(t, err)
	assert.Equal(t, "users", rels[0].RefTable)

	// 方言具备导航能力，但不支持序列这一具体 kind：仍需明确回传，供上层隐藏该树节点
	_, err = m.ListObjects(context.Background(), "", KindSequence)
	assert.ErrorIs(t, err, ErrUnsupportedKind)
}

// ===================== SearchTables（表名过滤：下推 / 回退，源码 metadata.go）=====================

// fakeTableProvider 不实现 TableSearcher，GetTables 返回固定清单 → Metadata.SearchTables 走回退子串过滤
type fakeTableProvider struct {
	fakeMetaProvider
}

func (fakeTableProvider) GetTables(...string) ([]Table, error) {
	return []Table{{TableName: "t_user"}, {TableName: "T_ORDER"}, {TableName: "user_role"}}, nil
}

// fakeTableSearcher 实现 TableSearcher，记录被调用以断言走了服务端下推而非回退
type fakeTableSearcher struct {
	fakeMetaProvider
	called bool
	offset int
}

func (f *fakeTableSearcher) SearchTables(like string, limit int, offset int) ([]Table, error) {
	f.called = true
	f.offset = offset
	return []Table{{TableName: "pushed_" + like}}, nil
}

var (
	_ MetadataProvider = fakeTableProvider{}
	_ MetadataProvider = (*fakeTableSearcher)(nil)
	_ TableSearcher    = (*fakeTableSearcher)(nil)
)

// pushdownFailProvider 实现 TableSearcher 但 SearchTables 恒返回错误，GetTables 行为可控：
// 守护「下推失败静默回退全量+过滤」与「回退也失败时以回退真实错误为准」两条分支。
type pushdownFailProvider struct {
	fakeMetaProvider
	pushErr error
	tables  []Table
	getErr  error
}

func (p *pushdownFailProvider) SearchTables(string, int, int) ([]Table, error) { return nil, p.pushErr }
func (p *pushdownFailProvider) GetTables(...string) ([]Table, error)           { return p.tables, p.getErr }

var _ TableSearcher = (*pushdownFailProvider)(nil)

func tableNamesOf(ts []Table) []string {
	out := make([]string, 0, len(ts))
	for _, t := range ts {
		out = append(out, t.TableName)
	}
	return out
}

func TestMetadata_SearchTables(t *testing.T) {
	m := NewMetadataReader(nil, fakeTableProvider{}, nil, nil)

	// 空 like 等价全量 GetTables
	all, err := m.SearchTables("", 0, 0)
	assert.NoError(t, err)
	assert.Len(t, all, 3)

	// 空 like + limit：限量探测（回退路径截断），用于「表是否过多」判定而不全量渲染
	capped, err := m.SearchTables("", 2, 0)
	assert.NoError(t, err)
	assert.Len(t, capped, 2)

	// 无下推能力：回退 GetTables + 不区分大小写子串过滤，保序
	filtered, err := m.SearchTables("USER", 0, 0)
	assert.NoError(t, err)
	assert.Equal(t, []string{"t_user", "user_role"}, tableNamesOf(filtered))

	// 回退路径 offset：跳过首条命中取下一页（分页续载不重拉前缀）
	paged, err := m.SearchTables("USER", 1, 1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"user_role"}, tableNamesOf(paged))

	// 具备 TableSearcher：下推到 provider，不走回退；offset 透传
	fs := &fakeTableSearcher{}
	res, err := NewMetadataReader(nil, fs, nil, nil).SearchTables("x", 5, 7)
	assert.NoError(t, err)
	assert.True(t, fs.called, "应下推到 provider.SearchTables")
	assert.Equal(t, 7, fs.offset, "offset 应透传到下推实现")
	assert.Equal(t, "pushed_x", res[0].TableName)
}

func TestFilterTablesByLike(t *testing.T) {
	tables := []Table{{TableName: "t_user"}, {TableName: "T_ORDER"}, {TableName: "sys_log"}, {TableName: "user_role"}}

	// 大小写不敏感子串过滤，保持原顺序
	assert.Equal(t, []string{"t_user", "user_role"}, tableNamesOf(filterTablesByLike(tables, "USER", 0, 0)))
	// limit 截断
	assert.Len(t, filterTablesByLike(tables, "o", 1, 0), 1)
	// offset 跳过首条命中后再按 limit 截断
	assert.Equal(t, []string{"sys_log"}, tableNamesOf(filterTablesByLike(tables, "o", 1, 1)))
	// offset 超出命中总数 → 空页（前端据此判定已到底）
	assert.Empty(t, filterTablesByLike(tables, "o", 10, 99))
	// 无命中
	assert.Empty(t, filterTablesByLike(tables, "zzz", 0, 0))
}

// TestMetadata_SearchTables_PushdownFallback 守护下推失败的两条回退分支（本次改动引入 logx 留痕 + 以 ferr 为准）
func TestMetadata_SearchTables_PushdownFallback(t *testing.T) {
	pushErr := errors.New("pushdown: catalog sql incompatible")
	getErr := errors.New("connection refused")
	ordered := []Table{{TableName: "t_user"}, {TableName: "T_ORDER"}, {TableName: "user_role"}}

	// 下推失败但 GetTables 正常：静默回退为「全量 + 不区分大小写子串过滤」，对上层不应报错
	res, err := NewMetadataReader(nil, &pushdownFailProvider{pushErr: pushErr, tables: ordered}, nil, nil).
		SearchTables("USER", 0, 0)
	assert.NoError(t, err, "回退成功即不应向上抛错")
	assert.Equal(t, []string{"t_user", "user_role"}, tableNamesOf(res))

	// 下推失败且回退 GetTables 也失败：必须返回回退的真实错误(ferr)，而非已预期的下推错误——
	// 否则「连接已断」这类根因会被下推的 SQL 兼容性错误掩盖，误导上层与排障
	_, err = NewMetadataReader(nil, &pushdownFailProvider{pushErr: pushErr, getErr: getErr}, nil, nil).
		SearchTables("USER", 5, 0)
	assert.ErrorIs(t, err, getErr, "应以回退 GetTables 的错误为准")
	assert.NotErrorIs(t, err, pushErr, "不应把已预期且已留痕的下推错误当终态错误上抛")
}

// TestMetadata_SearchTables_OffsetNormalization 守护 offset 单点归一：limit<=0 时无意义、负值非法，均钳为 0，
// 保证下推路径（仅 limit>0 拼 OFFSET）与回退路径（filterTablesByLike）口径不分叉
func TestMetadata_SearchTables_OffsetNormalization(t *testing.T) {
	fs := &fakeTableSearcher{}
	_, err := NewMetadataReader(nil, fs, nil, nil).SearchTables("x", 0, 5)
	assert.NoError(t, err)
	assert.True(t, fs.called)
	assert.Equal(t, 0, fs.offset, "limit<=0 时 offset 应归一为 0")

	fs2 := &fakeTableSearcher{}
	_, err = NewMetadataReader(nil, fs2, nil, nil).SearchTables("x", 10, -3)
	assert.NoError(t, err)
	assert.Equal(t, 0, fs2.offset, "负 offset 应钳为 0（PG OFFSET 拒绝负值）")

	// 越界 offset → 钳到输入合法性上界（护栏，非业务分页上限）
	fs3 := &fakeTableSearcher{}
	_, err = NewMetadataReader(nil, fs3, nil, nil).SearchTables("x", 10, maxTableSearchOffset+99999)
	assert.NoError(t, err)
	assert.Equal(t, maxTableSearchOffset, fs3.offset, "越界 offset 应钳到上界")
}

// TestEscapeLikeWildcards 守护 LIKE 服务端下推的转义：用户输入的 %_\ 须按字面匹配（mysql/pg 共用），
// 反斜杠最先转义以免与后续 \% \_ 转义序列冲突；无通配符原样返回。
func TestEscapeLikeWildcards(t *testing.T) {
	assert.Equal(t, "", EscapeLikeWildcards(""))
	assert.Equal(t, "abc", EscapeLikeWildcards("abc"))                // 无通配符 → 原样
	assert.Equal(t, `100\%`, EscapeLikeWildcards("100%"))             // % → \%
	assert.Equal(t, `a\_b`, EscapeLikeWildcards("a_b"))               // _ → \_
	assert.Equal(t, `back\\slash`, EscapeLikeWildcards(`back\slash`)) // \ → \\
}
