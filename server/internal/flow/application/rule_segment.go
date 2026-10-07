package application

import (
	"context"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/internal/flow/domain/trigger"
	"mayfly-go/internal/flow/imsg"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/jsonx"
	"regexp"
	"strings"
)

// RuleSegment 可复用条件组：把一段判断（如「DBA 角色成员」「工作时段之外」）单独维护，
// 供触发策略与流程内的跳转/完成条件按标识引用。
//
// 只存条件树、不存处置级别：同一个判断在不同规则里可以定不同级别，级别属于「谁引用了它」
type RuleSegment interface {
	base.App[*entity.RuleSegment]

	GetPageList(condition *entity.RuleSegmentQuery) (*model.PageResult[*entity.RuleSegment], error)

	SaveRuleSegment(ctx context.Context, segment *entity.RuleSegment) error

	// DeleteRuleSegment 删除条件组。被任何规则引用时拒绝：
	// 留下悬空引用等于让一批策略在运行时集体判不出结果，那时已经有人在被误拦或误放
	DeleteRuleSegment(ctx context.Context, id uint64) error

	// Resolver 返回按标识展开条件组的取数函数。
	// 单次求值内结果就地缓存，避免同一条件组在一棵条件树里被反复查库
	Resolver(ctx context.Context) trigger.SegmentResolver
}

type ruleSegmentAppImpl struct {
	base.AppImpl[*entity.RuleSegment, repository.RuleSegment]

	procdefRepo repository.Procdef `inject:"T"`
}

var _ RuleSegment = (*ruleSegmentAppImpl)(nil)

