// Package shard 提供「单表 + 单列整型主键」数据集的范围分片规划能力。
//
// 为什么独立成包：分片规划原本是数据迁移（transfer）的私有实现，数据同步（sync）为了
// 把「一整轮只发一次的大 SELECT」拆成多条短查询（否则源库按 query_timeout 判定为慢查询并
// kill 掉整条流，大表同步永远跑不完）需要完全相同的区间切分算法。两处各写一份区间数学
// 迟早漂移，故收在此处，迁移与同步共用同一实现。
//
// 归属应用层而非 dbi 内核：按主键区间切分是「数据搬运的执行策略」，不是方言能力——
// 内核只提供 SQL 生成与元数据读取，方言差异仅体现在标识符引用（由调用方传入 quote）。
package shard

import (
	"context"
	"fmt"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/dbi/value"
	"mayfly-go/pkg/logx"
)

// TargetRows 单个分片的目标行数（~20万行/片）。
// var形式仅为集成测试可调小以验证多分片路径，生产代码勿修改；
// 需要按任务粒度控制单片大小时走 Plan/PlanTable 的 targetRows 参数（数据同步即如此）。
var TargetRows = 200000

// MinTargetRows 任务级自定义单片行数的下限：低于此值分片数会失控（每片一次往返的固定开销
// 远超数据读取本身），且单片行数太少时源库要重复做 N 次扫描规划。
const MinTargetRows = 1000

// MaxTargetRows 任务级自定义单片行数的上限：单片过大等于没分片，长查询被源库超时 kill 的
// 问题原样保留，把它拦在配置阶段比等到运行期报错更好定位。
const MaxTargetRows = 5000000

// MaxCount 单表最大分片数（防止主键区间极大时产生天文数字分片；
// 超出后单片跨度变大，仍能完整覆盖区间）
const MaxCount = 10000

// Shard 数据分片：按主键范围切分的搬运子任务
type Shard struct {
	// Where 分片过滤条件，如 `id >= 1 AND id <= 200000`；空表示无过滤（整表/单分片）
	Where string
}

// NormalizeTargetRows 归一化任务级单片行数配置：<=0 表示未配置，取包默认值；
// 显式配置值越界则钳制到 [MinTargetRows, MaxTargetRows]。
//
// 只钳任务值不钳包默认值：任务值是不可信的外部输入（表单/API 直连都能写），放任 1 行/片
// 会让分片数直接顶到上限，把一轮同步变成上万次往返；而 TargetRows 默认值是代码内常量、
// 仅供集成测试调小以在小夹具表上走出多分片路径，不属于外部输入面。
//
// 钳制而不报错：单片行数只是性能旋钮，配错量级不值得让整轮同步失败。
func NormalizeTargetRows(targetRows int) int {
	if targetRows <= 0 {
		return TargetRows
	}
	if targetRows < MinTargetRows {
		return MinTargetRows
	}
	if targetRows > MaxTargetRows {
		return MaxTargetRows
	}
	return targetRows
}

// DetectIntPrimaryKey 判定表是否存在**真实的单列整型主键**，返回主键列名；否则返回 ""（不应分片）。
//
// 注意：不依赖 GetPrimaryKeys（其只给列名，不含类型/宽度信息），而是直接读 GetColumns 的
// IsPrimaryKey + 数据类型判定——分片需"单列、整型"主键才能做数值范围切分，否则按下列规则不分片：
//   - 联合主键（多个IsPrimaryKey列）→ 不分片
//   - 无主键 → 不分片
//   - 非整型主键（varchar/复合类型等，无法做数值范围切分）→ 不分片
func DetectIntPrimaryKey(columns []dbi.Column) string {
	pk := ""
	pkCount := 0
	for _, c := range columns {
		if !c.IsPrimaryKey {
			continue
		}
		pkCount++
		pk = c.ColumnName
	}
	// 可空性不在这里判：很多方言的列元数据把「未显式声明 NOT NULL」一律报成可空（sqlite 即如此，
	// 真实 PK 也报可空），拿它当拒分片条件会让整个 sqlite 矩阵静默不分片（恰是本机制要治的形态）。
	// NULL 主键行会被任何比较谓词排除，因此正确修法是分片列表里额外带一片 `pk IS NULL`（见 withNullShard）。
	if pkCount != 1 || !isIntegerDataType(pkDataTypeOf(columns, pk)) {
		return ""
	}
	return pk
}

