package application

import (
	"context"
	"testing"

	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/trigger"

	"mayfly-go/pkg/base"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 条件组测试用的字段字典。
//
// 不复用 db/redis 的真实场景：那两个场景由各自模块的包级 init 注册，而它们又依赖 flow 应用层，
// flow 的测试再去导入会构成导入环。条件组的机制（存在性、场景一致、循环、悬空）与具体场景无关，
// 用一份本地注册的小字典反而能把这些判定与「某场景恰好有哪些字段」解耦
const (
	segTestBizType  = "flow_segment_test_flow"
	segOtherBizType = "flow_segment_other_flow"
)

func init() {
	RegisterTriggerBiz(trigger.BizMeta{
		BizType:    segTestBizType,
		Approvable: true,
		Fields: []trigger.TriggerField{
			{Key: "sql", TitleKey: "flow.field.sql", Group: "statement", Type: trigger.TypeString},
			{Key: "stmtType", TitleKey: "flow.field.stmtType", Group: "statement", Type: trigger.TypeEnum, Options: []trigger.FieldOption{{Value: "update"}, {Value: "delete"}}},
		},
	})
	RegisterTriggerBiz(trigger.BizMeta{
		BizType: segOtherBizType,
		Fields: []trigger.TriggerField{
			{Key: "cmd", TitleKey: "flow.field.cmd", Group: "command", Type: trigger.TypeString},
		},
	})
}

// 仓储替身：条件组的引用、循环、悬空判定与策略变更都要读库，
// 只补上各自接口比 base.Repo 多出的那几个方法，行为与 infra 里的真实实现一致
type segmentRepoStub struct {
	*base.RepoImpl[*entity.RuleSegment]
}

func (s *segmentRepoStub) GetPageList(condition *entity.RuleSegmentQuery, orderBy ...string) (*model.PageResult[*entity.RuleSegment], error) {
	qd := model.NewCond().Eq("biz_type", condition.BizType).Like("name", condition.Name).OrderBy(orderBy...)
	return s.PageByCond(qd, condition.PageParam)
}

func (s *segmentRepoStub) GetByRef(ref string) *entity.RuleSegment {
	segment := &entity.RuleSegment{Ref: ref}
	if err := s.GetByCond(segment); err != nil {
		return nil
	}
	return segment
}

type procdefRepoStub struct {
	*base.RepoImpl[*entity.Procdef]
}

func (p *procdefRepoStub) GetPageList(*entity.Procdef, model.PageParam, ...string) (*model.PageResult[*entity.ProcdefPagePO], error) {
	return nil, nil
}

type policyHisRepoStub struct {
	*base.RepoImpl[*entity.ProcdefPolicyHis]
}

func (r *policyHisRepoStub) ListByProcdef(procdefId uint64, limit int) ([]*entity.ProcdefPolicyHis, error) {
	page, err := r.PageByCond(model.NewCond().Eq("procdef_id", procdefId).OrderBy("id DESC"), model.PageParam{PageNum: 1, PageSize: limit})
	if err != nil {
		return nil, err
	}
	return page.List, nil
}

// setupRuleSegmentTest 内存 sqlite 建表并接管全局 db（base.RepoImpl 走 global.Db）
func setupRuleSegmentTest(t *testing.T) (*ruleSegmentAppImpl, *procdefAppImpl, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&entity.RuleSegment{}, &entity.Procdef{}, &entity.ProcdefPolicyHis{}))

	prev := global.Db
	global.Db = db
	t.Cleanup(func() { global.Db = prev })

	segmentApp := &ruleSegmentAppImpl{}
	segmentApp.Repo = &segmentRepoStub{RepoImpl: &base.RepoImpl[*entity.RuleSegment]{}}
	segmentApp.procdefRepo = &procdefRepoStub{RepoImpl: &base.RepoImpl[*entity.Procdef]{}}

	procdefApp := &procdefAppImpl{}
	procdefApp.Repo = &procdefRepoStub{RepoImpl: &base.RepoImpl[*entity.Procdef]{}}
	procdefApp.policyHisRepo = &policyHisRepoStub{RepoImpl: &base.RepoImpl[*entity.ProcdefPolicyHis]{}}

	return segmentApp, procdefApp, db
}

