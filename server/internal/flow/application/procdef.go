package application

import (
	"context"
	"mayfly-go/internal/flow/application/dto"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/internal/flow/domain/trigger"
	"mayfly-go/internal/flow/imsg"
	msgapp "mayfly-go/internal/msg/application"
	msgdto "mayfly-go/internal/msg/application/dto"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/jsonx"
)

type Procdef interface {
	base.App[*entity.Procdef]

	GetPageList(condition *entity.Procdef, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.ProcdefPagePO], error)

	// 保存流程实例信息
	SaveProcdef(ctx context.Context, def *dto.SaveProcdef) error

	// SaveFlowDef 保存流程定义流程信息
	SaveFlowDef(ctx context.Context, def *dto.SaveFlowDef) error

	// 删除流程实例信息
	DeleteProcdef(ctx context.Context, defId uint64) error

	// GetProcdefByCodePath 根据资源编号路径获取对应的流程定义
	GetProcdefByCodePath(ctx context.Context, codePaths ...string) *entity.Procdef

	// GetProcdefByResource 根据资源获取对应的流程定义
	GetProcdefByResource(ctx context.Context, resourceType int8, resourceCode string) *entity.Procdef

	// CheckTrigger 资源操作的前置触发策略校验，返回 nil 表示该资源未绑定流程或无需处置
	CheckTrigger(ctx context.Context, req *dto.TriggerRequest) (*trigger.Decision, error)

	// CheckProcdefTrigger 针对已定位的流程定义求值触发策略
	CheckProcdefTrigger(ctx context.Context, procdef *entity.Procdef, req *dto.TriggerRequest) (*trigger.Decision, error)

	// SimulateTrigger 试算触发策略，返回处置结论与引擎解析到的字段取值
	SimulateTrigger(ctx context.Context, req *dto.SimulateTrigger) (*dto.SimulateResult, error)

	// PolicySchema 下发场景的字段字典、检查项与可复用条件组，驱动前端条件编辑器
	PolicySchema(ctx context.Context) *dto.PolicySchema

	// ListPolicyHistory 返回某流程定义的策略变更时间线，最新的在前。
	// 合规要求「谁在什么时候把哪条规则改成了什么」必须可回溯，光有当前策略不够
	ListPolicyHistory(ctx context.Context, procdefId uint64, limit int) ([]*entity.ProcdefPolicyHis, error)
}

type procdefAppImpl struct {
	base.AppImpl[*entity.Procdef, repository.Procdef]

	procinstApp Procinst `inject:"T"`

	// policyHisRepo 记录策略变更前后快照，与流程定义保存在同一个事务里：
	// 分开写会留下「策略已改但记录没落库」的空档，而那正是审计最需要的一条
	policyHisRepo repository.ProcdefPolicyHis `inject:"T"`

	msgTmplBizApp    msgapp.MsgTmplBiz    `inject:"T"`
	tagTreeApp       tagapp.TagTreeReader `inject:"T"`
	tagTreeRelateApp tagapp.TagTreeRelate `inject:"T"`
}

var _ Procdef = (*procdefAppImpl)(nil)

var _ (Procdef) = (*procdefAppImpl)(nil)

func (p *procdefAppImpl) GetPageList(condition *entity.Procdef, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.ProcdefPagePO], error) {
	return p.Repo.GetPageList(condition, pageParam, orderBy...)
}

// validationOptions 组装保存期校验所需的上下文：条件组解析器 + 这份策略将要治理的场景。
//
// 场景由流程定义绑定的资源路径反推出来（各场景在注册表里声明自己治理哪几种资源类型），
// 这样才能挡住「给没有审批通道的资源配了兜底需审批」这种存进去永不生效的配置：
// 界面看着已配、运行时既不拦也不提示，出了事只会在日志里找不着北
func validationOptions(ctx context.Context, codePaths []string) []trigger.ValidateOption {
	options := []trigger.ValidateOption{trigger.WithSegmentResolver(ruleSegmentResolver(ctx))}
	if bound := boundResourcePaths(codePaths); len(bound) > 0 {
		// 兜底级别按「这些路径会命中哪些场景」判可落地性；同时把路径本体交给校验器，
		// 用来拦住「给某场景配了规则，但生效资源里根本没有它能治理的资源」这种永不命中的配置
		options = append(options, trigger.WithFallbackScenarios(trigger.GovernedBizTypes(bound)), trigger.WithBoundPaths(bound))
	}
	return options
}