// DetectIntShardKey 选出可作为「分片键」的整型列：优先单列整型主键，其次单列整型唯一索引。
//
// 为什么允许唯一索引而不只是主键：数仓表（StarRocks/Doris 的 Unique Key 模型）和只把 id 建成
// unique、没声明 PRIMARY KEY 的 MySQL 表都没有主键，但存在唯一列。分片 + 片内续扫的正确性只依赖
// 「该列在一行上唯一」，主键只是它的特例。
//
// 为什么不接受任意整型列（哪怕任务里显式指定）：片内续扫点记的是「最后读到的键值」，续扫谓词是
// `key > 该值`。列若可重复，同一键值的多行会被拆到两个查询里，`>` 会把后一个查询里同键值的未读行
// 跳过——静默漏行；改成 `>=` 补齐又会重复写入（追加模式下就是重复行）。唯一性是可证明的最小条件，
// 而且必须由数据库元数据（主键/唯一索引）保证，不能靠用户口头声明。
func DetectIntShardKey(columns []dbi.Column, indexes []dbi.Index) string {
	if pk := DetectIntPrimaryKey(columns); pk != "" {
		return pk
	}
	// 只有单列唯一索引可用：联合唯一索引的单个分量不唯一，拿它做续扫键会漏行
	for _, idx := range dbi.GroupIndexColumns(indexes) {
		if !idx.IsUnique || idx.IsPrimaryKey {
			continue
		}
		column := strings.TrimSpace(idx.ColumnName)
		if column == "" || strings.Contains(column, ",") {
			continue
		}
		c, ok := columnOf(columns, column)
		if !ok || !isIntegerDataType(c.DataType) {
			continue
		}
		return c.ColumnName
	}
	return ""
}

// columnOf 按列名（忽略大小写）从元数据里取出列定义。
func columnOf(columns []dbi.Column, columnName string) (dbi.Column, bool) {
	for _, c := range columns {
		if strings.EqualFold(c.ColumnName, columnName) {
			return c, true
		}
	}
	return dbi.Column{}, false
}

// pkNullable 判断主键列在元数据里是否可空（只用于决定要不要追加 NULL 片，不用于拒分片）。
func pkNullable(columns []dbi.Column, pk string) bool {
	for _, c := range columns {
		if c.ColumnName == pk {
			return c.Nullable
		}
	}
	return false
}

// withNullShard 在分片列表末尾追加一片 `pk IS NULL`（仅当真的探测到 NULL 主键行时）。
//
// 为什么必须可能有这一片：任何 `pk <= x` / `pk >= x` 都不匹配 NULL（比较运算遇 NULL 得
// UNKNOWN），主键为 NULL 的行在纯区间布局下会**静默不同步**——单次大查询读得到它们，分片后
// 读不到，这是分片引入的新丢失类别。补上后各片并集才覆盖整个值域（NULL ∪ 全键空间）。
//
// 不拿元数据的可空位直接决定加不加：很多方言把「未显式声明 NOT NULL」的列一律报成可空
// （sqlite 连真实主键也报可空），那样每次分片都会多一个永远为空的可检索片段；而反过来
// 拿它当拒分片的条件又会把整个 sqlite 矩阵静默打回不分片。所以先用可空位做廉价预筛，
// 再用一次索引扫描证实（见 pkHasNullRows）。
//
// 「整表一片」（Where 为空，无谓词）本身已包含 NULL 行，不重复追加。
func withNullShard(pkColumn string, shards []Shard, hasNullRows bool, quote func(string) string) []Shard {
	if !hasNullRows || len(shards) == 0 {
		return shards
	}
	if len(shards) == 1 && shards[0].Where == "" {
		return shards
	}
	return append(shards, Shard{Where: fmt.Sprintf("%s IS NULL", quote(pkColumn))})
}

