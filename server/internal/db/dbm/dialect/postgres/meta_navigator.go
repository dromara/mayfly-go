package postgres

import (
	"context"
	"fmt"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"

	"github.com/spf13/cast"
)

const (
	PGSQL_VIEWS_KEY        = "PGSQL_VIEWS"
	PGSQL_SEQUENCES_KEY    = "PGSQL_SEQUENCES"
	PGSQL_VIEW_DDL_KEY     = "PGSQL_VIEW_DDL"
	PGSQL_SEQUENCE_DDL_KEY = "PGSQL_SEQUENCE_DDL"
	PGSQL_TABLE_KEYS_KEY   = "PGSQL_TABLE_KEYS"
)

// PgsqlMetadata 按需实现内核的可选能力接口 MetadataNavigator：视图与序列内省；KeyProvider：主/唯一键内省。
// 查询均基于 pg 9.x 通用的系统表（pg_class.relkind / pg_get_viewdef / pg_constraint），以兼容 gauss/kingbase/vastbase 等国产库。
// 与 mysql 复用同一 dbi 契约（MetadataObject/ObjectKind/KeyConstraint）。
var (
	_ dbi.MetadataNavigator = (*PgsqlMetadata)(nil)
	_ dbi.KeyProvider       = (*PgsqlMetadata)(nil)
)

// SupportedKinds pg 的 MetadataNavigator 可列举视图与序列两类
func (pd *PgsqlMetadata) SupportedKinds() []dbi.ObjectKind {
	return []dbi.ObjectKind{dbi.KindView, dbi.KindSequence}
}

// ListObjects 支持视图与序列两类；其余 kind 回传 ErrUnsupportedKind。
// schema 为空则内省当前模式（current_schema()），非空则按指定模式过滤（下推至 SQL）。
func (pd *PgsqlMetadata) ListObjects(_ context.Context, schema string, kind dbi.ObjectKind) ([]dbi.MetadataObject, error) {
	switch kind {
	case dbi.KindView:
		return pd.listViews(schema)
	case dbi.KindSequence:
		return pd.listSequences(schema)
	default:
		return nil, dbi.ErrUnsupportedKind
	}
}

func (pd *PgsqlMetadata) listViews(schema string) ([]dbi.MetadataObject, error) {
	sql := fmt.Sprintf(metaSQL.Get(PGSQL_VIEWS_KEY), dbi.QuoteEscape(schema))
	_, res, err := pd.di.Query(sql)
	if err != nil {
		return nil, err
	}
	objects := make([]dbi.MetadataObject, 0, len(res))
	for _, re := range res {
		objects = append(objects, dbi.MetadataObject{
			Name:    cast.ToString(re["viewName"]),
			Kind:    dbi.KindView,
			Schema:  cast.ToString(re["schemaName"]),
			Comment: cast.ToString(re["viewComment"]),
			Attrs:   map[string]any{"definition": cast.ToString(re["viewDefinition"])},
		})
	}
	return objects, nil
}

func (pd *PgsqlMetadata) listSequences(schema string) ([]dbi.MetadataObject, error) {
	sql := fmt.Sprintf(metaSQL.Get(PGSQL_SEQUENCES_KEY), dbi.QuoteEscape(schema))
	_, res, err := pd.di.Query(sql)
	if err != nil {
		return nil, err
	}
	objects := make([]dbi.MetadataObject, 0, len(res))
	for _, re := range res {
		objects = append(objects, dbi.MetadataObject{
			Name:    cast.ToString(re["seqName"]),
			Kind:    dbi.KindSequence,
			Schema:  cast.ToString(re["schemaName"]),
			Comment: cast.ToString(re["seqComment"]),
			// 定义属性随列表一并返回，前端「属性面板」直接渲染，无需再按对象逐个查询
			Attrs: map[string]any{
				"dataType":    cast.ToString(re["dataType"]),
				"startValue":  cast.ToString(re["startValue"]),
				"incrementBy": cast.ToString(re["incrementBy"]),
				"minValue":    cast.ToString(re["minValue"]),
				"maxValue":    cast.ToString(re["maxValue"]),
				"cacheSize":   cast.ToString(re["cacheSize"]),
				"isCycle":     cast.ToString(re["isCycle"]),
				"lastValue":   cast.ToString(re["lastValue"]),
			},
		})
	}
	return objects, nil
}

// ObjectDDL 支持视图与序列：视图用 pg_get_viewdef 取定义；序列由 pg_sequence 目录列重建 CREATE SEQUENCE。
// 其余 kind 回传 ErrUnsupportedKind。schema 非空则限指定模式，空则回退 current_schema()。
func (pd *PgsqlMetadata) ObjectDDL(_ context.Context, schema string, kind dbi.ObjectKind, name string) (string, error) {
	var key, col, notFound string
	switch kind {
	case dbi.KindView:
		key, col, notFound = PGSQL_VIEW_DDL_KEY, "viewDefinition", "view"
	case dbi.KindSequence:
		key, col, notFound = PGSQL_SEQUENCE_DDL_KEY, "sequenceDdl", "sequence"
	default:
		return "", dbi.ErrUnsupportedKind
	}
	sql := fmt.Sprintf(metaSQL.Get(key), dbi.QuoteEscape(schema), dbi.QuoteEscape(name))
	_, res, err := pd.di.Query(sql)
	if err != nil {
		return "", err
	}
	if len(res) == 0 {
		return "", errorx.NewBizf("%s not found: %s", notFound, name)
	}
	return cast.ToString(res[0][col]), nil
}

// GetKeys 内省指定表的主键与唯一键约束（成员列按键内序号有序）。schema 为空则限 current_schema()。
func (pd *PgsqlMetadata) GetKeys(_ context.Context, schema, table string) ([]dbi.KeyConstraint, error) {
	sql := fmt.Sprintf(metaSQL.Get(PGSQL_TABLE_KEYS_KEY), dbi.QuoteEscape(schema), dbi.QuoteEscape(table))
	_, res, err := pd.di.Query(sql)
	if err != nil {
		return nil, err
	}
	return dbi.ParseKeyRows(res), nil
}
