package api

import (
	"strings"

	"mayfly-go/internal/agentops/api/form"
	"mayfly-go/internal/agentops/api/vo"
	"mayfly-go/internal/agentops/application"
	"mayfly-go/internal/agentops/domain/entity"
	sysapp "mayfly-go/internal/sys/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"

	"mayfly-go/internal/agentops/imsg"

	"mayfly-go/pkg/errorx"

	"github.com/spf13/cast"
)

// Trigger 事件驱动触发器 API：触发器 CRUD + 外部 webhook 推送入口。
//
// 告警联动无需在此暴露入口——alert 引擎通知成功后经 global.EventBus 发布中立事件，agentops
// 订阅后自动派发（见 init 装配）。webhook 入口供 Prometheus/Zabbix/自定义监控直接推送，
// 以 URL 路径令牌鉴权（路由级不校 token），归一化后同样派发只读诊断 Run。
type Trigger struct {
	triggerApp application.TriggerService `inject:"T"`
	// agentTaskApp 保存时断言绑定任务的可见归属（见 Save）：不注入则「绑他人任务」无闸可守
	agentTaskApp application.AgentTaskService `inject:"T"`
	// accountApp 把表单指定的 ownerAccountId 解析为账号名（仅 resolveRunOwner 代他人指定时使用）
	accountApp sysapp.Account `inject:"T"`
}

func (t *Trigger) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 触发器分页列表（不回传推送令牌，见 GetPageList）
		req.NewGet("", t.GetPageList),

		// 触发器详情（编辑回填；含明文 webhookToken，因此与保存同一权限点把守）
		req.NewGet("/:id", t.GetById).RequiredPermissionCode("agentops:trigger:save"),

		// 保存触发器（新建/更新）
		req.NewPost("", t.Save).Log(req.NewLogSaveI(imsg.LogTriggerSave)).RequiredPermissionCode("agentops:trigger:save"),

		// 删除触发器（批量，逗号分隔）
		req.NewDelete(":ids", t.Delete).Log(req.NewLogSaveI(imsg.LogTriggerDelete)).RequiredPermissionCode("agentops:trigger:del"),

		// 外部监控 webhook 推送入口：令牌即鉴权，路由级不校 token（外部系统无法附加 Authorization 头）。
		// 因此令牌只能从本模块的详情接口（受 permission 把守）拿到，列表不回传
		req.NewPost("/webhook/:token", t.Webhook).DontNeedToken(),
	}

	return req.NewConfs("/agentops/triggers", reqs[:]...)
}

// GetPageList 分页查询触发器。
//
// 列表一律清空 webhookToken：该令牌是免鉴权 webhook 路由的唯一鉴权因子，而列表 GET 按平台惯例
// 不带权限码（任意登录用户可达）。把凭据放在无鉴权出口的返回体里，等于把「驱动一次无人值守的
// 诊断运行、以绑定 owner 身份读监控数据」的能力交给任何登录账号，并可反复触发消耗 LLM 配额。
// 需要拼接推送地址的行操作走详情接口取令牌（与保存同一权限点）
func (t *Trigger) GetPageList(rc *req.Ctx) {
	cond, pageParam := rc.BindQueryAndPage[entity.Trigger]()
	// 列表已清空令牌，但可见面本身也要收窄（他人触发器的名称/匹配条件/绑定的任务都是信息）
	scopeOwnerToCaller(rc, func(accountId uint64) { cond.OwnerAccountId = accountId })

	pageRes, err := t.triggerApp.GetPageList(cond, pageParam, "id desc")
	biz.ErrIsNil(err)

	resVo := model.PageResultConv[*entity.Trigger, *vo.TriggerVO](pageRes)
	for _, item := range resVo.List {
		item.WebhookToken = ""
	}
	t.fillTaskNames(resVo.List)
	rc.ResData = resVo
}