func newSegment(ref, name, bizType string, node *entity.RuleNode) *entity.RuleSegment {
	segment := &entity.RuleSegment{Ref: ref, Name: name, BizType: bizType, RuleNode: node}
	segment.FillBaseInfo(model.IdGenTypeNone, nil)
	return segment
}

func conditionLeaf(field, op string, value any) *entity.RuleNode {
	return &entity.RuleNode{Kind: entity.NodeKindCondition, Field: field, Op: entity.OpName(op), Value: value}
}

func segmentLeaf(ref string) *entity.RuleNode {
	return &entity.RuleNode{Kind: entity.NodeKindSegment, Ref: ref}
}

func segmentGroup(logic entity.Logic, items ...*entity.RuleNode) *entity.RuleNode {
	return &entity.RuleNode{Kind: entity.NodeKindGroup, Logic: logic, Items: items}
}

func TestSaveRuleSegmentPersistsAndResolves(t *testing.T) {
	segmentApp, _, _ := setupRuleSegmentTest(t)
	ctx := context.Background()

	require.NoError(t, segmentApp.SaveRuleSegment(ctx, newSegment("dba_members", "DBA 成员", segTestBizType, conditionLeaf("sql", "contains", "orders"))))

	resolver := segmentApp.Resolver(ctx)
	def, ok := resolver("dba_members")
	require.True(t, ok)
	require.Equal(t, segTestBizType, def.BizType)
	require.Equal(t, "sql", def.Condition.Field)

	// 取不到时必须返回「不存在」而不是一个空定义：引用它的规则要按「无法判定」失败
	missing, ok := resolver("not_there")
	require.False(t, ok)
	require.Nil(t, missing)
}

func TestSaveRuleSegmentRejectsBadConfigs(t *testing.T) {
	segmentApp, _, _ := setupRuleSegmentTest(t)
	ctx := context.Background()

	cases := []struct {
		name   string
		seeded *entity.RuleSegment
		other  *entity.RuleSegment
	}{
		{name: "标识含大写", seeded: newSegment("DBA", "x", segTestBizType, conditionLeaf("sql", "contains", "a"))},
		{name: "标识过短", seeded: newSegment("a", "x", segTestBizType, conditionLeaf("sql", "contains", "a"))},
		{name: "场景未注册", seeded: newSegment("ghost_biz", "x", "ghost_flow", conditionLeaf("sql", "contains", "a"))},
		{name: "条件为空", seeded: newSegment("empty_cond", "x", segTestBizType, nil)},
		{name: "引用未注册字段", seeded: newSegment("ghost_field", "x", segTestBizType, conditionLeaf("nope", "eq", "a"))},
	}

	for _, tc := range cases {
		err := segmentApp.SaveRuleSegment(ctx, tc.seeded)
		require.Error(t, err, tc.name)
	}

	// 标识唯一：它是写进别人策略 JSON 里的外键
	require.NoError(t, segmentApp.SaveRuleSegment(ctx, newSegment("dba_members", "DBA 成员", segTestBizType, conditionLeaf("sql", "contains", "orders"))))
	duplicate := newSegment("dba_members", "另一个", segTestBizType, conditionLeaf("sql", "contains", "users"))
	require.Error(t, segmentApp.SaveRuleSegment(ctx, duplicate))
}

