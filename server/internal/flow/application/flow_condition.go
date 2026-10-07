package application

import (
	"context"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/trigger"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/utils/collx"

	"github.com/spf13/cast"
)

// FlowInstanceBizType 流程实例变量场景：连线跳转条件与用户任务完成条件的字段字典。
//
// 这两类判定与「资源操作要不要提工单」用的是同一个引擎（字段字典 + 条件树 + 操作符），
// 区别只在输入不是 SQL/命令而是引擎自己算好的流程变量，因此它 ConditionOnly：
// 不出现在触发策略编辑器的场景清单里，也不提供检查项与试算
const FlowInstanceBizType = "flow_instance"

// 流程变量字段 key。nrOfAll/nrOfCompleted 复用 node_usertask.go 里写入任务变量的常量，
// 保证「写入」与「被条件引用」两边永远同名
const (
	flowFieldNrOfAll           = NrOfAll
	flowFieldNrOfCompleted     = NrOfCompleted
	flowFieldNrOfCompletedRate = "nrOfCompletedRate"
	flowFieldApprovalResult    = "approvalResult"
	flowFieldBizType           = "bizType"
	flowFieldApplicant         = "applicant"
)

// approvalResult 的候选值。取值用符号而不是数据库里的数字状态码：
// 条件树会被存进流程定义 JSON 长期留存，状态码一旦调整历史流程就判错了
const (
	ApprovalResultCompleted  = "completed"  // 通过
	ApprovalResultProcessing = "processing" // 审批中
	ApprovalResultReject     = "reject"     // 拒绝
	ApprovalResultBack       = "back"       // 驳回
	ApprovalResultCanceled   = "canceled"   // 取消
)

// 数值字段可用的操作符：刻意不含「为空/非空」，数量型变量的 0 是有意义的取值
// （0 人已通过），按空值判定会把「还没有人审批」和「字段没提供」混成一谈
var flowCountOps = []entity.OpName{
	trigger.OpEq, trigger.OpNe, trigger.OpGt, trigger.OpGte, trigger.OpLt, trigger.OpLte, trigger.OpBetween,
}

func init() {
	RegisterTriggerBiz(trigger.BizMeta{
		BizType:          FlowInstanceBizType,
		Fields:           flowInstanceFields(),
		ConditionPresets: flowConditionPresets(),
		ConditionOnly:    true,
	})
}

func flowInstanceFields() []trigger.TriggerField {
	return []trigger.TriggerField{
		{Key: flowFieldNrOfAll, TitleKey: "flow.field.nrOfAll", Group: "approval", Type: trigger.TypeNumber, Ops: flowCountOps},
		{Key: flowFieldNrOfCompleted, TitleKey: "flow.field.nrOfCompleted", Group: "approval", Type: trigger.TypeNumber, Ops: flowCountOps},
		{
			// 完成比例是让「会签」可配置的关键：条件叶子只能拿字段和常量比较，
			// 「已完成 == 总数」这种两个字段之间的关系表达不出来，换成比例后
			// 会签是「比例 >= 1」、过半通过是「比例 >= 0.5」，都比旧实现更直白
			Key: flowFieldNrOfCompletedRate, TitleKey: "flow.field.nrOfCompletedRate", Group: "approval",
			Type: trigger.TypeNumber, Ops: flowCountOps,
			Resolve: resolveCompletedRate,
		},
		{
			Key: flowFieldApprovalResult, TitleKey: "flow.field.approvalResult", Group: "approval", Type: trigger.TypeEnum,
			Options: approvalResultOptions(),
			Resolve: resolveApprovalResult,
		},
		{Key: flowFieldBizType, TitleKey: "flow.field.bizType", Group: "instance", Type: trigger.TypeString},
		{Key: flowFieldApplicant, TitleKey: "flow.field.applicant", Group: "instance", Type: trigger.TypeString},
	}
}

// flowConditionPresets 固化审批模式的三种常见写法。
//
// 它们不再是 radio 背后的模板字符串：套用的结果就是一棵普通条件树，
// 管理员套完还能继续改（比如把会签的阈值调成「过半通过」），这是旧写法做不到的
func flowConditionPresets() []trigger.ConditionPreset {
	return []trigger.ConditionPreset{
		{
			Key: "orSign", TitleKey: "flow.orSign", DescriptionKey: "flow.orSignTip",
			RuleNode: &entity.RuleNode{Kind: entity.NodeKindCondition, Field: flowFieldNrOfCompleted, Op: trigger.OpGte, Value: 1},
		},
		{
			Key: "andSign", TitleKey: "flow.andSign", DescriptionKey: "flow.andSignTip",
			RuleNode: &entity.RuleNode{Kind: entity.NodeKindCondition, Field: flowFieldNrOfCompletedRate, Op: trigger.OpGte, Value: 1},
		},
		{
			Key: "majoritySign", TitleKey: "flow.majoritySign", DescriptionKey: "flow.majoritySignTip",
			RuleNode: &entity.RuleNode{Kind: entity.NodeKindCondition, Field: flowFieldNrOfCompletedRate, Op: trigger.OpGte, Value: 0.5},
		},
	}
}

