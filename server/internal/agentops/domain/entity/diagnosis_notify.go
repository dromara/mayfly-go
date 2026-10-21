package entity

import "strings"

// 诊断结果通知的参数键：notifyResult 下发、内置模板占位符引用，两处必须同源。
//
// 存在动机（真机实测）：agentops 触发器/任务收尾按 notifyTmplCode 发结果通知，参数是诊断专用的
// taskName/runNo/status/... ；而新装库唯一的模板是 flow 的「工单审批通知」（占位符 creator/
// procdefName 等），选它渲染出来全是 <no value>。根因是模板与参数键不同源，故把键与模板收在一处，
// 模板占位符全部由这些常量拼出——增删通知参数只改常量，模板随之变化，不会与 notifyResult 漂移
const (
	DiagNotifyKeyTaskName    = "taskName"
	DiagNotifyKeyRunNo       = "runNo"
	DiagNotifyKeyTriggerType = "triggerType"
	DiagNotifyKeyStatus      = "status"
	DiagNotifyKeyResult      = "result"
	DiagNotifyKeyError       = "error"
	DiagNotifyKeyTotalTokens = "totalTokens"
)

// DiagNotifyParamKeys 诊断结果通知的全部参数键。内置模板必须逐一覆盖，否则对应占位符渲染成
// <no value>（由 diagnosis_notify_test 断言锁死）
var DiagNotifyParamKeys = []string{
	DiagNotifyKeyTaskName, DiagNotifyKeyRunNo, DiagNotifyKeyTriggerType,
	DiagNotifyKeyStatus, DiagNotifyKeyResult, DiagNotifyKeyError, DiagNotifyKeyTotalTokens,
}

// DiagnosisNotifyTmpl 内置「运维诊断结果通知」模板文本（迁移 seed 与占位符校验共用同一真源）。
// error 为空时不渲染「失败原因」行（Go text/template 的 if 对空串为假）
var DiagnosisNotifyTmpl = strings.Join([]string{
	"【运维诊断结果】",
	"任务：{{." + DiagNotifyKeyTaskName + "}}",
	"状态：{{." + DiagNotifyKeyStatus + "}}",
	"运行号：{{." + DiagNotifyKeyRunNo + "}}",
	"触发方式：{{." + DiagNotifyKeyTriggerType + "}}",
	"诊断结论：{{." + DiagNotifyKeyResult + "}}",
	"{{if ." + DiagNotifyKeyError + "}}失败原因：{{." + DiagNotifyKeyError + "}}\n{{end}}消耗Tokens：{{." + DiagNotifyKeyTotalTokens + "}}",
}, "\n")

// NewDiagnosisNotifyParams 构造诊断结果通知参数（notifyResult 专用，键与模板同源）
func NewDiagnosisNotifyParams(taskName, runNo, triggerType, status, result, errMsg string, totalTokens int64) map[string]any {
	return map[string]any{
		DiagNotifyKeyTaskName:    taskName,
		DiagNotifyKeyRunNo:       runNo,
		DiagNotifyKeyTriggerType: triggerType,
		DiagNotifyKeyStatus:      status,
		DiagNotifyKeyResult:      result,
		DiagNotifyKeyError:       errMsg,
		DiagNotifyKeyTotalTokens: totalTokens,
	}
}
