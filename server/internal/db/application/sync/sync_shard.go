package sync

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"mayfly-go/internal/db/application/shard"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/logx"
)

// planSourceScan 决定本轮源读取是「一条流式大查询」还是「按主键区间分片的 N 条短查询」。
//
// 为什么必须有分片：doDataSync 把整轮数据装在同一条 SELECT 里流式消费，每批都要等目标库写完才继续
// 取行，于是这条 SELECT 在数据库服务端存活整轮时长。带查询级超时的引擎（StarRocks/Doris 的
// query_timeout、MySQL 8 的 max_execution_time、PG 的 statement_timeout）会在超时点直接 kill 掉它，
// 表现为 `Error 1105 ... query is timeout, killed by timeout checker`；而水位只在整轮成功后推进一次，
// 下一轮又从同样位置重放同样数据量 → 大表同步永远跑不完。拆成 N 条按主键闭区间限定的短查询后，
// 单条查询时长与整轮时长解耦，一条 SELECT 也不再要求服务端持续投递整表结果。
//
// 只在「结果行与源表行一一对应、且区间谓词能落进 WHERE 链」时启用，不满足一律保守退回历史行为：
//   - 顶层单表（无 JOIN/UNION/子查询入口）：区间条件必须能定位到某张物理表的主键列；
//   - 无 DISTINCT、无聚合/分组、无顶层 ORDER BY/LIMIT：前者把「整体去重」拆成「每片各自去重」
//     会静默变形结果集（`SELECT count(*)` 从 1 行变 N 行）；顶层子句则会让追加的区间条件落在
//     它们之后变成非法 SQL（与 validateDataSyncAppendable 同一约束，不能只在那条校验里守）；
//   - 顶层 WHERE 不以 OR/XOR（含 mysql 的 `||`）为根：区间以 `and (...)` 追加，AND 结合更紧时
//     只约束其中一支（形态一律按原文词法扫，见 scanSQLShape）；
//   - 投影不含窗口函数：OVER 的取值由本次查询可见行集决定，分片会静默改变它；
//   - 结果集原样带出主键列（见 projectsShardKey）：这是「分片后结果不变」可证的最小条件；
//   - 表存在单列整型分片键（shard.DetectIntShardKey：主键，或单列整型唯一索引）：非整型键无法做数值区间切分；
//     只认「数据库能证明唯一」的列，因为片内续扫靠 `key > 游标` 推进，可重复键会静默漏行；
//   - 分片键区间规划后确实 >1 片：小表单片本就是一条查询，不必引入额外探测。
//
// sourceScanPlan 本轮源读取计划。
//
// shards 恰好一个空串 = 整表单次查询（历史行为）；多个元素 = 按区间分片。
// shardKey/keyLabel 只在可分片时给值：片内续扫要用两者——键列拼谓词，标签从结果行里取游标值。
type sourceScanPlan struct {
	shards   []string
	shardKey string
	keyLabel string
	note     string
}

// singleScan 构造「整表单次查询」的扫描计划，reason 会写进运行日志（为什么没分片必须可见）。
func singleScan(reason string) sourceScanPlan {
	return sourceScanPlan{shards: []string{""}, note: "整表单次查询（" + reason + "）"}
}

