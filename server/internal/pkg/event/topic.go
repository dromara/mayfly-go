package event

const (
	EventTopicResourceOp     = "resource:op"      // 资源操作主题
	EventTopicBizMsgTmplSend = "biz:msgtmpl:send" // 发送业务关联的消息模板
	EventTopicMsgTmplSend    = "msgtmpl:send"     // 发送消息模板
	EventTopicOpsTrigger     = "ops:trigger"      // 中立运维触发事件（告警/webhook → 事件驱动 Agent 诊断）
)
