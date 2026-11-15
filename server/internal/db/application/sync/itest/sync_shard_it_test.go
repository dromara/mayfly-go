//go:build it

package itest

// 源侧主键分片扫描的端到端集成测试（黑盒，经导出入口 SyncTask 驱动完整链路）。
//
// 存在的理由：分片改变的是「读源结构」（一条大 SELECT → N 条按主键闭区间限定的短查询），
// 单测只能证明区间数学与语句拼装，而「各片结果并集恰好等于整表、不重不漏、水位不被片间交错拉回」
// 只有真库跑通才成立。同时钉住不可分片的形态必须退回单次流式并给出原因：退回路径若静默失效，
// 症状就是大表同步又被源库查询超时杀掉，而日志里看不出系统有没有尝试过分片。
//
// 运行：cd server && go test -tags it -count=1 -run TestITShard ./internal/db/application/sync/itest/

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/sync"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
)

const (
	shardSrcTable = "it_sync_shard_src"
	shardTgtTable = "it_sync_shard_tgt"
	shardRows     = 2400
	shardMaxUpd   = 999999
)

// setupShard 建源/目标表并灌入 shardRows 行；srcDDL 决定源表主键形态（整型 / 非整型）。
//
// 增量值刻意与主键倒挂：全局最大值 shardMaxUpd 落在 **第一片**（id=5）。若水位仍按
// 「最后一次读取的末行值」推进，末片给出的只是 shardRows，本用例的水位断言就会失败——
// 数据不这样设计，实现就没有鉴别力。
func setupShard(t *testing.T, srcDDL string) (*sync.DataSyncAppImpl, *dbi.DbConn, *dbi.DbConn) {
	t.Helper()
	src := mysqlConn(t)
	t.Cleanup(func() {
		mustExec(t, src, "DROP TABLE IF EXISTS "+quote(src, shardSrcTable))
		src.Close()
	})
	mustExec(t, src, "DROP TABLE IF EXISTS "+quote(src, shardSrcTable))
	mustExec(t, src, fmt.Sprintf("CREATE TABLE %s (%s)", quote(src, shardSrcTable), srcDDL))

	dst := pgConn(t)
	t.Cleanup(func() {
		mustExec(t, dst, "DROP TABLE IF EXISTS "+quote(dst, shardTgtTable))
		dst.Close()
	})
	mustExec(t, dst, "DROP TABLE IF EXISTS "+quote(dst, shardTgtTable))
	mustExec(t, dst, fmt.Sprintf("CREATE TABLE %s (id int primary key, name varchar(50), upd int)", quote(dst, shardTgtTable)))

	// 显式列名插入：srcDDL 改变列序（varchar 主键用例把 name 放在首列）时仍按语义写入
	for start := 1; start <= shardRows; start += 500 {
		end := min(start+499, shardRows)
		values := make([]string, 0, end-start+1)
		for id := start; id <= end; id++ {
			upd := id
			if id == 5 {
				upd = shardMaxUpd
			}
			values = append(values, fmt.Sprintf("(%d,'n%d',%d)", id, id, upd))
		}
		mustExec(t, src, fmt.Sprintf("INSERT INTO %s (id, name, upd) VALUES %s",
			quote(src, shardSrcTable), strings.Join(values, ",")))
	}

	app, _ := newApp()
	return app, src, dst
}

// newShardTask 单片 1000 行：2400 行必然切成多片。
// 生产默认 20 万行/片，在小夹具表上永远走不到分片分支，所以本用例必须显式调小。
func newShardTask(dataSQL string) *entity.DataSyncTask {
	task := &entity.DataSyncTask{
		Id: 9920, TaskName: "it-sync-shard", SyncMode: entity.DataSyncModeIncrementalMerge,
		SrcDbId: 1, TargetDbId: 2, TargetTableName: shardTgtTable,
		PageSize:    250,
		FieldMap:    `[{"src":"id","target":"id"},{"src":"name","target":"name"},{"src":"upd","target":"upd"}]`,
		UpdField:    "upd",
		UpdFieldVal: "0",
		DataSQL:     dataSQL,
	}
	task.SetShardTargetRows(1000)
	// 分片后各片按增量字段升序、片间交错，水位靠跨片取最大；Exclusive 与 Inclusive 在本用例
	// 的数据下取到同一最大值，显式 Exclusive 以免与游标边界语义的专项用例混在一起
	task.SetCursorInclusivity(entity.CursorInclusivityExclusive)
	return task
}

