package application

import (
	"context"
	"encoding/json"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/repository"
	"mayfly-go/internal/alert/imsg"
	labelapp "mayfly-go/internal/label/application"
	labelentity "mayfly-go/internal/label/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"strings"
)

// AlertNotifyPolicy 通知策略应用接口
type AlertNotifyPolicy interface {
	base.App[*entity.AlertNotifyPolicy]

	SaveAlertNotifyPolicy(ctx context.Context, policy *entity.AlertNotifyPolicy) error
	GetAlertNotifyPolicyList(condition *entity.AlertNotifyPolicyQuery, orderBy ...string) (*model.PageResult[*entity.AlertNotifyPolicy], error)
	DeleteNotifyPolicy(ctx context.Context, id uint64) error
	ChangeStatus(ctx context.Context, id uint64, status int8) error

	// ListEnabledWithLabels 返回所有启用策略并回填各策略的 MatchLabels（JSON）。
	// 调用方（告警引擎）拿到列表后用 matchRuleLabelsMap 自行匹配，
	// 与静默/升级策略的匹配方式保持一致，且便于引擎按周期缓存避免逐次通知全量查库
	ListEnabledWithLabels(ctx context.Context) ([]*entity.AlertNotifyPolicy, error)

	// GetChannelDistribution 按渠道统计关联的启用策略数（概览用）
	GetChannelDistribution() (map[string]int64, error)
}

type alertNotifyPolicyAppImpl struct {
	base.AppImpl[*entity.AlertNotifyPolicy, repository.AlertNotifyPolicy]

	labelBindingApp labelapp.LabelBinding `inject:"T"`
}

var _ AlertNotifyPolicy = (*alertNotifyPolicyAppImpl)(nil)

func (a *alertNotifyPolicyAppImpl) SaveAlertNotifyPolicy(ctx context.Context, policy *entity.AlertNotifyPolicy) error {
	if err := validateAlertNotifyPolicy(ctx, policy); err != nil {
		return err
	}

	// 提取 matchLabels 用于保存标签绑定，然后清空实体中的瞬态字段
	matchLabelsJSON := policy.MatchLabels
	policy.MatchLabels = ""

	var err error
	if policy.Id == 0 {
		err = a.Insert(ctx, policy)
	} else {
		err = a.UpdateById(ctx, policy)
	}
	if err != nil {
		return err
	}

	// 保存标签绑定到 t_label_binding 表
	if matchLabelsJSON != "" && matchLabelsJSON != "{}" {
		var labels map[string]string
		if err := json.Unmarshal([]byte(matchLabelsJSON), &labels); err == nil && len(labels) > 0 {
			if err := a.labelBindingApp.SaveBindingsByLabels(ctx, labelentity.LabelTargetAlertNotifyPolicy, policy.Id, labels); err != nil {
				return err
			}
		}
	} else {
		if err := a.labelBindingApp.DeleteByTarget(ctx, labelentity.LabelTargetAlertNotifyPolicy, policy.Id); err != nil {
			return err
		}
	}

	// 回填 MatchLabels 以便 API 响应包含标签信息
	policy.MatchLabels = matchLabelsJSON
	return nil
}

func (a *alertNotifyPolicyAppImpl) GetAlertNotifyPolicyList(condition *entity.AlertNotifyPolicyQuery, orderBy ...string) (*model.PageResult[*entity.AlertNotifyPolicy], error) {
	return a.GetRepo().GetAlertNotifyPolicyList(condition, orderBy...)
}

func (a *alertNotifyPolicyAppImpl) DeleteNotifyPolicy(ctx context.Context, id uint64) error {
	if _, err := a.GetById(id); err != nil {
		return errorx.NewBizI(ctx, imsg.ErrNotifyPolicyNotFound, "id", id)
	}
	if err := a.labelBindingApp.DeleteByTarget(ctx, labelentity.LabelTargetAlertNotifyPolicy, id); err != nil {
		return err
	}
	return a.DeleteById(ctx, id)
}

func (a *alertNotifyPolicyAppImpl) ChangeStatus(ctx context.Context, id uint64, status int8) error {
	policy, err := a.GetById(id)
	if err != nil {
		return errorx.NewBizI(ctx, imsg.ErrNotifyPolicyNotFound, "id", id)
	}
	if status != entity.AlertNotifyPolicyStatusEnable && status != entity.AlertNotifyPolicyStatusDisable {
		return errorx.NewBizI(ctx, imsg.ErrRuleStatusInvalid)
	}
	policy.Status = status
	return a.UpdateById(ctx, policy)
}

// ListEnabledWithLabels 返回所有启用策略并回填各策略的 MatchLabels（JSON）。
// 标签绑定批量加载（一次查询代替 N 次）；回填后的标签值由调用方用 parseRuleLabels 解析。
// 匹配语义：策略 MatchLabels 为空匹配所有事件，非空要求事件标签包含其全部键值对（AND 子集）
func (a *alertNotifyPolicyAppImpl) ListEnabledWithLabels(ctx context.Context) ([]*entity.AlertNotifyPolicy, error) {
	policies, err := a.GetRepo().ListEnabled()
	if err != nil {
		return nil, err
	}
	if len(policies) == 0 {
		return nil, nil
	}

	// 批量加载所有启用策略的标签绑定
	policyIds := make([]uint64, len(policies))
	for i, p := range policies {
		policyIds[i] = p.Id
	}
	allLabels, err := a.labelBindingApp.ListByTargets(ctx, labelentity.LabelTargetAlertNotifyPolicy, policyIds)
	if err != nil {
		return nil, err
	}

	for _, p := range policies {
		if labels := bindingsToMap(allLabels[p.Id]); len(labels) > 0 {
			b, err := json.Marshal(labels)
			if err != nil {
				return nil, err
			}
			p.MatchLabels = string(b)
		}
	}
	return policies, nil
}

func (a *alertNotifyPolicyAppImpl) GetChannelDistribution() (map[string]int64, error) {
	return a.GetRepo().GetChannelDistribution()
}

// validateAlertNotifyPolicy 通知策略保存校验
func validateAlertNotifyPolicy(ctx context.Context, policy *entity.AlertNotifyPolicy) error {
	if strings.TrimSpace(policy.Name) == "" {
		return errorx.NewBizI(ctx, imsg.ErrNotifyPolicyNameRequired)
	}
	if policy.Status != entity.AlertNotifyPolicyStatusEnable && policy.Status != entity.AlertNotifyPolicyStatusDisable {
		return errorx.NewBizI(ctx, imsg.ErrRuleStatusInvalid)
	}
	if len(policy.ChannelIds) == 0 && len(policy.ReceiverIds) == 0 {
		return errorx.NewBizI(ctx, imsg.ErrNotifyPolicyChannelRequired)
	}

	if policy.MatchLabels != "" && policy.MatchLabels != "{}" {
		var labels map[string]string
		if err := json.Unmarshal([]byte(policy.MatchLabels), &labels); err != nil {
			return errorx.NewBizI(ctx, imsg.ErrNotifyPolicyMatchLabelsInvalid)
		}
	}
	return nil
}