func planSourceScan(ctx context.Context, srcConn *dbi.DbConn, task *entity.DataSyncTask) sourceScanPlan {
	stmt, err := srcConn.GetDialect().GetSQLParser().Parse(task.DataSQL)
	if err != nil {
		return singleScan("源查询无法解析")
	}
	sel, ok := stmt.(*sqlstmt.SelectStmt)
	if !ok {
		return singleScan("源查询不是单条 SELECT")
	}
	if why, refuse := refuseShardScan(sel, task.DataSQL, srcConn.GetDialect().GetSQLSplitter()); refuse {
		return singleScan(why)
	}
	schema, table, ok := singleTableForIndex(sel)
	if !ok {
		return singleScan("源查询非单表或表名不是可直接定位的物理表名")
	}
	// 带库/架构限定时必须对上轴：元数据（主键、行数）与 MIN/MAX 都拿**不带限定**的表名按会话当前
	// 位置解析（pg meta.sql 用 current_schema()，mysql information_schema 用 database()），而
	// DataSQL 读的是限定名指向的表。轴弄错会出现两种后果：比对永不相等 → 静默不分片（恰是要治的
	// 超时场景）；更危险的是限定名不等于当前 schema 却碰巧等于库名（pg 允许库与架构同名）→ 拿 A 表
	// 的区间去扫 B 表，区间与真实主键分布错位、尾部区间外的行永久漏掉。故库名与 schema 两个定位维度
	// 都比对（无独立 schema 层的方言其限定段就是库名，Info.Schema 恒空），对不上宁可不分片。
	if schema != "" && !strings.EqualFold(schema, srcConn.Info.Database) && !strings.EqualFold(schema, srcConn.Info.Schema) {
		return singleScan(fmt.Sprintf("源查询限定在库/架构 [%s]，与连接所在库/schema 不一致，不能拿另一张表的区间来切分", schema))
	}

	columns, err := srcConn.Metadata().GetColumns(table)
	if err != nil {
		return singleScan(fmt.Sprintf("读取源表 [%s] 列元数据失败：%s", table, err.Error()))
	}
	// 索引元数据只为「无主键但存在单列整型唯一索引」的表准备（数仓 Unique Key 表、只建 unique 的 id 表）。
	// 取不到不影响主键路径：降级成「只用主键」，并在日志里留痕。
	indexes, idxErr := srcConn.Metadata().GetTableIndex(table)
	if idxErr != nil {
		logx.WarnfContext(ctx, "data sync: get index metadata of source table [%s] failed, shard by primary key only: %s", table, idxErr.Error())
		indexes = nil
	}
	shardKey := shard.DetectIntShardKey(columns, indexes)
	if shardKey == "" {
		return singleScan("源表既无单列整型主键、也无单列整型唯一索引，无法按区间切分")
	}
	if !projectsShardKey(sel, shardKey) {
		return singleScan(fmt.Sprintf("查询结果未原样带出分片键列 [%s]，分片后无法证明结果集与整表一致", shardKey))
	}
	// 结果集里分片键的标签（有别名时是别名）：片内续扫靠它从行里取游标值。
	// 取不到只是不能续扫，仍可正常分片。
	keyLabel := shardKeyLabel(sel, shardKey)

	// 行数估计只为决定「切几片」，取不到不影响正确性（shard.Plan 会退回按分片键区间跨度估算）
	estimatedRows := estimateSourceRows(srcConn, table)

	// 选型已读过列元数据，这里直接把 columns 传进去：「分片键选型 + MIN/MAX + 区间切分」
	// 因此只有一份输入，不会出现「选型看到的列」与「规划看到的列」不同源
	shards, err := shard.PlanTableByShardKey(ctx, srcConn, table, shardKey, columns, estimatedRows, task.GetShardTargetRows())
	if err != nil {
		logx.WarnfContext(ctx, "data sync: plan source shard scan for task [%s] failed: %s", task.TaskName, err.Error())
		return singleScan("分片键区间探测失败：" + err.Error())
	}
	if len(shards) < 2 {
		// 分片键已确认可分片，这里只可能是「区间足够小、一片就装下」。
		// 不拿 estimatedRows 开口：元数据取不到时它是 0，写「源表约 0 行」会把用户往「没数据」误引
		return singleScan("分片键区间一片即可覆盖，无需拆语句")
	}

	conditions := make([]string, 0, len(shards))
	for _, s := range shards {
		conditions = append(conditions, s.Where)
	}
	// 日志里报**生效值**而不是配置原值：未配置（0）时用的是包默认，直接写 0 会让用户以为
	// 「单片 0 行」这个自己没设过的数字。
	basis := fmt.Sprintf("单片目标 %d 行", shard.NormalizeTargetRows(task.GetShardTargetRows()))
	if estimatedRows <= 0 {
		// 元数据给不出行数时，片数是按分片键区间端点均分的，「单片目标行数」并不代表实际行数：
		// 稀疏/大量主键下每片可能只几十行，也可能几百万行。说清依据，用户才知道该不该调 shardTargetRows
		basis = "行数未知，按分片键区间均分"
	}
	resume := "，片内超预算时按该键续扫"
	if keyLabel == "" {
		resume = "，查询结果未带出分片键标签，片内不续扫"
	}
	return sourceScanPlan{
		shards:   conditions,
		shardKey: shardKey,
		keyLabel: keyLabel,
		note:     fmt.Sprintf("按分片键 [%s] 分 %d 片扫描（%s）%s，单条源查询不再受整轮时长拖累", shardKey, len(conditions), basis, resume),
	}
}

// shardKeyLabel 返回分片键在结果集里的列标签：有别名时是别名，否则是列名本身（SELECT * 同理）。
// 空串表示「按结果行定位不到该列」：仍可分片，只是不能片内续扫。
func shardKeyLabel(sel *sqlstmt.SelectStmt, shardKey string) string {
	for _, item := range sel.Items {
		if item.IsStar() {
			return shardKey
		}
		if item.Kind == sqlstmt.SelectItemColumn && strings.EqualFold(item.ColumnName, shardKey) {
			if item.Alias != "" {
				return item.Alias
			}
			return item.ColumnName
		}
	}
	return ""
}

