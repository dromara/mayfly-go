package api

import (
	"mayfly-go/internal/agentops/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"

	"mayfly-go/internal/agentops/imsg"

	"github.com/spf13/cast"
)

// Governance 治理 API：急停开关（Kill Switch）+ 治理统计看板 + 一键清场。
//
// 急停是全局性危险动作：写入（拉起/解除）与清场必须显式权限点
// （agentops:governance:killswitch）并记审计日志；统计与开关状态读取随菜单权限放行
// （GET 无需权限码，与任务/运行列表口径一致）。
type Governance struct {
	governanceApp application.Governance `inject:"T"`
	runService    application.RunService `inject:"T"`
}

func (g *Governance) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 治理看板统计（days 查询参数，<=0 用服务默认 30 天窗口）
		req.NewGet("/stats", g.Stats),

		// 拉起/解除急停（全局性动作：审计 + 独立权限点）
		req.NewPost("/kill-switch", g.SetKillSwitch).
			Log(req.NewLogSaveI(imsg.LogGovernanceKillSwitch)).
			RequiredPermissionCode("agentops:governance:killswitch"),

		// 一键取消全部非终态运行（急停清场附带动作）
		req.NewPost("/cancel-all", g.CancelAll).
			Log(req.NewLogSaveI(imsg.LogGovernanceCancelAll)).
			RequiredPermissionCode("agentops:governance:killswitch"),
	}

	return req.NewConfs("/agentops/governance", reqs[:]...)
}

// Stats 治理看板统计（运行 KPI 限定窗口 + 资源盘点全量）
func (g *Governance) Stats(rc *req.Ctx) {
	days := cast.ToInt(rc.Query("days"))
	stats, err := g.governanceApp.Stats(rc.MetaCtx, days)
	biz.ErrIsNil(err)
	rc.ResData = stats
}

// killSwitchForm 急停开关设置入参
type killSwitchForm struct {
	// Engaged true=拉起急停（所有 task/trigger 停止派发新运行）；false=解除恢复
	Engaged bool `json:"engaged"`
}

// SetKillSwitch 设置急停开关
func (g *Governance) SetKillSwitch(rc *req.Ctx) {
	form := rc.BindJson[killSwitchForm]()
	rc.ReqParam = form
	biz.ErrIsNil(g.governanceApp.SetKillSwitch(rc.MetaCtx, form.Engaged))
}

// CancelAll 一键取消全部非终态运行（急停清场），返回实际取消数量
func (g *Governance) CancelAll(rc *req.Ctx) {
	canceled, err := g.runService.CancelAllActive(rc.MetaCtx)
	biz.ErrIsNil(err)
	// 结果必须写进审计入参：这是全局性破坏动作，旧写法只留下一行「谁点了清场」，
	// 而 req_param/resp/extra 全空——事后无从知道到底取消了哪几条、取消了几条（旁边的
	// SetKillSwitch 就写了）。数量与界面 toast / 接口响应同一口径，不另造一份判据
	rc.ReqParam = collx.Kvs("canceled", canceled)
	rc.ResData = collx.Kvs("canceled", canceled)
}

var _ req.RouterApi = (*Governance)(nil)
