package schema

// 表结构对比与漂移检测：比较两侧库的表结构，摊开差异。
//
// 只报差异、不生成 DDL：跨方言生成 ALTER 要处理类型映射、生成列、表达式默认值、大小写与引号规则，
// 任何一处猜错都会变成「执行成功但结构被改坏」。差异交给人判断，真要改结构走各自的变更入口
// （那边有预检与审批）。
//
// 比较逻辑（DiffSnapshots）是纯函数：输入两侧的内省快照，不连库。跨方言的类型归一化规则
// （varchar 与 character varying 属同一类）能被各种数据形状单测覆盖，而「快照是否读得准」
// 由真实数据库集成测试负责，两件事不在一处混着验。

import (
	"sort"
	"strconv"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
)

// 表级差异类型
const (
	// TableOnlyInSource 源侧有、目标侧没有（目标缺这张表）
	TableOnlyInSource = "only-in-source"
	// TableOnlyInTarget 目标侧多出来的表（源侧已删除，或目标侧自己建的）
	TableOnlyInTarget = "only-in-target"
	// TableChanged 两侧都有但结构有差异
	TableChanged = "changed"
	// TableSame 两侧一致
	TableSame = "same"
)

// 列级差异类型
const (
	ColumnOnlyInSource = "column-only-in-source"
	ColumnOnlyInTarget = "column-only-in-target"
	ColumnTypeChanged  = "column-type-changed"
	ColumnSizeChanged  = "column-size-changed"
	ColumnNullableDiff = "column-nullable-diff"
	ColumnDefaultDiff  = "column-default-diff"
	ColumnAutoIncDiff  = "column-auto-increment-diff"
)

// TableSnapshot 一张表在某一侧的结构事实（由方言内省得到）
type TableSnapshot struct {
	Name    string
	Columns []dbi.Column

	// PrimaryKeys 有序主键列名（联合主键为多列）
	PrimaryKeys []string
	// PrimaryKeysAvailable 主键是否真的读到了。内省失败与「这张表没有主键」是两回事：
	// 都当成无主键会把一次权限受限的读取说成结构差异
	PrimaryKeysAvailable bool

	Indexes []dbi.Index
	// IndexesAvailable 同上，索引内省是否成功
	IndexesAvailable bool

	// DbType 该侧方言类型，用于按方言做类型归一化（否则跨方言比原始类型文本会到处误报）
	DbType string
}

// ColumnDiff 一列的差异
type ColumnDiff struct {
	Column string `json:"column"`
	Kind   string `json:"kind"`
	// Source/Target 两侧的可读描述；一侧没有该列时该侧为空
	Source string `json:"source,omitempty"`
	Target string `json:"target,omitempty"`
}

// IndexDiff 一个索引的差异（按列组合比对，不按索引名）
type IndexDiff struct {
	Kind    string `json:"kind"`
	Columns string `json:"columns"`
	Source  string `json:"source,omitempty"`
	Target  string `json:"target,omitempty"`
}

// KeyDiff 主键差异
type KeyDiff struct {
	Kind   string `json:"kind"`
	Source string `json:"source,omitempty"`
	Target string `json:"target,omitempty"`
}

// Unavailable 一处「比不了」的记录：不比较不等于一致
type Unavailable struct {
	Aspect string `json:"aspect"`
	Reason string `json:"reason"`
}

// TableDiff 一张表的对比结果
type TableDiff struct {
	Table      string       `json:"table"`
	Kind       string       `json:"kind"`
	Columns    []ColumnDiff `json:"columns,omitempty"`
	Indexes    []IndexDiff  `json:"indexes,omitempty"`
	PrimaryKey *KeyDiff     `json:"primaryKey,omitempty"`
	// Unavailable 该表比不了的方面（如索引内省失败、跨方言的自增属性不可比）
	Unavailable []Unavailable `json:"unavailable,omitempty"`
}