// pkHasNullRows 探测主键列是否存在 NULL 行（只有此时才需要额外的 NULL 片）。
//
// 计数必须用 COUNT(*) 而不是 COUNT(pk)：COUNT(列) 只统计非 NULL 值，而结果集里的 pk 全为 NULL，
// `COUNT(pk) ... WHERE pk IS NULL` 恒等于 0（本轮实测踩到：探测永远不命中，NULL 片永远补不上）。
//
// 也不写成 `SELECT 1 ... LIMIT 1`：本包跨方言共用，Oracle/达梦/MSSQL 不识别裸 LIMIT（会被当作
// 标识符或语法错误）；COUNT 形态无需分页改写，且 IS NULL 在带索引的主键列上就是往索引最左端
// 走一次（无 NULL 行时成本接近于零）。
//
// 报错往上抛：探测不成就是「无法证明分片能覆盖全部行」，由调用方降级不分片，不能默认无 NULL。
func pkHasNullRows(ctx context.Context, conn *dbi.DbConn, tableName, pk string, quote func(string) string) (bool, error) {
	sql := fmt.Sprintf("SELECT COUNT(*) AS nulls FROM %s WHERE %s IS NULL", quote(tableName), quote(pk))
	_, rows, err := conn.QueryContext(ctx, sql)
	if err != nil {
		return false, fmt.Errorf("probe null pk rows of table [%s] failed: %s", tableName, err.Error())
	}
	if len(rows) == 0 {
		return false, nil
	}
	valueRaw, ok := rowValueCI(rows[0], "nulls")
	if !ok || valueRaw == nil {
		return false, fmt.Errorf("probe null pk rows of table [%s] returned unexpected column labels", tableName)
	}
	nulls, ok := value.ValToInt64(valueRaw)
	if !ok {
		return false, fmt.Errorf("null pk count of table [%s] is not integrable: %v", tableName, valueRaw)
	}
	return nulls > 0, nil
}

// pkDataTypeOf 取指定列的数据类型
func pkDataTypeOf(columns []dbi.Column, columnName string) string {
	for _, c := range columns {
		if c.ColumnName == columnName {
			return c.DataType
		}
	}
	return ""
}

// isIntegerDataType 判断数据类型是否属于整型族（跨方言常用整型类型的白名单）
func isIntegerDataType(dataType string) bool {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "tinyint", "smallint", "mediumint", "int", "integer", "bigint",
		"int2", "int4", "int8": // pg别名
		return true
	}
	return false
}

// Plan 按主键闭区间规划分片：目标每片约 targetRows 行，生成
// `pk >= a AND pk <= b` 形式的闭区间过滤条件，分片间无重叠、并集覆盖 [pkMin, pkMax]。
//
// 参数：
//   - pkColumn: 整型主键列名（""返回nil）
//   - pkMin/pkMax: 主键最小/最大值（来自源库MIN/MAX查询；pkMax < pkMin返回nil）
//   - totalRows: 表行数估计（元数据；<=0时用主键区间跨度估计）
//   - targetRows: 单片目标行数，<=0 取包默认 TargetRows
//   - quote: 源库方言标识符引用函数
//
// 返回nil表示不可分片（调用方应整表单任务搬运）；单分片场景Where为空（无需过滤）。
func Plan(pkColumn string, pkMin, pkMax int64, totalRows, targetRows int, quote func(string) string) []Shard {
	if pkColumn == "" || pkMin > pkMax {
		return nil
	}
	targetRows = NormalizeTargetRows(targetRows)

	// 行数估计：优先元数据行数（对稀疏主键更准确），缺失时用主键区间跨度兜底。
	// 跨度用uint64运算（MaxInt64-MinInt64+1=2^64会溢出uint64，需封顶）
	diff := uint64(pkMax) - uint64(pkMin)
	span := diff
	if diff < ^uint64(0) {
		span = diff + 1
	}
	rowsEstimate := uint64(totalRows)
	if totalRows < 0 {
		// 行数估计只能来自元数据，负值是无效数据（uint64(负数) 会变成巨大值并顶到 MaxCount）
		rowsEstimate = 0
	}
	if rowsEstimate == 0 {
		rowsEstimate = span
	}

	shardCount := rowsEstimate / uint64(targetRows)
	if rowsEstimate%uint64(targetRows) != 0 {
		shardCount++
	}
	if shardCount < 1 {
		shardCount = 1
	}
	// 封顶：分片数不超过主键区间跨度与最大分片数
	if shardCount > span {
		shardCount = span
	}
	if shardCount > MaxCount {
		shardCount = MaxCount
	}
	if shardCount == 1 {
		return []Shard{{Where: ""}}
	}

	// 每片主键区间步长（向上取整，保证覆盖整个区间）
	step := span / shardCount
	if span%shardCount != 0 {
		step++
	}

	shards := make([]Shard, 0, shardCount)
	start := pkMin
	for i := uint64(0); i < shardCount; i++ {
		// end安全计算：步长能被start到pkMax的剩余跨度容纳时才前推，避免int64溢出
		end := pkMax
		if remain := uint64(pkMax) - uint64(start) + 1; remain > step {
			end = start + int64(step) - 1
		}
		isLast := end >= pkMax
		shards = append(shards, Shard{Where: rangePredicate(pkColumn, start, end, i == 0, isLast, quote)})
		if isLast {
			break
		}
		start = end + 1
	}
	return shards
}

