package sqldraft

// 结构快照：一次生成所用的「当前账号可见的库结构」。
//
// 三条与权限相关的事实必须在这里一次算清，后续所有环节都只读快照：
//   - tables 只含当前账号可见的表（VisibleTables）；
//   - hidden 记录「库里存在但当前账号不可见」的表名：用于把模型的「表不存在」
//     与「表被保护」区分开，但**只用于内部判定**，不回显给模型或界面；
//   - deniedColumns 记录无权列名：同理，命中时只回「该字段不在当前账号可见范围内」，
//     绝不把列名回给模型（否则就成了一条凭猜测试探列名的通道）。
//
// 结构细节（列、外键）一律**按需 ensure**：谁需要谁向快照要，快照负责去重与缓存。
// 这样召回策略/片段可以自取所需结构，而不必让编排层预知「哪一步要用哪份结构」——
// 编排层一旦预知，就长出 `if id == "examples"` 这类把策略知识泄漏进管线的分支。

import (
	"context"
	"errors"
	"strings"

	"mayfly-go/internal/db/application/sqlutil"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/logx"
)

// columnInfo 列结构（上下文与核对都用它）
type columnInfo struct {
	Name       string
	Type       string
	Comment    string
	PrimaryKey bool
}

// tableInfo 表结构（Columns 按需装载：候选表只需要名字与注释，选中后才取列）
type tableInfo struct {
	Name    string
	Comment string
	Columns []columnInfo
}

// fkEdge 外键关系（用于把 JOIN 需要的那侧表也召回）
type fkEdge struct {
	FromTable  string
	FromColumn string
	ToTable    string
	ToColumn   string
}

// snapshot 结构快照
type snapshot struct {
	db      string
	dialect string
	version string
	// quotePrefix 标识符引用符（从 Quoter 实测派生，不另维护方言表）
	quotePrefix string

	tables []*tableInfo
	byName map[string]*tableInfo
	// visible 可见表名（小写）：find 的高频判定走 map，不做线性扫描
	visible map[string]struct{}
	// hidden 库里存在但当前账号不可见的表（小写表名）
	hidden map[string]struct{}
	// deniedColumns 无权列名（小写）；只用于「受限」判定，绝不回显
	deniedColumns map[string]struct{}
	// colLoaded 已查询过列明细的表（小写）：无列的表也只查一次
	colLoaded map[string]struct{}
	// relLoaded 已查询过外键的表（小写）：ensureRelations 的去重账本
	relLoaded map[string]struct{}
	// relUnsupported 方言未实现外键内省：记一次就不再重复发注定失败的查询
	relUnsupported bool
	// fkSeen 外键边去重（两端都被选中时同一条边会被读到两次）
	fkSeen map[string]struct{}
	fks    []fkEdge
}

// TableAcl 表列权限切片（消费方窄接口：只声明本包真正用到的三个方法）
//
// 单独声明在这里而不是复用 sqlexec.TableAcl：本包只需要「可见表 / 无权列 / dry 判定」
// 三件事，其余（规则 CRUD、结果裁剪）与本包无关。
type TableAcl interface {
	// VisibleTables 过滤出当前账号可见的表
	VisibleTables(ctx context.Context, dbId uint64, tables []string) ([]string, error)
	// DeniedColumns 当前账号在这些表上无权看到/使用的列名（小写）
	DeniedColumns(ctx context.Context, dbId uint64, tables []string) (map[string]struct{}, error)
	// AssertStatementAllowed 语句级判定的 dry 调用（只用于提示，不拦截生成）
	AssertStatementAllowed(ctx context.Context, dbId uint64, tables []string, resolvable bool, refs []sqlutil.ColumnRef) error
}

// buildSnapshot 读取库结构并按权限裁剪。
//
// 只取表清单（轻）+ 无权列（一次），列明细留给 ensureColumns 按选中表拉取：
// 大库上「先全量取列」会让一次生成退化成几秒的结构洪泛，而真正进上下文的表不超过 12 张。
func buildSnapshot(ctx context.Context, meta metadataReader, dialect dbi.Dialect, dbType dbi.DbType, acl TableAcl, dbId uint64, db string) (*snapshot, error) {
	all, err := meta.GetTables()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(all))
	byNameAll := make(map[string]*tableInfo, len(all))
	for _, table := range all {
		info := &tableInfo{Name: table.TableName, Comment: strings.TrimSpace(table.TableComment)}
		names = append(names, table.TableName)
		byNameAll[strings.ToLower(table.TableName)] = info
	}

	visible := names
	if acl != nil {
		visible, err = acl.VisibleTables(ctx, dbId, names)
		if err != nil {
			// 权限判定读不出来时不能退化成「全部可见」：那正是越权路径。宁可整次失败
			return nil, err
		}
	}
	visibleSet := make(map[string]struct{}, len(visible))
	tables := make([]*tableInfo, 0, len(visible))
	for _, name := range visible {
		key := strings.ToLower(name)
		visibleSet[key] = struct{}{}
		if info, ok := byNameAll[key]; ok {
			tables = append(tables, info)
		}
	}
	hidden := make(map[string]struct{}, len(byNameAll))
	for key := range byNameAll {
		if _, ok := visibleSet[key]; !ok {
			hidden[key] = struct{}{}
		}
	}

	denied := map[string]struct{}{}
	if acl != nil && len(visible) > 0 {
		denied, err = acl.DeniedColumns(ctx, dbId, visible)
		if err != nil {
			return nil, err
		}
	}

	snap := &snapshot{
		db:            db,
		dialect:       string(dbType),
		tables:        tables,
		byName:        byNameAll,
		visible:       visibleSet,
		hidden:        hidden,
		deniedColumns: denied,
		colLoaded:     make(map[string]struct{}, len(tables)),
		relLoaded:     make(map[string]struct{}, len(tables)),
		fkSeen:        make(map[string]struct{}, len(tables)),
	}
	if server, sErr := meta.GetDbServer(); sErr == nil && server != nil {
		snap.version = server.Version
	}
	// 引用符从方言的 Quoter 实测派生：`Quote("x")` 的首字符就是该库的标识符包裹符。
	// 不另维护「方言 → 引用符」表——那必然是第二份真源，且新增方言时静默漏配。
	if dialect != nil {
		if quoted := dialect.Quoter().Quote("x"); len(quoted) >= 2 {
			snap.quotePrefix = quoted[:1]
		}
	}
	return snap, nil
}

