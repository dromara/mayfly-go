// Package sqldraft 自然语言 SQL 草稿：把一句描述变成可核对的 SQL 文本。
//
// 定位：产物是**草稿**，不执行、不放行。写语句仍要回到既有执行链（触发策略 / 工单审批 /
// 表列权限），本包不提供任何执行入口——「AI 直接执行」这条路刻意不存在。
//
// 管线（每一段都是可替换接缝，新增意图/召回/片段/校验只加一行注册，不改管线）：
//
//	intent 注册表 → 表召回（提议阶段：表名匹配 / 注释词面 / 模型选择题；
//	                      扩展阶段：外键邻接 / 同族表补齐）
//	→ 结构快照（按表列权限裁剪，列与外键按需 ensure）→ 片段按预算装配
//	→ 一次生成（textgen 中立端口）→ 契约解析 → 校验（解析 / 标识符 / 策略 / 权限 / 计划）
//	→ 有界自纠（结构化 Finding 回喂）→ 结论（verified / unverified / invalid）
//
// 三条红线（改动时必须保持）：
//  1. 生成永不执行；写语句的「要不要审批」由 CheckSqlsWithoutTicket 的权威事实决定，
//     不采信模型自评（模型返回的 confidence/审批建议一律丢弃）；
//  2. 当前账号无权的表列既不进上下文，也不出现在任何提示文案里（ACL 拒绝原因收敛为通用码）；
//  3. 上下文超预算时如实告知「被裁剪」，不静默降级成一份看起来完整的结构。
package sqldraft

import (
	"context"
	"time"

	"mayfly-go/internal/ai/pkg/textgen"
)

// 生成约束的默认值。
//
// 刻意不做成系统配置项：这些属于「算法参数」而非运维开关，暴露给管理员只会多一个
// 没人会调的旋钮；真正需要运营侧控制的是模型配置本身与入口权限码。
const (
	// maxTables 进入上下文的表数上限（召回结果按分截断）
	maxTables = 12
	// maxTableCards 发给模型做「选择题」的候选表上限。
	//
	// 200 张表约 2k+ 字符，光把清单读完就要额外延迟；改成「先按描述相关性预筛，再截断」，
	// 上限收到 80——真机上描述相关表几乎总在前几十张内
	maxTableCards = 80
	// maxRepairRounds 自纠轮数上限（只回喂结构化 Finding，不做无界重试）
	maxRepairRounds = 1
	// candidateTimeout 候选补采的单次超时。
	//
	// 刻意短于 callTimeout：补采是「同一道题再要一份答案」，一份慢答案不如不要——
	// 网关拥塞时再等一个 55s 只会把整体体感拖到分钟级，而首轮结果已经在手上
	candidateTimeout = 20 * time.Second
	// maxOutputTokens 单次生成的输出上限
	maxOutputTokens = 1200
	// contextTokenBudget 上下文片段总预算（按 token 估算裁剪，超限丢低优先片段）
	contextTokenBudget = 6000
	// callTimeout 单次生成调用超时。
	//
	// 必须小于模型配置里的客户端超时（默认 60s）：超 60s 时先触发的是 HTTP 客户端超时，
	// 用户看到的就是一句 Go/HTTP 原文。压在 55s 让"我们的超时"稳定先到，从而能给出
	// 本地化、可执行的提示（真机实测：复杂需求 + 思考模式下 60s 客户端超时会真实发生）。
	callTimeout = 55 * time.Second
	// chooseTimeout 表选择题的超时。
	//
	// 取值权衡：太小（30s）会在网关抖动时先超时，召回退化后模型硬编垃圾 SQL；
	// 太大（45s）用户就是干等。实测正常在 4–8s，25s 只作极端兜底。
	chooseTimeout = 25 * time.Second
	// confidentScore 确定性召回的「足够可信」门槛：低于它就必须再问一次模型做选择题
	confidentScore = 8
	// maxSiblingTables 同族表补齐的上限：同后缀表可能有十几张（*_log 之类），
	// 全带进来会把上下文预算吃光；三张足以让模型对照列结构做选择
	maxSiblingTables = 3
	// maxColumnPreviewTables 选表选择题里「补列名预览」的卡片数上限。
	//
	// 只对注释为空的卡片补：注释齐全时预览是纯开销。上限压住召回阶段的元数据往返，
	// 而排序已把最相关的表排在前面，前若干张正是最需要区分的
	maxColumnPreviewTables = 16
	// maxColumnPreviewCols 每张预览卡片列出的列名数
	maxColumnPreviewCols = 12
	// maxMaterialRunes 单次请求的材料总长上限（描述 + 选区 + 光标语句）。
	//
	// contextTokenBudget 只管装配后的上下文片段，管不到任务正文：用户把整个脚本
	// 当选区提交时，正文会原样进提示词，既拖慢也挤爆输出预算。超限直接拒绝并说明，
	// 不静默截断——截断后的需求会生成一份看起来正常、实际答非所问的草稿
	maxMaterialRunes = 8000
)

