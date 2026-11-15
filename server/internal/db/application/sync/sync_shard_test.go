package sync

import (
	"strings"
	"testing"
	"time"

	"mayfly-go/internal/db/application/shard"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/domain/entity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------- refuseShardScan ----------------

func TestRefuseShardScan(t *testing.T) {
	cases := []struct {
		name    string
		sql     string
		wantRef bool
	}{
		{"plain star", "SELECT * FROM t", false},
		{"columns with where", "SELECT id, name FROM t WHERE status = 1", false},
		{"distinct", "SELECT DISTINCT name FROM t", true},
		{"group by", "SELECT status, COUNT(*) FROM t GROUP BY status", true},
		{"having", "SELECT status FROM t GROUP BY status HAVING COUNT(*) > 1", true},
		{"top level order by", "SELECT * FROM t ORDER BY name", true},
		{"top level limit", "SELECT * FROM t LIMIT 10", true},
		{"union", "SELECT * FROM t UNION SELECT * FROM t2", true},
		// 聚合/标量函数不在此处判（解析器只留表达式原文，按名单拒绝会漏方言聚合），
		// 由 projectsShardKey 用「结果是否原样带出主键」这一可证条件兜住
		{"scalar function left to key check", "SELECT UPPER(name) FROM t", false},
		// 区间谓词以 `and (...)` 追加：顶层 OR/XOR 根下 AND 结合更紧，只约束其中一支，
		// 另一支的每一行会在每一片重复返回（N 片 = N 倍数据）
		{"top level or root", "SELECT id, name FROM t WHERE a = 1 OR b = 2", true},
		{"top level xor root", "SELECT id, name FROM t WHERE a = 1 XOR b = 2", true},
		{"or with in-subquery branch", "SELECT id FROM t WHERE a IN (SELECT id FROM t2) OR b = 2", true},
		// 解析器在不认识的运算符（% MOD XOR）处会静默丢弃后的表达式，AST 判形看不到那个 OR：
		// 这一例就是本轮审查发现的误放行原形（真实执行时该语句会产出 N 倍重复行）
		{"or after unmodelled operator", "SELECT id, name FROM t WHERE id % 2 = 0 OR id = 1", true},
		// 括号已经把整个条件包起来：追加的 AND 作用于括号结果，语义仍正确，剥掉括号会误拒
		{"parenthesised or root stays shardable", "SELECT id, name FROM t WHERE (a = 1 OR b = 2)", false},
		{"and root with nested or", "SELECT id, name FROM t WHERE a = 1 AND (b = 2 OR c = 3)", false},
		{"not root", "SELECT id, name FROM t WHERE NOT (a = 1 OR b = 2)", false},
		// 窗口列取值由本次查询可见行集决定，分片会静默改变它
		{"window function", "SELECT id, ROW_NUMBER() OVER (ORDER BY amount DESC) rn FROM t", true},
		{"window function lowercase", "SELECT id, SUM(x) OVER () s FROM t", true},
		// 取模本身不是风险：它只让 AST 截断，没有顶层 OR 时仍应可分片
		{"unmodelled operator without or", "SELECT id, name FROM t WHERE id % 2 = 0", false},
		// 本轮发现的另一个误放行原形：解析器在 % 处截断后，其后的 ORDER BY/LIMIT 根本不进 AST，
		// 拿 AST 拦不到 → 区间拼到 LIMIT 之后直接语法错误（模式 3 下发生在 TRUNCATE 之后）
		{"order by after unmodelled operator", "SELECT id, name FROM t WHERE id % 2 = 0 ORDER BY name", true},
		{"limit after unmodelled operator", "SELECT id, name FROM t WHERE id % 2 = 0 LIMIT 10", true},
		// mysql/Doris/StarRocks 默认 sql_mode 下 `||` 就是 OR，同样会被追加的 AND 越过；
		// 分片侧多拒一次只是退回单次流式，不值得为它拿 N 倍重复冒险
		{"pipes as or", "SELECT id, name FROM t WHERE a = 1 || b = 2", true},
		// 无括号 CASE 表达式内的 OR 不构成风险（CASE 已在比较运算符左侧收口），误拒就是本末倒置
		{"or inside case expression", "SELECT id, name FROM t WHERE CASE WHEN a = 1 OR b = 2 THEN 1 ELSE 0 END = 1", false},
		// 子查询内部的尾部子句处于括号里，不影响外层拼接位置
		{"limit inside subquery", "SELECT id FROM (SELECT id FROM t LIMIT 10) x", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// dataSQL 取同一原文：部分运算符在 AST 里不可见，只有原文能兼到
			why, ref := refuseShardScan(parseSelect(t, c.sql), c.sql, mysqlSplitter())
			assert.Equal(t, c.wantRef, ref, "拒绝原因=%q", why)
			if ref {
				assert.NotEmpty(t, why, "拒绝必须给出可写进运行日志的原因")
			}
		})
	}
}

