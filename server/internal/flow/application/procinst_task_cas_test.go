package application

import (
	"context"
	"testing"

	"os"
	"strings"

	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// procinstTaskRepoStub 只补 GetPageList，其余能力来自嵌入的通用实现（与 rule_segment_test 同法）
type procinstTaskRepoStub struct {
	*base.RepoImpl[*entity.ProcinstTask]
}

func (s *procinstTaskRepoStub) GetPageList(*entity.ProcinstTaskQuery, ...string) (*model.PageResult[*entity.ProcinstTaskPO], error) {
	return nil, nil
}

func newCasTaskApp() *procinstTaskAppImpl {
	app := &procinstTaskAppImpl{}
	app.Repo = &procinstTaskRepoStub{RepoImpl: &base.RepoImpl[*entity.ProcinstTask]{}}
	return app
}

func setupCasTestRepo(t *testing.T) base.Repo[*entity.ProcinstTask] {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&entity.ProcinstTask{}))

	// base.RepoImpl 走全局 db，测试期间接管并在结束后归还
	previous := global.Db
	global.Db = db
	t.Cleanup(func() { global.Db = previous })

	return &base.RepoImpl[*entity.ProcinstTask]{}
}

// TestUpdateByCondIsCompareAndSet 「抢到这次流转」必须能被区分出来。
//
// 或签 / 会签下两个审批人可能同时点同一个任务，双方读到的都是「审批中」；
// 若条件更新不回报行数，两边都会认为自己完成成功，工单被批两次、
// 批后回放的业务（一条 SQL、一条命令）被执行两次
func TestUpdateByCondIsCompareAndSet(t *testing.T) {
	repo := setupCasTestRepo(t)
	ctx := context.Background()

	// 不预设主键：Insert 走创建分支才会补齐 create_time 等审计字段，随后用生成的 id 做 CAS 条件
	task := &entity.ProcinstTask{NodeKey: "approve", Status: entity.ProcinstTaskStatusProcess}
	require.NoError(t, repo.Insert(ctx, task))
	require.NotZero(t, task.Id)

	cond := model.NewCond().Eq("id", task.Id).Eq("status", entity.ProcinstTaskStatusProcess)

	rows, err := repo.UpdateByCond(ctx, &entity.ProcinstTask{Status: entity.ProcinstTaskStatusCompleted, Remark: "a"}, cond)
	require.NoError(t, err)
	require.Equal(t, int64(1), rows, "the first transition must win the row")

	rows, err = repo.UpdateByCond(ctx, &entity.ProcinstTask{Status: entity.ProcinstTaskStatusReject, Remark: "b"}, cond)
	require.NoError(t, err)
	require.Zero(t, rows, "a second concurrent handler must lose, otherwise the task is approved twice")

	var stored entity.ProcinstTask
	require.NoError(t, global.Db.First(&stored, task.Id).Error)
	require.Equal(t, entity.ProcinstTaskStatusCompleted, stored.Status, "the loser must not overwrite the winner")
	require.Equal(t, "a", stored.Remark)
}

// TestUpdateByCondInsideTransaction 事务内也必须报出「有没有抢到」。
//
// 生产路径是 p.Tx 里执行，取的是 ctx 上的事务 db（与全局 db 是两条分支），
// 只测非事务分支就会让事务分支里的错误静默通过
func TestUpdateByCondInsideTransaction(t *testing.T) {
	repo := setupCasTestRepo(t)
	ctx := context.Background()

	task := &entity.ProcinstTask{NodeKey: "approve", Status: entity.ProcinstTaskStatusProcess}
	require.NoError(t, repo.Insert(ctx, task))

	tx := global.Db.Begin()
	require.NoError(t, tx.Error)
	txCtx := base.NewCtxWithDb(ctx, tx)

	rows, err := repo.UpdateByCond(txCtx, &entity.ProcinstTask{Status: entity.ProcinstTaskStatusCompleted}, model.NewCond().Eq("id", task.Id).Eq("status", entity.ProcinstTaskStatusProcess))
	require.NoError(t, err)
	require.Equal(t, int64(1), rows, "inside a transaction the first CAS must still win")

	rows, err = repo.UpdateByCond(txCtx, &entity.ProcinstTask{Status: entity.ProcinstTaskStatusReject}, model.NewCond().Eq("id", task.Id).Eq("status", entity.ProcinstTaskStatusProcess))
	require.NoError(t, err)
	require.Zero(t, rows, "the same transaction must not be able to take the same transition twice")
	require.NoError(t, tx.Rollback().Error)
}

// TestCompleteInstTaskRejectsLoserHandler 后到的审批人必须被拒，而不是「成功但什么也没改」。
//
// 这里守的是 completeInstTask 自己：条件里的 status=Process 一旦被去掉，
// 第二个审批人就会当成通过成功，工单被批两次、批后回放的业务被执行两次
func TestCompleteInstTaskRejectsLoserHandler(t *testing.T) {
	repo := setupCasTestRepo(t)
	app := newCasTaskApp()
	ctx := context.Background()

	task := &entity.ProcinstTask{NodeKey: "approve", Status: entity.ProcinstTaskStatusProcess}
	require.NoError(t, repo.Insert(ctx, task))

	winner := &entity.ProcinstTask{NodeKey: "approve", Status: entity.ProcinstTaskStatusCompleted, Remark: "first"}
	winner.SetId(task.Id)
	require.NoError(t, app.completeInstTask(ctx, winner))

	loser := &entity.ProcinstTask{NodeKey: "approve", Status: entity.ProcinstTaskStatusReject, Remark: "second"}
	loser.SetId(task.Id)
	err := app.completeInstTask(ctx, loser)
	require.Error(t, err, "a handler arriving after the task reached a terminal state must be rejected")
	require.Contains(t, err.Error(), "已被其他审批人处理")

	var stored entity.ProcinstTask
	require.NoError(t, global.Db.First(&stored, task.Id).Error)
	require.Equal(t, entity.ProcinstTaskStatusCompleted, stored.Status)
	require.Equal(t, "first", stored.Remark)
}

// TestTerminalWritesUseCompareAndSet 三处终态写入必须走 CAS。
//
// 「未满足完成条件时只保存计数」那处仍是普通保存（任务状态没变，不该被 CAS 拒），
// 所以这里守的是数量与位置，而不是禁止一切 Save
func TestTerminalWritesUseCompareAndSet(t *testing.T) {
	content, err := os.ReadFile("procinst_task.go")
	require.NoError(t, err)
	source := string(content)

	require.Equal(t, 3, strings.Count(source, "p.completeInstTask(ctx, instTask)"), "通过/拒绝/退回三条终态写入都要走 CAS")
	require.Equal(t, 1, strings.Count(source, "return p.Save(ctx, instTask)"), "只允许保留「未完成任务只存计数」这一处普通保存")
	require.NotContains(t, source, "if err := p.Save(ctx, instTask)", "终态写入不得退回无条件更新")
}