// fillTaskNames 批量回查关联任务名并填进 VO（列表与详情共用同一份取数）。
//
// 为什么在后端补而不是前端拿任务列表去映射：触发器列表与任务列表各自按调用者收窄，
// 两个可见面并不相同；前端自行映射会为「看得到触发器、看不到那个任务」的行留一个裸 id，
// 而且每页都要多拉一次全量任务。这里一次 IN 查询而不是逐行回查（N+1）；
// 基座自动附加软删过滤，已删除的任务查不回来→TaskName 为空，由前端渲染成「任务已不存在」
func (t *Trigger) fillTaskNames(items []*vo.TriggerVO) {
	taskIds := make([]uint64, 0, len(items))
	for _, item := range items {
		if item.TaskId != 0 {
			taskIds = append(taskIds, item.TaskId)
		}
	}
	if len(taskIds) == 0 {
		return
	}
	tasks, err := t.agentTaskApp.GetByIds(taskIds, "name")
	biz.ErrIsNil(err)

	names := make(map[uint64]string, len(tasks))
	for _, task := range tasks {
		names[task.Id] = task.Name
	}
	for _, item := range items {
		item.TaskName = names[item.TaskId]
	}
}

// GetById 触发器详情（含完整 instruction/matchCond/webhookToken，供编辑表单回填与拼接推送地址）
func (t *Trigger) GetById(rc *req.Ctx) {
	id := cast.ToUint64(rc.PathParam("id"))
	trigger, err := t.triggerApp.GetById(id)
	biz.ErrIsNil(err)
	// 详情会带出完整 webhook 令牌（列表已清空），所以归属断言必须在此：
	// 否则任何持详情权限的账号都能拿到他人令牌并伪造事件触发运行
	biz.ErrIsNil(assertResourceOwned(rc, trigger.OwnerAccountId))

	resVo := model.PageResultConv[*entity.Trigger, *vo.TriggerVO](&model.PageResult[*entity.Trigger]{List: []*entity.Trigger{trigger}})
	// 与列表同源补任务名：只补列表会漏下编辑抽屉（它读详情接口，关联任务下拉就回到裸 id）
	t.fillTaskNames(resVo.List)
	rc.ResData = resVo.List[0]
}

// Save 保存触发器：内联模式（taskId=0）的运行身份经 resolveRunOwner 解析
// （缺省当前登录账号；指定他人需超管或 owner-impersonate 权限点）；关联任务模式以任务 owner 为准。
func (t *Trigger) Save(rc *req.Ctx) {
	triggerForm := rc.BindJson[form.TriggerForm]()
	rc.ReqParam = triggerForm

	ownerId, ownerName := uint64(0), ""
	// 仅内联模式需要在触发器上落运行身份；关联任务模式运行身份取自任务
	if triggerForm.TaskId == 0 {
		owner, err := resolveRunOwner(rc, t.accountApp, triggerForm.OwnerAccountId)
		biz.ErrIsNil(err)
		ownerId, ownerName = owner.AccountId, owner.Username
	} else {
		// 绑定任务必须对调用者可见：派发以任务 owner 身份入队，不拦就等于任何持 trigger:save
		// 的账号都能把他人（如 admin）任务绑上免鉴权 webhook，拿令牌以高权身份无限驱动诊断运行
		task, terr := t.agentTaskApp.GetById(triggerForm.TaskId)
		biz.ErrIsNil(terr)
		biz.ErrIsNil(assertResourceOwned(rc, task.OwnerAccountId))
	}

	// 更新分支断言归属（与详情/删除同口径）：否则可改他人触发器的匹配条件/绑定任务/启用态，
	// 甚至把 owner 改成自己；令牌侧的保留保护拦不住这两条等价能力
	if triggerForm.Id != 0 {
		old, gerr := t.triggerApp.GetById(triggerForm.Id)
		biz.ErrIsNil(gerr)
		biz.ErrIsNil(assertResourceOwned(rc, old.OwnerAccountId))
	}

	// enabled 仅 1/-1 合法；0 不是状态值而是「未传」哨兵（int8 零值无法区分两者），故按默认启用处理，
	// 避免「保存成功却永不生效」。前端状态下拉只提供 1/-1，不会显式传 0；真要禁用请传 -1
	// （手工构造 enabled:0 的请求会看到它变 1，这是设计而非缺陷）
	enabled := triggerForm.Enabled
	if enabled == 0 {
		enabled = entity.TriggerEnabled
	}

	trigger := &entity.Trigger{
		Name:        triggerForm.Name,
		SourceType:  triggerForm.SourceType,
		MatchCond:   triggerForm.MatchCond,
		TaskId:      triggerForm.TaskId,
		Instruction: triggerForm.Instruction,
		// 芯片段随指令一并落库（Extra）：不收则内联触发器派发给模型的指令没有资源定位信息，
		// 模型只能凭空猜 id（任务侧早已收，这里是对称漏项）
		InstructionSegments: triggerForm.InstructionSegments,
		ProfileId:           triggerForm.ProfileId,
		// 结果通知配置：不取则 webhook/告警驱动的诊断运行永远无人可通知
		NotifyType:      triggerForm.NotifyType,
		NotifyTmplCode:  triggerForm.NotifyTmplCode,
		OwnerAccountId:  ownerId,
		OwnerUsername:   ownerName,
		CooldownSeconds: triggerForm.CooldownSeconds,
		Enabled:         enabled,
		Remark:          triggerForm.Remark,
	}
	trigger.Id = triggerForm.Id

	biz.ErrIsNil(t.triggerApp.SaveTrigger(rc.MetaCtx, trigger))
	// 令牌只在两个时刻回传：新建（服务端刚生成，无旧地址可坏）与显式轮换。
	// 常规更新一律保留旧值且不回传（见 resolveWebhookToken）
	resp := map[string]any{"id": trigger.Id, "webhookToken": ""}
	if triggerForm.RotateWebhookToken {
		token, err := t.triggerApp.RotateWebhookToken(rc.MetaCtx, trigger.Id)
		biz.ErrIsNil(err)
		resp["webhookToken"] = token
	} else if triggerForm.Id == 0 {
		resp["webhookToken"] = trigger.WebhookToken
	}
	rc.ResData = resp
}