// mysqlSplitter 取一份真实方言的切割器：词法扫描必须跑在真能力表上，
// 零值 DialectConfig 会让引用/注释语义与生产不一致，测过了也不算数。
func mysqlSplitter() sqlparser.SQLSplitter {
	return dbi.GetDialect("mysql").GetSQLSplitter()
}

// ---------------- scanSQLShape ----------------

// 分片区间与增量/软删谓词同一个拼接面，形态事实只扫一次：这里钉住「什么算危险」的边界，
// 避免为了安全把所有带 OR/子句的查询一刀切拒掉（那就本末倒置了）。
func TestScanSQLShape(t *testing.T) {
	cases := []struct {
		sql       string
		wantOr    bool
		wantPipes bool
		wantTail  bool
	}{
		{"SELECT id FROM t", false, false, false},
		{"SELECT id FROM t WHERE a = 1", false, false, false},
		{"SELECT id FROM t WHERE a = 1 AND b = 2", false, false, false},
		{"SELECT id FROM t WHERE (a = 1 OR b = 2)", false, false, false},
		{"SELECT id FROM t WHERE a = 1 AND (b = 2 OR c = 3)", false, false, false},
		{"SELECT id FROM t WHERE NOT (a = 1 OR b = 2)", false, false, false},
		{"SELECT id FROM t WHERE a = 1 OR b = 2", true, false, false},
		{"SELECT id FROM t WHERE a = 1 OR b = 2 OR c = 3", true, false, false},
		{"select id from t where a = 1 or b = 2", true, false, false},
		{"SELECT id FROM t WHERE a = 1 XOR b = 2", true, false, false},
		// 解析器不认识的运算符后面的 OR：AST 在此截断，词法扫描仍能看到（本轮回归的原形）
		{"SELECT id FROM t WHERE id % 2 = 0 OR id = 1", true, false, false},
		{"SELECT id FROM t WHERE id MOD 2 = 0 OR id = 1", true, false, false},
		// 子查询里的 OR 处于括号内，不影响外层 WHERE 的整体形状
		{"SELECT id FROM t WHERE a IN (SELECT x FROM t2 WHERE y = 1 OR z = 2)", false, false, false},
		// 字符串字面量与注释里的 or 不是运算符
		{"SELECT id FROM t WHERE note = 'a OR b'", false, false, false},
		{"SELECT id FROM t WHERE a = 1 -- cond OR drop\n", false, false, false},
		// 列名含 or/xor 子串不误命中
		{"SELECT id FROM t WHERE color = 1 AND xor_flag = 2", false, false, false},
		// 无括号 CASE 内部的 OR 已被 CASE 收口，追加 AND 仍作用于整个 WHERE
		{"SELECT id FROM t WHERE CASE WHEN a = 1 OR b = 2 THEN 1 ELSE 0 END = 1", false, false, false},
		// `||`：mysql 系默认语义即 OR（只在分片门禁保守），pg/oracle 下它是拼接
		{"SELECT id FROM t WHERE a = 1 || b = 2", false, true, false},
		// 尾部子句：AST 能看到的与因截断看不到的都要记上
		{"SELECT id FROM t WHERE a = 1 ORDER BY id", false, false, true},
		{"SELECT id FROM t WHERE id % 2 = 0 ORDER BY id LIMIT 10", false, false, true},
		{"SELECT id FROM t WHERE a = 1 GROUP BY id", false, false, true},
		// WHERE 子句结束后（尾部子句之后）的 OR 不属于它，但尾部子句本身已命中
		{"SELECT id FROM t WHERE a = 1 ORDER BY (CASE WHEN x OR y THEN 1 ELSE 2 END)", false, false, true},
		// 子查询内部的尾部子句处于括号里，不算外层形态
		{"SELECT id FROM (SELECT id FROM t ORDER BY x LIMIT 5) s WHERE id = 1", false, false, false},
	}
	for _, c := range cases {
		shape := scanSQLShape(c.sql, mysqlSplitter())
		assert.Equal(t, c.wantOr, shape.topLevelOr, "topLevelOr sql=%s", c.sql)
		assert.Equal(t, c.wantPipes, shape.topLevelPipes, "topLevelPipes sql=%s", c.sql)
		assert.Equal(t, c.wantTail, shape.topLevelTail, "topLevelTail sql=%s", c.sql)
	}

	// 没有词法能力表时无法区分字面量与运算符，三项均按最危处理（不能默认放行）
	assert.Equal(t, worstCaseShape, scanSQLShape("SELECT id FROM t WHERE a = 1", nil))
	// 括号不配对同理：子句边界无从推断
	assert.Equal(t, worstCaseShape, scanSQLShape("SELECT id FROM t WHERE (a = 1 OR b = 2", mysqlSplitter()))
}

