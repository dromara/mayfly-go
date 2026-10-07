package trigger

import (
	"context"
	"errors"
	"fmt"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/imsg"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/utils/collx"
	"strconv"
	"strings"
)

// Error 策略校验失败。
//
// Source 定位到具体规则供界面高亮，MsgId + Attrs 是分类后的原因：
// 直接把 Go 的 fmt 文本丢给前端会让中文界面出现半中半英的句子，
// 所以原因也走 i18n，渲染时按请求语言出文案
type Error struct {
	// Source 出错的规则定位，如 check[db.dml-without-where].params.rows
	Source string
	// MsgId 原因的消息 id
	MsgId i18n.MsgId
	// Attrs 原因的插值参数（字段名、取值、上下界等）
	Attrs collx.M
}

// Reason 按上下文语言渲染原因文本
func (e *Error) Reason(ctx context.Context) string {
	if e.MsgId == 0 {
		return i18n.TC(ctx, imsg.ReasonUnknownKind)
	}
	attrs := make([]any, 0, len(e.Attrs)*2)
	for key, value := range e.Attrs {
		attrs = append(attrs, key, value)
	}
	return i18n.TC(ctx, e.MsgId, attrs...)
}

func (e *Error) Error() string {
	return fmt.Sprintf("trigger policy is invalid at %s: %s", e.Source, e.Reason(context.Background()))
}

// invalid 生成一条带定位与国际化原因的校验错误。
// attrs 用 k, v, k, v 交替传入，既作为 i18n 插值参数也用于日志定位
func invalid(source string, msgId i18n.MsgId, attrs ...any) *Error {
	values := collx.M{}
	for index := 0; index+1 < len(attrs); index += 2 {
		values[attrs[index].(string)] = attrs[index+1]
	}
	return &Error{Source: source, MsgId: msgId, Attrs: values}
}

// AsError 取出结构化的校验/求值错误，非该类型时返回 nil。
//
// 应用层渲染与测试断言都按原因分类（MsgId）而不是比对文案：
// 断言英文句子的测试会在改文案时假失败，比对分类才稳
func AsError(err error) *Error {
	var policyErr *Error
	if errors.As(err, &policyErr) {
		return policyErr
	}
	return nil
}

// ValidateOption 保存校验的可选项
type ValidateOption func(*validation)

// WithFallbackScenarios 声明这份策略将要治理的场景，用于校验兜底级别能不能落地。
//
// 不传时跳过该项校验：校验器本身不知道流程定义绑了哪些资源，
// 把「谁治理什么」交给调用方给出，策略域才不用反向依赖标签域
func WithFallbackScenarios(bizTypes []string) ValidateOption {
	return func(val *validation) { val.fallbackScenarios = bizTypes }
}

// WithSegmentResolver 提供可复用条件组的取数函数。
//
// 条件树里出现 segment 引用时必须提供，否则只能校验到「引用了一个标识」这一层，
// 无法确认它存在、字段字典是否匹配、是否会循环展开
// WithBoundPaths 声明这份策略将要治理的资源路径，用于校验「配置的场景是否真能命中」。
//
// 不传即跳过：校验器不该反向依赖标签域，路径由调用方从绑定关系里剥出来
func WithBoundPaths(paths [][]int8) ValidateOption {
	return func(v *validation) { v.boundPaths = paths }
}

func WithSegmentResolver(resolver SegmentResolver) ValidateOption {
	return func(v *validation) { v.segments = resolver }
}

// validation 一次校验所需的上下文：字段字典按场景不同，条件组按引用惰性展开
type validation struct {
	segments          SegmentResolver
	fallbackScenarios []string
	// boundPaths 流程定义实际绑定的资源路径（空表示尚未绑定，此时跳过命中性校验）
	boundPaths [][]int8
}

