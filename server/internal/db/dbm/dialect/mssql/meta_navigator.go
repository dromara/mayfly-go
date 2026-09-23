package mssql

import (
	"context"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"

	"github.com/spf13/cast"
)

const (
	MSSQL_VIEWS_KEY        = "MSSQL_VIEWS"
	MSSQL_SEQUENCES_KEY    = "MSSQL_SEQUENCES"
	MSSQL_VIEW_DDL_KEY     = "MSSQL_VIEW_DDL"
	MSSQL_SEQUENCE_DDL_KEY = "MSSQL_SEQUENCE_DDL"
)

// MssqlMetadata 按需实现内核可选能力接口 MetadataNavigator：视图与序列内省；KeyProvider：主/唯一键内省。
// 视图取 sys.views + sys.sql_modules（原始 CREATE VIEW 语句），序列取 sys.sequences 并重建 CREATE SEQUENCE，
// 与 mysql/pg/oracle 复用同一 dbi 契约。schema 经 `?` 绑定参数下推，空则回退当前默认架构。
var (
	_ dbi.MetadataNavigator = (*MssqlMetadata)(nil)
	_ dbi.KeyProvider       = (*MssqlMetadata)(nil)
)

// SupportedKinds mssql 的 MetadataNavigator 可列举视图与序列两类
func (md *MssqlMetadata) SupportedKinds() []dbi.ObjectKind {
	return []dbi.ObjectKind{dbi.KindView, dbi.KindSequence}
}

// resolveSchema 空 schema 回退当前默认架构（mssql 以 `?` 绑定，需具体值而非 SQL 端判空）
func (md *MssqlMetadata) resolveSchema(schema string) string {
	if schema == "" {
		return md.di.CurrentSchema()
	}
	return schema
}

// ListObjects 支持视图与序列；其余 kind 回传 ErrUnsupportedKind。
func (md *MssqlMetadata) ListObjects(_ context.Context, schema string, kind dbi.ObjectKind) ([]dbi.MetadataObject, error) {
	sch := md.resolveSchema(schema)
	switch kind {
	case dbi.KindView:
		_, res, err := md.di.Query(metaSQL.Get(MSSQL_VIEWS_KEY), sch)
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
	case dbi.KindSequence:
		_, res, err := md.di.Query(metaSQL.Get(MSSQL_SEQUENCES_KEY), sch)
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
					"dataType":    cast.ToString(re["dataType"]),
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
	default:
		return nil, dbi.ErrUnsupportedKind
	}
}

// ObjectDDL 视图取 sys.sql_modules 原始定义；序列由 sys.sequences 重建 CREATE SEQUENCE。
func (md *MssqlMetadata) ObjectDDL(_ context.Context, schema string, kind dbi.ObjectKind, name string) (string, error) {
	sch := md.resolveSchema(schema)
	var key, col, notFound string
	switch kind {
	case dbi.KindView:
		key, col, notFound = MSSQL_VIEW_DDL_KEY, "viewDefinition", "view"
	case dbi.KindSequence:
		key, col, notFound = MSSQL_SEQUENCE_DDL_KEY, "sequenceDdl", "sequence"
	default:
		return "", dbi.ErrUnsupportedKind
	}
	_, res, err := md.di.Query(metaSQL.Get(key), name, sch)
	if err != nil {
		return "", err
	}
	if len(res) == 0 {
		return "", errorx.NewBizf("%s not found: %s", notFound, name)
	}
	return cast.ToString(res[0][col]), nil
}

// GetKeys 内省指定表的主键与唯一键约束（成员列按键内序号有序）。schema 为空回退当前默认架构。
// SQL Server 的 KEY_COLUMN_USAGE.ORDINAL_POSITION 即键内位置（与 information_schema 标准一致），可直接用。
func (md *MssqlMetadata) GetKeys(_ context.Context, schema, table string) ([]dbi.KeyConstraint, error) {
	sql := `SELECT tc.CONSTRAINT_NAME AS keyName, tc.CONSTRAINT_TYPE AS keyType, k.COLUMN_NAME AS columnName, k.ORDINAL_POSITION AS ordinal
FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE k
  ON k.CONSTRAINT_NAME = tc.CONSTRAINT_NAME AND k.TABLE_SCHEMA = tc.TABLE_SCHEMA AND k.TABLE_NAME = tc.TABLE_NAME
WHERE tc.TABLE_SCHEMA = ? AND tc.TABLE_NAME = ? AND tc.CONSTRAINT_TYPE IN ('PRIMARY KEY', 'UNIQUE')
ORDER BY tc.CONSTRAINT_NAME, k.ORDINAL_POSITION`
	_, res, err := md.di.Query(sql, md.resolveSchema(schema), table)
	if err != nil {
		return nil, err
	}
	return dbi.ParseKeyRows(res), nil
}