// Result 一次结构对比的结果
type Result struct {
	Source string `json:"source"`
	Target string `json:"target"`
	// Compared 实际参与比对的表数（截断后）
	Compared int `json:"compared"`
	// DiffTables 有差异或被判定为「比不了」的表数
	DiffTables int          `json:"diffTables"`
	Tables     []*TableDiff `json:"tables,omitempty"`
	// Note 全局如实说明，如比对的表数达到上限被截断
	Note string `json:"note,omitempty"`
}

// diffContext 一次比对的两侧方言信息
type diffContext struct {
	srcType dbi.DbType
	dstType dbi.DbType
	// dstDialect 目标侧方言，用于拿迁移侧已有的类型映射规则判等价（见 sameTypeAcrossDialect）
	dstDialect dbi.Dialect
	// sameDialect 是否同方言：跨方言时一些属性（自增）的呈现方式根本不同，不能直接判等
	sameDialect bool
}

func newDiffContext(src, dst []TableSnapshot) diffContext {
	ctx := diffContext{sameDialect: true}
	if len(src) > 0 {
		ctx.srcType = dbi.DbType(src[0].DbType)
	}
	if len(dst) > 0 {
		ctx.dstType = dbi.DbType(dst[0].DbType)
	}
	ctx.sameDialect = ctx.srcType == ctx.dstType
	if !ctx.sameDialect {
		ctx.dstDialect = dbi.GetDialect(ctx.dstType)
	}
	return ctx
}

// DiffSnapshots 比对两侧表快照。
//
// 表名按大小写不敏感匹配：mysql 的表名大小写敏感性取决于配置、pg 默认折叠成小写，
// 按原样比对会让同一张表被判成「一侧多一侧少」
func DiffSnapshots(src, dst []TableSnapshot) []*TableDiff {
	ctx := newDiffContext(src, dst)
	srcByName := indexSnapshots(src)
	dstByName := indexSnapshots(dst)

	names := make([]string, 0, len(srcByName))
	seen := make(map[string]bool, len(srcByName)+len(dstByName))
	for _, table := range src {
		key := strings.ToLower(table.Name)
		if !seen[key] {
			seen[key] = true
			names = append(names, key)
		}
	}
	for _, table := range dst {
		key := strings.ToLower(table.Name)
		if !seen[key] {
			seen[key] = true
			names = append(names, key)
		}
	}
	sort.Strings(names)

	diffs := make([]*TableDiff, 0, len(names))
	for _, name := range names {
		source, srcOk := srcByName[name]
		target, dstOk := dstByName[name]
		switch {
		case srcOk && !dstOk:
			diffs = append(diffs, &TableDiff{Table: source.Name, Kind: TableOnlyInSource})
		case !srcOk && dstOk:
			diffs = append(diffs, &TableDiff{Table: target.Name, Kind: TableOnlyInTarget})
		default:
			if diff := diffTable(ctx, source, target); diff != nil {
				diffs = append(diffs, diff)
			}
		}
	}
	return diffs
}

func indexSnapshots(snapshots []TableSnapshot) map[string]TableSnapshot {
	res := make(map[string]TableSnapshot, len(snapshots))
	for _, snapshot := range snapshots {
		res[strings.ToLower(snapshot.Name)] = snapshot
	}
	return res
}

// diffTable 比对两侧都存在的表；无差异返回 nil（不制造一条 same 记录，界面上只列差异）
func diffTable(ctx diffContext, src, dst TableSnapshot) *TableDiff {
	diff := &TableDiff{Table: src.Name, Kind: TableChanged}
	// 一侧读不到任何列时不比列：那会把对面每一列都报成「只在一侧存在」，
	// 而真实情况是一次内省没拿到数据，两者对用户的处置完全不同
	if len(src.Columns) == 0 || len(dst.Columns) == 0 {
		diff.Unavailable = append(diff.Unavailable, Unavailable{Aspect: "columns", Reason: unavailableSideText(len(src.Columns), len(dst.Columns))})
	} else {
		var uncomparable []string
		diff.Columns, uncomparable = diffColumns(ctx, src, dst)
		if len(uncomparable) > 0 {
			diff.Unavailable = append(diff.Unavailable, Unavailable{
				Aspect: "column-types",
				Reason: "types not recognised by the dialect engine, so they were not compared: " + strings.Join(uncomparable, ", "),
			})
		}
	}
	diff.Indexes, diff.Unavailable = diffIndexes(src, dst, diff.Unavailable...)
	// 主键读不到与真的没有主键是两回事：前者只能标「比不了」，
	// 当成无主键会把一次权限受限的内省说成结构缺陷
	if !src.PrimaryKeysAvailable || !dst.PrimaryKeysAvailable {
		diff.Unavailable = append(diff.Unavailable, Unavailable{Aspect: "primary-key", Reason: unavailableKeyReason(src, dst)})
	}
	diff.PrimaryKey = diffPrimaryKey(src, dst)

	if len(diff.Columns) == 0 && len(diff.Indexes) == 0 && diff.PrimaryKey == nil && len(diff.Unavailable) == 0 {
		return nil
	}
	return diff
}