// 自引用必须在保存时就挡住。
// 条件组还没落库，校验用的必须是「待保存的内容」，否则拿旧内容判会以为没环
func TestSaveRuleSegmentRejectsSelfReference(t *testing.T) {
	segmentApp, _, _ := setupRuleSegmentTest(t)
	ctx := context.Background()

	self := newSegment("loop_a", "自引用", segTestBizType, segmentGroup(entity.LogicAll, segmentLeaf("loop_a")))
	err := segmentApp.SaveRuleSegment(ctx, self)
	require.Error(t, err)
	require.Contains(t, err.Error(), "循环引用自身")
}

func TestSaveRuleSegmentRejectsCycleBetweenSegments(t *testing.T) {
	segmentApp, _, db := setupRuleSegmentTest(t)
	ctx := context.Background()

	// 先各存一个互不引用的合法条件组，再把 a 改成引用 b：此时 b 已引用 a，环只在改完之后出现
	require.NoError(t, segmentApp.SaveRuleSegment(ctx, newSegment("seg_a", "A", segTestBizType, conditionLeaf("sql", "contains", "orders"))))
	require.NoError(t, segmentApp.SaveRuleSegment(ctx, newSegment("seg_b", "B", segTestBizType, segmentLeaf("seg_a"))))

	var segA entity.RuleSegment
	require.NoError(t, db.Where("ref = ?", "seg_a").First(&segA).Error)

	edited := newSegment("seg_a", "A", segTestBizType, segmentLeaf("seg_b"))
	edited.Id = segA.Id
	err := segmentApp.SaveRuleSegment(ctx, edited)
	require.Error(t, err)
	require.Contains(t, err.Error(), "循环引用自身")
}

// 条件组只能在同场景的规则里引用：跨场景时字段同名不同义，判出来的结果没有任何保证
func TestSaveRuleSegmentRejectsForeignScenario(t *testing.T) {
	segmentApp, _, _ := setupRuleSegmentTest(t)
	ctx := context.Background()

	require.NoError(t, segmentApp.SaveRuleSegment(ctx, newSegment("db_leaf", "DB 条件", segTestBizType, conditionLeaf("sql", "contains", "orders"))))

	err := segmentApp.SaveRuleSegment(ctx, newSegment("redis_leaf", "Redis 里引用 DB 条件", segOtherBizType, segmentLeaf("db_leaf")))
	require.Error(t, err)
	require.Contains(t, err.Error(), segOtherBizType)
}

func TestSaveRuleSegmentKeepsRefImmutable(t *testing.T) {
	segmentApp, _, db := setupRuleSegmentTest(t)
	ctx := context.Background()

	segment := newSegment("stable_ref", "原名", segTestBizType, conditionLeaf("sql", "contains", "orders"))
	require.NoError(t, segmentApp.SaveRuleSegment(ctx, segment))

	renamed := newSegment("other_ref", "改名", segTestBizType, conditionLeaf("sql", "contains", "users"))
	renamed.Id = segment.Id
	err := segmentApp.SaveRuleSegment(ctx, renamed)
	require.Error(t, err)

	// 名称与条件可以改，标识不动
	same := newSegment("stable_ref", "新名", segTestBizType, conditionLeaf("sql", "contains", "users"))
	same.Id = segment.Id
	require.NoError(t, segmentApp.SaveRuleSegment(ctx, same))

	var reloaded entity.RuleSegment
	require.NoError(t, db.First(&reloaded, segment.Id).Error)
	require.Equal(t, "新名", reloaded.Name)
}