// rangePredicate 组装一片的区间谓词，并把**首片去掉下界、末片去掉上界**。
//
// 为什么要去掉：MIN/MAX 只在规划那一刻有效，落在区间外的新行不属于任何一片。对非单调主键
// （业务号、哈希分桶 id，Doris/StarRocks 常见）与迁移中的活表，这意味着「本轮期间落入 [MIN,MAX]
// 之外」的新行被**永久漏掉**——单次大查询没有区间谓词，不存在这一失败类别。去掉两端后各片并集
// 从 [MIN,MAX] 变成整个键空间：末片跑在最后，自增/雪花 id 的新行（高位那一侧）能被它带走；
// 首片先跑，只能带走它之前已落入的低位新行——两者都是严格优于封住边界的布局。
//
// 代价几乎为零：区间端点本来就是真实的 MIN/MAX，去掉一个边界条件不改变索引扫描范围。
// NULL 主键行仍会被任一比较谓词排除（比较运算遇 NULL 得 UNKNOWN），那一部分由
// PlanTable 探测后额外补上的 `pk IS NULL` 片带走（见 withNullShard），不靠列可空位拒分片。
func rangePredicate(pkColumn string, start, end int64, isFirst, isLast bool, quote func(string) string) string {
	col := quote(pkColumn)
	switch {
	case isFirst && isLast:
		// 一片就覆盖全表：不加任何谓词（与 shardCount==1 的早期返回同义，这里带走步长恰好覆盖全区间的情形）
		return ""
	case isFirst:
		return fmt.Sprintf("%s <= %d", col, end)
	case isLast:
		return fmt.Sprintf("%s >= %d", col, start)
	default:
		return fmt.Sprintf("%s >= %d AND %s <= %d", col, start, col, end)
	}
}

// PlanTable 在给定连接上规划单表的主键范围分片。
//
// 与 Plan 的分工：Plan 是纯区间数学，本函数负责取「只有连上库才拿得到」的事实——
// 主键列的类型判定（元数据）与主键取值区间（MIN/MAX）。
//
// 返回 (nil, nil) 表示该表不可分片（无单列整型主键、空表、主键区间过小），
// 调用方应按整表单任务搬运；返回 error 表示探测本身失败（元数据不可达、MIN/MAX 查询报错
// 或返回形态未知），由调用方决定是降级还是中止——两者语义不同，不可合并成一个 nil：
// 把「探测没成功」当成「一片就装得下」，调用方会把一个自己并未验证过的结论写给用户。
//
// 参数 estimatedRows 为元数据给出的行数估计（<=0 时按主键区间跨度估算），
// targetRows 为单片目标行数（<=0 取包默认）。
//
// 探测成本：一次元数据读（多数方言有缓存）+ 一条 MIN/MAX 聚合，相对整表扫描可忽略；
// MIN/MAX 只在**规划时刻**有效，但首片无下界、末片无上界（见 rangePredicate）使并集覆盖整个
// 键空间，规划后才落入 [MIN,MAX] 之外的新行仍会被两端某一片带走，只剩「扫描尚未走到那一端
// 时已落入中间」这一很窄的窗口可能本轮漏掉，下一轮（或下一次同步）补回。
func PlanTable(ctx context.Context, conn *dbi.DbConn, tableName string, estimatedRows, targetRows int) ([]Shard, error) {
	columns, err := conn.Metadata().GetColumns(tableName)
	if err != nil {
		return nil, fmt.Errorf("get columns of table [%s] failed: %s", tableName, err.Error())
	}
	// 迁移侧的口径保持不变：只按**主键**分片（搬运的冲突键/回填都建立在主键上）
	return PlanTableByShardKey(ctx, conn, tableName, DetectIntPrimaryKey(columns), columns, estimatedRows, targetRows)
}