func unavailableSideText(srcCount, dstCount int) string {
	switch {
	case srcCount == 0 && dstCount == 0:
		return "no columns could be read from either side"
	case srcCount == 0:
		return "no columns could be read from the source"
	default:
		return "no columns could be read from the target"
	}
}

// unavailableKeyReason 指出是哪一侧没读到主键：「确实无主键」是合法结构（PrimaryKeysAvailable=true
// 且长度为 0），只能把「读不到」标成比不了，否则一次权限受限的内省就成了结构缺陷
func unavailableKeyReason(src, dst TableSnapshot) string {
	switch {
	case !src.PrimaryKeysAvailable && !dst.PrimaryKeysAvailable:
		return "primary keys could not be read from either side"
	case !src.PrimaryKeysAvailable:
		return "primary keys could not be read from the source"
	default:
		return "primary keys could not be read from the target"
	}
}

func diffColumns(ctx diffContext, src, dst TableSnapshot) ([]ColumnDiff, []string) {
	srcCols := columnsByName(src.Columns)
	dstCols := columnsByName(dst.Columns)

	diffs := make([]ColumnDiff, 0, len(srcCols))
	uncomparable := make([]string, 0, 2)
	for _, name := range sortedColumnNames(srcCols, dstCols) {
		source, inSrc := srcCols[name]
		target, inDst := dstCols[name]
		switch {
		case inSrc && !inDst:
			diffs = append(diffs, ColumnDiff{Column: source.ColumnName, Kind: ColumnOnlyInSource, Source: describeColumn(ctx.srcType, source)})
		case !inSrc && inDst:
			diffs = append(diffs, ColumnDiff{Column: target.ColumnName, Kind: ColumnOnlyInTarget, Target: describeColumn(ctx.dstType, target)})
		default:
			pairDiffs, unknown := diffColumnPair(ctx, source, target)
			diffs = append(diffs, pairDiffs...)
			if unknown {
				uncomparable = append(uncomparable, source.ColumnName)
			}
		}
	}
	return diffs, uncomparable
}

// diffColumnPair 两侧都有该列时逐项比：类型（归一化后）→ 长度/精度 → 可空 → 默认值 → 自增。
// 第二个返回值表示该列的类型认不出来（调用方需把它标成比不了）
func diffColumnPair(ctx diffContext, source, target dbi.Column) ([]ColumnDiff, bool) {
	name := source.ColumnName
	diffs := make([]ColumnDiff, 0, 3)

	srcCategory, srcKnown := resolveCategory(ctx.srcType, source.DataType)
	dstCategory, dstKnown := resolveCategory(ctx.dstType, target.DataType)
	if !srcKnown || !dstKnown {
		// 认不出名字就不比类型也不报差异：GetDbDataType 会把未注册名静默当成 varchar，
		// 拿那个归类去比会把一个自定义/空间类型列与对面的 varchar 判成「相同」
		return diffColumnRest(ctx, source, target, nil, nil), !srcKnown || !dstKnown
	}
	if srcCategory != dstCategory {
		if sameTypeAcrossDialect(ctx, source, target, dstCategory) {
			// 差异可由迁移侧的既有类型映射解释（如 mysql blob → pg bytea）：那是本来就会做的转换，不是漂移
			return diffColumnRest(ctx, source, target, nil, nil), false
		}
		diffs = append(diffs, ColumnDiff{
			Column: name,
			Kind:   ColumnTypeChanged,
			Source: describeColumn(ctx.srcType, source),
			Target: describeColumn(ctx.dstType, target),
		})
		// 类型都不同就不再比长度/精度：那是同一条差异的第二种说法，只会让界面重复
		return diffs, false
	}
	return diffColumnRest(ctx, source, target, &srcCategory, &dstCategory), false
}