// 被引用的条件组删不掉：删掉会让引用它的策略在运行期集体报「无法判定」
func TestDeleteRuleSegmentBlockedWhileReferenced(t *testing.T) {
	segmentApp, procdefApp, db := setupRuleSegmentTest(t)
	ctx := context.Background()

	segment := newSegment("dba_members", "DBA 成员", segTestBizType, conditionLeaf("sql", "contains", "orders"))
	require.NoError(t, segmentApp.SaveRuleSegment(ctx, segment))

	// 未引用时可删
	require.NoError(t, segmentApp.DeleteRuleSegment(ctx, segment.Id))

	recreated := newSegment("dba_members", "DBA 成员", segTestBizType, conditionLeaf("sql", "contains", "orders"))
	require.NoError(t, segmentApp.SaveRuleSegment(ctx, recreated))
	// 删除后同一标识可以重新使用：唯一性只约束未删除的行
	require.NotEqual(t, segment.Id, recreated.Id)

	procdef := &entity.Procdef{Name: "生产库审批", DefKey: "prod_db_flow", Status: entity.ProcdefStatusEnable}
	procdef.TriggerPolicy = &entity.TriggerPolicy{
		Version:         entity.TriggerPolicyVersion,
		DefaultSeverity: entity.SeverityRequired,
		Customs: []*entity.CustomCondition{{
			BizType:  segTestBizType,
			Severity: entity.SeverityRequired,
			When:     segmentGroup(entity.LogicAll, segmentLeaf("dba_members")),
		}},
	}
	procdef.FillBaseInfo(model.IdGenTypeNone, nil)
	require.NoError(t, db.Create(procdef).Error)

	err := segmentApp.DeleteRuleSegment(ctx, recreated.Id)
	require.Error(t, err)
	require.Contains(t, err.Error(), "生产库审批")

	// 只被流程图的跳转条件引用同样要拦住
	procdef.TriggerPolicy = nil
	procdef.FlowDef = `{"nodes":[],"edges":[{"name":"通过","key":"e1","sourceNodeKey":"a","targetNodeKey":"b","extra":{"condition":{"kind":"segment","ref":"dba_members"}}}]}`
	procdef.Name = "流程图引用"
	require.NoError(t, procdefApp.UpdateById(ctx, procdef))
	err = segmentApp.DeleteRuleSegment(ctx, recreated.Id)
	require.Error(t, err)
	require.Contains(t, err.Error(), "流程图引用")

	// 撤掉两处引用后才能真正删除。这里直连 db 清列：UpdateById 走的是「只写非零字段」，
	// 把字段设成 nil 再保存并不会把库里那列清空，测试若依赖它就会出现「已经清了却仍报被引用」
	require.NoError(t, db.Model(&entity.Procdef{}).Where("id = ?", procdef.Id).Updates(map[string]any{"flow_def": `{"nodes":[],"edges":[]}`, "trigger_policy": nil}).Error)
	require.NoError(t, segmentApp.DeleteRuleSegment(ctx, recreated.Id))
}

// 触发策略里的悬空引用要在保存时就拒掉，而不是等真实操作撞上
func TestTriggerPolicySaveChecksSegmentReferences(t *testing.T) {
	segmentApp, _, _ := setupRuleSegmentTest(t)
	ctx := context.Background()

	policy := &entity.TriggerPolicy{
		Version:         entity.TriggerPolicyVersion,
		DefaultSeverity: entity.SeverityRequired,
		Customs: []*entity.CustomCondition{{
			BizType:  segTestBizType,
			Severity: entity.SeverityRequired,
			When:     segmentGroup(entity.LogicAll, segmentLeaf("dba_members")),
		}},
	}

	err := trigger.ValidatePolicy(policy, trigger.WithSegmentResolver(segmentApp.Resolver(ctx)))
	require.Error(t, err)
	require.Contains(t, err.Error(), "不存在")

	require.NoError(t, segmentApp.SaveRuleSegment(ctx, newSegment("dba_members", "DBA 成员", segTestBizType, conditionLeaf("sql", "contains", "orders"))))
	require.NoError(t, trigger.ValidatePolicy(policy, trigger.WithSegmentResolver(segmentApp.Resolver(ctx))))
}

