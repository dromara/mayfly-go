package mask

import (
	"context"

	"mayfly-go/internal/db/domain/entity"
	masksvc "mayfly-go/internal/db/domain/mask"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/model"
)

// 脱敏规则与列标签的 CRUD。

// GetRulePageList 分页获取脱敏规则
func (m *MaskAppImpl) GetRulePageList(condition *entity.DbMaskRuleQuery, orderBy ...string) (*model.PageResult[*entity.DbMaskRule], error) {
	return m.maskRuleRepo.GetPageList(condition, orderBy...)
}

// SaveRule 保存脱敏规则，校验算法与参数合法性
func (m *MaskAppImpl) SaveRule(ctx context.Context, rule *entity.DbMaskRule) error {
	if rule.MatchType != entity.MaskMatchTypeRegex && rule.MatchType != entity.MaskMatchTypeExact && rule.MatchType != entity.MaskMatchTypePrefix {
		return errorx.NewBizf("invalid mask rule match type: %d", rule.MatchType)
	}
	if _, err := masksvc.Get(rule.Algorithm); err != nil {
		return err
	}
	if _, err := parseMaskParams(rule.Params); err != nil {
		return err
	}
	if rule.MatchType == entity.MaskMatchTypeRegex && rule.Pattern != "" {
		if _, err := masksvc.NewPlan([]*masksvc.Rule{{MatchType: masksvc.MatchTypeRegex, Pattern: rule.Pattern}}, nil); err != nil {
			return err
		}
	}
	if err := m.maskRuleRepo.Save(ctx, rule); err != nil {
		return err
	}
	m.invalidate()
	return nil
}

// DeleteRule 删除脱敏规则
func (m *MaskAppImpl) DeleteRule(ctx context.Context, id uint64) error {
	if err := m.maskRuleRepo.DeleteById(ctx, id); err != nil {
		return err
	}
	m.invalidate()
	return nil
}

// GetTagPageList 分页获取列标签
func (m *MaskAppImpl) GetTagPageList(condition *entity.DbMaskColumnQuery, orderBy ...string) (*model.PageResult[*entity.DbMaskColumn], error) {
	return m.maskColumnRepo.GetPageList(condition, orderBy...)
}

// GetTagById 根据id获取列标签
func (m *MaskAppImpl) GetTagById(id uint64) (*entity.DbMaskColumn, error) {
	return m.maskColumnRepo.GetById(id)
}

// SaveTag 保存列标签，校验动作对应的算法/规则配置
func (m *MaskAppImpl) SaveTag(ctx context.Context, tag *entity.DbMaskColumn) error {
	if tag.InstanceId == 0 {
		return errorx.NewBizf("mask column tag instance id is required")
	}
	switch tag.Action {
	case entity.MaskColumnActionBind:
		if tag.Algorithm == "" && tag.RuleId == 0 {
			return errorx.NewBizI(ctx, imsg.ErrMaskTagNeedAlgoOrRule)
		}
		if tag.Algorithm != "" {
			if _, err := masksvc.Get(tag.Algorithm); err != nil {
				return err
			}
		} else if _, err := m.maskRuleRepo.GetById(tag.RuleId); err != nil {
			return err
		}
	case entity.MaskColumnActionExempt:
	default:
		return errorx.NewBizf("invalid mask column action: %d", tag.Action)
	}
	if _, err := parseMaskParams(tag.Params); err != nil {
		return err
	}
	if err := m.maskColumnRepo.Save(ctx, tag); err != nil {
		return err
	}
	m.invalidate()
	return nil
}

// DeleteTag 删除列标签
func (m *MaskAppImpl) DeleteTag(ctx context.Context, id uint64) error {
	if err := m.maskColumnRepo.DeleteById(ctx, id); err != nil {
		return err
	}
	m.invalidate()
	return nil
}

// GetMaskApp 获取脱敏应用门面（组合接口，向后兼容）
func GetMaskApp() MaskApp {
	return ioc.Get[MaskApp]()
}

// GetMaskEngine 获取通用脱敏引擎（供任意模块注入使用）
func GetMaskEngine() MaskEngine {
	return ioc.Get[MaskEngine]()
}

// GetMaskRuleApp 获取脱敏规则管理服务
func GetMaskRuleApp() MaskRuleApp {
	return ioc.Get[MaskRuleApp]()
}