// boundResourcePaths 把流程定义绑定的标签路径转成「资源类型路径」序列（剥掉纯标签段）。
//
// 治理粒度是路径而不是单个类型：绑在实例层会连带治理其下的库，只比末级类型会误判成不治理
func boundResourcePaths(codePaths []string) [][]int8 {
	paths := make([][]int8, 0, len(codePaths))
	for _, codePath := range codePaths {
		path := make([]int8, 0, 3)
		for _, section := range tagentity.CodePath(codePath).GetPathSections() {
			if section.Type == tagentity.TagTypeTag {
				continue
			}
			path = append(path, int8(section.Type))
		}
		if len(path) > 0 {
			paths = append(paths, path)
		}
	}
	return paths
}

func (p *procdefAppImpl) SaveProcdef(ctx context.Context, defParam *dto.SaveProcdef) error {
	def := defParam.Procdef
	if err := entity.ProcdefStatusEnum.Valid(def.Status); err != nil {
		return err
	}
	if err := trigger.ValidatePolicy(def.TriggerPolicy, validationOptions(ctx, defParam.CodePaths)...); err != nil {
		return toPolicyInvalidError(ctx, err)
	}

	if def.Id == 0 {
		if p.GetByCond(&entity.Procdef{DefKey: def.DefKey}) == nil {
			return errorx.NewBizI(ctx, imsg.ErrProcdefKeyExist)
		}
	} else {
		// 防止误修改key
		def.DefKey = ""
	}

	action := entity.ProcdefPolicyActionCreate
	var before *entity.TriggerPolicy
	if def.Id != 0 {
		action = entity.ProcdefPolicyActionUpdate
		original, err := p.GetById(def.Id)
		if err != nil {
			return err
		}
		before = original.TriggerPolicy
	}
	defName := def.Name

	return p.Tx(ctx, func(ctx context.Context) error {
		if err := p.Save(ctx, def); err != nil {
			return err
		}
		return p.recordPolicyChange(ctx, def.Id, defName, action, before, def.TriggerPolicy)
	}, func(ctx context.Context) error {
		// 保存通知消息模板
		if err := p.msgTmplBizApp.SaveBizTmpl(ctx, &msgdto.MsgTmplBizSave{
			TmplId:  defParam.MsgTmplId,
			BizType: FlowTaskNotifyBizKey,
			BizId:   def.Id,
		}); err != nil {
			return err
		}
		return p.tagTreeRelateApp.RelateTag(ctx, tagentity.TagRelateTypeFlowDef, def.Id, defParam.CodePaths...)
	})
}

func (p *procdefAppImpl) SaveFlowDef(ctx context.Context, def *dto.SaveFlowDef) error {
	if err := validateFlowDef(ctx, def.FlowDef); err != nil {
		return err
	}

	procdef := &entity.Procdef{
		FlowDef: jsonx.ToStr(def.FlowDef),
	}
	procdef.Id = def.Id
	return p.Save(ctx, procdef)
}

func (p *procdefAppImpl) DeleteProcdef(ctx context.Context, defId uint64) error {
	if err := p.canModify(ctx, defId); err != nil {
		return err
	}

	procdef, err := p.GetById(defId)
	if err != nil {
		return err
	}
	return p.Tx(ctx, func(ctx context.Context) error {
		// 删除本身就是最重的一次策略变更（所有资源的审批随之消失），必须留痕。
		// 变更记录不随流程定义删除而清理：合规问的是「当时是谁撤掉的」
		if err := p.recordPolicyChange(ctx, procdef.Id, procdef.Name, entity.ProcdefPolicyActionDelete, procdef.TriggerPolicy, nil); err != nil {
			return err
		}
		return p.DeleteById(ctx, defId)
	})
}

func (p *procdefAppImpl) ListPolicyHistory(ctx context.Context, procdefId uint64, limit int) ([]*entity.ProcdefPolicyHis, error) {
	return p.policyHisRepo.ListByProcdef(procdefId, limit)
}

// policyTextChanged 判断两份策略是否真的不同。
//
// 按序列化结果整体比较而不是逐字段判空：策略里全是列表与嵌套条件树，
// 逐字段比较很容易漏掉「级别改了但规则名没变」这类必须记录的情况
func policyTextChanged(before, after *entity.TriggerPolicy) bool {
	return jsonx.ToStr(before) != jsonx.ToStr(after)
}