// ValidatePolicy 保存前的全量静态校验。
//
// 注册表只在这里参与判定：字段是否存在、操作符是否适用、参数是否越界、正则是否可编译都在落库前拦下，
// 求值路径因此无需再查注册表结构，也不会出现「配置错误静默退化成一律审批」
func ValidatePolicy(policy *entity.TriggerPolicy, options ...ValidateOption) error {
	if policy == nil {
		return nil
	}
	cfg := &validation{}
	for _, option := range options {
		option(cfg)
	}

	if policy.Version > entity.TriggerPolicyVersion {
		return invalid("version", imsg.ReasonPolicyVersion, "version", policy.Version, "support", entity.TriggerPolicyVersion)
	}
	if err := entity.SeverityEnum.Valid(policy.DefaultSeverity); err != nil {
		return invalid("defaultSeverity", imsg.ReasonSeverityInvalid)
	}
	// 兜底级别也要过能力位：规则级早就拦了这件事，兜底级漏拦会让「需审批」
	// 存进一个没有审批通道的场景，运行时既不拦也不提示（求值期另有收敛兜住，但那是最后一道防线）
	// 「直接放行」是兜底特有的合法取值（规则级不允许它），因此只在配了需要落地能力的级别时校验。
	//
	// 只要有一个治理场景能落地就放行：一份策略可以同时治理数据库（有审批通道）与机器命令（没有），
	// 管理员对数据库侧的「未命中规则也要审批」是合法意图，不能因为机器侧落不了地就整份拒存；
	// 机器侧由求值期按场景各自收敛（抬到该场景最严的可用级别）
	if policy.DefaultSeverity != entity.SeverityDisabled && len(cfg.fallbackScenarios) > 0 {
		honourable := 0
		for _, bizType := range cfg.fallbackScenarios {
			if err := validateSeverity("defaultSeverity", bizType, policy.DefaultSeverity); err == nil {
				honourable++
			}
		}
		if honourable == 0 {
			// 一个能落地的场景都没有：这份兜底存进去就是「看着配了策略、实际一条不拦」
			return invalid("defaultSeverity", imsg.ReasonSeverityNoChannel, "bizType", strings.Join(cfg.fallbackScenarios, ", "))
		}
	}

	// 配了规则的场景必须至少能被一条绑定路径命中。
	//
	// 策略编辑器会列出所有已注册场景，管理员给机器命令配好规则却只绑了数据库实例时，
	// 运行时永远解析不到那个场景：不拦住就等于「看着管住了，实际一条都没管」
	if err := validateGovernedScenarios(policy, cfg); err != nil {
		return err
	}

	if err := validateChecks(policy); err != nil {
		return err
	}
	return validateCustoms(policy, cfg)
}

// ValidateCondition 校验一棵独立使用的条件树（流程连线跳转条件、用户任务完成条件）。
//
// 与触发策略共用同一套字段字典、操作符与条件组校验：条件语义两处必须一致，
// 否则会出现「策略页能存、流程图存不进去」或反过来在求值时才报错的分叉
func ValidateCondition(bizType string, node *entity.RuleNode, options ...ValidateOption) error {
	cfg := &validation{}
	for _, option := range options {
		option(cfg)
	}
	count, depth := countNodes(node)
	if count > MaxNodeCount {
		return invalid("condition", imsg.ReasonNodeCountExceeded, "count", count, "limit", MaxNodeCount)
	}
	if depth > MaxNodeDepth {
		return invalid("condition", imsg.ReasonNodeDepthExceeded, "depth", depth, "limit", MaxNodeDepth)
	}
	return validateNode("condition", bizType, node, cfg, nil)
}

func validateChecks(policy *entity.TriggerPolicy) error {
	seen := make(map[string]bool, len(policy.Checks))
	for index, config := range policy.Checks {
		source := fmt.Sprintf("checks[%d]", index)
		if config == nil || config.Key == "" {
			return invalid(source, imsg.ReasonCheckKeyRequired)
		}
		source = fmt.Sprintf("checks[%s@%s]", config.Key, config.BizType)
		if config.BizType == "" {
			return invalid(source, imsg.ReasonCheckBizRequired)
		}
		if _, registered := BizMetaOf(config.BizType); !registered {
			return invalid(source, imsg.ReasonBizUnregistered, "bizType", config.BizType)
		}
		def, registered := CheckOf(config.Key)
		if !registered {
			return invalid(source, imsg.ReasonCheckUnregistered)
		}
		if !checkAppliesTo(&def, config.BizType) {
			return invalid(source, imsg.ReasonCheckNotForBiz, "bizType", config.BizType)
		}
		if err := validateSeverity(source+".severity", config.BizType, config.Severity); err != nil {
			return err
		}
		deduplicate := config.Key + "@" + config.BizType
		if seen[deduplicate] {
			return invalid(source, imsg.ReasonCheckDuplicated)
		}
		seen[deduplicate] = true

		if err := validateParams(source+".params", def.Params, config.Params); err != nil {
			return err
		}
	}
	return nil
}

