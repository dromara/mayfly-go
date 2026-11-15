package itest

// 自增主键跨库迁移/恢复的真实链路集成测试（驱动生产 DumpDbScript + ImportDumpStream）。
//
// 为何必须真实库端到端验证：自增列迁移有两个只有真实数据库才会暴露的失败模式——
//   - 目标列静默退化为普通整型（映射丢失）：迁移后应用插入不带id的行直接报 NOT NULL 违反；
//   - 目标序列/计数器未校正：向serial列显式插入id**不会**推进pg序列，若不校正则迁移后
//     第一条业务插入即与已迁移数据主键冲突（Error: duplicate key value violates 主键）。
//     pg的DumpTxnWrapper.AfterInsert为此产出 `SELECT setval(...)`，必须验证它能穿过真实导入
//     链路（切割→批级事务→执行）生效，而不是只验证导出文本。
//
// 断言：导入后插入不带id的行必须成功，且新行id = 已迁移最大id + 1。
//
// 运行：cd server && go test -tags it -count=1 -run TestITAutoIncrement ./internal/db/application/transfer/

import (
	"context"
	"fmt"
	"mayfly-go/internal/db/application/shard"
	"mayfly-go/internal/db/application/transfer"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/dbi/value"
)

const itAiTable = "it_ai_tbl"

// itAiDDL 各方言自增主键建表列定义
var itAiDDL = map[string]string{
	"mysql":    "(id INT AUTO_INCREMENT PRIMARY KEY, val VARCHAR(64))",
	"postgres": "(id serial PRIMARY KEY, val VARCHAR(64))",
	"sqlite":   "(id INTEGER PRIMARY KEY, val TEXT)",
	"mssql":    "(id INT IDENTITY(1,1) PRIMARY KEY, val NVARCHAR(64))",
}

// itAiCombos 全方言×全方言组合（同构组合的源库即目标库，dump内含DROP重建故可直接导入）
func itAiCombos() []itPair {
	return itPairAll(itAllNodes)
}

// itAiPrepare 建自增主键表并可选写入id=1..rowCount的行（显式id，验证迁移后能否续接自增）
func itAiPrepare(t *testing.T, conn *dbi.DbConn, rowCount int) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(itAiTable)))
	require.NoError(t, err, "[%s] 清理源表失败", conn.Info.Type)
	_, err = conn.Exec(fmt.Sprintf("CREATE TABLE %s %s", quote(itAiTable), itAiDDL[string(conn.Info.Type)]))
	require.NoError(t, err, "[%s] 建表失败", conn.Info.Type)

	if rowCount == 0 {
		return
	}
	values := make([]string, 0, rowCount)
	for i := 1; i <= rowCount; i++ {
		values = append(values, fmt.Sprintf("(%d, 'v%02d')", i, i))
	}
	insert := fmt.Sprintf("INSERT INTO %s (id, val) VALUES %s", quote(itAiTable), strings.Join(values, ","))
	_, err = conn.Exec(itIdentityBatch(t, conn, itAiTable, true, insert))
	require.NoError(t, err, "[%s] 写入%d行失败", conn.Info.Type, rowCount)
}

// itAiMaxId 读回最大id（空表为0）
func itAiMaxId(t *testing.T, conn *dbi.DbConn) int64 {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.Query(fmt.Sprintf("SELECT COALESCE(MAX(id), 0) AS mid FROM %s", quote(itAiTable)))
	require.NoError(t, err, "[%s] 查询最大id失败", conn.Info.Type)
	require.Len(t, rows, 1)
	mid, ok := value.ValToInt64(rows[0]["mid"])
	require.True(t, ok, "最大id取值形态异常: %T", rows[0]["mid"])
	return mid
}

// itAiInsertNext 插入不带id的行，返回是否成功（迁移后业务写入的真实验证）
func itAiInsertNext(t *testing.T, conn *dbi.DbConn) error {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, err := conn.Exec(fmt.Sprintf("INSERT INTO %s (val) VALUES ('after-import')", quote(itAiTable)))
	return err
}

