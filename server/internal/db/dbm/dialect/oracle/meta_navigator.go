package oracle

import (
	"context"
	"fmt"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"

	"github.com/spf13/cast"
)

const (
	ORACLE_VIEWS_KEY     = "ORACLE_VIEWS"
	ORACLE_SEQUENCES_KEY = "ORACLE_SEQUENCES"
)

// OracleMetadata 按需实现内核可选能力接口 MetadataNavigator：视图与序列内省；KeyProvider：主/唯一键内省。
// OracleMetadata11 内嵌本类型，自动继承这些方法，故 11/12 两版本共享同一实现（能力声明真实）。
// 与 mysql、pg、dm 复用同一 dbi 契约（MetadataObject/ObjectKind/KeyConstraint）。
var (
	_ dbi.MetadataNavigator = (*OracleMetadata)(nil)
	_ dbi.KeyProvider       = (*OracleMetadata)(nil)
)

// SupportedKinds oracle 的 MetadataNavigator 可列举视图与序列两类
func (od *OracleMetadata) SupportedKinds() []dbi.ObjectKind {
	return []dbi.ObjectKind{dbi.KindView, dbi.KindSequence}
}

// ListObjects 支持视图与序列两类；其余 kind 回传 ErrUnsupportedKind。
// schema 为空则内省当前模式（USERENV CURRENT_SCHEMA），非空则按指定 owner 过滤（下推至 SQL）。
func (od *OracleMetadata) ListObjects(_ context.Context, schema string, kind dbi.ObjectKind) ([]dbi.MetadataObject, error) {
	switch kind {
	case dbi.KindView:
		return od.listViews(schema)
	case dbi.KindSequence:
		return od.listSequences(schema)
	default:
		return nil, dbi.ErrUnsupportedKind
	}
}

func (od *OracleMetadata) listViews(schema string) ([]dbi.MetadataObject, error) {
	sql := fmt.Sprintf(metaSQL.Get(ORACLE_VIEWS_KEY), dbi.QuoteEscape(schema))
	_, res, err := od.di.Query(sql)
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

func (od *OracleMetadata) listSequences(schema string) ([]dbi.MetadataObject, error) {
	sql := fmt.Sprintf(metaSQL.Get(ORACLE_SEQUENCES_KEY), dbi.QuoteEscape(schema))
	_, res, err := od.di.Query(sql)
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
			// 定义属性随列表返回，前端序列属性面板直接渲染（与 pg 的 attrs 契约一致）
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

// ObjectDDL 用 DBMS_METADATA.GET_DDL 取对象权威 DDL（oracle 取 DDL 的标准做法，与既有建表 DDL 同源）。
// 对象名/schema 均以 QuoteEscape 作值转义嵌入（标识符值匹配，非拼接执行上下文）；
// schema 为空时第 3 参为 ”（oracle 视作 NULL），GET_DDL 回退当前模式。
func (od *OracleMetadata) ObjectDDL(_ context.Context, schema string, kind dbi.ObjectKind, name string) (string, error) {
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
	_, res, err := od.di.Query(sql)
	if err != nil {
		return "", err
	}
	if len(res) == 0 {
		return "", errorx.NewBizf("%s not found: %s", objectType, name)
	}
	return cast.ToString(res[0]["ddl"]), nil
}

// GetKeys 内省指定表的主键与唯一键约束（成员列按键内位置有序）。schema 为空则限当前模式。
// ALL_CONS_COLUMNS.POSITION 为约束内列位置（键内序）；owner/table 以 UPPER 匹配 oracle 大写字典视图。
func (od *OracleMetadata) GetKeys(_ context.Context, schema, table string) ([]dbi.KeyConstraint, error) {
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
	_, res, err := od.di.Query(sql)
	if err != nil {
		return nil, err
	}
	return dbi.ParseKeyRows(res), nil
}