// TestITShardScanCoversEveryRowOnce 分片读取不重不漏，且水位取跨片最大值。
func TestITShardScanCoversEveryRowOnce(t *testing.T) {
	app, src, dst := setupShard(t, "id int primary key, name varchar(50), upd int")
	ctx := context.Background()
	task := newShardTask(fmt.Sprintf("SELECT id, name, upd FROM %s", shardSrcTable))

	syncLog := &entity.DataSyncLog{}
	metrics := sync.NewSyncMetrics()
	require.NoError(t, app.SyncTask(ctx, task, src, dst, syncLog, metrics), syncLog.ErrText)

	// 分片确实发生（而不是静默退回单次流式）：运行日志里出现第 2 片才说明语句真被拆开了
	assert.Contains(t, syncLog.RunLog, "片扫描", "运行日志未记录分片计划: %s", syncLog.RunLog)
	assert.Contains(t, syncLog.RunLog, "分片 2/", "单片 1000 行、2400 行数据应至少切成 2 片: %s", syncLog.RunLog)

	// 整轮 SQL（执行日志展示用）不带分片区间：区间逐片写在分片段里，避免只看到最后一片
	assert.NotContains(t, syncLog.DataSQLFull, ">= 1 AND", "DataSQLFull 不应固化某一片的区间: %s", syncLog.DataSQLFull)

	assert.Equal(t, shardRows, countRows(t, dst, shardTgtTable), "目标行数应等于源行数")
	ids := queryIds(t, dst, shardTgtTable)
	require.Len(t, ids, shardRows)
	for i, id := range ids {
		require.EqualValues(t, i+1, id, "第 %d 个主键不连续，分片间存在丢行或重复", i+1)
	}

	// 水位取跨片最大值：全局最大刻意落在第一片，按「末次读取末行」实现会停在 shardRows
	assert.Equal(t, fmt.Sprint(shardMaxUpd), task.UpdFieldVal, "水位必须跨分片取最大，否则下一轮整表重放")
	assert.Equal(t, shardRows, metrics.TotalRows)

	// 第二轮只应带走真正越过水位的新行：反证上一轮水位没有推得过分（否则这些行永久漏掉）
	mustExec(t, src, fmt.Sprintf("INSERT INTO %s (id, name, upd) VALUES (3001,'late-a',%d),(3002,'late-b',%d)",
		quote(src, shardSrcTable), shardMaxUpd+1, shardMaxUpd+2))
	log2 := &entity.DataSyncLog{}
	require.NoError(t, app.SyncTask(ctx, task, src, dst, log2, nil), log2.ErrText)
	assert.Equal(t, 2, log2.ResNum, "越过水位的新行必须被同步")
	assert.Equal(t, shardRows+2, countRows(t, dst, shardTgtTable))
	assert.Equal(t, fmt.Sprint(shardMaxUpd+2), task.UpdFieldVal)
}