// itAiRowCount 读回表行数
func itAiRowCount(t *testing.T, conn *dbi.DbConn) int64 {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.Query(fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %s", quote(itAiTable)))
	require.NoError(t, err, "[%s] 查询行数失败", conn.Info.Type)
	require.Len(t, rows, 1)
	cnt, ok := value.ValToInt64(rows[0]["cnt"])
	require.True(t, ok, "行数取值形态异常: %T", rows[0]["cnt"])
	return cnt
}

// itAiDrop 清理用例表
func itAiDrop(t *testing.T, conn *dbi.DbConn) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, _ = conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(itAiTable)))
}

// TestITAutoIncrementMigrateKeepsSequence 自增表 4×4 迁移后必须能继续自增且不撞已迁移主键
func TestITAutoIncrementMigrateKeepsSequence(t *testing.T) {
	const rows = 5
	for _, c := range itAiCombos() {
		c := c
		t.Run(c.src.name+"->"+c.tgt.name, func(t *testing.T) {
			srcConn, tgtConn := c.src.conn(t), c.tgt.conn(t)
			defer srcConn.Close()
			defer tgtConn.Close()

			itAiPrepare(t, srcConn, rows)
			require.Equal(t, int64(rows), itAiMaxId(t, srcConn), "前置数据未写入，本用例断言无效")

			script := itDumpTable(t, srcConn, itAiTable, c.tgt.dbType, true, true, "")
			require.NotEmpty(t, script)
			if c.src.name != c.tgt.name {
				defer itAiDrop(t, srcConn)
			}
			app := &transfer.DbTransferAppImpl{}
			require.NoError(t, app.ImportDumpStream(context.Background(), 0, tgtConn, strings.NewReader(script)),
				"自增表迁移导入失败, dump脚本:\n%s", itTruncate(script, 4000))

			assert.Equal(t, int64(rows), itAiMaxId(t, tgtConn), "迁移后行数/最大id不一致")

			// 迁移后首条业务写入：不得因目标自增列退化或序列未校正而失败
			if err := itAiInsertNext(t, tgtConn); !assert.NoError(t, err,
				"[%s->%s] 迁移后插入不带id的行失败，自增列可能未迁移或序列未校正", c.src.name, c.tgt.name) {
				return
			}
			assert.Equal(t, int64(rows+1), itAiMaxId(t, tgtConn),
				"迁移后自增未续接已迁移数据（新行id应=%d）", rows+1)

			// 目标元数据仍须标记自增，否则后续结构再迁移会静默丢失该语义
			cols, err := tgtConn.Metadata().GetColumns(itAiTable)
			require.NoError(t, err, "读取目标列元数据失败")
			require.NotEmpty(t, cols)
			hasIncr := false
			for _, col := range cols {
				if col.ColumnName == "id" && col.AutoIncrement {
					hasIncr = true
				}
			}
			assert.True(t, hasIncr, "[%s] 目标id列不再是自增列, 列元数据: %+v", c.tgt.name, cols)

			itAiDrop(t, tgtConn)
		})
	}
}

// TestITAutoIncrementEmptyTableMigrate 空自增表迁移：导出的序列校正语句在零行数据下必须仍可执行。
//
// 空表时 `SELECT setval(seq, (SELECT max(id) FROM t))` 的子查询为NULL，
// 若目标库拒绝NULL参数则整条导入链路失败——大表分片迁移中不含数据的分片段正是该形态
func TestITAutoIncrementEmptyTableMigrate(t *testing.T) {
	for _, c := range itAiCombos() {
		c := c
		t.Run(c.src.name+"->"+c.tgt.name, func(t *testing.T) {
			srcConn, tgtConn := c.src.conn(t), c.tgt.conn(t)
			defer srcConn.Close()
			defer tgtConn.Close()

			itAiPrepare(t, srcConn, 0)
			if c.src.name != c.tgt.name {
				defer itAiDrop(t, srcConn)
			}
			script := itDumpTable(t, srcConn, itAiTable, c.tgt.dbType, true, true, "")
			itAiDrop(t, tgtConn)

			app := &transfer.DbTransferAppImpl{}
			require.NoError(t, app.ImportDumpStream(context.Background(), 0, tgtConn, strings.NewReader(script)),
				"空自增表迁移导入失败, dump脚本:\n%s", itTruncate(script, 4000))

			assert.Equal(t, int64(0), itAiMaxId(t, tgtConn), "空表迁移后应无数据")
			require.NoError(t, itAiInsertNext(t, tgtConn), "空表迁移后首条业务写入失败")
			assert.Equal(t, int64(1), itAiMaxId(t, tgtConn), "空表迁移后自增应从1开始")
			itAiDrop(t, tgtConn)
		})
	}
}

