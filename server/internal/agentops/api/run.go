package api

import (
	"mayfly-go/internal/agentops/api/vo"
	"mayfly-go/internal/agentops/application"
	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/structx"

	"github.com/spf13/cast"
	"mayfly-go/internal/agentops/imsg"
)

// PermissionRunQueryAll 允许查询他人名下运行记录的权限点。
//
// 运行列表 GET 按平台惯例不挂权限码（否则所有人看不到执行历史），所以可见面改由
// owner 收窄：未持本码且非超管者，只能看到自己作为运行身份（owner）的记录。
// 不复用写侧的 PermissionRunAsOwner：「能看别人跑出了什么结论」与「能以别人身份跑」是两回事
const PermissionRunQueryAll = "agentops:run:queryAll"

// Run 运行记录（执行历史）API：查询三类触发（cron/event/manual）产出的运行、查看结论、
// 点击跳会话回放（前端用 conversationId 定位），以及取消排队/运行中的运行（急停）。
type Run struct {
	runApp application.RunService `inject:"T"`
}

func (r *Run) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 运行记录分页列表（执行历史）
		req.NewGet("", r.GetPageList),

		// 运行详情（含完整 instruction/input/summary）
		req.NewGet("/:id", r.GetById),

		// 取消排队/运行中的运行（急停/人工取消）
		req.NewPost("/:id/cancel", r.Cancel).Log(req.NewLogSaveI(imsg.LogRunCancel)).RequiredPermissionCode("agentops:run:cancel"),

		// 人类反馈（点赞/点踩 + 备注）：落库并沉淀经验记忆（学习闭环，阶段 4.3）
		req.NewPost("/:id/feedback", r.Feedback).Log(req.NewLogSaveI(imsg.LogRunFeedback)).RequiredPermissionCode("agentops:run:feedback"),

		// 离线评测回放：用诊断档重跑原始输入并由审查档对比打分（历史事件重放，阶段 4.3）
		req.NewPost("/:id/eval", r.ReplayEval).Log(req.NewLogSaveI(imsg.LogRunEval)).RequiredPermissionCode("agentops:run:eval"),
	}

	return req.NewConfs("/agentops/runs", reqs[:]...)
}

// GetPageList 分页查询运行记录，裁剪大字段后返回列表视图
func (r *Run) GetPageList(rc *req.Ctx) {
	cond, pageParam := rc.BindQueryAndPage[entity.Run]()
	// 收窄必须发生在查询前：绑定进来的 ownerAccountId 由这里覆写，非授权者无法借查询参数读到他人运行
	scopeRunsToCaller(rc, cond)

	pageRes, err := r.runApp.GetPageList(cond, pageParam, "id desc")
	biz.ErrIsNil(err)

	rc.ResData = model.PageResultConv[*entity.Run, *vo.RunVO](pageRes)
}

// GetById 运行详情（含完整提示词/输入/结论，供详情抽屉展示）
func (r *Run) GetById(rc *req.Ctx) {
	run := r.mustGetVisible(rc, cast.ToUint64(rc.PathParam("id")))

	detail := &vo.RunDetailVO{}
	biz.ErrIsNil(structx.Copy(detail, run))
	rc.ResData = detail
}

// mustGetVisible 取运行并断言调用者可见（读详情/取消/反馈/评测四处共用同一口径）。
//
// 为什么写入口也必须断言：application 层签名（Cancel/Feedback/ReplayEval）不带调用者身份，
// 条件更新只认 run id——知道 id 就能掐断他人正在跑的变更、改写他人反馈、以他人身份重放其提示词。
// 越权统一按不存在处理，不用错误码差异泄露资源存在性
func (r *Run) mustGetVisible(rc *req.Ctx, id uint64) *entity.Run {
	run, err := r.runApp.GetById(id)
	biz.ErrIsNil(err)
	biz.IsTrue(canQueryAllRuns(rc) || run.OwnerAccountId == runCallerId(rc), "run not found")
	return run
}