// diffColumnRest 类型层面不比之后剩下的可空/默认值/自增三项（它们与类型归类无关，仍能如实比对）
func diffColumnRest(ctx diffContext, source, target dbi.Column, srcCategory, dstCategory *dbi.TypeCategory) []ColumnDiff {
	name := source.ColumnName
	diffs := make([]ColumnDiff, 0, 3)
	if srcCategory != nil && dstCategory != nil {
		if sizeDiff, ok := compareSize(*srcCategory, *dstCategory, source, target); ok {
			diffs = append(diffs, ColumnDiff{Column: name, Kind: ColumnSizeChanged, Source: sizeDiff.src, Target: sizeDiff.dst})
		}
	}

	if source.Nullable != target.Nullable {
		diffs = append(diffs, ColumnDiff{
			Column: name,
			Kind:   ColumnNullableDiff,
			Source: nullableText(source.Nullable),
			Target: nullableText(target.Nullable),
		})
	}

	if defaultDiff, ok := compareDefault(source, target); ok {
		diffs = append(diffs, ColumnDiff{Column: name, Kind: ColumnDefaultDiff, Source: defaultDiff.src, Target: defaultDiff.dst})
	}

	// 自增只在同方言内比：mysql 的 AUTO_INCREMENT 与 pg 的 identity/serial 在元数据里形态不同，
	// 跨方言判等会把每一列都报成差异
	if ctx.sameDialect && source.AutoIncrement != target.AutoIncrement {
		diffs = append(diffs, ColumnDiff{
			Column: name,
			Kind:   ColumnAutoIncDiff,
			Source: boolText(source.AutoIncrement),
			Target: boolText(target.AutoIncrement),
		})
	}
	return diffs
}

type pair struct{ src, dst string }

// compareSize 比较长度或精度：只在两侧同类且两侧都给出数值时比。
//
// 0 在这里的含义是「该方言没提供」而不是「长度为 0」，拿它判等会造出一批假差异
func compareSize(srcCategory, dstCategory dbi.TypeCategory, source, target dbi.Column) (pair, bool) {
	if srcCategory != dstCategory {
		return pair{}, false
	}
	switch {
	case isStringCategory(srcCategory) && isStringCategory(dstCategory):
		if source.CharMaxLength > 0 && target.CharMaxLength > 0 && source.CharMaxLength != target.CharMaxLength {
			return pair{src: strconv.Itoa(source.CharMaxLength), dst: strconv.Itoa(target.CharMaxLength)}, true
		}
	case dbi.IsNumericCategory(srcCategory) && dbi.IsNumericCategory(dstCategory):
		if source.NumPrecision > 0 && target.NumPrecision > 0 && source.NumPrecision != target.NumPrecision {
			return pair{src: strconv.Itoa(source.NumPrecision), dst: strconv.Itoa(target.NumPrecision)}, true
		}
		if source.NumScale > 0 && target.NumScale > 0 && source.NumScale != target.NumScale {
			return pair{src: strconv.Itoa(source.NumScale), dst: strconv.Itoa(target.NumScale)}, true
		}
	}
	return pair{}, false
}

// compareDefault 比较默认值。表达式默认值不参与比较：各方言的函数名与语法互不相通
// （pg 的 gen_random_uuid()、mysql 的 current_timestamp(3)），跨库比出来的一定是「不同」，
// 而那不是漂移
func compareDefault(source, target dbi.Column) (pair, bool) {
	if source.IsExprDefault || target.IsExprDefault {
		return pair{}, false
	}
	src, dst := normalizeDefault(source.ColumnDefault), normalizeDefault(target.ColumnDefault)
	if src == dst {
		return pair{}, false
	}
	return pair{src: src, dst: dst}, true
}