// TestITShardScanRefusesOrRootWithoutDuplication 顶层 WHERE 以 OR 为根时必须拒分片。
//
// 区间谓词以 `and (...)` 追加，AND 比 OR 结合更紧：`WHERE id%2=0 OR id=1 and (区间)` 被源库
// 解成 `id%2=0 OR (id=1 AND 区间)`，于是偶数 id 的每一行在**每一片**都重复返回。
// 目标表有主键且 UPSERT 幂等，最终数据看上去仍然正确——鉴别力必须落在写入行数上：不拦就是
// 3 片 × 1200 行的重复投递（ResNum 变近 3 倍），自标成功、白跑一场且日志里没有错误。
//
// 本用例故意不配增量字段（appends=false）：只让分片门禁参与，不跟 Save/Run 的谓词拦截在一起。
func TestITShardScanRefusesOrRootWithoutDuplication(t *testing.T) {
	app, src, dst := setupShard(t, "id int primary key, name varchar(50), upd int")
	// 1..2400 里的偶数行 + id=1
	wantRows := shardRows/2 + 1

	cases := []struct {
		name       string
		dataSQL    string
		wantShard  bool
		wantReason string
	}{
		{
			name:       "顶层 OR 根：拒分片",
			dataSQL:    fmt.Sprintf("SELECT id, name, upd FROM %s WHERE id %% 2 = 0 OR id = 1", shardSrcTable),
			wantReason: "OR/XOR 为根",
		},
		{
			// 解析器在 % 处截断了 WHERE 表达式（Where.Text 只剩 "id"），AST 判形看不到那个 OR；
			// 本轮审查发现的误放行原形就是这个形状，词法扫描必须同样拒掉
			name:       "取模后的顶层 OR：仍拒分片",
			dataSQL:    fmt.Sprintf("SELECT id, name, upd FROM %s WHERE id %% 2 = 0 OR id = 1 AND name IS NOT NULL", shardSrcTable),
			wantReason: "OR/XOR 为根",
		},
		{
			// 括号把整个条件包起来后，追加的 AND 作用于括号结果，语义仍正确：
			// 不能为了安全把所有带 OR 的查询一刀切拒掉（那就本末倒置了）
			name:       "括号包裹的 OR：仍分片",
			dataSQL:    fmt.Sprintf("SELECT id, name, upd FROM %s WHERE (id %% 2 = 0 OR id = 1)", shardSrcTable),
			wantShard:  true,
			wantReason: "分片 2/",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			task := newShardTask(c.dataSQL)
			// 全量刷新且不配增量字段：运行期不拼任何水位谓词，只有分片区间依赖这一判形
			task.SyncMode = entity.DataSyncModeFullRefresh
			task.UpdField = ""
			task.UpdFieldVal = ""
			syncLog := &entity.DataSyncLog{}
			metrics := sync.NewSyncMetrics()
			require.NoError(t, app.SyncTask(context.Background(), task, src, dst, syncLog, metrics), syncLog.ErrText)

			assert.Contains(t, syncLog.RunLog, c.wantReason, "运行日志必须说清分了片还是为何没分: %s", syncLog.RunLog)
			if c.wantShard {
				assert.Contains(t, syncLog.RunLog, "片扫描", "括号包裹的 OR 仍应走分片: %s", syncLog.RunLog)
			} else {
				assert.Contains(t, syncLog.RunLog, "整表单次查询", "OR 根应退回单次流式: %s", syncLog.RunLog)
			}
			assert.Equal(t, wantRows, syncLog.ResNum, "写入行数应与源命中行一致，多出的部分就是重复投递: %s", syncLog.RunLog)
			assert.Equal(t, wantRows, metrics.TotalRows, "指标同样不得计入重复行")
			assert.Equal(t, wantRows, countRows(t, dst, shardTgtTable), "目标数据必须完整不重")
		})
	}
}

// TestITShardScanWithTrailingLineComment 用户 DataSQL 以行注释结尾时，分片区间必须仍然生效。
//
// 这是本轮审查里最严重的一类形态：片段用空格拼接时，`and (区间)` 与 tail 会整段落进尾行注释，
// 语句完全合法、不报错、不丢行，但每一片读的都是**同一张整表** → N 片 = N 倍重复写入，
// 目标行数与 ResNum 一一对得上，任务自标成功——单靠看日志或比对行数根本看不出来。
// 所以本用例的鉴别力必须落在「写入总行数 == 源行数」上，而不是日志里有没有「分片」字样
// （片段被注释吞掉时那句日志照样会打）。
func TestITShardScanWithTrailingLineComment(t *testing.T) {
	app, src, dst := setupShard(t, "id int primary key, name varchar(50), upd int")
	ctx := context.Background()

	for _, comment := range []string{"-- 只看有效行", "# mysql 风格行注释"} {
		t.Run(comment, func(t *testing.T) {
			mustExec(t, dst, fmt.Sprintf("TRUNCATE TABLE %s", quote(dst, shardTgtTable)))
			task := newShardTask(fmt.Sprintf("SELECT id, name, upd FROM %s WHERE id > 0 %s", shardSrcTable, comment))
			// 全量刷新且不配增量字段：拼接只可能来自分片区间与排序，变量单一
			task.SyncMode = entity.DataSyncModeFullRefresh
			task.UpdField = ""
			task.UpdFieldVal = ""

			syncLog := &entity.DataSyncLog{}
			metrics := sync.NewSyncMetrics()
			require.NoError(t, app.SyncTask(ctx, task, src, dst, syncLog, metrics), syncLog.ErrText)

			// 确实走了分片（片数 >1），否则下面的行数断言就没有意义
			assert.Contains(t, syncLog.RunLog, "分片 2/", "夹具应切成多片: %s", syncLog.RunLog)
			// 关键断言：注入没被注释吞掉时 total == 源行数；被吞掉则是 shardRows × 片数
			assert.Equal(t, shardRows, syncLog.ResNum, "每片必须只读自己的区间，否则就是整表重复投递: %s", syncLog.RunLog)
			assert.Equal(t, shardRows, metrics.TotalRows)
			assert.Equal(t, shardRows, countRows(t, dst, shardTgtTable), "目标行数必须与源一一对应")
		})
	}
}

