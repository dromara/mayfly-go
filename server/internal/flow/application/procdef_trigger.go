package application

import (
	"context"
	"mayfly-go/internal/flow/application/dto"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/trigger"
	"mayfly-go/internal/flow/imsg"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
)

// CheckTrigger 资源操作的前置触发策略校验：先按资源路径就近定位流程定义，再求值。
//
// 未绑定流程定义时返回 nil 结论表示无需处置；业务方据此决定自己的提示文案，
// 因此新增资源类型接入审批只需注册场景 + 调用本方法，无需重复实现定位与求值逻辑
func (p *procdefAppImpl) CheckTrigger(ctx context.Context, req *dto.TriggerRequest) (*trigger.Decision, error) {
	procdef := p.GetProcdefByCodePath(ctx, req.CodePaths...)
	if procdef == nil {
		return nil, nil
	}
	return p.CheckProcdefTrigger(ctx, procdef, req)
}

// CheckProcdefTrigger 针对已定位到的流程定义求值触发策略。
// 批量执行场景（如 SQL 脚本逐条判定）可复用已取到的流程定义，避免逐条重复查库
func (p *procdefAppImpl) CheckProcdefTrigger(ctx context.Context, procdef *entity.Procdef, req *dto.TriggerRequest) (*trigger.Decision, error) {
	meta, ok := trigger.BizMetaOf(req.BizType)
	if !ok {
		return nil, errorx.NewBizf("the flow biz type %q is not registered", req.BizType)
	}
	if procdef == nil {
		return nil, nil
	}
	// 首次真实校验时核对一次「声明的审批通道」与「已注册的业务回调」是否一致
	checkTriggerApprovalChannels()

	tc := trigger.NewContext(ctx, req.BizType, procdef.Id, req.Raw, contextx.GetLoginAccount(ctx)).
		WithAttributes(req.Attributes).WithCodePaths(req.CodePaths).
		WithSegments(ruleSegmentResolver(ctx))
	decision := evaluateTriggerPolicy(ctx, procdef.TriggerPolicy, procdef.Id, procdef.Name, tc)
	logTriggerDiagnostics(ctx, meta.BizType, tc)
	logTriggerWarnings(ctx, decision)
	return decision, nil
}

// logTriggerWarnings 提醒级结论不阻断执行，但必须落日志：
// 业务入口只按「需审批 / 禁止」决定放行，若不在此留痕，管理员配的「仅提醒」就成了无出口的静默配置
func logTriggerWarnings(ctx context.Context, decision *trigger.Decision) {
	for _, finding := range decision.Findings {
		if finding.Severity != entity.SeverityWarning {
			continue
		}
		logx.WarnfContext(ctx, "flow trigger: procdef=%d rule=%s matched, warned only (not blocking) by policy", decision.ProcdefId, finding.Source)
	}
}

// SimulateTrigger 试算触发策略：只针对请求里带来的策略草稿求值，不落库也不回读已保存策略。
//
// 试算的用途是「边改边看这份草稿会怎么判」，因此刻意不接受 procdefId 换已保存策略这条路：
// 那既会让管理员看到别人配置的策略内容，也和草稿试算的真实意图不符
func (p *procdefAppImpl) SimulateTrigger(ctx context.Context, req *dto.SimulateTrigger) (*dto.SimulateResult, error) {
	if req.Policy == nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrTriggerPolicyRequired)
	}
	// 试算必须与保存走同一份校验：条件组引用不解析就会把一份本来配错的策略判成「可以试算」，
	// 到真实执行时才炸
	if err := trigger.ValidatePolicy(req.Policy, trigger.WithSegmentResolver(ruleSegmentResolver(ctx))); err != nil {
		return nil, toPolicyInvalidError(ctx, err)
	}

	facts, err := resolveSimulateInput(ctx, req.BizType, req.Raw)
	if err != nil {
		return nil, err
	}

	// 与 CheckProcdefTrigger 同构：Raw、派生事实、资源路径三者缺一不可，
	// 少一样就会有「试算说没事、真实执行被拦」的判定漂移
	tc := trigger.NewContext(ctx, facts.Meta.BizType, 0, facts.Raw, contextx.GetLoginAccount(ctx)).
		WithAttributes(facts.Attributes).WithCodePaths(facts.CodePaths).WithSegments(ruleSegmentResolver(ctx))
	decision := trigger.Evaluate(ctx, req.Policy, tc)
	// 字段字典全量解析也会补出「解析不出取值」的字段，Unknown 必须在这之后再取，
	// 否则这批字段不会进试算的「未能判定」清单，管理员只看到一个没有依据的结论
	fields := resolveAllFields(facts.Meta, tc)
	decision.Unknown = tc.Unknown
	return &dto.SimulateResult{Decision: decision, Fields: fields}, nil
}

