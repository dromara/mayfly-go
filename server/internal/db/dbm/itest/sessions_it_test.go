//go:build it

package itest

// 会话内省与取消（可选能力 SessionInspector）的真实数据库集成测试。
//
// 覆盖三件只有在真库上才能证伪的事：
//  1. 长查询确实出现在活动会话列表里，且各字段（标识/库/账号/已执行时长/语句文本）取的是库侧事实；
//  2. 取消能立刻中断它（不是等 sleep 自然结束）——这是「服务端主动取消」存在的意义；
//  3. 列表不能包含发起这次内省的自己，否则界面会多出一条永远取消不掉的幽灵会话。
//
// 已接 IT 节点的方言：mysql、postgres、mssql、oracle（不可达则 SkipOrRequire 跳过）。
// clickhouse/dm 也已实现并声明 FeatSessions（护栏保证声明⟺接口实现），但本环境无可达实例、未接 IT 节点，
// 其系统视图列名（system.processes / V$SESSIONS）与取消语义（KILL QUERY / SP_CLOSE_SESSION）需在真实实例上跑通后补节点。
//
// 运行：cd server && go test -tags it -count=1 -run TestITSession ./internal/db/dbm/itest/

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

// sessionSleepNode 各方言的长查询语句与用于在列表里认出它的文本片段
type sessionSleepNode struct {
	name      string
	conn      func(t *testing.T) *dbi.DbConn
	sleepSql  string
	match     string
	selfMatch string // 内省语句自身的特征文本，用于断言列表不含自己
	// hasStartedAt 该方言能否给出「语句开始时间」：mysql 的 processlist.TIME 只是持续秒数，
	// 拿 now()-elapsed 反推出来的是一个看上去真实的时间点，属于虚构
	hasStartedAt bool
	// quickSql 一条只会跑几十毫秒的短语句，用于反复制造「对方刚刚才开始」的窗口。
	// 只有「时长 = 两个时间戳相减」的方言需要（pg/mssql）：mysql/oracle 拿的是库侧
	// 已给定的非负秒数，算不出负值，因此为空、不做加压
	quickSql string
	// useExec 该方言的长跑语句要走 Exec 而不是 Query 起。
	//
	// 只有 oracle 需要：Oracle 的 KILL SESSION 是协作式的，目标会话经 go-ora 的游标（Query）路径
	// 执行时被标记待杀（ORA-00031）后并不会中断当前调用——真库实测 DBMS_SESSION.SLEEP(8) 与一条长 SELECT
	// 经 QueryContext 执行时，KILL 之后仍跑满全程才返回；同一条语句走 ExecContext 则会被立刻终止
	//（ORA-00028，~30ms）。因此要在 oracle 上验证「取消立刻中断一个在跑的操作」，就用 Exec 路径起会话——
	// 这也正是生产里取消长跑 DML 的真实场景。游标路径的协作式终止是 Oracle + 驱动固有语义，
	// 已在产品代码 sessions.go 的 CancelSession 处注明，不由本用例强行断言
	useExec bool
}

func sessionNodes() []sessionSleepNode {
	return []sessionSleepNode{
		{name: "mysql", conn: mysqlConn, sleepSql: "SELECT SLEEP(8)", match: "SLEEP(", selfMatch: "processlist"},
		{name: "pg", conn: pgConn, sleepSql: "SELECT pg_sleep(8)", match: "pg_sleep", selfMatch: "pg_stat_activity", hasStartedAt: true, quickSql: "SELECT pg_sleep(0.03)"},
		{name: "mssql", conn: gcMssqlConn, sleepSql: "WAITFOR DELAY '00:00:08'", match: "WAITFOR", selfMatch: "dm_exec_sessions", hasStartedAt: true, quickSql: "WAITFOR DELAY '00:00:00.05'"},
		// oracle 23ai FREE 上 DBMS_SESSION.SLEEP 不能当 SQL 函数调用（真库实测 ORA-00904：
		// "DBMS_SESSION"."SLEEP": invalid identifier），只能走匿名 PL/SQL 块；
		// 用例前提不变（必须有一条长跑语句可看见可取消），只是方言内的写法不同。
		// useExec：Oracle KILL 对游标路径是协作式的（不会中断在跑的调用），走 Exec 才会被立刻终止，详见字段说明
		{name: "oracle", conn: oracleConn, sleepSql: "BEGIN DBMS_SESSION.SLEEP(8); END;", match: "SLEEP", selfMatch: "v$session", quickSql: "BEGIN DBMS_SESSION.SLEEP(0.05); END;", useExec: true},
	}
}

// startBusySession 在独立 goroutine 里跑一条长查询，返回其错误通道。
// node.useExec 为真时走 ExecContext（oracle 需要，原因见 sessionSleepNode.useExec 说明）
func startBusySession(conn *dbi.DbConn, node sessionSleepNode) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		var err error
		if node.useExec {
			_, err = conn.ExecContext(context.Background(), node.sleepSql)
		} else {
			_, _, err = conn.QueryContext(context.Background(), node.sleepSql)
		}
		errCh <- err
	}()
	return errCh
}

