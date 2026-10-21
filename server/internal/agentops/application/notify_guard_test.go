package application

import (
	"context"
	"errors"
	"testing"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/trigger"
	"mayfly-go/internal/ai/pkg/headless"
	msgapp "mayfly-go/internal/msg/application"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tmplReject 模拟 msg 侧「模板发不出去」的业务错误（真实文案由 msg/imsg 按请求语言生成）
var tmplReject = errors.New("消息模板[broken]未关联任何消息渠道，通知发不出去")

// fakeMsgTmplApp 只实现 CheckSendable 的消息模板服务替身。
//
// 内嵌 nil 接口即可满足 msgapp.MsgTmpl：本组用例只应触达 CheckSendable，其余方法一旦被调到
// 就当场 panic——那说明保存校验走偏了路径，比返回零值静默通过更有信号
type fakeMsgTmplApp struct {
	msgapp.MsgTmpl

	checked []string
	err     error
}

func (f *fakeMsgTmplApp) CheckSendable(_ context.Context, tmplCode string) error {
	f.checked = append(f.checked, tmplCode)
	return f.err
}

// notifyTask 构造一个「除通知配置外全部合法」的任务，便于把失败原因唯一归到通知配置上
func notifyTask(tmplCode string) *entity.AgentTask {
	return &entity.AgentTask{
		Name: "通知校验任务", OwnerAccountId: 9, OwnerUsername: "dba",
		Instruction:    "检查慢查询",
		ProfileId:      headless.ProfileReviewer,
		NotifyType:     entity.TaskNotifyAlways,
		NotifyTmplCode: tmplCode,
		Status:         entity.AgentTaskStatusDisable, // 不自动调度，隔离 cron 绑定对断言的干扰
	}
}

// inlineTrigger 构造一个内联模式的触发器（除通知配置外全部合法）
func inlineTrigger(tmplCode string) *entity.Trigger {
	return &entity.Trigger{
		Name: "通知校验触发器", SourceType: trigger.SourceTypeAlert,
		OwnerAccountId: 9, OwnerUsername: "dba", Instruction: "诊断该机器",
		ProfileId: headless.ProfileDiagnostician, Enabled: entity.TriggerEnabled,
		NotifyType: entity.TaskNotifyAlways, NotifyTmplCode: tmplCode,
	}
}

// TestCheckNotifyTmplSendableSkipsWhenNothingToSend 三种「无模板可校验」的情况都必须放行且不查库。
//
// 最后一条尤其关键：msgTmplApp 未注入（单测装配、或 IOC 缺失）时必须跳过而不是 panic，
// 否则保存任务/触发器这条主链路会被一个「只是没法校验」的依赖整个拖死
func TestCheckNotifyTmplSendableSkipsWhenNothingToSend(t *testing.T) {
	fake := &fakeMsgTmplApp{err: errors.New("不应被调用")}
	ctx := context.Background()

	require.NoError(t, checkNotifyTmplSendable(ctx, fake, entity.TaskNotifyNone, "anyTmpl"))
	require.NoError(t, checkNotifyTmplSendable(ctx, fake, entity.TaskNotifyAlways, ""))
	require.NoError(t, checkNotifyTmplSendable(ctx, nil, entity.TaskNotifyAlways, "anyTmpl"))

	assert.Empty(t, fake.checked, "不通知 / 无模板 / 未注入模板服务时都不该去查模板")
}

// TestCheckNotifyTmplSendablePropagatesRejection 模板不可发时原样上抛：
// msg 侧已按请求语言译好文案，这里再包一层会把译文污染成中英混排
func TestCheckNotifyTmplSendablePropagatesRejection(t *testing.T) {
	fake := &fakeMsgTmplApp{err: tmplReject}

	err := checkNotifyTmplSendable(context.Background(), fake, entity.TaskNotifyAlways, "broken")
	require.ErrorIs(t, err, tmplReject)
	assert.Equal(t, []string{"broken"}, fake.checked)
}

// TestSaveTaskRejectsUnsendableNotifyTmpl 任务保存期就得拒掉发不出去的模板，且不留半条记录。
//
// 真机复现的原始缺陷面：模板存在但未关联渠道时保存「操作成功」，直到运行收尾才在
// 服务端日志里留一条 ERROR，配置人在界面上得不到任何信号
func TestSaveTaskRejectsUnsendableNotifyTmpl(t *testing.T) {
	_, taskSvc, _ := setupTriggerSvc(t)
	fake := &fakeMsgTmplApp{err: tmplReject}
	taskSvc.msgTmplApp = fake

	task := notifyTask("broken")
	err := taskSvc.SaveTask(runTestCtx(), task)

	require.ErrorIs(t, err, tmplReject, "模板发不出去却保存成功，等于把失败推迟成运行收尾的一条服务端日志")
	assert.Equal(t, []string{"broken"}, fake.checked)
	assert.Zero(t, task.Id, "被拒的保存不该留下记录")
}

// TestSaveTaskAcceptsSendableNotifyTmpl 模板可发时照常保存（校验不得误伤正常配置）
func TestSaveTaskAcceptsSendableNotifyTmpl(t *testing.T) {
	_, taskSvc, _ := setupTriggerSvc(t)
	fake := &fakeMsgTmplApp{}
	taskSvc.msgTmplApp = fake

	task := notifyTask("agentopsDiagnosisResult")
	require.NoError(t, taskSvc.SaveTask(runTestCtx(), task))
	assert.NotZero(t, task.Id)
	assert.Equal(t, []string{"agentopsDiagnosisResult"}, fake.checked)
}

// TestSaveTriggerRejectsUnusableInlineNotifyConfig 内联触发器的结果通知配置：
// 缺模板与模板不可发都必须在保存期拒绝。
//
// 「缺模板」过去只有前端 required 兜着，后端不校验——而 webhook/告警派发是无人值守链路，
// 绕过界面直接调 API 就能存下一个「开了通知却没有模板」的触发器，事件来了 Run 也跑成了，
// 通知静默丢失（notifyResult 对空模板是直接 return，连日志都不留）
func TestSaveTriggerRejectsUnusableInlineNotifyConfig(t *testing.T) {
	tests := []struct {
		name     string
		tmplCode string
		tmplErr  error
		wantErr  error
	}{
		{name: "开了通知却没填模板", tmplCode: "", wantErr: nil},
		{name: "模板发不出去", tmplCode: "broken", tmplErr: tmplReject, wantErr: tmplReject},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			triggerSvc, _, _ := setupTriggerSvc(t)
			fake := &fakeMsgTmplApp{err: tt.tmplErr}
			triggerSvc.msgTmplApp = fake

			tr := inlineTrigger(tt.tmplCode)
			err := triggerSvc.SaveTrigger(runTestCtx(), tr)

			require.Error(t, err, "内联通知配置不可用时不该保存成功")
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			}
			assert.Zero(t, tr.Id, "被拒的保存不该留下记录")
		})
	}
}