// TestITAutoIncrementShardDataOnlyImport 分片数据阶段导入：每个分片的仅数据产物都会尾部产出序列校正，
// 全部分片按生产顺序（结构阶段→逐分片数据阶段）导入后，自增必须续接全表最大id。
//
// 对应 transferTableData2Db 的 where 分片形态：分片段产物不含DDL（缺陷C回归守卫），
// 且尾部的 setval 校正基于目标表全表max，必须仍能把序列推到最终值
func TestITAutoIncrementShardDataOnlyImport(t *testing.T) {
	const rows = 20
	for _, node := range itAllNodes {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()

			itAiPrepare(t, conn, rows)
			app := &transfer.DbTransferAppImpl{}

			// 结构阶段与数据阶段的产物均从有数据的源表导出（结构阶段导入会DROP重建，必须预先采集）
			ddlScript := itDumpTable(t, conn, itAiTable, node.dbType, true, false, "")
			bounds := []string{"id <= 7", "id > 7 AND id <= 14", "id > 14"}
			shards := make([]string, 0, len(bounds))
			for _, where := range bounds {
				shard := itDumpTable(t, conn, itAiTable, node.dbType, false, true, where)
				require.NotContains(t, shard, "CREATE TABLE", "分片段产物不应含DDL")
				shards = append(shards, shard)
			}

			// 结构阶段：DROP重建，表清空但自增语义保留
			require.NoError(t, app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(ddlScript)),
				"结构阶段导入失败")
			require.Equal(t, int64(0), itAiMaxId(t, conn), "结构阶段不应携带数据")

			// 数据阶段：按主键分3片逐片导入（与生产分片迁移同一标志位与where语义）
			for i, shard := range shards {
				require.NotEmpty(t, strings.TrimSpace(shard))
				require.NoError(t, app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(shard)),
					"分片%d导入失败, dump脚本:\n%s", i+1, itTruncate(shard, 3000))
			}

			assert.Equal(t, int64(rows), itAiMaxId(t, conn), "分片导入后最大id不一致")
			assert.Equal(t, int64(rows), itAiRowCount(t, conn), "分片导入后行数不一致（分片间不得重叠或遗漏）")

			require.NoError(t, itAiInsertNext(t, conn), "分片导入后自增未续接（首条业务写入失败）")
			assert.Equal(t, int64(rows+1), itAiMaxId(t, conn), "分片导入后新行id不一致")
			itAiDrop(t, conn)
		})
	}
}