// canQueryAllRuns 是否可查询他人运行：超级管理员，或持有 PermissionRunQueryAll。
//
// 取登录账号用 rc.GetLoginAccount()：它无登录态直接中断请求（fail-closed），
// 所以这里不需要也不会出现「拿不到账号」的可达分支
func canQueryAllRuns(rc *req.Ctx) bool {
	la := rc.GetLoginAccount()
	if la.Id == consts.AdminId {
		return true
	}
	registry := req.GetPermissionCodeRegistery()
	if registry == nil {
		return false // 注册表未装配时 fail-closed，不默认放行全量
	}
	// 同时接受 impersonate 码：能以任何账号身份派发运行的人，逻辑上就已能看全量运行；
	// 而任务/触发器/提案的归属断言复用的正是这个码（见 assertResourceOwned），
	// 两边不同源会出现「能改他人任务却看不到它的运行」这种无法解释的中间态
	return registry.HasCode(la.Id, PermissionRunQueryAll) || registry.HasCode(la.Id, PermissionRunAsOwner)
}

// runCallerId 取当前登录账号 id（无登录态由 GetLoginAccount 直接中断）。
//
// 不能「拿不到就当 0」：NewCond 的 Eq 会忽略零值条件，「按 owner 收窄」会静默变成「不过滤」
func runCallerId(rc *req.Ctx) uint64 {
	return rc.GetLoginAccount().Id
}

// scopeRunsToCaller 把运行查询条件收窄到调用者可见面：非授权者强制按 owner 过滤
func scopeRunsToCaller(rc *req.Ctx, cond *entity.Run) {
	if canQueryAllRuns(rc) {
		return
	}
	cond.OwnerAccountId = runCallerId(rc)
}

// Cancel 取消一个排队/运行中的运行；已终态则无操作（幂等）
func (r *Run) Cancel(rc *req.Ctx) {
	id := cast.ToUint64(rc.PathParam("id"))
	rc.ReqParam = id
	// 取消会真实中断在途执行（cancel 本实例 ctx），归属断言不可省：非归属者只能取消自己的
	r.mustGetVisible(rc, id)
	biz.ErrIsNil(r.runApp.Cancel(rc.MetaCtx, id))
}

// runFeedbackForm 运行反馈表单
type runFeedbackForm struct {
	// Feedback 反馈取值：up=点赞 down=点踩
	Feedback string `json:"feedback" binding:"required"`
	// Note 反馈备注（原因说明，随经验沉淀进记忆系统）
	Note string `json:"note"`
}

// Feedback 对终态运行点赞/点踩：落库 + 沉淀为操作员经验记忆（fail-open，记忆失败不影响落库）
func (r *Run) Feedback(rc *req.Ctx) {
	id := cast.ToUint64(rc.PathParam("id"))
	body := rc.BindJson[runFeedbackForm]()
	rc.ReqParam = map[string]any{"id": id, "feedback": body.Feedback, "note": body.Note}
	// 反馈随 extra 整列写回且会沉淀记忆：不拦则任何人可给他人运行写任意反馈
	r.mustGetVisible(rc, id)
	biz.ErrIsNil(r.runApp.Feedback(rc.MetaCtx, id, body.Feedback, body.Note))
}

// ReplayEval 离线评测回放：返回评测运行（异步执行，前端可跳执行历史观察进度）
func (r *Run) ReplayEval(rc *req.Ctx) {
	id := cast.ToUint64(rc.PathParam("id"))
	rc.ReqParam = id
	// 评测以原运行 owner 身份重放其完整指令/输入（工具权限按该 owner 收敛）：
	// 不断言可见面就等于低权账号给个 id 就能冒他人身份读资源、把他人提示词连数据带走
	r.mustGetVisible(rc, id)
	evalRun, err := r.runApp.ReplayEval(rc.MetaCtx, id)
	biz.ErrIsNil(err)
	rc.ResData = map[string]any{"runId": evalRun.Id, "runNo": evalRun.RunNo}
}

var _ req.RouterApi = (*Run)(nil)