// 求值路径要能展开条件组，并且结论与直接写那棵树一模一样
func TestTriggerEvaluationExpandsSegment(t *testing.T) {
	segmentApp, _, _ := setupRuleSegmentTest(t)
	ctx := context.Background()
	require.NoError(t, segmentApp.SaveRuleSegment(ctx, newSegment("orders_only", "只碰 orders", segTestBizType, conditionLeaf("sql", "contains", "orders"))))

	policy := &entity.TriggerPolicy{
		Version:         entity.TriggerPolicyVersion,
		DefaultSeverity: entity.SeverityDisabled,
		Customs: []*entity.CustomCondition{{
			BizType:  segTestBizType,
			Severity: entity.SeverityRequired,
			When:     segmentGroup(entity.LogicAll, segmentLeaf("orders_only")),
		}},
	}

	segment := segmentApp.Resolver(ctx)
	hit := trigger.NewContext(ctx, segTestBizType, 1, map[string]string{"sql": "delete from orders"}, nil).WithSegments(segment)
	decision := trigger.Evaluate(ctx, policy, hit)
	require.Equal(t, entity.SeverityRequired, decision.Severity)
	require.True(t, decision.Matched)

	miss := trigger.NewContext(ctx, segTestBizType, 1, map[string]string{"sql": "select 1"}, nil).WithSegments(segment)
	require.Equal(t, entity.SeverityDisabled, trigger.Evaluate(ctx, policy, miss).Severity)

	// 条件组被删掉后策略不会静默放行，而是 fail-closed
	require.NoError(t, segmentApp.SaveRuleSegment(ctx, newSegment("other", "其它", segTestBizType, conditionLeaf("sql", "contains", "x"))))
	var stored entity.RuleSegment
	require.NoError(t, segmentApp.GetByCond(&stored))
	require.NoError(t, segmentApp.DeleteRuleSegment(ctx, stored.Id))

	after := trigger.NewContext(ctx, segTestBizType, 1, map[string]string{"sql": "delete from orders"}, nil).WithSegments(segmentApp.Resolver(ctx))
	decision = trigger.Evaluate(ctx, policy, after)
	require.Equal(t, entity.SeverityRequired, decision.Severity)
	require.NotEmpty(t, decision.Findings)
}

func TestPolicyHistorySnapshots(t *testing.T) {
	_, procdefApp, db := setupRuleSegmentTest(t)
	ctx := context.Background()

	before := &entity.TriggerPolicy{Version: entity.TriggerPolicyVersion, DefaultSeverity: entity.SeverityRequired}
	after := &entity.TriggerPolicy{
		Version:         entity.TriggerPolicyVersion,
		DefaultSeverity: entity.SeverityRequired,
		Checks:          []*entity.CheckConfig{{Key: "db.dml-without-where", BizType: segTestBizType, Severity: entity.SeverityRequired}},
	}

	// 没变化不写记录，否则时间线会被「什么都没改」的条目淹没
	require.NoError(t, procdefApp.recordPolicyChange(ctx, 7, "生产库审批", entity.ProcdefPolicyActionUpdate, after, after))
	records, err := procdefApp.ListPolicyHistory(ctx, 7, 10)
	require.NoError(t, err)
	require.Empty(t, records)

	require.NoError(t, procdefApp.recordPolicyChange(ctx, 7, "生产库审批", entity.ProcdefPolicyActionCreate, nil, after))
	require.NoError(t, procdefApp.recordPolicyChange(ctx, 7, "生产库审批", entity.ProcdefPolicyActionUpdate, after, before))

	records, err = procdefApp.ListPolicyHistory(ctx, 7, 10)
	require.NoError(t, err)
	require.Len(t, records, 2)

	// 最新在前，且每条都带前后两份快照：diff 由展示侧现算，文案才不会随 i18n 演进而失真
	require.Equal(t, entity.ProcdefPolicyActionUpdate, records[0].Action)
	require.Empty(t, records[0].After.Checks)
	require.Len(t, records[1].After.Checks, 1)
	require.Nil(t, records[1].Before)

	records, err = procdefApp.ListPolicyHistory(ctx, 999, 10)
	require.NoError(t, err)
	require.Empty(t, records, "another process definition must not leak into this timeline")

	var count int64
	require.NoError(t, db.Model(&entity.ProcdefPolicyHis{}).Count(&count).Error)
	require.EqualValues(t, 2, count)
}

