package dm

import (
	"context"
	"fmt"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"

	"github.com/spf13/cast"
)

const (
	DM_VIEWS_KEY     = "DM_VIEWS"
	DM_SEQUENCES_KEY = "DM_SEQUENCES"
)

// DMMetadata 按需实现内核可选能力接口 MetadataNavigator：视图与序列内省；KeyProvider：主/唯一键内省。
// 达梦高度兼容 Oracle，复用 ALL_VIEWS / ALL_SEQUENCES / ALL_CONSTRAINTS 数据字典与 DBMS_METADATA.GET_DDL，
// 与 oracle 复用同一 dbi 契约（MetadataObject/ObjectKind/KeyConstraint）。
var (
	_ dbi.MetadataNavigator = (*DMMetadata)(nil)
	_ dbi.KeyProvider       = (*DMMetadata)(nil)
)

// SupportedKinds dm 的 MetadataNavigator 可列举视图与序列两类
func (dd *DMMetadata) SupportedKinds() []dbi.ObjectKind {
	return []dbi.ObjectKind{dbi.KindView, dbi.KindSequence}
}

// ListObjects 支持视图与序列；其余 kind 回传 ErrUnsupportedKind。
// schema 为空则内省当前模式（SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID)），非空则按指定 owner 过滤。
func (dd *DMMetadata) ListObjects(_ context.Context, schema string, kind dbi.ObjectKind) ([]dbi.MetadataObject, error) {
	switch kind {
	case dbi.KindView:
		return dd.listViews(schema)
	case dbi.KindSequence:
		return dd.listSequences(schema)
	default:
		return nil, dbi.ErrUnsupportedKind
	}
}

func (dd *DMMetadata) listViews(schema string) ([]dbi.MetadataObject, error) {
	sql := fmt.Sprintf(metaSQL.Get(DM_VIEWS_KEY), dbi.QuoteEscape(schema))
	_, res, err := dd.di.Query(sql)
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
		})
	}
	return objects, nil
}

func (dd *DMMetadata) listSequences(schema string) ([]dbi.MetadataObject, error) {
	sql := fmt.Sprintf(metaSQL.Get(DM_SEQUENCES_KEY), dbi.QuoteEscape(schema))
	_, res, err := dd.di.Query(sql)
	if err != nil {
		return nil, err
	}
	objects := make([]dbi.MetadataObject, 0, len(res))
	for _, re := range res {
		cycle := "false"
		if cast.ToString(re["cycleFlag"]) == "Y" {
			cycle = "true"
		}
		objects = append(objects, dbi.MetadataObject{
			Name:   cast.ToString(re["seqName"]),
			Kind:   dbi.KindSequence,
			Schema: cast.ToString(re["schemaName"]),
			Attrs: map[string]any{
				"minValue":    cast.ToString(re["minValue"]),
				"maxValue":    cast.ToString(re["maxValue"]),
				"incrementBy": cast.ToString(re["incrementBy"]),
				"cacheSize":   cast.ToString(re["cacheSize"]),
				"lastValue":   cast.ToString(re["lastValue"]),
				"isCycle":     cycle,
			},
		})
	}
	return objects, nil
}

// ObjectDDL 用 Oracle 兼容的 DBMS_METADATA.GET_DDL 取视图/序列权威 DDL。
// schema 为空时第 3 参为 ”（达梦视作 NULL，回退当前模式）。
func (dd *DMMetadata) ObjectDDL(_ context.Context, schema string, kind dbi.ObjectKind, name string) (string, error) {
	var objectType string
	switch kind {
	case dbi.KindView:
		objectType = "VIEW"
	case dbi.KindSequence:
		objectType = "SEQUENCE"
	default:
		return "", dbi.ErrUnsupportedKind
	}
	sql := fmt.Sprintf(`SELECT DBMS_METADATA.GET_DDL('%s', '%s', '%s') AS "ddl" FROM DUAL`, objectType, dbi.QuoteEscape(name), dbi.QuoteEscape(schema))
	_, res, err := dd.di.Query(sql)
	if err != nil {
		return "", err
	}
	if len(res) == 0 {
		return "", errorx.NewBizf("%s not found: %s", objectType, name)
	}
	return cast.ToString(res[0]["ddl"]), nil
}

// GetKeys 内省指定表的主键与唯一键约束（成员列按键内位置有序）。schema 为空则限当前模式。
// 达梦复用 Oracle 的 ALL_CONSTRAINTS / ALL_CONS_COLUMNS 字典视图，POSITION 即键内列位置。
func (dd *DMMetadata) GetKeys(_ context.Context, schema, table string) ([]dbi.KeyConstraint, error) {
	sql := fmt.Sprintf(`SELECT c.CONSTRAINT_NAME "keyName",
       CASE c.CONSTRAINT_TYPE WHEN 'P' THEN 'PRIMARY KEY' WHEN 'U' THEN 'UNIQUE' END "keyType",
       cc.COLUMN_NAME "columnName",
       cc.POSITION "ordinal"
FROM ALL_CONSTRAINTS c
JOIN ALL_CONS_COLUMNS cc ON cc.OWNER = c.OWNER AND cc.CONSTRAINT_NAME = c.CONSTRAINT_NAME
WHERE c.OWNER = UPPER(COALESCE(NULLIF('%s', ''), SYS_CONTEXT('USERENV', 'CURRENT_SCHEMA')))
  AND c.TABLE_NAME = UPPER('%s')
  AND c.CONSTRAINT_TYPE IN ('P', 'U')
ORDER BY c.CONSTRAINT_NAME, cc.POSITION`, dbi.QuoteEscape(schema), dbi.QuoteEscape(table))
	_, res, err := dd.di.Query(sql)
	if err != nil {
		return nil, err
	}
	return dbi.ParseKeyRows(res), nil
}