// TestITAutoIncrementParallelShardImport 生产阶段2的真实形态：**分片并行**导入到自增表。
//
// DbTransferAppImpl.Run 把一张表的多个分片丢进errgroup并发池（SetLimit=任务并发度），
// 每个分片段尾部都带一句序列校正，而校正写在本分片事务内、提交前执行，
// 因而只能读到“本段已写入 + 他人已提交”的max。此处的不变式是：
// 无论分片以何种顺序完成，序列终值必须不低于全表最大id；否则迁移后第一条业务写入即撞主键（duplicate key）。
// 该性质不能只靠「写入值与本段读到的max及序列当前值取GREATEST」：读当前值与写回不是一个原子动作，
// 并发下仍会丢更新（本机 pg16 实测：B 读到旧值后 A 写 100 并提交，B 仍写回 GREATEST(50,1)=50）。
// 语句现在用事务级咨询锁把「读→写」围成临界区（见 postgres.AfterInsert），竞态本身由
// TestITPgSequenceCorrectionHoldsAdvisoryLock 确定性钉死；本用例则从结果侧守住全链路的行数与终值
// （多轮×多分片×满并发采样不同交错，断言只依赖结果，不引入抖动）
func TestITAutoIncrementParallelShardImport(t *testing.T) {
	const (
		rows        = 300
		shardRows   = 40 // 300/40 → 8个分片，放大并行交错空间
		cycles      = 3  // 重复整轮，采样不同完成顺序
		Concurrency = 8  // 与分片数同量级，让全部分片真正并发
	)
	for _, node := range itAllNodes {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()

			origTargetRows := shard.TargetRows
			shard.TargetRows = shardRows
			defer func() { shard.TargetRows = origTargetRows }()

			app := &transfer.DbTransferAppImpl{}
			for cycle := 0; cycle < cycles; cycle++ {
				itAiPrepare(t, conn, rows)

				wheres := app.PlanTableShards(context.Background(), 0, conn, itAiTable, rows)
				require.GreaterOrEqual(t, len(wheres), 3,
					"第%d轮：主键分片未生效（并行导入退化为单任务，本用例失去鉴别力）", cycle+1)

				// 结构与分片数据脚本均在源表有数据时采集：结构段导入会DROP重建
				ddlScript := itDumpTable(t, conn, itAiTable, node.dbType, true, false, "")
				shards := make([]string, 0, len(wheres))
				for _, where := range wheres {
					shard := itDumpTable(t, conn, itAiTable, node.dbType, false, true, where)
					require.NotContains(t, shard, "CREATE TABLE", "分片段产物不应含DDL")
					shards = append(shards, shard)
				}

				require.NoError(t, app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(ddlScript)),
					"第%d轮：结构阶段导入失败", cycle+1)
				require.Equal(t, int64(0), itAiRowCount(t, conn), "第%d轮：结构阶段不应携带数据", cycle+1)

				eg := new(errgroup.Group)
				eg.SetLimit(Concurrency)
				for i, shard := range shards {
					i, shard := i, shard
					eg.Go(func() error {
						if err := app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(shard)); err != nil {
							return fmt.Errorf("分片%d导入失败: %w\n%s", i+1, err, itTruncate(shard, 2000))
						}
						return nil
					})
				}
				require.NoError(t, eg.Wait(), "第%d轮：分片并行导入失败", cycle+1)

				assert.Equal(t, int64(rows), itAiRowCount(t, conn), "第%d轮：并行导入后行数不一致（丢行或重复）", cycle+1)
				assert.Equal(t, int64(rows), itAiMaxId(t, conn), "第%d轮：并行导入后最大id不一致", cycle+1)
				assert.Equal(t, int64(rows), itAiDistinctIdCount(t, conn), "第%d轮：id存在重叠或空洞", cycle+1)

				require.NoError(t, itAiInsertNext(t, conn),
					"第%d轮：并行分片导入后自增/序列未续接到全表最大id（首条业务写入即撞主键）", cycle+1)
				assert.Equal(t, int64(rows+1), itAiMaxId(t, conn), "第%d轮：并行导入后新行id不一致", cycle+1)
				itAiDrop(t, conn)
			}
		})
	}
}

// itAiDistinctIdCount 去重后的id数，用于证明并行分片既无重叠也无遗漏
func itAiDistinctIdCount(t *testing.T, conn *dbi.DbConn) int64 {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.Query(fmt.Sprintf("SELECT COUNT(DISTINCT id) AS cnt FROM %s", quote(itAiTable)))
	require.NoError(t, err, "[%s] 查询去重id数失败", conn.Info.Type)
	require.Len(t, rows, 1)
	cnt, ok := value.ValToInt64(rows[0]["cnt"])
	require.True(t, ok, "去重id数取值形态异常: %T", rows[0]["cnt"])
	return cnt
}

// itAiCorrectionStmt 从真实 dump 产物里取出自增序列校正那一句（用例要跑的就是生产语句原文，
// 而不是自己拼一份看似等价的副本）
func itAiCorrectionStmt(t *testing.T, conn *dbi.DbConn) string {
	t.Helper()
	script := itDumpTable(t, conn, itAiTable, "postgres", false, true, "")
	for _, line := range strings.Split(script, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "SELECT setval(") {
			return strings.TrimSpace(line)
		}
	}
	t.Fatalf("dump 产物未含序列校正语句:\n%s", itTruncate(script, 2000))
	return ""
}