// 解析器在不认识的运算符处静默截断——钉住这个行为本身，否则后来人仍会以为 AST 判形可用。
// 重点不只是 Where.Text 变短，而是其后的顶层子句也从 AST 里彻底消失（那才是分片误放行的根源）。
func TestParserTruncatesUnmodelledOperators(t *testing.T) {
	sel := parseSelect(t, "SELECT id FROM t WHERE a = 1 XOR b = 2")
	require.NotNil(t, sel.Where, "解析器必须仍给出 Where 节点")
	assert.NotContains(t, sel.Where.Text, "XOR", "实测：解析器在 XOR 处丢弃了后半段表达式: %s", sel.Where.Text)

	truncated := parseSelect(t, "SELECT id FROM t WHERE id % 2 = 0 ORDER BY id LIMIT 10")
	require.NotNil(t, truncated.Where)
	assert.Equal(t, "id", truncated.Where.Text, "实测：取模之后的表达式被整段丢弃")
	assert.Empty(t, truncated.OrderBy, "实测：截断后的顶层 ORDER BY 不进 AST，拿 AST 拦不到")
	assert.Nil(t, truncated.Limit, "实测：截断后的顶层 LIMIT 不进 AST，拿 AST 拦不到")

	// 同一条语句去掉 % 后子句完整解析，证明上面的空值来自截断而非本方言不支持这些子句
	intact := parseSelect(t, "SELECT id FROM t WHERE status = 1 ORDER BY id LIMIT 10")
	require.Len(t, intact.OrderBy, 1)
	require.NotNil(t, intact.Limit)
}

