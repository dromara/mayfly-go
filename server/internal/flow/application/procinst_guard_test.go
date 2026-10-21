package application

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"

	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/model"

	sysapp "mayfly-go/internal/sys/application"
	sysentity "mayfly-go/internal/sys/domain/entity"

	"github.com/stretchr/testify/require"
)

// ==================== 测试桩 ====================

type fakeProcinstRepo struct {
	repository.Procinst
	procinst *entity.Procinst
}

func (f *fakeProcinstRepo) GetById(_ uint64, _ ...string) (*entity.Procinst, error) {
	return f.procinst, nil
}

type fakeCandidateRepo struct {
	repository.ProcinstTaskCandidate
	rows []*entity.ProcinstTaskCandidate
}

func (f *fakeCandidateRepo) SelectByCond(_ any, _ ...string) ([]*entity.ProcinstTaskCandidate, error) {
	return f.rows, nil
}

type fakeRoleApp struct {
	sysapp.Role
	rolesByAccount map[uint64][]uint64
}

func (f *fakeRoleApp) GetAccountRoles(accountId uint64) ([]*sysentity.AccountRole, error) {
	ids := f.rolesByAccount[accountId]
	roles := make([]*sysentity.AccountRole, 0, len(ids))
	for _, id := range ids {
		roles = append(roles, &sysentity.AccountRole{RoleId: id})
	}
	return roles, nil
}

func newProcinstAppForView(p *entity.Procinst, rows []*entity.ProcinstTaskCandidate, rolesByAccount map[uint64][]uint64) *procinstAppImpl {
	app := &procinstAppImpl{}
	app.Repo = &fakeProcinstRepo{procinst: p}
	app.procinstTaskCandidateRepo = &fakeCandidateRepo{rows: rows}
	app.roleApp = &fakeRoleApp{rolesByAccount: rolesByAccount}
	return app
}

func ctxWith(id uint64, username string) context.Context {
	return contextx.WithLoginAccount(context.Background(), &model.LoginAccount{Id: id, Username: username})
}

// ==================== B2：读接口可见性 ====================

// TestCanViewProcinst 详情/操作历史的可见性必须收窄到「发起人 / 管理员 / 办理人 / 候选人」。
//
// 缺陷类（真实发生过）：详情与时间线只按自增 id 取数，任意登录账号遍历 id 即可读到
// bizForm（SQL 全文 / 变更提案）与全部审批意见。
func TestCanViewProcinst(t *testing.T) {
	procinst := &entity.Procinst{Id: 42, CreatorId: 7}
	handler := "alice"
	rows := []*entity.ProcinstTaskCandidate{
		{ProcinstId: 42, Candidate: "11"},
		{ProcinstId: 42, Candidate: "role:3", Handler: &handler},
	}

	// 只有账号 5 属于角色 3（桩不能给所有账号都发角色，否则「陌生人」会被放行）
	app := newProcinstAppForView(procinst, rows, map[uint64][]uint64{5: {3}})

	require.NoError(t, app.CanViewProcinst(ctxWith(7, "creator"), 42), "发起人必须能看")
	require.NoError(t, app.CanViewProcinst(ctxWith(1, "admin"), 42), "管理员必须能看")
	require.NoError(t, app.CanViewProcinst(ctxWith(11, "cand"), 42), "直接候选人必须能看")
	require.NoError(t, app.CanViewProcinst(ctxWith(9, "alice"), 42), "处理人（按账号名匹配）必须能看")
	require.NoError(t, app.CanViewProcinst(ctxWith(5, "role-guy"), 42), "角色候选人必须能看")

	require.Error(t, app.CanViewProcinst(ctxWith(99, "stranger"), 42), "无关账号不得读取")
	require.Error(t, app.CanViewProcinst(context.Background(), 42), "无登录态不得读取")
}

// TestReadHandlersCallCanView 两个读接口必须在取数前调用可见性判定（源码守卫）。
func TestReadHandlersCallCanView(t *testing.T) {
	for _, file := range []string{"procinst.go", "his_procinst_op.go"} {
		raw, err := os.ReadFile("../api/" + file)
		require.NoErrorf(t, err, "读取 %s 失败", file)
		text := stripLineCommentsForFlowGuard(string(raw))
		require.Containsf(t, text, "CanViewProcinst(rc.MetaCtx, instId)",
			"%s 缺少可见性判定：读接口会退化成按自增 id 任意读取", file)
	}
}

// ==================== B1：策略定位失败必须 fail-closed ====================

// TestProcdefLookupDoesNotSwallowError 定位流程定义时读失败不得等同于「未绑定」。
func TestProcdefLookupDoesNotSwallowError(t *testing.T) {
	raw, err := os.ReadFile("procdef.go")
	require.NoError(t, err)
	text := stripLineCommentsForFlowGuard(string(raw))

	require.NotContains(t, text, "if err != nil || len(relateIds) == 0",
		"读错误被并入「未绑定」：标签库抖动一次，策略门禁就在该请求上整体打开")
	require.Contains(t, text, "func (p *procdefAppImpl) GetProcdefByCodePath(ctx context.Context, codePaths ...string) (*entity.Procdef, error)",
		"定位函数必须把失败以 error 上抛")
}

// ==================== H1：完成写入必须 CAS ====================

// TestCompletedProcUsesCAS 完成实例的终态写入必须带前置状态并按行数判赢。
//
// 缺陷类（真实发生过）：`CompletedProc` 只排除 Completed/Terminated，漏了 Cancelled，
// 且用整行 Save——与用户取消并发时会把取消覆盖回「已完成」，随后 AfterCommit 还会执行一次业务回放。
func TestCompletedProcUsesCAS(t *testing.T) {
	raw, err := os.ReadFile("procinst.go")
	require.NoError(t, err)
	text := stripLineCommentsForFlowGuard(string(raw))

	body := regexp.MustCompile(`(?s)(func \(p \*procinstAppImpl\) CompletedProc\(.*?\n\})`).FindStringSubmatch(text)
	require.Len(t, body, 2, "找不到 CompletedProc 函数体")
	fn := body[1]

	require.NotContains(t, fn, "p.Save(ctx, procinst)",
		"终态写入退回整行 Save：会覆盖并发的取消")
	require.Contains(t, fn, `EqStrict("status", entity.ProcinstStatusActive)`,
		"未把前置状态写进更新条件：并发完成/取消无人判赢")
	require.Contains(t, fn, "entity.ProcinstStatusCancelled",
		"终态判断漏了 Cancelled：取消可被回写成已完成")
}

// stripLineCommentsForFlowGuard 去掉行注释（说明注释会引用被禁写法）
func stripLineCommentsForFlowGuard(source string) string {
	var kept []string
	for _, line := range strings.Split(source, "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}