// TestSaveTriggerAcceptsSendableInlineNotifyConfig 内联通知配置合法时照常保存
func TestSaveTriggerAcceptsSendableInlineNotifyConfig(t *testing.T) {
	triggerSvc, _, _ := setupTriggerSvc(t)
	fake := &fakeMsgTmplApp{}
	triggerSvc.msgTmplApp = fake

	tr := inlineTrigger("agentopsDiagnosisResult")
	require.NoError(t, triggerSvc.SaveTrigger(runTestCtx(), tr))
	assert.NotZero(t, tr.Id)
	assert.Equal(t, []string{"agentopsDiagnosisResult"}, fake.checked)
}

// TestSaveTriggerSkipsNotifyCheckWhenTaskBound 关联任务模式下，触发器行里残留的通知配置
// 既不生效也不该校验。
//
// resolveRunConfig 在 task_id>0 时用任务的 notify* 覆盖触发器自带的值，前端也在绑定任务时
// 隐藏这两个字段。若仍按触发器行里的旧值校验，就会出现「改一个与通知无关的字段，却被一条
// 早已不生效的残留配置拦下」——错误无法归因，用户也无从修正（界面上根本没有那个输入框）。
// 任务侧配置的可发性由 SaveTask 自己负责
func TestSaveTriggerSkipsNotifyCheckWhenTaskBound(t *testing.T) {
	triggerSvc, taskSvc, _ := setupTriggerSvc(t)
	fake := &fakeMsgTmplApp{err: tmplReject}
	triggerSvc.msgTmplApp = fake
	taskSvc.msgTmplApp = fake

	// 任务自身「不通知」，保存必然通过，也就不会污染 fake.checked
	bound := notifyTask("")
	bound.NotifyType = entity.TaskNotifyNone
	require.NoError(t, taskSvc.SaveTask(runTestCtx(), bound))
	require.NotZero(t, bound.Id)

	tr := inlineTrigger("staleTmplFromInlineDays")
	tr.TaskId = bound.Id
	require.NoError(t, triggerSvc.SaveTrigger(runTestCtx(), tr), "关联任务时不该按触发器行里的残留通知配置拒绝保存")
	assert.Empty(t, fake.checked, "关联任务模式不该触发模板可发性校验")
}