// 增量/软删谓词同样以 `and (...)` 追加，顶层 OR 根时保存必须拦住：拼接后只约束其中一支，
// 属于存量任务里已经在静默多读数据的形态，给出可执行的改法而不是默默少同步。
func TestValidateAppendableRejectsOrRoot(t *testing.T) {
	newTask := func(mutate func(*entity.DataSyncTask)) *entity.DataSyncTask {
		task := &entity.DataSyncTask{SyncMode: entity.DataSyncModeIncrementalMerge, DataSQL: "SELECT id FROM t WHERE a = 1 OR b = 2"}
		mutate(task)
		return task
	}
	orSQLs := []string{
		"SELECT id FROM t WHERE a = 1 OR b = 2",
		"SELECT id FROM t WHERE a = 1 XOR b = 2",
		"SELECT id FROM t WHERE id % 2 = 0 OR id = 1",
	}
	for _, sql := range orSQLs {
		task := newTask(func(t *entity.DataSyncTask) { t.UpdField = "upd"; t.DataSQL = sql })
		err := validateDataSyncAppendable(mysqlSplitter(), parseSelect(t, sql), task)
		require.Error(t, err, "sql=%s", sql)
		assert.Contains(t, err.Error(), "OR/XOR", "报错必须说清原因并给出括号改法")
	}

	// 括号包裹同样追加谓词，但语义正确，不能误拒已有任务
	parenSQL := "SELECT id FROM t WHERE (a = 1 OR b = 2)"
	require.NoError(t, validateDataSyncAppendable(mysqlSplitter(), parseSelect(t, parenSQL),
		newTask(func(t *entity.DataSyncTask) { t.UpdField = "upd"; t.DataSQL = parenSQL })))

	// 没配增量字段且已有 WHERE 时不追加任何谓词，OR 根也无从伤害
	require.NoError(t, validateDataSyncAppendable(mysqlSplitter(), parseSelect(t, orSQLs[0]), newTask(func(*entity.DataSyncTask) {})))

	// 模式 3（全量刷新）运行期不拼增量谓词，只拼 order by：顶层 OR 根对它是安全的。
	// 拿旧的宽判据（UpdField != "" 就拦）会误拒这类存量任务，且 runSync 每轮都拦 = 从此跑不动
	fullRefresh := newTask(func(t *entity.DataSyncTask) {
		t.SyncMode = entity.DataSyncModeFullRefresh
		t.UpdField = "upd"
	})
	require.NoError(t, validateDataSyncAppendable(mysqlSplitter(), parseSelect(t, fullRefresh.DataSQL), fullRefresh))

	// 模式 4 只配软删字段（无增量字段）：运行期真的会注入 `and (flag <> v OR flag IS NULL)`，
	// 顶层 OR 必须拦住——这是静默错数据，不能因为「没配增量字段」就放行
	softDel := newTask(func(t *entity.DataSyncTask) {
		t.SyncMode = entity.DataSyncModeIncrementalSoftDel
		t.SoftDeleteField = "del_flag"
		t.SoftDeleteValue = "Y"
	})
	err := validateDataSyncAppendable(mysqlSplitter(), parseSelect(t, softDel.DataSQL), softDel)
	require.Error(t, err, "软删过滤注入同样走 WHERE 链，顶层 OR 必须拦")
	assert.Contains(t, err.Error(), "OR/XOR")

	// 注入判定本身也要钉住（它与 planSyncQuery 共用同一口径）
	assert.False(t, appendsWherePredicate(&entity.DataSyncTask{SyncMode: entity.DataSyncModeFullRefresh, UpdField: "upd"}))
	assert.True(t, appendsWherePredicate(&entity.DataSyncTask{SyncMode: entity.DataSyncModeIncrementalMerge, UpdField: "upd"}))
	assert.True(t, appendsWherePredicate(softDel))
	assert.False(t, appendsWherePredicate(&entity.DataSyncTask{SyncMode: entity.DataSyncModeIncrementalAppend}))
}

// ---------------- projectsShardKey ----------------

func TestProjectsShardKey(t *testing.T) {
	cases := []struct {
		sql      string
		pk       string
		wantProj bool
	}{
		{"SELECT * FROM t", "id", true},
		{"SELECT t.* FROM t", "id", true},
		{"SELECT id, name FROM t", "id", true},
		{"SELECT name, id FROM t", "id", true},
		{"SELECT id AS pk_id, name FROM t", "id", true},
		{"SELECT t.id, name FROM t", "id", true},
		// 主键被表达式/聚合包住：行数与行归属都不再与整表等价
		{"SELECT MAX(id) FROM t", "id", false},
		{"SELECT CONCAT(id, '-') FROM t", "id", false},
		{"SELECT name FROM t", "id", false},
		// 主键列名大小写与书写差异不应影响判定
		{"SELECT ID FROM t", "id", true},
	}
	for _, c := range cases {
		t.Run(c.sql, func(t *testing.T) {
			assert.Equal(t, c.wantProj, projectsShardKey(parseSelect(t, c.sql), c.pk))
		})
	}
}

// ---------------- syncQueryPlan.sqlFor ----------------

