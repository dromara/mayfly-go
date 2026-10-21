package application

import (
	"context"

	"mayfly-go/internal/agentops/domain/entity"
	msgapp "mayfly-go/internal/msg/application"
)

// checkNotifyTmplSendable 结果通知模板的保存期「可发性」校验（任务与触发器共用）。
//
// 为什么在「模板非空」之外还要再加这一层（真机实测）：只校验非空拦不住「模板已停用」和
// 「模板未关联任何渠道」，而这两种配置在运行收尾时同样发不出去——notifyResult 只把失败写成
// 一条服务端 ERROR，界面上早已弹出「操作成功」。实测：告警触发器绑了 flow 的「工单审批通知」
// 模板（未关联渠道），alert→trigger→Run 全链成功，结果通知静默丢失，配置人得不到任何信号。
//
// 取向与本模块 validateDispatchProfile 一致：能在保存期拦住的配置错误，不留到派发期只写日志。
// tmplApp 为 nil（单测绕过 IOC 装配）、「不通知」或模板为空时放行——空模板由调用方各自的
// 「要通知必须有模板」校验负责，判据不在这里重复
func checkNotifyTmplSendable(ctx context.Context, tmplApp msgapp.MsgTmpl, notifyType int8, tmplCode string) error {
	if tmplApp == nil || notifyType == entity.TaskNotifyNone || tmplCode == "" {
		return nil
	}
	return tmplApp.CheckSendable(ctx, tmplCode)
}