func diffPrimaryKey(src, dst TableSnapshot) *KeyDiff {
	if !src.PrimaryKeysAvailable || !dst.PrimaryKeysAvailable {
		return nil
	}
	srcKey := strings.Join(lowerAll(src.PrimaryKeys), ",")
	dstKey := strings.Join(lowerAll(dst.PrimaryKeys), ",")
	if srcKey == dstKey {
		return nil
	}
	return &KeyDiff{Kind: "primary-key-changed", Source: srcKey, Target: dstKey}
}

// diffIndexes 按「索引覆盖的列组合」比对，不比索引名：
// 索引名是各库自取的（mysql 用 PRIMARY、pg 用表名+_pkey），按名比对会把同一个索引说成两个
func diffIndexes(src, dst TableSnapshot, carried ...Unavailable) ([]IndexDiff, []Unavailable) {
	if !src.IndexesAvailable || !dst.IndexesAvailable {
		// 内省失败不等于「这一侧没有索引」：报成缺失会引导用户去重建一个本来就存在的索引
		side := "target"
		switch {
		case !src.IndexesAvailable && !dst.IndexesAvailable:
			side = "both sides"
		case !src.IndexesAvailable:
			side = "source"
		}
		return nil, append(carried, Unavailable{Aspect: "indexes", Reason: "index introspection failed on the " + side})
	}

	srcByKey := indexByKey(src.Indexes)
	dstByKey := indexByKey(dst.Indexes)

	diffs := make([]IndexDiff, 0, len(srcByKey))
	for _, key := range sortedKeys(srcByKey, dstByKey) {
		source, inSrc := srcByKey[key]
		target, inDst := dstByKey[key]
		switch {
		case inSrc && !inDst:
			diffs = append(diffs, IndexDiff{Kind: "index-only-in-source", Columns: key, Source: indexText(source)})
		case !inSrc && inDst:
			diffs = append(diffs, IndexDiff{Kind: "index-only-in-target", Columns: key, Target: indexText(target)})
		case firstUnique(source) != firstUnique(target):
			diffs = append(diffs, IndexDiff{
				Kind:    "index-unique-diff",
				Columns: key,
				Source:  boolText(firstUnique(source)),
				Target:  boolText(firstUnique(target)),
			})
		}
	}
	return diffs, carried
}

// indexByKey 把平铺的索引记录按（列组合）归并；主键索引不参与（另有 diffPrimaryKey）
func indexByKey(indexes []dbi.Index) map[string][]dbi.Index {
	grouped := make(map[string][]dbi.Index, len(indexes))
	byName := make(map[string][]dbi.Index)
	for _, index := range indexes {
		if index.IsPrimaryKey {
			continue
		}
		name := strings.ToLower(index.IndexName)
		byName[name] = append(byName[name], index)
	}
	for _, records := range byName {
		sort.Slice(records, func(i, j int) bool { return records[i].SeqInIndex < records[j].SeqInIndex })
		columns := strings.ToLower(joinColumnNames(records))
		if columns == "" {
			continue
		}
		grouped[columns] = append(grouped[columns], records...)
	}
	return grouped
}

func joinColumnNames(records []dbi.Index) string {
	names := make([]string, 0, len(records))
	for _, record := range records {
		names = append(names, record.ColumnName)
	}
	return strings.Join(names, ",")
}

func firstUnique(records []dbi.Index) bool {
	for _, record := range records {
		return record.IsUnique
	}
	return false
}

func indexText(records []dbi.Index) string {
	if len(records) == 0 {
		return ""
	}
	name := records[0].IndexName
	if firstUnique(records) {
		return name + " (unique)"
	}
	return name
}

func columnsByName(columns []dbi.Column) map[string]dbi.Column {
	res := make(map[string]dbi.Column, len(columns))
	for _, column := range columns {
		res[strings.ToLower(column.ColumnName)] = column
	}
	return res
}