// TestSyncQueryPlanSqlFor 钉住分片区间的落点：必须在 WHERE 链内、且在排序/增量条件之前。
//
// 拼装位置错了就是「语法错误」或「水位条件被区间挤到 order by 之后」，两种都是整轮失败，
// 而失败点离拼装处很远，只能靠这组断言就地锁死。
func TestSyncQueryPlanSqlFor(t *testing.T) {
	const cond = "`id` >= 1 AND `id` <= 5"

	// 源查询无 WHERE：planSyncQuery 会补 where 1=1，区间条件以 and 接其后。
	// 片段之间必须是换行而非空格（见 sqlFor 里的行注释说明），否则尾行注释会吞掉后续所有片段
	noWhere := newSyncQueryPlan("select * from t", "where 1=1", "and id > 9", "order by id asc")
	assert.Equal(t, "select * from t\nwhere 1=1\nand (`id` >= 1 AND `id` <= 5)\nand id > 9\norder by id asc", noWhere.sqlFor(cond))

	// 源查询已有 WHERE：不补 1=1，区间条件仍接在 WHERE 链里，也不拼出双空格
	withWhere := newSyncQueryPlan("select * from t where status = 1", "", "", "order by id asc")
	assert.Equal(t, "select * from t where status = 1\nand (`id` >= 1 AND `id` <= 5)\norder by id asc", withWhere.sqlFor(cond))

	// 空条件即整轮 SQL：执行日志与不分片路径都取这个值
	assert.Equal(t, "select * from t where status = 1\norder by id asc", withWhere.sqlFor(""))

	// 区间条件必须早于排序，否则是非法 SQL（追加在语句尾部会落在 order by 之后）
	shardSQL := noWhere.sqlFor(cond)
	assert.Less(t, strings.Index(shardSQL, "`id` >= 1"), strings.Index(shardSQL, "order by"), "区间条件必须落在排序子句之前")

	// 尾行注释是本函数最关键的形态：用空格连接时，拼接的区间与 tail 会整段落进注释里，
	// 语句仍然合法但区间完全消失——每一片读的都是同一张过滤后的整表，N 片 = N 倍重复写入，
	// 行数与目标一一对得上、任务自标成功（本轮审查发现的静默错数据原形）
	commented := newSyncQueryPlan("select * from t where status = 1 -- 只看有效行", "", "", "order by id asc")
	for _, got := range []string{commented.sqlFor(cond), commented.sqlFor("")} {
		commentAt := strings.Index(got, "--")
		require.GreaterOrEqual(t, commentAt, 0, "夹具本身必须带行注释")
		tail := got[commentAt:]
		if strings.Contains(got, "and (") {
			assert.Contains(t, tail, "\nand (", "注入的区间必须另起一行，不能落在注释同一行")
		}
		assert.Contains(t, tail, "\norder by", "tail 也必须另起一行，否则排序子句被注释掉")
	}
}

// ---------------- 水位跨分片取最大 ----------------

// TestAdvanceSyncWatermarkTakesRoundMax 分片扫描下增量值在片间交错，水位必须取本轮最大值。
//
// 原实现取「最后处理那一批的末行值」：不分片时源查询恒按增量字段升序，末行即最大值，两者等价；
// 分片后每片各自升序，最后一片的末行可能比前一片小 —— 按末行覆盖会把水位拉回过去，
// 下一轮大量重放，Append 模式下直接多写重复行。
func TestAdvanceSyncWatermarkTakesRoundMax(t *testing.T) {
	sec := &syncExecContext{task: &entity.DataSyncTask{UpdField: "id", UpdFieldVal: "0"}, updFieldName: "id"}

	// 片1 末行 id=5
	require.NoError(t, advanceSyncWatermark([]map[string]any{{"id": int64(5)}}, sec))
	assert.Equal(t, "5", sec.task.UpdFieldVal)

	// 片2 末行 id=3（增量值回绕）：水位必须留在 5，不能被覆盖成 3
	require.NoError(t, advanceSyncWatermark([]map[string]any{{"id": int64(3)}}, sec))
	assert.Equal(t, "5", sec.task.UpdFieldVal, "更小值不得覆盖已推进的水位")

	// 片3 末行 id=9：正常前进
	require.NoError(t, advanceSyncWatermark([]map[string]any{{"id": int64(9)}}, sec))
	assert.Equal(t, "9", sec.task.UpdFieldVal)
}

func TestAdvanceSyncWatermarkKeepsOnError(t *testing.T) {
	sec := &syncExecContext{task: &entity.DataSyncTask{UpdField: "id", UpdFieldVal: "7"}, updFieldName: "id"}

	// 末行缺增量列 / 增量为 NULL：报错并保持水位与轮内最大值不变（调用方按 warn 处理）
	require.Error(t, advanceSyncWatermark([]map[string]any{{"name": "x"}}, sec))
	assert.Equal(t, "7", sec.task.UpdFieldVal)
	require.Error(t, advanceSyncWatermark([]map[string]any{{"id": nil}}, sec))
	assert.Equal(t, "7", sec.task.UpdFieldVal)
	assert.Nil(t, sec.watermarkVal, "探测失败不得污染轮内最大值")

	// 空批直接返回，不报错（尾批可能整批被过滤掉）
	require.NoError(t, advanceSyncWatermark(nil, sec))
}