// refuseShardScan 判定顶层 SELECT 的子句形态是否会让「按主键区间拆分读取」改变结果集或产生非法 SQL。
// dataSQL 为源查询原文、splitter 提供方言词法能力表：拼接落在原文尾部，而 AST 在不认识的
// 运算符处会静默截断（见 scanSQLShape），所以子句形态以词法扫描为准；只有解析器在子句
// 关键字**之前**就已置好的标志（DISTINCT、UNION）才继续用 AST，它们不会被表达式截断波及。
// 返回 (原因, 是否拒绝)。
func refuseShardScan(sel *sqlstmt.SelectStmt, dataSQL string, splitter sqlparser.SQLSplitter) (string, bool) {
	if sel.Distinct {
		return "SELECT DISTINCT 去重范围会由整表缩为每片", true
	}
	if len(sel.Unions) > 0 {
		return "含 UNION 集合运算", true
	}
	shape := scanSQLShape(dataSQL, splitter)
	// 区间谓词以 `and (...)` 追加进已有 WHERE 链：顶层以 OR/XOR 为根时 AND 结合更紧，区间只约束
	// 其中一支，另一支的每一行会在**每一片**重复返回（N 片 = N 倍数据）。不能只靠子句存在性校验：
	// 全量刷新/对账模式不拼任何增量谓词，历史行为下这类 SQL 一个字符都不追加，因而完全安全。
	if shape.topLevelOr {
		return "顶层 WHERE 以 OR/XOR 为根，追加的区间条件会被 AND 优先级越过、只约束其中一支", true
	}
	// `||` 在 mysql/Doris/StarRocks 默认 sql_mode 下就是 OR；pg/oracle 下它是字符串拼接，这里
	// 多拒一次只损失分片（仍退回历史行为），换来的是不对 N 倍重复放行。保存校验不用这一位，
	// 因为那边误拒会直接让存量合法任务跑不动（见 sqlShape.topLevelPipes）。
	if shape.topLevelPipes {
		return "顶层 WHERE 含 ||（mysql/Doris/StarRocks 默认语义即 OR），追加的区间条件可能只约束其中一支", true
	}
	// 尾部子句靠词法而不是 AST：解析器在不认识的运算符（% & ? # 等）处截断后，其后的
	// ORDER BY/LIMIT 根本不进 AST，拿 AST 拦会误放行，区间拼到 LIMIT 之后直接语法错误
	if shape.topLevelTail {
		return "含顶层 GROUP BY/HAVING/ORDER BY/LIMIT 等尾部子句，区间条件只能追加在语句尾部，会落在这些子句之后", true
	}
	if hasWindowFunction(sel) {
		return "投影含窗口函数（OVER），分片会按主键区间重算分区内取值", true
	}
	return "", false
}

// windowOverReg 匹配投影原文里的 OVER 关键字（词边界，大小写不敏感）。
//
// 走文本而非 AST：解析器把带 OVER 的投影归入 SelectItemExpr 且不建模窗口子句（实测
// `ROW_NUMBER() OVER (...)` 只留下原文），结构化判据无从下手。列名恰好叫 over 会误命中，
// 代价只是退回不分片，不改变数据。
var windowOverReg = regexp.MustCompile(`(?i)\bover\b`)

// hasWindowFunction 判断投影里是否出现窗口函数。窗口列的取值由「本次查询可见的行集」决定，
// 分片把该行集从整表缩到一个主键区间会**静默改变**结果（不报错、不重复、不丢失），
// 是分片语义下最难发现的一类失真。
func hasWindowFunction(sel *sqlstmt.SelectStmt) bool {
	for _, item := range sel.Items {
		if windowOverReg.MatchString(item.Text) {
			return true
		}
	}
	return false
}

// projectsShardKey 判断查询结果是否原样带出了分片主键列。
//
// 为什么需要这一步：分片正确性的前提是「每条源行恰好落进一个区间，且该行的结果行也跟着落在同一片」。
// 投影里主键以普通列引用出现（id / t.id / id AS pk_id）或 `SELECT *` 时，这一前提成立；
// 若主键被函数/表达式/聚合包住（MAX(id)、CONCAT(id, '-')），行数与行的归属都不再与整表等价，
// 退回单次流式更安全。比「按函数名单拒绝聚合」可靠：解析器只给表达式原文，名单会漏方言自带聚合。
func projectsShardKey(sel *sqlstmt.SelectStmt, pk string) bool {
	for _, item := range sel.Items {
		if item.IsStar() {
			return true
		}
		if item.Kind == sqlstmt.SelectItemColumn && strings.EqualFold(item.ColumnName, pk) {
			return true
		}
	}
	return false
}

// estimateSourceRows 取元数据里的行数估计（用于决定分片数，不参与正确性判定）。
// 取不到返回 0，由 shard.Plan 按主键区间跨度估算。
func estimateSourceRows(srcConn *dbi.DbConn, table string) int {
	tables, err := srcConn.Metadata().GetTables(table)
	if err != nil {
		logx.Warnf("data sync: get row estimate of source table [%s] failed: %s", table, err.Error())
		return 0
	}
	for _, tb := range tables {
		if strings.EqualFold(tb.TableName, table) {
			return tb.TableRows
		}
	}
	return 0
}
