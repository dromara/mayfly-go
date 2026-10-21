package entity

import (
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/jsonx"
)

// Extra 键单点：展示型辅助字段（关联账号名、备注、反馈备注）与「非查询维度的结构化附件」
// 都不落独立列，统一收敛在 model.ExtraData 的 JSON 容器里（判据见 docs/server/domain.md
// 「ExtraData 使用原则」：前端展示字段、关联名称、可选扩展信息用 ExtraData；
// 查询条件/排序/分组统计/索引字段才独立列）。
//
// 键名集中在本文件声明，避免各处写字面量字符串漂移；字段与 Extra 的双向同步由各实体上的
// BeforeSave / AfterFind 钩子完成（Extra 是唯一落库真源，VO 按名拷贝与 JSON 输出都走瞬态字段，
// 因此接口契约不变），业务代码不要直接拼这些键。
//
// 写入一律走 model.ExtraData.SetExtraValue：基座对零值（空串/false/0/nil）是删键，所以
// 「把备注改回空」这类编辑能真实落库；读侧 GetExtraString 对缺键返回空串，两者对外等价。
const (
	// ExtraKeyOwnerUsername 运行/归属账号名快照（列表与详情展示用；账号 id 才是查询维度，仍独立列）
	ExtraKeyOwnerUsername = "ownerUsername"
	// ExtraKeyRemark 备注（可选扩展信息）
	ExtraKeyRemark = "remark"
	// ExtraKeyFeedbackNote 运行反馈备注（随经验沉淀进记忆系统的说明文本）
	ExtraKeyFeedbackNote = "feedbackNote"
	// ExtraKeyInstructionSegments 任务/触发器指令里的芯片引用段（技能码、资源定位信息）。
	// 文本指令里的 `@机器 aliyun` 对模型不可直接取参（工具要的是 id/dbName/ip/port），
	// 故结构化段随指令一并保存，派发时展开为模型可直接取用的定位文本
	ExtraKeyInstructionSegments = "instructionSegments"
	// ExtraKeyNotifyType 内联触发器的结果通知渠道（0=不通知）。
	//
	// 为什么走 Extra 而不是像 AgentTask 那样开独立列：它从来不是查询/排序/统计维度，
	// 只在派发时被读一次传给 Enqueue；按 docs/server/domain.md 的 ExtraData 使用原则，
	// 这类字段不该进表结构（加列还要走迁移，风险与成本都高一个量级）。
	// 没有它，内联触发器（webhook/告警的主用法）派发的诊断运行结果永远无处可达
	ExtraKeyNotifyType = "notifyType"
	// ExtraKeyNotifyTmplCode 结果通知的消息模板码（语义同 ExtraKeyNotifyType）
	ExtraKeyNotifyTmplCode = "notifyTmplCode"
	// ExtraKeyRunRole 运行形态（编排的规划/步骤/审查、评测的重放、提案的执行/校验）。
	//
	// 为什么走 Extra 而不是把形态写进标题前缀（旧做法）：标题是展示数据，中文前缀落库后英文界面
	// 永远看到中文，且存量迁移曾拿 `title LIKE '[评测%'` 当分类谓词——改一次措辞就静默漏判。
	// 它也从来不是查询/统计维度（粗粒度归属由 trigger_type 独立列承担），按 docs/server/domain.md
	// 的 ExtraData 使用原则不该加列。
	//
	// 键名必须与 Run.RunRole 的 json tag 一致（前端按 runRole 读）：改常量标识符不会自动改字符串值，
	// 这类「改名漏改值」的漂移已由回填用例抓过
	ExtraKeyRunRole = "runRole"
)

// InstructionSegment 指令芯片引用段（技能 / 资源）。
//
// 与 ai/protocol.ContentSegment 同形但独立声明：domain 层不依赖其它模块的协议包，
// 转成协议段发生在应用层派发处（见 application.instructionText），芯片→文本的渲染器仍归 ai 所有
type InstructionSegment struct {
	// Type 段类型：input_text / skill / resource
	Type string `json:"type"`
	// Text 段文本（芯片为其标签，如 "db-ops-guide"、"aliyun"）
	Text string `json:"text"`
	// Extra 定位信息（skill: id；resource: resourceType/id/code/ip/port/db/username 等）
	Extra map[string]any `json:"extra,omitempty"`
}

// setInstructionSegments 写入指令芯片段：空列表等价于「没有引用」，删键而非留 `[]`
// （否则旧段会在指令已被改成纯文本后继续被展开，给模型喂一组已经不存在的引用）
func setInstructionSegments(e *model.ExtraData, segs []InstructionSegment) {
	if len(segs) == 0 {
		e.SetExtraValue(ExtraKeyInstructionSegments, nil)
		return
	}
	e.SetExtraValue(ExtraKeyInstructionSegments, segs)
}

// instructionSegments 读回指令芯片段；未配置或容器里是脏数据时返回 nil（按纯文本指令处理，
// 不能让一条坏数据把整次运行派发打挂）
func instructionSegments(e *model.ExtraData) []InstructionSegment {
	raw := e.GetExtraVal(ExtraKeyInstructionSegments)
	if raw == nil {
		return nil
	}
	parsed, err := jsonx.ToByStr[[]InstructionSegment](jsonx.ToStr(raw))
	if err != nil || parsed == nil {
		return nil
	}
	return *parsed
}