// TestWatermarkAheadTimeNotLexicographic 时间点必须按绝对时刻比，不能按字符串布局比。
//
// 两个不同 UTC 偏移的 time.Time，字典序与时间序相反：判反方向会把水位推得过分，
// 中间那段增量值的行被永久跳过（静默漏数据，比多读严重得多）。
func TestWatermarkAheadTimeNotLexicographic(t *testing.T) {
	// 2025-12-31 23:00 +0800 == 15:00 UTC，字典序却大于 16:00 UTC 的写法
	later := time.Date(2025, 12, 31, 16, 0, 0, 0, time.UTC)
	earlier := time.Date(2025, 12, 31, 23, 0, 0, 0, time.FixedZone("CST", 8*3600))
	require.True(t, later.After(earlier), "夹具前提：later 的绝对时刻更晚")
	require.Greater(t, earlier.String(), later.String(), "夹具前提：字典序与时间序相反")

	assert.True(t, watermarkAhead(later, earlier), "更晚时间点应判为更靠后")
	assert.False(t, watermarkAhead(earlier, later))

	// 同列不应出现一半时间一半非时间：保守判「不更靠后」，宁可下轮重放
	assert.False(t, watermarkAhead("2026-01-01 00:00:00", earlier))
	assert.False(t, watermarkAhead(earlier, "2026-01-01 00:00:00"))

	// 数值按数值语义（10 > 9），字符串日期退回字典序（同一列同布局下即时间序）
	assert.True(t, watermarkAhead(int64(10), int64(9)))
	assert.True(t, watermarkAhead("2026-01-02", "2026-01-01"))
}

// ---------------- shardTargetRows 配置 ----------------

func TestSetShardTargetRowsExtra(t *testing.T) {
	task := &entity.DataSyncTask{}
	assert.Equal(t, 0, task.GetShardTargetRows(), "未配置即自动")

	task.SetShardTargetRows(50000)
	assert.Equal(t, 50000, task.GetShardTargetRows())
	assert.Equal(t, 50000, task.Extra[entity.ExtraKeyShardTargetRows])

	// <=0 必须清 key：留 0 会让「自动」与「显式 0」两种语义在前端回填时分叉
	task.SetShardTargetRows(0)
	assert.Equal(t, 0, task.GetShardTargetRows())
	_, existed := task.Extra[entity.ExtraKeyShardTargetRows]
	assert.False(t, existed, "0 时必须清除 key")
}

// TestValidateShardTargetRowsBounds 越界配置在保存阶段就拒，而不是运行期静默钳制。
func TestValidateShardTargetRowsBounds(t *testing.T) {
	// 用真方言：DataSQL 的单语句/可拼接校验依赖解析器，nil 方言会在后面的子句里空指针
	dialect := dbi.GetDialect("mysql")
	require.NotNil(t, dialect, "mysql dialect must be registered via blank import of dbm")

	for _, rows := range []int{1, shard.MinTargetRows - 1, shard.MaxTargetRows + 1, 99999999} {
		task := &entity.DataSyncTask{DataSQL: "select * from t"}
		task.SetShardTargetRows(rows)
		err := validateDataSyncSQL(dialect, task)
		require.Error(t, err, "shardTargetRows=%d 应被拒绝", rows)
		assert.Contains(t, err.Error(), "shardTargetRows")
	}

	// 合法值与未配置都不该命中这条守卫（也不应报任何其他错误）
	for _, rows := range []int{0, shard.MinTargetRows, 50000, shard.MaxTargetRows} {
		task := &entity.DataSyncTask{DataSQL: "select * from t"}
		task.SetShardTargetRows(rows)
		assert.NoError(t, validateDataSyncSQL(dialect, task), "shardTargetRows=%d 应合法", rows)
	}
}

// TestShardTargetRowsEffectiveValueIsClamped 保存校验放行后的运行期口径：
// 生效值由 shard 包单点钳制，同步侧不再抄一份阈值。
func TestShardTargetRowsEffectiveValueIsClamped(t *testing.T) {
	assert.Equal(t, shard.MinTargetRows, shard.NormalizeTargetRows(1))
	assert.Equal(t, shard.TargetRows, shard.NormalizeTargetRows(0), "未配置取包默认")
}

// ---------------- 片内续扫 ----------------

