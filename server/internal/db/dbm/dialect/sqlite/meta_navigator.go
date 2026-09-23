package sqlite

import (
	"context"
	"fmt"
	"sort"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"

	"github.com/spf13/cast"
)

const (
	SQLITE_VIEWS_KEY    = "SQLITE_VIEWS"
	SQLITE_VIEW_DDL_KEY = "SQLITE_VIEW_DDL"
)

// SQLiteMetadata 按需实现内核可选能力接口 MetadataNavigator（视图内省，sqlite 无序列概念）+ KeyProvider（主/唯一键）。
// sqlite_master.type='view' 的 sql 列即完整 CREATE VIEW 原文，列举与 DDL 同源。
var (
	_ dbi.MetadataNavigator = (*SQLiteMetadata)(nil)
	_ dbi.KeyProvider       = (*SQLiteMetadata)(nil)
)

// SupportedKinds sqlite 仅支持视图（无序列）
func (sd *SQLiteMetadata) SupportedKinds() []dbi.ObjectKind {
	return []dbi.ObjectKind{dbi.KindView}
}

// ListObjects 仅支持视图；sqlite 单文件库无 schema 维度，schema 参数忽略。
func (sd *SQLiteMetadata) ListObjects(_ context.Context, _ string, kind dbi.ObjectKind) ([]dbi.MetadataObject, error) {
	if kind != dbi.KindView {
		return nil, dbi.ErrUnsupportedKind
	}
	_, res, err := sd.di.Query(metaSQL.Get(SQLITE_VIEWS_KEY))
	if err != nil {
		return nil, err
	}
	objects := make([]dbi.MetadataObject, 0, len(res))
	for _, re := range res {
		objects = append(objects, dbi.MetadataObject{
			Name:  cast.ToString(re["viewName"]),
			Kind:  dbi.KindView,
			Attrs: map[string]any{"definition": cast.ToString(re["viewDefinition"])},
		})
	}
	return objects, nil
}

// ObjectDDL 取 sqlite_master.sql 中的 CREATE VIEW 原文。
func (sd *SQLiteMetadata) ObjectDDL(_ context.Context, _ string, kind dbi.ObjectKind, name string) (string, error) {
	if kind != dbi.KindView {
		return "", dbi.ErrUnsupportedKind
	}
	_, res, err := sd.di.Query(metaSQL.Get(SQLITE_VIEW_DDL_KEY), name)
	if err != nil {
		return "", err
	}
	if len(res) == 0 {
		return "", errorx.NewBizf("view not found: %s", name)
	}
	return cast.ToString(res[0]["viewDefinition"]), nil
}

// GetKeys 内省 sqlite 表的主键与唯一键约束（成员列按键内序号有序）。
// 主键取 PRAGMA table_info 的 pk 序号（>0 即键内位置）；唯一键取 index_list 中 unique 且
// origin!='pk'（排除主键自动索引）且非 partial 的索引，再经 index_info 按 seqno 取有序列。
// sqlite 无 schema 维度，schema 参数忽略。
func (sd *SQLiteMetadata) GetKeys(_ context.Context, _, table string) ([]dbi.KeyConstraint, error) {
	quoter := sd.di.GetDialect().Quoter().QuoteIdent
	qt := quoter(table)

	keys := make([]dbi.KeyConstraint, 0, 4)

	// 主键：table_info.pk 为 1 起的键内序号
	pkRes, err := sd.queryPragma(fmt.Sprintf("PRAGMA table_info(%s)", qt))
	if err != nil {
		return nil, err
	}
	pk := dbi.KeyConstraint{Name: "PRIMARY", Type: dbi.KeyTypePrimary}
	for _, re := range pkRes {
		if seq := cast.ToInt(re["pk"]); seq > 0 {
			pk.Columns = append(pk.Columns, dbi.KeyColumn{Name: cast.ToString(re["name"]), Ordinal: seq})
		}
	}
	// table_info 按物理列序返回，须按键内序号 pk 重排（否则联合主键 (a,b) 会退化为建表列序）
	sort.Slice(pk.Columns, func(i, j int) bool { return pk.Columns[i].Ordinal < pk.Columns[j].Ordinal })
	if len(pk.Columns) > 0 {
		keys = append(keys, pk)
	}

	// 唯一键：index_list 过滤 unique=1、origin!='pk'、partial=0
	idxRes, err := sd.queryPragma(fmt.Sprintf("PRAGMA index_list(%s)", qt))
	if err != nil {
		return nil, err
	}
	for _, ix := range idxRes {
		if cast.ToInt(ix["unique"]) != 1 || cast.ToString(ix["origin"]) == "pk" || cast.ToInt(ix["partial"]) != 0 {
			continue
		}
		name := cast.ToString(ix["name"])
		infoRes, err := sd.queryPragma(fmt.Sprintf("PRAGMA index_info(%s)", quoter(name)))
		if err != nil {
			return nil, err
		}
		uk := dbi.KeyConstraint{Name: name, Type: dbi.KeyTypeUnique}
		for _, c := range infoRes {
			uk.Columns = append(uk.Columns, dbi.KeyColumn{Name: cast.ToString(c["name"]), Ordinal: cast.ToInt(c["seqno"]) + 1})
		}
		if len(uk.Columns) > 0 {
			keys = append(keys, uk)
		}
	}
	return keys, nil
}

// queryPragma 执行 PRAGMA 并返回行（PRAGMA 表名不能作绑定参数，调用方须已引用标识符）。
func (sd *SQLiteMetadata) queryPragma(sql string) ([]map[string]any, error) {
	_, res, err := sd.di.Query(sql)
	return res, err
}