func checkAppliesTo(def *CheckDef, bizType string) bool {
	if len(def.BizTypes) == 0 {
		return true
	}
	for _, support := range def.BizTypes {
		if support == bizType {
			return true
		}
	}
	return false
}

func validateCustoms(policy *entity.TriggerPolicy, cfg *validation) error {
	seen := make(map[string]bool, len(policy.Customs))
	for index, custom := range policy.Customs {
		source := fmt.Sprintf("customs[%d]", index)
		if custom == nil || custom.BizType == "" {
			return invalid(source, imsg.ReasonCheckBizRequired)
		}
		source = fmt.Sprintf("customs[%s]", custom.BizType)
		if _, registered := BizMetaOf(custom.BizType); !registered {
			return invalid(source, imsg.ReasonBizUnregistered, "bizType", custom.BizType)
		}
		if seen[custom.BizType] {
			return invalid(source, imsg.ReasonCustomDuplicated)
		}
		seen[custom.BizType] = true
		if custom.When == nil && custom.Unless == nil {
			return invalid(source, imsg.ReasonCustomRequired)
		}
		if err := validateSeverity(source+".severity", custom.BizType, custom.Severity); err != nil {
			return err
		}

		count, depth := countNodes(custom.When)
		itemCount, itemDepth := countNodes(custom.Unless)
		count += itemCount
		if itemDepth > depth {
			depth = itemDepth
		}
		if count > MaxNodeCount {
			return invalid(source, imsg.ReasonNodeCountExceeded, "count", count, "limit", MaxNodeCount)
		}
		if depth > MaxNodeDepth {
			return invalid(source, imsg.ReasonNodeDepthExceeded, "depth", depth, "limit", MaxNodeDepth)
		}

		if err := validateNode(source+".when", custom.BizType, custom.When, cfg, nil); err != nil {
			return err
		}
		if err := validateNode(source+".unless", custom.BizType, custom.Unless, cfg, nil); err != nil {
			return err
		}
	}
	return nil
}

// validateNode refs 记录本条判定路径上已展开的条件组标识，用于在保存时就挡住循环引用：
// 留到求值才发现的话，被拦下的会是线上的一次真实操作，而不是配置它的那个人
func validateNode(source, bizType string, node *entity.RuleNode, cfg *validation, refs []string) error {
	if node == nil {
		return nil
	}
	switch node.Kind {
	case entity.NodeKindGroup:
		return validateGroupNode(source, bizType, node, cfg, refs)
	case entity.NodeKindCondition:
		return validateConditionNode(source, bizType, node)
	case entity.NodeKindSegment:
		return validateSegmentNode(source, bizType, node, cfg, refs)
	default:
		return invalid(source, imsg.ReasonUnknownKind, "kind", node.Kind)
	}
}

func validateGroupNode(source, bizType string, node *entity.RuleNode, cfg *validation, refs []string) error {
	switch node.Logic {
	case entity.LogicAll, entity.LogicAny, entity.LogicNone:
	default:
		return invalid(source, imsg.ReasonUnknownLogic, "logic", node.Logic)
	}
	if len(node.Items) == 0 {
		return invalid(source, imsg.ReasonGroupEmpty)
	}
	for index, item := range node.Items {
		if err := validateNode(fmt.Sprintf("%s.items[%d]", source, index), bizType, item, cfg, refs); err != nil {
			return err
		}
	}
	return nil
}

func validateSegmentNode(source, bizType string, node *entity.RuleNode, cfg *validation, refs []string) error {
	if node.Ref == "" {
		return invalid(source, imsg.ReasonSegmentRefRequired)
	}
	for _, ref := range refs {
		if ref == node.Ref {
			return invalid(source, imsg.ReasonSegmentCycle, "ref", node.Ref)
		}
	}
	if len(refs) >= MaxSegmentDepth {
		return invalid(source, imsg.ReasonSegmentTooDeep, "limit", MaxSegmentDepth)
	}
	if cfg == nil || cfg.segments == nil {
		return invalid(source, imsg.ReasonSegmentNoResolver, "ref", node.Ref)
	}
	def, ok := cfg.segments(node.Ref)
	if !ok || def == nil {
		return invalid(source, imsg.ReasonSegmentUnregistered, "ref", node.Ref)
	}
	if def.BizType != bizType {
		return invalid(source, imsg.ReasonSegmentCrossBiz, "ref", node.Ref, "owner", def.BizType, "used", bizType)
	}
	return validateNode(source+".condition", bizType, def.Condition, cfg, append(refs, node.Ref))
}