// resolveAllFields 试算时把该场景字段字典全部解析一遍。
//
// 求值本身只需要被条件引用到的字段（其余按惰性跳过以省算力），但试算面板要回答的是
// 「引擎把我粘贴的这条输入理解成了什么」：只列出被树引用的派生字段，管理员就看不到
// 命令名、目标表这类被检查项内部消费的解析结果，无从判断结论为何与预期不符
func resolveAllFields(meta trigger.BizMeta, tc *trigger.Context) map[string]any {
	fields := tc.ResolvedFields()
	for _, field := range meta.Fields {
		value, found, err := tc.FieldValue(field)
		if err != nil || !found {
			continue
		}
		fields[field.Key] = value
	}
	return fields
}

// PolicySchema 把触发引擎注册表映射为对外 schema。只暴露事实（key/类型/候选值/参数边界），
// 不携带求值函数与文案，展示文案由前端 i18n 按 key 承载。
//
// 一次返回全部场景：触发策略编辑器取 Scenarios，流程图里的条件编辑器取 ConditionScenarios，
// 两者共用同一个请求，不需要按场景再发一次
func (p *procdefAppImpl) PolicySchema(ctx context.Context) *dto.PolicySchema {
	metas := trigger.BizMetas()
	segments := p.policySegments(ctx)
	schema := &dto.PolicySchema{
		Scenarios:          make([]*dto.PolicyScenario, 0, len(metas)),
		ConditionScenarios: make([]*dto.PolicyScenario, 0),
		GovernPaths:        trigger.DeclaredGovernPaths(),
	}
	for _, meta := range metas {
		scenario := &dto.PolicyScenario{
			BizType:            meta.BizType,
			Severities:         trigger.AvailableSeverities(meta.BizType),
			FailClosedSeverity: trigger.FailClosedSeverity(meta.BizType),
			Fields:             make([]*dto.PolicyField, 0, len(meta.Fields)),
			Checks:             make([]*dto.PolicyCheck, 0),
			Presets:            meta.Presets,
			SimulateFields:     policySimulateInputs(meta.SimulateFields),
			ConditionPresets:   meta.ConditionPresets,
		}
		for _, field := range meta.Fields {
			scenario.Fields = append(scenario.Fields, policyFieldSchema(field))
		}
		for _, check := range trigger.ChecksOf(meta.BizType) {
			scenario.Checks = append(scenario.Checks, policyCheckSchema(check))
		}
		for _, segment := range segments[meta.BizType] {
			scenario.Segments = append(scenario.Segments, segment)
		}
		// 流程内部条件的字段字典单独一栏：触发策略编辑器遍历 Scenarios 时不会拿到一张空卡片
		if meta.ConditionOnly {
			schema.ConditionScenarios = append(schema.ConditionScenarios, scenario)
			continue
		}
		schema.Scenarios = append(schema.Scenarios, scenario)
	}
	return schema
}

// policySegments 按场景分组取可复用条件组的引用候选项。
// 一次 schema 请求只查一遍条件组表，而不是每个场景各查一次
func (p *procdefAppImpl) policySegments(ctx context.Context) map[string][]*dto.PolicySegment {
	grouped := map[string][]*dto.PolicySegment{}
	app := ioc.GetBeansByType[RuleSegment]()
	if len(app) == 0 {
		return grouped
	}
	segments, err := app[0].ListByCond(model.NewCond())
	if err != nil {
		logx.WarnfContext(ctx, "flow policy schema: failed to list reusable condition groups, the reference picker stays empty: %v", err)
		return grouped
	}
	for _, segment := range segments {
		grouped[segment.BizType] = append(grouped[segment.BizType], &dto.PolicySegment{
			Ref: segment.Ref, Name: segment.Name, Remark: segment.Remark,
		})
	}
	return grouped
}

// editorKeyOf 取字段类型对应的值编辑器标识，未注册的类型退回与类型同名的标识。
//
// numericOf 该类型的期望值是否按数值比较，与 editorKeyOf 一样只在这里解析一次。
//
// 参数侧与条件侧曾各自让前端按 `type` 推导控件与比较方式，等于在两边各存一份
// 「类型 -> 控件 / 是否数值」映射，新增一种字段类型就会出现「后端改了、前端还猜旧」的分叉
func numericOf(fieldType trigger.FieldType) bool {
	def, ok := trigger.TypeOf(fieldType)
	return ok && def.Numeric
}

// editorKeyOf 取字段类型对应的值编辑器标识，未注册的类型退回与类型同名的标识
func editorKeyOf(fieldType trigger.FieldType) string {
	if def, ok := trigger.TypeOf(fieldType); ok {
		return def.EditorKey
	}
	return string(fieldType)
}

// policySimulateInputs 给试算输入补上控件标识：场景显式声明的 EditorKey 优先
// （资源引用类输入靠它换成资源树选择），未声明的按条件字段同一套类型注册表推导
func policySimulateInputs(inputs []trigger.SimulateInput) []*dto.PolicySimulateInput {
	result := make([]*dto.PolicySimulateInput, 0, len(inputs))
	for _, input := range inputs {
		editorKey := input.EditorKey
		if editorKey == "" {
			editorKey = editorKeyOf(input.Type)
		}
		result = append(result, &dto.PolicySimulateInput{
			Key: input.Key, TitleKey: input.TitleKey, Type: input.Type,
			EditorKey: editorKey, NameKey: input.NameKey,
			PlaceholderKey: input.PlaceholderKey,
			Required:       input.Required, Multiline: input.Multiline,
		})
	}
	return result
}