// PlanTableByShardKey 用调用方选定的分片键规划区间。
//
// 与 PlanTable 的区别只在「键怎么选」：同步侧用 DetectIntShardKey（主键或单列整型唯一索引），
// 迁移侧仍只认主键；一旦选好，MIN/MAX 探测、区间数学、NULL 片、日志都是同一份实现。
// columns 由调用方传入：选型时已经读过列元数据，这里再读一次既多一次往返，也会出现
// 「选型看到的列」与「规划看到的列」不同源。
//
// 返回语义与 PlanTable 完全一致（(nil, nil) = 该键不可分片）。
func PlanTableByShardKey(ctx context.Context, conn *dbi.DbConn, tableName, shardKey string, columns []dbi.Column, estimatedRows, targetRows int) ([]Shard, error) {
	if shardKey == "" {
		return nil, nil
	}

	quote := conn.GetDialect().Quoter().QuoteIdent
	minMaxSQL := fmt.Sprintf("SELECT MIN(%s) AS mn, MAX(%s) AS mx FROM %s", quote(shardKey), quote(shardKey), quote(tableName))
	_, rows, err := conn.QueryContext(ctx, minMaxSQL)
	if err != nil {
		return nil, fmt.Errorf("query shard key range of table [%s] failed: %s", tableName, err.Error())
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("query shard key range of table [%s] returned no row", tableName)
	}
	// 聚合查询必然返回一行：取不到键即返回形态与预期不符（列标签不是 mn/mx），
	// 静默退回整表单任务会让分片默默失效且无人知晓，明确报错交给调用方降级并留痕
	pkMinRaw, hasMin := rowValueCI(rows[0], "mn")
	pkMaxRaw, hasMax := rowValueCI(rows[0], "mx")
	if !hasMin || !hasMax {
		return nil, fmt.Errorf("query shard key range of table [%s] returned unexpected column labels, want mn/mx", tableName)
	}
	// MIN/MAX 为 NULL 即空表：不可分片（可能 0 行，搬运侧自然无数据），不是探测失败
	if pkMinRaw == nil || pkMaxRaw == nil {
		return nil, nil
	}
	pkMin, ok1 := value.ValToInt64(pkMinRaw)
	pkMax, ok2 := value.ValToInt64(pkMaxRaw)
	if !ok1 || !ok2 {
		// 分片键列取到值但不可归一为整数（返回形态未知）：不能当成「区间足够小」
		return nil, fmt.Errorf("shard key range of table [%s] is not integrable: min=%v max=%v", tableName, pkMinRaw, pkMaxRaw)
	}
	shards := Plan(shardKey, pkMin, pkMax, estimatedRows, targetRows, quote)
	// 可空分片键列先做廉价预筛，再证实真的存在 NULL 行才补一片 `<key> IS NULL`：区间谓词永远不匹配
	// NULL，不补就会静默漏行；无 NULL 时不补，避免每次分片白产一个空片段（见 withNullShard）
	if pkNullable(columns, shardKey) {
		hasNulls, err := pkHasNullRows(ctx, conn, tableName, shardKey, quote)
		if err != nil {
			return nil, err
		}
		shards = withNullShard(shardKey, shards, hasNulls, quote)
	}
	// 规划结果留一条可检索的痕：区间数学没有日志时，调用方各自决定怎么说，同一条规划会有两种措辞
	// （旧 transfer 实现有这条日志，下沉时随文件一并丢过）
	logx.Debugf("db shard plan: table=%s key=%s min=%d max=%d estimatedRows=%d targetRows=%d shards=%d", tableName, shardKey, pkMin, pkMax, estimatedRows, NormalizeTargetRows(targetRows), len(shards))
	return shards, nil
}

// rowValueCI 大小写不敏感取结果行的列值。
//
// `AS mn` 是未加引号的标识符，Oracle/达梦会按自动折叠为大写列标签（MN/MX），而结果 map
// 的键就是驱动返回的列标签原样、不做归一；直接 `rows[0]["mn"]` 在这些方言上恒 miss，
// 会把「本可分片的大表」默默变成整表单任务（恰是本机制要治的超时场景）。
func rowValueCI(row map[string]any, column string) (any, bool) {
	for k, v := range row {
		if strings.EqualFold(k, column) {
			return v, true
		}
	}
	return nil, false
}