func validateConditionNode(source, bizType string, node *entity.RuleNode) error {
	if node.Field == "" {
		return invalid(source, imsg.ReasonFieldRequired)
	}
	field, registered := FieldOf(bizType, node.Field)
	if !registered {
		return invalid(source, imsg.ReasonFieldUnregistered, "field", node.Field)
	}
	if node.Op == "" {
		return invalid(source, imsg.ReasonOperatorRequired)
	}
	op, registered := OpOf(node.Op)
	if !registered {
		return invalid(source, imsg.ReasonOperatorUnregistered, "op", node.Op)
	}
	if !op.appliesTo(field.Type) {
		return invalid(source, imsg.ReasonOperatorMismatch, "op", node.Op, "field", node.Field, "type", field.Type)
	}
	if !containsOp(field.AvailableOps(), node.Op) {
		return invalid(source, imsg.ReasonOperatorNotAllowed, "op", node.Op, "field", node.Field)
	}

	switch op.ValueKind {
	case ValueNone:
		return nil
	case ValueMulti:
		if len(toList(node.Value)) == 0 {
			return invalid(source+".value", imsg.ReasonValueMultiRequired, "op", node.Op)
		}
	case ValueRange:
		if _, _, ok := splitRange(node.Value); !ok {
			return invalid(source+".value", imsg.ReasonValueRangeRequired, "op", node.Op)
		}
	case ValueSingle:
		if isEmptyValue(node.Value) {
			return invalid(source+".value", imsg.ReasonValueRequired)
		}
	}

	if err := validateConditionValue(source+".value", field, op, node.Value); err != nil {
		return err
	}
	if op.ValidateValue != nil {
		if err := op.ValidateValue(node.Value); err != nil {
			return invalid(source+".value", imsg.ReasonRegexInvalid, "detail", err.Error())
		}
	}
	return nil
}

// validateConditionValue 按字段类型校验期望值形态。
//
// 关键约束：布尔字段的期望值必须是真布尔。求值时任一侧为布尔就会按布尔归一比较，
// 若允许在这里写 1，「1 == true」会让条件语义变得不可预期
func validateConditionValue(source string, field TriggerField, op OpDef, value any) error {
	if op.ValueKind == ValueNone {
		return nil
	}
	items := []any{value}
	if op.ValueKind == ValueRange {
		lower, upper, _ := splitRange(value)
		items = []any{lower, upper}
	} else if op.ValueKind == ValueMulti {
		items = toList(value)
	}

	for _, item := range items {
		// 数值型（含时间戳、时长）只看能否转成数值，判定依据来自类型注册表
		if typeNumeric(field.Type) {
			if _, ok := toFloat(item); !ok {
				return invalid(source, imsg.ReasonValueNumberExpected, "field", field.Key)
			}
			continue
		}
		switch field.Type {
		case TypeBool:
			if !isBool(item) {
				return invalid(source, imsg.ReasonValueBoolExpected, "field", field.Key)
			}
		case TypeEnum:
			if !containsOption(field.Options, item) {
				return invalid(source, imsg.ReasonValueEnumInvalid, "value", canonical(item), "field", field.Key)
			}
		}
	}
	return nil
}

// validateSeverity 级别必须落在该场景真正能落地的集合内：
// 「不处置」由「规则不出现在配置里」表达，「需审批」要求该场景有审批通过后的执行回调
func validateSeverity(source, bizType string, severity entity.Severity) error {
	for _, allowed := range AvailableSeverities(bizType) {
		if severity == allowed {
			return nil
		}
	}
	if severity == entity.SeverityRequired {
		return invalid(source, imsg.ReasonSeverityNoChannel, "bizType", bizType)
	}
	return invalid(source, imsg.ReasonSeverityNotConfig, "severity", severity)
}

