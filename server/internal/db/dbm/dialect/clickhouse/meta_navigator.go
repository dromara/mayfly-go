package clickhouse

import (
	"context"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"

	"github.com/spf13/cast"
)

const (
	CLICKHOUSE_VIEWS_KEY    = "CLICKHOUSE_VIEWS"
	CLICKHOUSE_VIEW_DDL_KEY = "CLICKHOUSE_VIEW_DDL"
)

// ClickHouseMetadata 按需实现内核可选能力接口 MetadataNavigator：视图内省（clickhouse 无序列概念）；
// KeyProvider：主键内省（取 ORDER BY/primary_key 有序列；clickhouse 无列级唯一键约束清单，唯一键返回空）。
// system.tables 中引擎为视图类（View/MaterializedView/...）者，其 create_table_query 即完整 CREATE VIEW 语句。
var (
	_ dbi.MetadataNavigator = (*ClickHouseMetadata)(nil)
	_ dbi.KeyProvider       = (*ClickHouseMetadata)(nil)
)

// SupportedKinds clickhouse 仅支持视图（无序列）
func (cm *ClickHouseMetadata) SupportedKinds() []dbi.ObjectKind {
	return []dbi.ObjectKind{dbi.KindView}
}

// ListObjects 仅支持视图；clickhouse 以 database 界定，schema 参数忽略。
func (cm *ClickHouseMetadata) ListObjects(_ context.Context, _ string, kind dbi.ObjectKind) ([]dbi.MetadataObject, error) {
	if kind != dbi.KindView {
		return nil, dbi.ErrUnsupportedKind
	}
	_, res, err := cm.di.Query(metaSQL.Get(CLICKHOUSE_VIEWS_KEY), cm.di.GetDatabase())
	if err != nil {
		return nil, err
	}
	objects := make([]dbi.MetadataObject, 0, len(res))
	for _, re := range res {
		objects = append(objects, dbi.MetadataObject{
			Name:    cast.ToString(re["viewName"]),
			Kind:    dbi.KindView,
			Comment: cast.ToString(re["viewComment"]),
			Attrs:   map[string]any{"definition": cast.ToString(re["viewDefinition"])},
		})
	}
	return objects, nil
}

// ObjectDDL 取 system.tables.create_table_query 的完整 CREATE VIEW 语句。
func (cm *ClickHouseMetadata) ObjectDDL(_ context.Context, _ string, kind dbi.ObjectKind, name string) (string, error) {
	if kind != dbi.KindView {
		return "", dbi.ErrUnsupportedKind
	}
	_, res, err := cm.di.Query(metaSQL.Get(CLICKHOUSE_VIEW_DDL_KEY), cm.di.GetDatabase(), name)
	if err != nil {
		return "", err
	}
	if len(res) == 0 {
		return "", errorx.NewBizf("view not found: %s", name)
	}
	return cast.ToString(res[0]["viewDefinition"]), nil
}

// GetKeys 内省 clickhouse 表的主键（ORDER BY/primary_key 有序列）。
// clickhouse 的 primary_key 是表引擎设置里的有序列清单；无关系型数据库那种列级唯一键约束，
// 故仅返回主键（primary_key 为空即 MergeTree 未设主键时返回空列表，不虚构）。
func (cm *ClickHouseMetadata) GetKeys(_ context.Context, _, table string) ([]dbi.KeyConstraint, error) {
	_, res, err := cm.di.Query("SELECT primary_key FROM system.tables WHERE database = ? AND name = ?", cm.di.GetDatabase(), table)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}
	pk := dbi.KeyConstraint{Name: "PRIMARY KEY", Type: dbi.KeyTypePrimary}
	for i, part := range strings.Split(cast.ToString(res[0]["primary_key"]), ",") {
		if col := strings.TrimSpace(part); col != "" {
			pk.Columns = append(pk.Columns, dbi.KeyColumn{Name: col, Ordinal: i + 1})
		}
	}
	if len(pk.Columns) == 0 {
		return nil, nil
	}
	return []dbi.KeyConstraint{pk}, nil
}