// Req SQL 草稿请求
type Req struct {
	// DbId 数据源 id
	DbId uint64
	// Db 库名（空表示用资产默认库，由连接层解析）
	Db string
	// Intent 意图 id（见 intent.go 的注册清单）
	Intent string
	// Prompt 用户描述（自然语言）
	Prompt string
	// Selection 编辑器选区原文（可为空）
	Selection string
	// AnchorStatement 光标所在完整语句（explain/fix 意图的材料来源）
	AnchorStatement string
	// RefineOf 上一版生成的语句（多轮细化）：非空时本次需求按「在它基础上追加」理解。
	//
	// 它与 AnchorStatement 不是一回事：后者是编辑器里已存在的 SQL（用户写的），
	// 前者是本对话框上一轮产物（模型写的）。混用一个字段会让「重新生成」与「继续细化」
	// 两种意图在管线里无法区分
	RefineOf string
	// IncludePlans 是否取执行计划做只读校验（失败只降级为提示，不影响草稿产出）
	IncludePlans bool
}

// 校验结论三态（由确定性校验推导，不采信模型自评）。
//
// 空串是第四态「无结论」：散文产物（explain）没有可校验的语句，前端据此不渲染标签
const (
	verificationVerified   = "verified"
	verificationUnverified = "unverified"
	verificationInvalid    = "invalid"
)

// Statement 一条生成的语句
type Statement struct {
	SQL         string   `json:"sql"`
	Notes       string   `json:"notes"`
	Assumptions []string `json:"assumptions,omitempty"`
}

// Severity 校验结论级别
type Severity string

const (
	// SeverityInvalid 致命：解析失败或引用了确定不存在的标识符，默认不写回编辑器
	SeverityInvalid Severity = "invalid"
	// SeverityUnverified 无法确认（授权拒绝 / 计划不可得 / CTE 无法静态核对），可写回但必须标出
	SeverityUnverified Severity = "unverified"
	// SeverityNote 提示（如「这条会走审批」），不影响可用性
	SeverityNote Severity = "note"
)

// 校验结论码。前端按码出文案，不解析 message 文本
const (
	CodeParseFailed       = "parse_failed"
	CodeUnknownTable      = "unknown_table"
	CodeUnknownColumn     = "unknown_column"
	CodeTableRestricted   = "table_restricted"
	CodeColumnRestricted  = "column_restricted"
	CodePolicyTicket      = "policy_ticket"
	CodeAclDenied         = "acl_denied"
	CodePlanUnavailable   = "plan_unavailable"
	CodePlanFullScan      = "plan_full_scan"
	CodeUncheckable       = "uncheckable"
	CodeWriteStatement    = "write_statement"
	CodeContextTruncated  = "context_truncated"
	CodeChooserTruncated  = "chooser_truncated"
	CodeValueNotSampled   = "value_not_sampled"
	CodeTargetUnconfirmed = "target_table_unconfirmed"
	CodeNoTableSource     = "no_table_source"
)

// 闸门结论字符串（与 sqlexec.PolicyBlockKind 的返回值一一对应；
// 这里按字符串消费，既不 import sqlexec 也不 import flow，依赖方向保持单向）
const (
	policyBlockApproval = "approval"
	policyBlockWarnAck  = "warn_ack"
)

// Finding 一条校验结论
type Finding struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	// Message 面向用户的说明（服务端已本地化）
	Message string `json:"message"`
	// Identifier 触发该项的标识符（受限标识符不回显，见 CodeColumnRestricted）
	Identifier string `json:"identifier,omitempty"`
	// Candidates 同库/同表里最相近的候选标识符（自纠与人工修正的依据）
	Candidates []string `json:"candidates,omitempty"`
}

// ContextTable 本次上下文里带上的表（界面作为「依据」展示，可跳转表结构）
type ContextTable struct {
	Name    string `json:"name"`
	Comment string `json:"comment,omitempty"`
	Columns int    `json:"columns"`
	Reason  string `json:"reason,omitempty"`
}

// Draft 草稿结果
type Draft struct {
	Intent string `json:"intent"`
	// Statements 生成结果（按出现顺序）。产物是散文的意图（explain）为空
	Statements []Statement `json:"statements"`
	// Explanation 散文产物（explain 意图）：解释类结果不是 SQL，不能塞进 statements
	Explanation string `json:"explanation,omitempty"`
	// Verification verified | unverified | invalid（由确定性校验推导，不采信模型自评）。
	// 散文产物（explain）为空串：没有可校验的语句，界面据此不显示校验标签
	Verification string `json:"verification"`
	// Checks 全部校验结论
	Checks []Finding `json:"checks"`
	// ContextTables 本次上下文里的表
	ContextTables []ContextTable `json:"contextTables"`
	// Warnings 上下文/召回层面的告知（裁剪、截断、示例丢弃等）
	Warnings []string `json:"warnings,omitempty"`
	// Usage 本轮 token 用量（多轮自纠累计；模型未上报时为 nil）
	Usage *textgen.Usage `json:"usage,omitempty"`
	// Repairs 实际发生的自纠轮数
	Repairs int `json:"repairs"`
	// Candidates 未通过校验时实际补采并通过解析的候选数（0 表示没补采）。
	//
	// 必须回传而不是只进日志：补采会多花一次模型调用，界面上不写清「为什么这次更慢/更贵」
	// 就等于把成本藏在用户看不见的地方
	Candidates int `json:"candidates"`
	// Dialect 目标方言（含版本）
	Dialect string `json:"dialect"`
}

// Service SQL 草稿应用服务
type Service interface {
	// Draft 生成一份草稿。模型未配置、库不可达、意图未知均返回错误，不做静默降级
	Draft(ctx context.Context, req *Req) (*Draft, error)
}