func policyFieldSchema(field trigger.TriggerField) *dto.PolicyField {
	result := &dto.PolicyField{
		Key:       field.Key,
		TitleKey:  field.TitleKey,
		Group:     field.Group,
		Type:      field.Type,
		EditorKey: editorKeyOf(field.Type),
		Numeric:   numericOf(field.Type),
		Options:   make([]string, 0, len(field.Options)),
		Ops:       make([]*dto.PolicyOperator, 0),
	}
	for _, option := range field.Options {
		result.Options = append(result.Options, option.Value)
	}
	for _, opName := range field.AvailableOps() {
		def, ok := trigger.OpOf(opName)
		if !ok {
			// 字段声明了未注册的操作符，不降级静默下发，避免前端渲染出无法求值的条件
			continue
		}
		result.Ops = append(result.Ops, &dto.PolicyOperator{Name: def.Name, LabelKey: def.LabelKey, ValueKind: def.ValueKind})
	}
	return result
}

func policyCheckSchema(check trigger.CheckDef) *dto.PolicyCheck {
	result := &dto.PolicyCheck{
		Key:            check.Key,
		TitleKey:       check.TitleKey,
		DescriptionKey: check.DescriptionKey,
		Default:        check.Default,
	}
	for _, param := range check.Params {
		options := make([]string, 0, len(param.Options))
		for _, option := range param.Options {
			options = append(options, option.Value)
		}
		result.Params = append(result.Params, &dto.PolicyCheckParam{
			Key:       param.Key,
			TitleKey:  param.TitleKey,
			Type:      param.Type,
			EditorKey: editorKeyOf(param.Type),
			Numeric:   numericOf(param.Type),
			Default:   param.Default,
			Min:       param.Min,
			Max:       param.Max,
			Options:   options,
			Required:  param.Required,
		})
	}
	return result
}

// resolveSimulateInput 用场景注册的解析能力把管理员粘贴的原始输入补全为求值所需的字段取值
func resolveSimulateInput(ctx context.Context, bizType string, raw map[string]string) (*simulateBundle, error) {
	meta, ok := trigger.BizMetaOf(bizType)
	if !ok {
		return nil, errorx.NewBizf("the flow biz type %q is not registered", bizType)
	}
	if meta.Simulate == nil {
		return &simulateBundle{Meta: meta, Raw: raw}, nil
	}
	facts, err := meta.Simulate(ctx, raw)
	if err != nil {
		return nil, errorx.NewBizf("failed to parse the simulated input: %s", err.Error())
	}
	if facts == nil {
		facts = &trigger.SimulatedFacts{}
	}
	if len(facts.Raw) == 0 {
		facts.Raw = raw
	}
	return &simulateBundle{Meta: meta, Raw: facts.Raw, Attributes: facts.Attributes, CodePaths: facts.CodePaths}, nil
}

// simulateBundle 试算一次求值所需的完整上下文：场景元数据 + 原始输入 + 派生事实 + 资源路径。
// 与真实执行路径 CheckProcdefTrigger 喂给上下文的内容保持同一形状
type simulateBundle struct {
	Meta       trigger.BizMeta
	Raw        map[string]string
	Attributes map[string]any
	CodePaths  []string
}

// evaluateTriggerPolicy 求值并把生效流程定义回填到结论，供拦截提示与审计定位来源
func evaluateTriggerPolicy(ctx context.Context, policy *entity.TriggerPolicy, procdefId uint64, procdefName string, tc *trigger.Context) *trigger.Decision {
	decision := trigger.Evaluate(ctx, policy, tc)
	decision.ProcdefId = procdefId
	decision.ProcdefName = procdefName
	return decision
}

// logTriggerDiagnostics 字段取值失败必须留痕，否则「策略没生效」和「策略判定为不风险」无法区分
func logTriggerDiagnostics(ctx context.Context, bizType string, tc *trigger.Context) {
	if len(tc.Unknown) > 0 {
		logx.WarnfContext(ctx, "flow trigger: bizType=%s could not resolve fields=%v, the related conditions were treated as not matched", bizType, tc.Unknown)
	}
}

// toPolicyInvalidError 把校验失败转换为带定位信息的业务错误
func toPolicyInvalidError(ctx context.Context, err error) error {
	return errorx.NewBizI(ctx, imsg.ErrTriggerPolicyInvalid, "source", errorSource(err), "reason", errorReason(ctx, err))
}

// errorSource 定位串：结构化错误自带，其它错误统一归到 policy
func errorSource(err error) string {
	if policyErr := trigger.AsError(err); policyErr != nil {
		return policyErr.Source
	}
	return "policy"
}

// errorReason 原因文本按请求语言渲染；非结构化错误（如实体解析失败）原样带出，
// 它们含技术细节，本就不该被翻译掉
func errorReason(ctx context.Context, err error) string {
	if policyErr := trigger.AsError(err); policyErr != nil {
		return policyErr.Reason(ctx)
	}
	return err.Error()
}