// segmentRefPattern 条件组标识规则。
//
// 标识会原样出现在策略 JSON 里，被删除校验按文本反查引用，因此限定为小写字母数字与 -_，
// 不接受引号、反斜杠等会破坏 JSON 或模糊匹配的形状
var segmentRefPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,63}$`)

func (r *ruleSegmentAppImpl) GetPageList(condition *entity.RuleSegmentQuery) (*model.PageResult[*entity.RuleSegment], error) {
	return r.Repo.GetPageList(condition, "id DESC")
}

func (r *ruleSegmentAppImpl) SaveRuleSegment(ctx context.Context, segment *entity.RuleSegment) error {
	if !segmentRefPattern.MatchString(segment.Ref) {
		return errorx.NewBizI(ctx, imsg.ErrRuleSegmentRefInvalid, "ref", segment.Ref)
	}
	if _, ok := trigger.BizMetaOf(segment.BizType); !ok {
		return errorx.NewBizI(ctx, imsg.ErrRuleSegmentBizUnknown, "bizType", segment.BizType)
	}
	if segment.RuleNode == nil {
		return errorx.NewBizI(ctx, imsg.ErrRuleSegmentConditionRequired)
	}

	// 校验自己的树时用「待保存的内容」覆盖库里那份：
	// 否则改出一条指向自身的引用会拿旧内容去判，循环引用要到运行时才暴露
	resolve := r.resolverWithOverride(&trigger.SegmentDefinition{Ref: segment.Ref, BizType: segment.BizType, Condition: segment.RuleNode})
	if err := trigger.ValidateCondition(segment.BizType, segment.RuleNode, trigger.WithSegmentResolver(resolve)); err != nil {
		return toPolicyInvalidError(ctx, err)
	}

	if segment.Id == 0 {
		if existing := r.Repo.GetByRef(segment.Ref); existing != nil && existing.Id != segment.Id {
			return errorx.NewBizI(ctx, imsg.ErrRuleSegmentRefExist, "ref", segment.Ref)
		}
		return r.Save(ctx, segment)
	}

	// 标识是引用方写进策略 JSON 的外键，改它等于把所有引用一次性悬空
	original, err := r.GetById(segment.Id)
	if err != nil {
		return err
	}
	if original.Ref != segment.Ref {
		return errorx.NewBizI(ctx, imsg.ErrRuleSegmentRefImmutable, "ref", original.Ref)
	}
	return r.UpdateById(ctx, segment)
}

func (r *ruleSegmentAppImpl) DeleteRuleSegment(ctx context.Context, id uint64) error {
	segment, err := r.GetById(id)
	if err != nil {
		return err
	}
	referrers, err := r.listReferrers(segment.Ref)
	if err != nil {
		return err
	}
	if len(referrers) > 0 {
		return errorx.NewBizI(ctx, imsg.ErrRuleSegmentReferenced, "ref", segment.Ref, "referrers", strings.Join(referrers, ", "))
	}
	return r.DeleteById(ctx, id)
}

// listReferrers 找出引用该条件组的规则名称。
//
// 条件树以 JSON 文本存列，标识又被限定为不含引号与安全字符的字面量，
// 因此按 "ref":"<ref>" 这个 JSON 片段反查即可精确命中，不需要拉全表逐条反序列化
func (r *ruleSegmentAppImpl) listReferrers(ref string) ([]string, error) {
	// 引用写在 JSON 文本里，而排版不受我们控制：设计器写的是紧凑 JSON，但手工修过、
	// 或历史导入的数据可能带空格。只认一种写法会让引用漏检，条件组被删后要到运行期
	// 才报「无法判定」，那时被拦下的是正在提单的人，而不是改配置的管理员
	refPattern := regexp.MustCompile(`"ref"\s*:\s*"` + regexp.QuoteMeta(ref) + `"`)
	referrers := make([]string, 0)

	procdefs, err := r.procdefRepo.SelectByCond(model.NewCond())
	if err != nil {
		return nil, err
	}
	for _, procdef := range procdefs {
		policyText := ""
		if procdef.TriggerPolicy != nil {
			policyText = jsonx.ToStr(procdef.TriggerPolicy)
		}
		if refPattern.MatchString(policyText) || refPattern.MatchString(procdef.FlowDef) {
			referrers = append(referrers, procdef.Name)
		}
	}

	// 条件组可以引用条件组，因此本表也要一起反查
	// LIKE 只能按子串粗筛（含 ref 字样不等于引用了它），命中后仍用同一个正则判定
	segments, err := r.Repo.SelectByCond(model.NewCond().Like("rule_node", "%"+ref+"%"))
	if err != nil {
		return nil, err
	}
	for _, segment := range segments {
		if refPattern.MatchString(jsonx.ToStr(segment.RuleNode)) {
			referrers = append(referrers, segment.Name)
		}
	}
	return referrers, nil
}

func (r *ruleSegmentAppImpl) Resolver(ctx context.Context) trigger.SegmentResolver {
	return r.resolverWithOverride(nil)
}

// resolverWithOverride 构造条件组取数函数；override 非空时让该标识直接返回尚未落库的内容
// （见 SaveRuleSegment），这样自引用与「改完才成环」都能按新内容判出来。
// cache 让同一次求值里对同一标识的多次引用只查一次库
func (r *ruleSegmentAppImpl) resolverWithOverride(override *trigger.SegmentDefinition) trigger.SegmentResolver {
	cache := make(map[string]*trigger.SegmentDefinition)
	return func(ref string) (*trigger.SegmentDefinition, bool) {
		if override != nil && override.Ref == ref {
			return override, true
		}
		if cached, ok := cache[ref]; ok {
			return cached, cached != nil
		}
		segment := r.Repo.GetByRef(ref)
		if segment == nil {
			cache[ref] = nil
			return nil, false
		}
		def := &trigger.SegmentDefinition{Ref: segment.Ref, BizType: segment.BizType, Condition: segment.RuleNode}
		cache[ref] = def
		return def, true
	}
}

// GetRuleSegmentApp 取可复用条件组应用
func GetRuleSegmentApp() RuleSegment {
	return ioc.Get[RuleSegment]()
}