// visibleTables 可见表名清单（给「选择题」与召回用）
func (s *snapshot) visibleTables() []*tableInfo {
	return s.tables
}

// visibleNames 可见表名（候选池，用于给出「最相近的表名」修复提示）
func (s *snapshot) visibleNames() []string {
	names := make([]string, 0, len(s.tables))
	for _, table := range s.tables {
		names = append(names, table.Name)
	}
	return names
}

func (s *snapshot) find(name string) (*tableInfo, bool) {
	key := strings.ToLower(strings.TrimSpace(name))
	if _, ok := s.visible[key]; !ok {
		return nil, false
	}
	info, ok := s.byName[key]
	return info, ok
}

func (s *snapshot) isHidden(name string) bool {
	_, ok := s.hidden[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

func (s *snapshot) isColumnDenied(name string) bool {
	_, ok := s.deniedColumns[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

// ensureColumns 按需装载表的列明细（含主键标记，剔除无权列）。
//
// 逐表调用而**不是**一次批量取：MetadataReader 对单表列查询走跨请求的进程内 schema 缓存，
// 多表批量只有请求级缓存。批量看似少几次往返，实际会绕过缓存、每次生成都真查一遍
// information_schema，还顺带丢掉「为补全子系统预热列缓存」的副作用。
//
// 不返回 error：单表取列失败（视图/权限受限的表会一直失败）不终止整次生成，
// 让整次失败等于「一张坏表挡住整个库」。缺列的事实由校验层如实标出
// （见 identifierChecker → columnExists：无列明细时不判「列不存在」）。
func (s *snapshot) ensureColumns(ctx context.Context, meta metadataReader, tables []string) {
	pending := make([]*tableInfo, 0, len(tables))
	for _, name := range tables {
		info, ok := s.find(name)
		if !ok {
			continue
		}
		key := strings.ToLower(info.Name)
		if _, loaded := s.colLoaded[key]; loaded {
			continue
		}
		s.colLoaded[key] = struct{}{}
		pending = append(pending, info)
	}
	for _, info := range pending {
		columns, err := meta.GetColumns(info.Name)
		if err != nil {
			logx.WarnfContext(ctx, "[sqldraft] load columns of table %s failed: %s", info.Name, err.Error())
			continue
		}
		list := make([]columnInfo, 0, len(columns))
		for _, column := range columns {
			if s.isColumnDenied(column.ColumnName) {
				continue
			}
			list = append(list, columnInfo{
				Name:       column.ColumnName,
				Type:       column.ColumnType,
				Comment:    strings.TrimSpace(column.ColumnComment),
				PrimaryKey: column.IsPrimaryKey,
			})
		}
		info.Columns = list
	}
}

// ensureRelations 按需装载表的外键关系（幂等：同一张表只查一次）。
//
// 召回阶段的外键邻接策略与渲染阶段的关系片段都要用它，因此装载入口收在快照上：
// 谁需要谁 ensure，编排层不必预知「哪一步要用外键」。
func (s *snapshot) ensureRelations(ctx context.Context, meta metadataReader, tables []string) {
	if s.relUnsupported {
		return
	}
	for _, name := range tables {
		info, ok := s.find(name)
		if !ok {
			continue
		}
		key := strings.ToLower(info.Name)
		if _, loaded := s.relLoaded[key]; loaded {
			continue
		}
		s.relLoaded[key] = struct{}{}
		edges, err := meta.GetForeignKeys(ctx, "", info.Name)
		if err != nil {
			if errors.Is(err, dbi.ErrUnsupportedKind) {
				// 方言未实现外键内省属正常边界：没有关系片段，但表结构片段仍然可用。
				// 记一次就不再对其余表重复发注定失败的查询
				s.relUnsupported = true
				return
			}
			logx.WarnfContext(ctx, "[sqldraft] load foreign keys of table %s failed: %s", info.Name, err.Error())
			continue
		}
		for _, edge := range edges {
			edgeKey := strings.ToLower(edge.Table + "." + edge.Column + ">" + edge.RefTable + "." + edge.RefColumn)
			if _, dup := s.fkSeen[edgeKey]; dup {
				continue
			}
			s.fkSeen[edgeKey] = struct{}{}
			s.fks = append(s.fks, fkEdge{
				FromTable: edge.Table, FromColumn: edge.Column,
				ToTable: edge.RefTable, ToColumn: edge.RefColumn,
			})
		}
	}
}

// dialectLabel 方言展示标签（如 mysql 8.0.36）
func (s *snapshot) dialectLabel() string {
	if s.version == "" {
		return s.dialect
	}
	return s.dialect + " " + s.version
}