// waitSessionFor 轮询等待长查询出现在会话列表里（系统表可见有毫秒级延迟，不轮询会偶发失败）
func waitSessionFor(t *testing.T, conn *dbi.DbConn, match string) *dbi.SessionInfo {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		sessions, err := conn.Metadata().ListSessions(context.Background())
		require.NoError(t, err)
		for _, s := range sessions {
			if strings.Contains(s.Query, match) {
				return s
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("长查询未出现在活动会话列表中，会话内省链路不通")
	return nil
}

func TestITSessionListAndCancel(t *testing.T) {
	for _, node := range sessionNodes() {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()

			errCh := startBusySession(conn, node)
			target := waitSessionFor(t, conn, node.match)

			assert.NotEmpty(t, target.Id, "会话标识必须可用于取消")
			assert.NotEmpty(t, target.User, "会话账号应如实取自库侧系统表")
			// 时长是「已经跑了多久」，长查询被识别到时至少要已计时 0ms 且不超过语句总时长
			assert.GreaterOrEqual(t, target.ElapsedMs, int64(0))
			assert.Less(t, target.ElapsedMs, int64(9000))
			// 时间点必须与方言的真实能力一致：能给的不能缺，给不了的不能造
			if node.hasStartedAt {
				require.NotNil(t, target.StartedAt, "该方言能给出语句开始时间，必须填上而不是留空")
				assert.False(t, target.StartedAt.IsZero(), "开始时间不得是零值")
			} else {
				assert.Nil(t, target.StartedAt, "该方言只给出已持续时长，不得虚构时间点")
			}

			cancelledAt := time.Now()
			require.NoError(t, conn.Metadata().CancelSession(context.Background(), target.Id))

			select {
			case err := <-errCh:
				require.Error(t, err, "被取消的查询必须以错误结束，不能静默返回成功")
				// sleep 语句本身要 8 秒；若取消只是「等它跑完」，这里会耗满 8 秒
				assert.Less(t, time.Since(cancelledAt), 5*time.Second, "取消必须立即生效")
			case <-time.After(8 * time.Second):
				t.Fatal("取消后长查询仍未结束")
			}
		})
	}
}

// TestITSessionListExcludesSelf 内省列表不得包含发起内省自身的会话：
// mysql/pg 都在 SQL 里排除了当前连接，这里用「列表里没有含内省语句特征的会话」来钉住
func TestITSessionListExcludesSelf(t *testing.T) {
	for _, node := range sessionNodes() {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()

			sessions, err := conn.Metadata().ListSessions(context.Background())
			require.NoError(t, err)
			for _, s := range sessions {
				assert.NotContains(t, s.Query, node.selfMatch, "会话列表不应包含本次内省自身")
			}
		})
	}
}

// TestITSessionElapsedNeverNegative 「已执行时长」对操作者永远是一个非负的读数。
//
// 真库跑出来的真实缺陷：时长是「本语句的快照时间 - 另一条会话的语句开始时间」，两者来自不同快照，
// 对方刚开始跑时就会算出负值（pg 实测到 -7ms），界面上直接渲染成「已执行 -7ms」。
// 这里不只测目标会话，而是把实例上当时刻看到的一切行（含后台进程、空闲会话、刚开始的请求）都过一遍：
// 单点断言只有在这种全量口径下才有牙，否则要靠恰好踩中时序才会红
func TestITSessionElapsedNeverNegative(t *testing.T) {
	for _, node := range sessionNodes() {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()

			// 加压：并发跑一批短语句，同时不停内省。负时长只出现在「对方在我们取到快照时间之后
			// 才开始」这个窄窗口里，靠自然流量赌不到；加压后每一轮都能造出几十个这样的窗口
			if node.quickSql != "" {
				stop := make(chan struct{})
				var wg sync.WaitGroup
				for i := 0; i < 4; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						for {
							select {
							case <-stop:
								return
							default:
								// 短语句失败不影响本用例：只需要它的开始时间落在窗口里
								_, _, _ = conn.QueryContext(context.Background(), node.quickSql)
							}
						}
					}()
				}
				defer func() {
					close(stop)
					wg.Wait()
				}()
			}

			// 多轮取量：负值只在「对方刚开始」那个窄窗口里出现，一轮取不到不代表没问题
			for round := 0; round < 12; round++ {
				sessions, err := conn.Metadata().ListSessions(context.Background())
				require.NoError(t, err)
				for _, s := range sessions {
					if s.ElapsedMs < 0 {
						t.Fatalf("会话 id=%s 的已执行时长为负: %dms（语句：%s）", s.Id, s.ElapsedMs, s.Query)
					}
				}
				time.Sleep(20 * time.Millisecond)
			}
			if node.quickSql == "" {
				t.Logf("[%s] 该方言的时长由库侧直接给出非负秒数，不做时间戳相减，无需加压验证", node.name)
			}
		})
	}
}

// TestITSessionCancelInvalidId 非法会话标识必须本地拒绝，不把请求内容交给数据库
func TestITSessionCancelInvalidId(t *testing.T) {
	for _, node := range sessionNodes() {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()

			for _, bad := range []string{"", "0", "-1", "1; DROP TABLE it_probe", "abc"} {
				err := conn.Metadata().CancelSession(context.Background(), bad)
				require.Error(t, err, "非法会话标识 %q 必须被拒绝", bad)
				assert.True(t, errors.Is(err, dbi.ErrInvalidSessionId),
					"非法标识必须是可判定的 ErrInvalidSessionId，而不是数据库报错: %v", err)
			}
		})
	}
}

// TestITSessionUnsupportedDialect 未实现该能力的方言必须返回 ErrUnsupportedKind，
// 而不是空列表——空列表会让上层把「不支持」误读成「没有会话」
func TestITSessionUnsupportedDialect(t *testing.T) {
	conn := sqliteConn(t)
	defer conn.Close()

	_, err := conn.Metadata().ListSessions(context.Background())
	require.Error(t, err)
	assert.True(t, errors.Is(err, dbi.ErrUnsupportedKind), "sqlite 未实现会话内省，应报不支持: %v", err)
	assert.True(t, errors.Is(conn.Metadata().CancelSession(context.Background(), "1"), dbi.ErrUnsupportedKind))
}