func sortedColumnNames(maps ...map[string]dbi.Column) []string {
	seen := make(map[string]bool)
	names := make([]string, 0, len(maps[0]))
	for _, m := range maps {
		for key := range m {
			if !seen[key] {
				seen[key] = true
				names = append(names, key)
			}
		}
	}
	sort.Strings(names)
	return names
}

func sortedKeys(maps ...map[string][]dbi.Index) []string {
	seen := make(map[string]bool)
	keys := make([]string, 0, len(maps[0]))
	for _, m := range maps {
		for key := range m {
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	sort.Strings(keys)
	return keys
}

// sameTypeAcrossDialect 跨方言时用迁移侧已有的类型映射规则判等价：把源列转到目标方言后再比类别。
//
// 不自建一张「类型等价表」：那是第二套真相，与迁移规则一旦不同步，报出的差异就不可信；
// mysql 的 blob 对 pg 的 bytea 正是迁移本来的产物，当成漂移报出来只会淹没真差异
func sameTypeAcrossDialect(ctx diffContext, source, target dbi.Column, dstCategory dbi.TypeCategory) bool {
	if ctx.sameDialect || ctx.dstDialect == nil {
		return false
	}
	converted := source
	if err := dbi.ConvToTargetDbColumn(ctx.srcType, ctx.dstType, ctx.dstDialect, &converted); err != nil {
		// 映射不过去（目标方言接不住这个类别）就是真差异，不当作等价
		return false
	}
	convertedCategory, ok := resolveCategory(ctx.dstType, converted.DataType)
	return ok && convertedCategory == dstCategory
}

// isStringCategory 字符类列：长度（CharMaxLength）对它们才有可比的意义
func isStringCategory(category dbi.TypeCategory) bool {
	switch category {
	case dbi.TCVarchar, dbi.TCChar:
		return true
	}
	return false
}

// resolveCategory 列类型的跨方言归类。known=false 表示该方言的类型引擎认不出这个名字。
//
// 不直接用 GetDbDataType：它对未注册名静默回落成 DefaultDbDataType(TCVarchar)，
// 拿那个归类去比会把一个自定义/空间类型列与对面的 varchar 判成「相同」——
// 认不出来既不是相同也不是不同，只能不比并标出来
func resolveCategory(dbType dbi.DbType, dataType string) (dbi.TypeCategory, bool) {
	engine := dbi.GetTypeEngine(dbType)
	if engine == nil {
		return dbi.TCUnknown, false
	}
	resolved, ok := engine.ResolveType(dataType)
	if !ok {
		return dbi.TCUnknown, false
	}
	return resolved.Category(), true
}

// describeColumn 差异里给一侧列的可读描述：优先用驱动给出的完整类型（含长度/精度），
// 没给则退到基础类型名；两者都没有就留空，不拿归类名去冒充库侧类型
func describeColumn(_ dbi.DbType, column dbi.Column) string {
	if text := strings.TrimSpace(column.ColumnType); text != "" {
		return text
	}
	return strings.TrimSpace(column.DataType)
}

func normalizeDefault(value string) string {
	trimmed := strings.TrimSpace(value)
	// 两侧对同一个「无默认值」的写法不同（mysql 给 NULL 字符串、pg 给 'NULL'、有的驱动给空串），
	// 不归一就会造出一批「默认值不同」的假差异
	if strings.EqualFold(trimmed, "null") {
		return ""
	}
	// 字符串默认值各方言是否带引号不一致，去掉包裹引号后比较内容
	if len(trimmed) >= 2 {
		if (trimmed[0] == '\'' && trimmed[len(trimmed)-1] == '\'') || (trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"') {
			return trimmed[1 : len(trimmed)-1]
		}
	}
	return trimmed
}

func lowerAll(values []string) []string {
	res := make([]string, 0, len(values))
	for _, value := range values {
		res = append(res, strings.ToLower(strings.TrimSpace(value)))
	}
	return res
}

func nullableText(nullable bool) string {
	if nullable {
		return "NULL"
	}
	return "NOT NULL"
}

func boolText(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