// Delete 批量删除触发器（逗号分隔 id）
func (t *Trigger) Delete(rc *req.Ctx) {
	idsStr := rc.PathParam("ids")
	rc.ReqParam = idsStr
	for _, id := range strings.Split(idsStr, ",") {
		// 跳过空段：批量删除末尾空串会变成 id=0 的无意义查询
		if id == "" {
			continue
		}
		triggerId := cast.ToUint64(id)
		trigger, err := t.triggerApp.GetById(triggerId)
		biz.ErrIsNil(err)
		biz.ErrIsNil(assertResourceOwned(rc, trigger.OwnerAccountId))
		biz.ErrIsNil(t.triggerApp.Delete(rc.MetaCtx, triggerId))
	}
}

// Webhook 外部监控推送入口：按 URL 令牌定位触发器 → 归一化原始 JSON → 匹配/冷却 → 派生诊断 Run。
// 令牌即鉴权（无效/禁用/非 webhook 来源一律拒绝），运行身份取触发器绑定 owner，无需登录态。
func (t *Trigger) Webhook(rc *req.Ctx) {
	token := rc.PathParam("token")
	rc.ReqParam = map[string]any{"token": token}

	// 刻意不用 rc.BindJson：它在解析失败时 panic，于是空 body 会被外部监控收到一个只带追踪码的
	// 500「server error」——既读不出该改什么，又污染错误率告警。推送体不合法是调用方的错，
	// 必须留在业务层给 400：拿不到 JSON 对象（空 body / 截断 / 数组 / 标量）统一回一条准确文案；
	// Normalize 里那条「必须是 JSON 对象」作为分层防御保留（本层不再依赖它）
	var payload map[string]any
	if bindErr := rc.BindJSON(&payload); bindErr != nil {
		biz.ErrIsNil(errorx.NewBizI(rc.MetaCtx, imsg.ErrWebhookPayloadUnparsable))
	}
	run, err := t.triggerApp.DispatchWebhook(rc.MetaCtx, token, payload)
	biz.ErrIsNil(err)

	rc.ResData = map[string]any{"runId": run.Id, "runNo": run.RunNo}
}

var _ req.RouterApi = (*Trigger)(nil)