// TestITIncrementalRejectsOrRootWhere 增量模式（运行期要追加 `and upd > 水位`）碰上顶层 OR 根
// 时必须直接报错，而不是跑完并自标成功：拼接后只约束 OR 的一支，另一支的全部历史行会每轮
// 重复同步（重复写 + 水位形同失效）。这是分片之前就存在的同源缺陷，报错要给出可执行改法。
func TestITIncrementalRejectsOrRootWhere(t *testing.T) {
	app, src, dst := setupShard(t, "id int primary key, name varchar(50), upd int")
	task := newShardTask(fmt.Sprintf("SELECT id, name, upd FROM %s WHERE id %% 2 = 0 OR id = 1", shardSrcTable))

	syncLog := &entity.DataSyncLog{}
	err := app.SyncTask(context.Background(), task, src, dst, syncLog, nil)
	require.Error(t, err, "顶层 OR + 增量谓词必须报错，不得静默重复同步")
	assert.Contains(t, err.Error(), "OR/XOR", "报错必须说清原因: %s", err.Error())
	assert.Zero(t, countRows(t, dst, shardTgtTable), "拦住就不该写入任何数据")
}

// TestITShardScanRefusedFallsBackToSingleQuery 不可分片的形态必须退回单次流式并在日志给出原因。
//
// 两条拒绝路径：结果集未原样带出分片键列（聚合与表达式包住主键也走这里）、源表无单列整型主键。
// 两轮最终数据都必须完整——退回路径若把语句拼错，症状是同步失败或静默漏行。
func TestITShardScanRefusedFallsBackToSingleQuery(t *testing.T) {
	cases := []struct {
		name       string
		srcDDL     string
		dataSQL    string
		wantReason string
	}{
		{
			name: "主键未原样带出",
			// 主键被表达式包住（id + 0）：解析器不再认出普通列引用，分片无法证明行归属；
			// 取位仍等于 id，所以目标数据应完整——拒绝分片但不能拒掉同步本身
			srcDDL:     "id int primary key, name varchar(50), upd int",
			dataSQL:    fmt.Sprintf("SELECT id + 0 AS id, name, upd FROM %s", shardSrcTable),
			wantReason: "未原样带出分片键列",
		},
		{
			name:       "无单列整型主键",
			srcDDL:     "name varchar(50) primary key, id int, upd int",
			dataSQL:    fmt.Sprintf("SELECT id, name, upd FROM %s", shardSrcTable),
			wantReason: "无单列整型主键",
		},
		{
			name: "含 DISTINCT",
			// DISTINCT 保留 id 列，否则投影不带主键会先命中「未原样带出分片键列」那条拒绝路径
			srcDDL:     "id int primary key, name varchar(50), upd int",
			dataSQL:    fmt.Sprintf("SELECT DISTINCT id, name, upd FROM %s", shardSrcTable),
			wantReason: "SELECT DISTINCT 去重范围会由整表缩为每片",
		},
		{
			name: "含窗口函数",
			// 投影带出主键且顶层无 ORDER BY（OVER 内的排序不属于顶层子句），前三道守卫全过；
			// 窗口列却按「本次可见行集」重算——不拒就是每片各自编号，结果静默失真
			srcDDL:     "id int primary key, name varchar(50), upd int",
			dataSQL:    fmt.Sprintf("SELECT id, name, upd, ROW_NUMBER() OVER (ORDER BY upd DESC) rn FROM %s", shardSrcTable),
			wantReason: "投影含窗口函数（OVER）",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			app, src, dst := setupShard(t, c.srcDDL)
			task := newShardTask(c.dataSQL)
			syncLog := &entity.DataSyncLog{}
			require.NoError(t, app.SyncTask(context.Background(), task, src, dst, syncLog, nil), syncLog.ErrText)

			assert.Contains(t, syncLog.RunLog, "整表单次查询", "不可分片时应退回单次流式: %s", syncLog.RunLog)
			assert.Contains(t, syncLog.RunLog, c.wantReason, "拒绝原因必须写进运行日志: %s", syncLog.RunLog)
			// 三例的投影都保留 id 且行行唯一（upd 与 id 一一对应），退回单次流式后目标应拿到全部源行
			assert.Equal(t, shardRows, countRows(t, dst, shardTgtTable), "退回路径同样必须同步完整")
		})
	}
}