// recordPolicyChange 写入一条策略变更记录。策略没变则不写：
// 只改名称、换标签的保存也会进来，时间线就会被「什么都没改」的条目淹没
func (p *procdefAppImpl) recordPolicyChange(ctx context.Context, procdefId uint64, procdefName string, action entity.ProcdefPolicyAction, before, after *entity.TriggerPolicy) error {
	if action == entity.ProcdefPolicyActionUpdate && !policyTextChanged(before, after) {
		return nil
	}
	return p.policyHisRepo.Insert(ctx, &entity.ProcdefPolicyHis{
		ProcdefId:   procdefId,
		ProcdefName: procdefName,
		Action:      action,
		Before:      before,
		After:       after,
	})
}

// GetProcdefByCodePath 按资源标签路径解析生效的流程定义（策略治理的入口），取不到即不受治理。
//
// 这里是两趟查询（标签关联 + 定义行）。实测本地 warm 连接下合计约 0.76ms/次（p95 约 1ms），
// 相对一次远端 Redis/MySQL 命令的往返可忽略，故**不加缓存**：
// 策略是安全控制，缓存意味着管理员改了之后存在不生效窗口，代价不对等。
// 若将来部署形态变为高 RTT 的远端存储再评估，方向应是「同请求内复用」（已做：
// Redis 批量与 SQL 文件导入都只解析一次），而不是跨请求 TTL
func (p *procdefAppImpl) GetProcdefByCodePath(ctx context.Context, codePaths ...string) *entity.Procdef {
	relateIds, err := p.tagTreeRelateApp.GetGovernRelateIds(ctx, tagentity.TagRelateTypeFlowDef, codePaths...)
	if err != nil || len(relateIds) == 0 {
		return nil
	}

	procdefId := relateIds[len(relateIds)-1]
	procdef, err := p.GetById(procdefId)
	if err != nil {
		return nil
	}
	if procdef.Status == entity.ProcdefStatusDisable {
		return nil
	}
	return procdef
}

func (p *procdefAppImpl) GetProcdefByResource(ctx context.Context, resourceType int8, resourceCode string) *entity.Procdef {
	resourceCodePaths := p.tagTreeApp.ListTagPathByTypeAndCode(resourceType, resourceCode)
	return p.GetProcdefByCodePath(ctx, resourceCodePaths...)
}

// 判断该流程实例是否可以执行修改操作
func (p *procdefAppImpl) canModify(ctx context.Context, prodefId uint64) error {
	if activeInstCount := p.procinstApp.CountByCond(&entity.Procinst{ProcdefId: prodefId, Status: entity.ProcinstStatusActive}); activeInstCount > 0 {
		return errorx.NewBizI(ctx, imsg.ErrExistProcinstRunning)
	}
	if suspInstCount := p.procinstApp.CountByCond(&entity.Procinst{ProcdefId: prodefId, Status: entity.ProcinstStatusSuspended}); suspInstCount > 0 {
		return errorx.NewBizI(ctx, imsg.ErrExistProcinstSuspended)
	}
	return nil
}

// validateFlowDef 校验流程定义信息
func validateFlowDef(ctx context.Context, p *entity.FlowDef) error {
	// 检查是否有开始节点
	startNodes := p.GetNodeByType(FlowNodeTypeStart)
	if len(startNodes) != 1 {
		return errorx.NewBiz("not one start node")
	}

	// 检查是否有结束节点
	endNodes := p.GetNodeByType(FlowNodeTypeEnd)
	if len(endNodes) != 1 {
		return errorx.NewBiz("not one end node")
	}

	// 校验节点自身逻辑
	for _, node := range p.Nodes {
		nh, _ := nodeBehaviorRegistry.GetNode(node.Type)
		if nh != nil {
			if err := nh.Validate(ctx, p, node); err != nil {
				return err
			}
		}
	}

	return validateFlowEdges(ctx, p)
}

// validateFlowEdges 校验连线上配置的跳转条件。
//
// 必须在保存流程图时就拦下：条件写坏（引用不存在的字段、条件组被删）在运行期只能选择
// 「按不成立处理」让流程走进没有出口的分支，或「按成立处理」跳过审批，两种都不是正确行为
func validateFlowEdges(ctx context.Context, flowDef *entity.FlowDef) error {
	resolver := ruleSegmentResolver(ctx)
	for _, edge := range flowDef.Edges {
		condition, err := edge.Condition()
		if err != nil {
			return flowConditionError(ctx, flowEdgeLabel(edge), err)
		}
		if err := trigger.ValidateCondition(FlowInstanceBizType, condition, trigger.WithSegmentResolver(resolver)); err != nil {
			return flowConditionError(ctx, flowEdgeLabel(edge), err)
		}
	}
	return nil
}