func approvalResultOptions() []trigger.FieldOption {
	values := []string{
		ApprovalResultCompleted, ApprovalResultProcessing, ApprovalResultReject, ApprovalResultBack, ApprovalResultCanceled,
	}
	options := make([]trigger.FieldOption, 0, len(values))
	for _, value := range values {
		options = append(options, trigger.FieldOption{Value: value})
	}
	return options
}

// resolveCompletedRate 由「已完成 / 总数」派生完成比例。
//
// 总数缺失或为 0 时返回 nil（而不是 0）：0 会被读成「一个人都没通过」，
// 而这个场景真正的问题是「无从判断」，交给引擎记入 Unknown
func resolveCompletedRate(tc *trigger.Context) (any, error) {
	all, ok := flowVarNumber(tc, flowFieldNrOfAll)
	if !ok || all <= 0 {
		return nil, nil
	}
	completed, ok := flowVarNumber(tc, flowFieldNrOfCompleted)
	if !ok {
		return nil, nil
	}
	return completed / all, nil
}

// resolveApprovalResult 把任务状态码归一成符号取值，并把写入时已是符号的情况原样返回
func resolveApprovalResult(tc *trigger.Context) (any, error) {
	raw, found := flowVarValue(tc, flowFieldApprovalResult)
	if !found {
		return nil, nil
	}
	if text, ok := raw.(string); ok {
		return text, nil
	}
	status := entity.ProcinstTaskStatus(cast.ToInt8(raw))
	for _, item := range approvalResultMapping {
		if item.status == status {
			return item.value, nil
		}
	}
	// 认不出的状态码按「无法判定」处理：猜一个结果会让流程走错分支，
	// 而分支走错的代价是跳过审批
	return nil, nil
}

var approvalResultMapping = []struct {
	status entity.ProcinstTaskStatus
	value  string
}{
	{entity.ProcinstTaskStatusCompleted, ApprovalResultCompleted},
	{entity.ProcinstTaskStatusProcess, ApprovalResultProcessing},
	{entity.ProcinstTaskStatusReject, ApprovalResultReject},
	{entity.ProcinstTaskStatusBack, ApprovalResultBack},
	{entity.ProcinstTaskStatusCanceled, ApprovalResultCanceled},
}

// flowVarValue 读取一个流程变量。流程变量既可能来自任务/执行流/实例变量（结构化），
// 也可能由调用方作为原始输入给出，因此两个来源都要看
func flowVarValue(tc *trigger.Context, key string) (any, bool) {
	if value, ok := tc.RawValue(key); ok {
		return value, true
	}
	return tc.Attribute(key)
}

// flowVarNumber 读取数值型流程变量。变量经 JSON 存取后整数会变成 float64，
// 直接写任务变量时又是 int，两种形态都要能归一
func flowVarNumber(tc *trigger.Context, key string) (float64, bool) {
	value, found := flowVarValue(tc, key)
	if !found {
		return 0, false
	}
	number, err := cast.ToFloat64E(value)
	if err != nil {
		return 0, false
	}
	return number, true
}

// newFlowConditionContext 构造流程内条件的求值上下文。
//
// 与触发策略共用 Context，字段取值同样走三态（未提供 / 取到 / 取失败），
// 新增一类流程变量只需在字段字典里加一项，判定入口不用改
func newFlowConditionContext(ctx context.Context, vars collx.M, procdefId uint64) *trigger.Context {
	tc := trigger.NewContext(ctx, FlowInstanceBizType, procdefId, nil, contextx.GetLoginAccount(ctx)).WithAttributes(vars)
	if resolver := ruleSegmentResolver(ctx); resolver != nil {
		tc.WithSegments(resolver)
	}
	return tc
}

// ruleSegmentResolver 取可复用条件组的展开函数。
//
// 用 GetBeansByType 而不是 ioc.Get：条件组是可选能力，未注册（如单测、精简部署）时
// 返回 nil 让引用它的条件按「无法判定」失败，而不是把整个流程操作打成 panic
func ruleSegmentResolver(ctx context.Context) trigger.SegmentResolver {
	apps := ioc.GetBeansByType[RuleSegment]()
	if len(apps) == 0 {
		return nil
	}
	return apps[0].Resolver(ctx)
}