// TestContinuationCond 续扫谓词：必须从本片原始区间重建，且游标必须是严格大于。
func TestContinuationCond(t *testing.T) {
	quote := func(s string) string { return "`" + s + "`" }

	next, ok := continuationCond("`id` >= 1 AND `id` <= 500", "id", quote, int64(200))
	require.True(t, ok)
	assert.Equal(t, "`id` >= 1 AND `id` <= 500 AND `id` > 200", next, "区间必须保留，且在尾部追加严格大于")

	next, ok = continuationCond("", "id", quote, int64(7))
	require.True(t, ok)
	assert.Equal(t, "`id` > 7", next, "无区间时（整表一片）只需游标谓词")

	// 其他驱动返回形态（pg 的 int 可能给 int64/int32/字符串）：能归一就续，不能就拒
	next, ok = continuationCond("", "id", quote, int32(9))
	require.True(t, ok)
	assert.Equal(t, "`id` > 9", next)
	_, ok = continuationCond("", "id", quote, nil)
	assert.False(t, ok, "游标为 NULL 时不得算出谓词（否则会拼出 `> 0` 式错误区间）")
	_, ok = continuationCond("", "id", quote, "not-a-number")
	assert.False(t, ok, "非整型游标必须拒续扫，不能静默少读")
}

// TestSqlForOrderOrderByShardKey 分片模式用分片键排序（续扫无损的前提），默认排序保持不变。
func TestSqlForOrderOrderByShardKey(t *testing.T) {
	plan := newSyncQueryPlan("select id, name from t", "where 1=1", "and upd > '2026-01-01'", "order by upd asc")

	defaultSQL := plan.sqlFor("")
	assert.Contains(t, defaultSQL, "order by upd asc", "默认仍按增量字段排序")
	assert.NotContains(t, defaultSQL, "order by `id`")

	shardSQL := plan.sqlForOrder("`id` >= 1 AND `id` <= 100", "order by `id` asc")
	assert.Contains(t, shardSQL, "and (`id` >= 1 AND `id` <= 100)")
	assert.Contains(t, shardSQL, "and upd > '2026-01-01'", "增量条件不得因换排序而丢失")
	assert.Contains(t, shardSQL, "order by `id` asc")
	assert.NotContains(t, shardSQL, "order by upd asc", "分片模式必须且只能按分片键排序")
	assert.Less(t, strings.Index(shardSQL, "and upd >"), strings.Index(shardSQL, "order by"), "谓词必须在排序之前")
}

// TestShardKeyLabel 游标取值用的列标签：别名优先，SELECT * 用列名。
func TestShardKeyLabel(t *testing.T) {
	dialect := dbi.GetDialect("mysql")
	require.NotNil(t, dialect)
	parse := func(sql string) *sqlstmt.SelectStmt {
		stmt, err := dialect.GetSQLParser().Parse(sql)
		require.NoError(t, err)
		sel, ok := stmt.(*sqlstmt.SelectStmt)
		require.True(t, ok)
		return sel
	}

	assert.Equal(t, "id", shardKeyLabel(parse("select * from t"), "id"), "SELECT * 的标签就是列名")
	assert.Equal(t, "id", shardKeyLabel(parse("select id, name from t"), "id"))
	assert.Equal(t, "ID", shardKeyLabel(parse("select ID, name from t"), "id"), "按元数据写法返回（比对大小写不敏感）")
	assert.Equal(t, "pk_id", shardKeyLabel(parse("select id as pk_id, name from t"), "id"), "有别名时游标在别名列上")
	assert.Equal(t, "", shardKeyLabel(parse("select name from t"), "id"), "投影里没有分片键")
}

// TestShardScanOrder 续扫不丢行的充分条件：分片查询必须按分片键升序；
// 不可续扫时不得额外插排序（沿用默认增量排序）。
func TestShardScanOrder(t *testing.T) {
	quote := func(s string) string { return "`" + s + "`" }
	assert.Equal(t, "order by `id` asc", shardScanOrder(true, quote, "id"))
	assert.Equal(t, "", shardScanOrder(false, quote, "id"), "不可续扫时不得改排序")
	assert.Equal(t, "", shardScanOrder(true, quote, ""), "无分片键时无从排序")
}

func TestBudgetStep(t *testing.T) {
	assert.Equal(t, budgetCheckRows, (&syncSourceScanner{batchSize: 1000}).budgetStep())
	assert.Equal(t, budgetCheckRows, (&syncSourceScanner{batchSize: budgetCheckRows}).budgetStep())
	assert.Equal(t, 100, (&syncSourceScanner{batchSize: 100}).budgetStep())
	assert.Equal(t, budgetCheckRows, (&syncSourceScanner{batchSize: 0}).budgetStep(), "batchSize 未归一成 0 时不得返回 0 步长")
}