// 流程变量的条件判定与条件组配合：跳转条件里引用条件组时，字段字典必须是流程实例那份
func TestFlowConditionWithSegment(t *testing.T) {
	segmentApp, _, _ := setupRuleSegmentTest(t)
	ctx := context.Background()

	require.NoError(t, segmentApp.SaveRuleSegment(ctx, newSegment("all_approved", "全部通过", FlowInstanceBizType, conditionLeaf(flowFieldNrOfCompletedRate, "gte", 1))))

	// 服务就绪时 newFlowConditionContext 会从 IOC 取到条件组应用；
	// 单测里没有容器，这里显式挂上同一个取数函数，验证的是「挂上之后能不能正确展开」
	resolver := segmentApp.Resolver(ctx)
	withResolver := func(vars collx.M) *trigger.Context {
		return newFlowConditionContext(ctx, vars, 0).WithSegments(resolver)
	}

	matched, err := trigger.MatchCondition(segmentLeaf("all_approved"), withResolver(collx.M{flowFieldNrOfAll: 3, flowFieldNrOfCompleted: 3}))
	require.NoError(t, err)
	require.True(t, matched)

	notMatched, err := trigger.MatchCondition(segmentLeaf("all_approved"), withResolver(collx.M{flowFieldNrOfAll: 3, flowFieldNrOfCompleted: 2}))
	require.NoError(t, err)
	require.False(t, notMatched)
}

// TestDeleteBlockedRegardlessOfJsonSpacing 反查引用不能只认一种 JSON 排版。
//
// 引用写在 flow_def 的原文里：设计器输出紧凑 JSON，但人工修过或历史导入的数据会带空格
// （`"ref": "x"`）。精确匹配那一种写法就会漏检，条件组被删掉后要到运行期才报「无法判定」，
// 那时被拦下的是正在提单的人，而不是当初删掉它的管理员
func TestDeleteBlockedRegardlessOfJsonSpacing(t *testing.T) {
	segmentApp, procdefApp, _ := setupRuleSegmentTest(t)
	ctx := context.Background()

	segment := newSegment("dba_members", "DBA 成员", segTestBizType, conditionLeaf("sql", "contains", "orders"))
	require.NoError(t, segmentApp.SaveRuleSegment(ctx, segment))

	procdef := &entity.Procdef{Name: "带空格的流程图", DefKey: "spaced_flow", FlowDef: `{"nodes": [], "edges": [{"extra": { "ref" : "dba_members" }}]}`}
	require.NoError(t, procdefApp.Save(ctx, procdef))

	err := segmentApp.DeleteRuleSegment(ctx, segment.Id)
	require.Error(t, err, "带空格的引用也必须被反查出来")
	require.Contains(t, err.Error(), "带空格的流程图")
}

// TestDeleteProtectionIgnoresMentions 反查认结构而不是认子串。
//
// 粗筛用 LIKE（含 ref 字样），命中后仍要按 `"ref":"..."` 判定：
// 流程名、备注里出现同样的字符串不是引用，误判会把管理员锁在删除入口外
func TestDeleteProtectionIgnoresMentions(t *testing.T) {
	segmentApp, procdefApp, _ := setupRuleSegmentTest(t)
	ctx := context.Background()

	segment := newSegment("dba_members", "DBA 成员", segTestBizType, conditionLeaf("sql", "contains", "orders"))
	require.NoError(t, segmentApp.SaveRuleSegment(ctx, segment))

	procdef := &entity.Procdef{Name: "只是提到它", DefKey: "mention_flow", FlowDef: `{"nodes":[],"remark":"dba_members 这个条件组以后可能用"}`}
	require.NoError(t, procdefApp.Save(ctx, procdef))

	require.NoError(t, segmentApp.DeleteRuleSegment(ctx, segment.Id))
}
