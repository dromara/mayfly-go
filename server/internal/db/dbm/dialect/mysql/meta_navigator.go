package mysql

import (
	"context"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"

	"github.com/spf13/cast"
)

const (
	MYSQL_VIEWS_KEY           = "MYSQL_VIEWS"
	MYSQL_TABLE_RELATIONS_KEY = "MYSQL_TABLE_RELATIONS"
	MYSQL_TABLE_KEYS_KEY      = "MYSQL_TABLE_KEYS"
)

// MysqlMetadata 按需实现内核的可选能力接口：视图导航 + 外键关系内省 + 主/唯一键约束内省。
// 编译期断言确保签名与 dbi 契约一致；不实现这些接口的其余方言完全不受影响。
var (
	_ dbi.MetadataNavigator  = (*MysqlMetadata)(nil)
	_ dbi.ForeignKeyProvider = (*MysqlMetadata)(nil)
	_ dbi.KeyProvider        = (*MysqlMetadata)(nil)
)

// SupportedKinds mysql 的 MetadataNavigator 目前可列举视图类别
func (md *MysqlMetadata) SupportedKinds() []dbi.ObjectKind {
	return []dbi.ObjectKind{dbi.KindView}
}

// ListObjects 目前支持视图类别；其余 kind 明确回传 ErrUnsupportedKind（供上层隐藏树节点）。
// schema 为空则内省当前库，非空则按指定库过滤（下推至 SQL，见 MYSQL_VIEWS）。
func (md *MysqlMetadata) ListObjects(_ context.Context, schema string, kind dbi.ObjectKind) ([]dbi.MetadataObject, error) {
	if kind != dbi.KindView {
		return nil, dbi.ErrUnsupportedKind
	}
	_, res, err := md.di.Query(metaSQL.Get(MYSQL_VIEWS_KEY), schema)
	if err != nil {
		return nil, err
	}
	objects := make([]dbi.MetadataObject, 0, len(res))
	for _, re := range res {
		objects = append(objects, dbi.MetadataObject{
			Name:    cast.ToString(re["viewName"]),
			Kind:    dbi.KindView,
			Schema:  cast.ToString(re["viewSchema"]),
			Comment: cast.ToString(re["viewComment"]),
			Attrs:   map[string]any{"definition": cast.ToString(re["viewDefinition"])},
		})
	}
	return objects, nil
}

// ObjectDDL 用 SHOW CREATE VIEW 取权威的视图重建 DDL（比用 view_definition 拼接更保真）。
// schema 非空时以 `库`.`视图` 限定，跨当前库内省指定库的视图。
func (md *MysqlMetadata) ObjectDDL(_ context.Context, schema string, kind dbi.ObjectKind, name string) (string, error) {
	if kind != dbi.KindView {
		return "", dbi.ErrUnsupportedKind
	}
	quoter := md.di.GetDialect().Quoter()
	viewRef := quoter.QuoteIdent(name)
	if schema != "" {
		viewRef = quoter.QuoteIdent(schema) + "." + quoter.QuoteIdent(name)
	}
	_, res, err := md.di.Query("SHOW CREATE VIEW " + viewRef)
	if err != nil {
		return "", err
	}
	if len(res) == 0 {
		return "", errorx.NewBizf("view not found: %s", name)
	}
	return cast.ToString(res[0]["Create View"]), nil
}

// GetForeignKeys 内省指定表的外键关系（ER 图、迁移建表拓扑排序消费）。
// schema 为空则限当前库，非空则限指定库。
func (md *MysqlMetadata) GetForeignKeys(_ context.Context, schema, table string) ([]dbi.ForeignKey, error) {
	_, res, err := md.di.Query(metaSQL.Get(MYSQL_TABLE_RELATIONS_KEY), schema, table)
	if err != nil {
		return nil, err
	}
	rels := make([]dbi.ForeignKey, 0, len(res))
	for _, re := range res {
		rels = append(rels, dbi.ForeignKey{
			Name:      cast.ToString(re["fkName"]),
			Table:     table,
			Column:    cast.ToString(re["columnName"]),
			RefTable:  cast.ToString(re["refTable"]),
			RefColumn: cast.ToString(re["refColumn"]),
			OnUpdate:  cast.ToString(re["updateRule"]),
			OnDelete:  cast.ToString(re["deleteRule"]),
		})
	}
	return rels, nil
}

// GetKeys 内省指定表的主键与唯一键约束（成员列按键内序号有序）。schema 为空限当前库。
func (md *MysqlMetadata) GetKeys(_ context.Context, schema, table string) ([]dbi.KeyConstraint, error) {
	_, res, err := md.di.Query(metaSQL.Get(MYSQL_TABLE_KEYS_KEY), schema, table)
	if err != nil {
		return nil, err
	}
	return dbi.ParseKeyRows(res), nil
}