// itAiSeqLastValue 读序列当前值（仅 pg：本用例钉的就是 pg 的 setval 读-改-写）
func itAiSeqLastValue(t *testing.T, conn *dbi.DbConn) int64 {
	t.Helper()
	_, rows, err := conn.Query(`SELECT last_value FROM "it_ai_tbl_id_seq"`)
	require.NoError(t, err, "读序列当前值失败")
	require.Len(t, rows, 1)
	v, ok := value.ValToInt64(rows[0]["last_value"])
	require.True(t, ok, "序列当前值形态异常: %T", rows[0]["last_value"])
	return v
}

// TestITPgSequenceCorrectionHoldsAdvisoryLock 并发分片导入下，校正语句的「读序列当前值→写回」
// 必须是一个事务级咨询锁保护的临界区。
//
// 竞态原型（本机 pg16 psql 实测，无需任何巧合调度）：
//   - B 先执行，读到 last_value=1；
//   - A 随后 setval(100) 并提交；
//   - B 再写回 GREATEST(50, 1)=50 → 序列终值 50 低于全表最大 100。
//
// setval 本身立即生效且不随事务回滚，所以「只增不减」的 GREATEST 仍拦不住：丢的是**读到的那一份**
// 旧值（旧注释把“setval 立即可见”误当成“读与写不可分割”）。不修则 pg 目标库上的并行分片迁移
// 会随机产出「迁移后第一条业务插入即撞主键」的表。
//
// 本用例不用概率性并发，而是直接钉住机制：A 持有事务不提交时，B 的同一句必须阻塞在锁上；
// A 提交后 B 才能进入临界区，因而它读到的当前值必然不低于 A 刚写入的值。去掉锁后两个断言都会变红。
func TestITPgSequenceCorrectionHoldsAdvisoryLock(t *testing.T) {
	conn := itPg.conn(t)
	defer conn.Close()
	itAiPrepare(t, conn, 0)
	defer itAiDrop(t, conn)
	quote := conn.GetDialect().Quoter().QuoteIdent

	// 基线行 id=50 已提交：并发侧（B）看到的 max 就是 50，比 A 写入的 100 小
	_, err := conn.Exec(fmt.Sprintf("INSERT INTO %s (id, val) VALUES (50, 'base')", quote(itAiTable)))
	require.NoError(t, err)
	correction := itAiCorrectionStmt(t, conn)

	app := &transfer.DbTransferAppImpl{}

	// A：事务内写入 id=100（未提交但自身可见）→ 执行校正 → 故意不提交，持有锁到用例放行
	txA, err := conn.Begin()
	require.NoError(t, err)
	_, err = conn.TxExecContext(context.Background(), txA,
		fmt.Sprintf("INSERT INTO %s (id, val) VALUES (100, 'holder')", quote(itAiTable)))
	require.NoError(t, err)
	_, err = conn.TxExecContext(context.Background(), txA, correction)
	require.NoError(t, err, "A 侧执行生产校正语句失败（语句形态本身不合法？）")

	// B：走真实导入链路（自带事务与语句切割），它在 A 提交前不得完成
	done := make(chan error, 1)
	go func() {
		done <- app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(correction))
	}()
	select {
	case err := <-done:
		t.Fatalf("并发侧未被咨询锁阻塞，「读→写」不是临界区（丢更新竞态仍存在）: err=%v seq=%d", err, itAiSeqLastValue(t, conn))
	case <-time.After(400 * time.Millisecond):
	}

	require.NoError(t, dbi.CommitTargetTx(conn, txA))
	select {
	case err := <-done:
		require.NoError(t, err, "A 提交后并发侧执行校正失败")
	case <-time.After(30 * time.Second):
		t.Fatal("A 已提交但并发侧仍未完成：事务级咨询锁未随事务释放")
	}

	// 结果侧不变式：没有锁时 B 会用旧值把序列拉回 50
	maxId := itAiMaxId(t, conn)
	require.GreaterOrEqual(t, maxId, int64(100), "夹具前提：全表最大 id 应为 100")
	assert.GreaterOrEqual(t, itAiSeqLastValue(t, conn), maxId, "序列终值必须不低于全表最大id")
	require.NoError(t, itAiInsertNext(t, conn), "首条业务写入仍撞主键")
}