func validateParams(source string, schema []CheckParam, params map[string]any) error {
	defined := make(map[string]*CheckParam, len(schema))
	for index := range schema {
		param := &schema[index]
		defined[param.Key] = param
	}

	for key := range params {
		if _, ok := defined[key]; !ok {
			return invalid(source+"."+key, imsg.ReasonParamUndeclared)
		}
	}
	for index := range schema {
		param := &schema[index]
		value, provided := params[param.Key]
		if !provided || value == nil {
			if param.Required {
				return invalid(source+"."+param.Key, imsg.ReasonParamRequired, "param", param.Key)
			}
			continue
		}
		if err := validateParamValue(source, param, value); err != nil {
			return err
		}
	}
	return nil
}

func validateParamValue(source string, param *CheckParam, value any) error {
	at := func(msgId i18n.MsgId, attrs ...any) error {
		return invalid(source+"."+param.Key, msgId, append([]any{"param", param.Key}, plainAttrs(attrs)...)...)
	}
	// 数值型参数统一按数值比较并套 min/max，时间戳与时长同理（注册表里就标着数值）
	if typeNumeric(param.Type) {
		number, ok := toFloat(value)
		if !ok {
			return at(imsg.ReasonParamNotNumber)
		}
		if param.Min != nil && number < *param.Min {
			return at(imsg.ReasonParamTooSmall, "min", *param.Min)
		}
		if param.Max != nil && number > *param.Max {
			return at(imsg.ReasonParamTooLarge, "max", *param.Max)
		}
		return nil
	}
	switch param.Type {
	case TypeBool:
		if !isBool(value) {
			return at(imsg.ReasonParamNotBool)
		}
	case TypeEnum:
		items := toList(value)
		if len(items) == 0 {
			return at(imsg.ReasonParamOptionRequired)
		}
		for _, item := range items {
			if !containsOption(param.Options, item) {
				return at(imsg.ReasonParamOptionInvalid, "value", canonical(item))
			}
		}
	case TypeString, TypeStringList, TypeNumberList:
		if len(toList(value)) == 0 {
			return at(imsg.ReasonParamValueRequired)
		}
	}
	return nil
}

// typeNumeric 该字段类型的期望值是否按数值比较（未注册的类型按非数值处理）
func typeNumeric(name FieldType) bool {
	def, ok := TypeOf(name)
	return ok && def.Numeric
}

func containsOption(options []FieldOption, value any) bool {
	for _, option := range options {
		if looseEq(option.Value, value) {
			return true
		}
	}
	return false
}

func containsOp(ops []entity.OpName, target entity.OpName) bool {
	for _, op := range ops {
		if op == target {
			return true
		}
	}
	return false
}

// plainAttrs 把插值参数里的数值转成十进制原文。
//
// 边界值直接传 float64 时模板按 %v 输出，1.048576e+06 这种写法管理员对着输入框
// 根本对不上自己填的数；整数同样走这条路径，避免出现两种数字风格
func plainAttrs(attrs []any) []any {
	converted := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		switch value := attr.(type) {
		case float64:
			converted = append(converted, strconv.FormatFloat(value, 'f', -1, 64))
		case float32:
			converted = append(converted, strconv.FormatFloat(float64(value), 'f', -1, 32))
		case int:
			converted = append(converted, strconv.Itoa(value))
		case int64:
			converted = append(converted, strconv.FormatInt(value, 10))
		default:
			converted = append(converted, attr)
		}
	}
	return converted
}

// validateGovernedScenarios 策略里出现过的场景，是否在该流程定义绑定的资源上真能命中
func validateGovernedScenarios(policy *entity.TriggerPolicy, cfg *validation) error {
	if len(cfg.boundPaths) == 0 {
		return nil
	}
	bizTypes := make([]string, 0, len(policy.Checks)+len(policy.Customs))
	for _, config := range policy.Checks {
		if config != nil {
			bizTypes = append(bizTypes, config.BizType)
		}
	}
	for _, custom := range policy.Customs {
		if custom != nil {
			bizTypes = append(bizTypes, custom.BizType)
		}
	}
	unmatched := make([]string, 0, len(bizTypes))
	seen := make(map[string]bool, len(bizTypes))
	for _, bizType := range bizTypes {
		if seen[bizType] {
			continue
		}
		seen[bizType] = true
		if !ScenarioGoverned(bizType, cfg.boundPaths) {
			unmatched = append(unmatched, bizType)
		}
	}
	if len(unmatched) == 0 {
		return nil
	}
	return invalid("bizType", imsg.ReasonScenarioNotGoverned, "bizType", strings.Join(unmatched, ", "))
}
