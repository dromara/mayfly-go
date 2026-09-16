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
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
)

type AlertInhibition interface {
	base.App[*entity.AlertInhibition]

	SaveAlertInhibition(ctx context.Context, inhibition *entity.AlertInhibition) error
	GetAlertInhibitionList(condition *entity.AlertInhibitionQuery, orderBy ...string) (*model.PageResult[*entity.AlertInhibition], error)
	ChangeStatus(ctx context.Context, id uint64, status int8) error
	DeleteInhibition(ctx context.Context, id uint64) error
	// IsInhibited 检查目标事件是否被源事件抑制
	IsInhibited(sourceEvent *entity.AlertEvent, targetEvent *entity.AlertEvent) bool
}

type alertInhibitionAppImpl struct {
	base.AppImpl[*entity.AlertInhibition, repository.AlertInhibition]
	labelBindingApp labelapp.LabelBinding `inject:"T"`
}

var _ AlertInhibition = (*alertInhibitionAppImpl)(nil)

func (a *alertInhibitionAppImpl) SaveAlertInhibition(ctx context.Context, inhibition *entity.AlertInhibition) error {
	// 校验源匹配标签
	if inhibition.SourceMatch == "" {
		return errorx.NewBizI(ctx, imsg.ErrInhibitionMatchLabelsInvalid)
	}
	// 校验目标匹配标签
	if inhibition.TargetMatch == "" {
		return errorx.NewBizI(ctx, imsg.ErrInhibitionMatchLabelsInvalid)
	}

	if err := a.Save(ctx, inhibition); err != nil {
		return err
	}

	// 先清理旧绑定，避免更新时旧键残留（P0 修复）
	_ = a.labelBindingApp.DeleteByTarget(ctx, labelentity.LabelTargetAlertInhibitionSource, inhibition.Id)
	_ = a.labelBindingApp.DeleteByTarget(ctx, labelentity.LabelTargetAlertInhibitionTarget, inhibition.Id)

	// 保存源标签绑定到 t_label_binding 表
	if inhibition.SourceMatch != "" && inhibition.SourceMatch != "{}" {
		var labels map[string]string
		if err := json.Unmarshal([]byte(inhibition.SourceMatch), &labels); err == nil && len(labels) > 0 {
			logx.Debugf("[alert] SaveAlertInhibition: inhibition[%d] saving %d source label bindings", inhibition.Id, len(labels))
			if err := a.labelBindingApp.SaveBindingsByLabels(ctx, labelentity.LabelTargetAlertInhibitionSource, inhibition.Id, labels); err != nil {
				logx.Errorf("[alert] SaveAlertInhibition: inhibition[%d] source SaveBindingsByLabels error: %s", inhibition.Id, err.Error())
				return err
			}
		}
	}

	// 保存目标标签绑定到 t_label_binding 表
	if inhibition.TargetMatch != "" && inhibition.TargetMatch != "{}" {
		var labels map[string]string
		if err := json.Unmarshal([]byte(inhibition.TargetMatch), &labels); err == nil && len(labels) > 0 {
			logx.Debugf("[alert] SaveAlertInhibition: inhibition[%d] saving %d target label bindings", inhibition.Id, len(labels))
			if err := a.labelBindingApp.SaveBindingsByLabels(ctx, labelentity.LabelTargetAlertInhibitionTarget, inhibition.Id, labels); err != nil {
				logx.Errorf("[alert] SaveAlertInhibition: inhibition[%d] target SaveBindingsByLabels error: %s", inhibition.Id, err.Error())
				return err
			}
		}
	}

	return nil
}

func (a *alertInhibitionAppImpl) GetAlertInhibitionList(condition *entity.AlertInhibitionQuery, orderBy ...string) (*model.PageResult[*entity.AlertInhibition], error) {
	return a.GetRepo().GetAlertInhibitionList(condition, orderBy...)
}

func (a *alertInhibitionAppImpl) ChangeStatus(ctx context.Context, id uint64, status int8) error {
	return a.GetRepo().UpdateById(ctx, &entity.AlertInhibition{
		Model:  model.Model{Id: id},
		Status: status,
	})
}

func (a *alertInhibitionAppImpl) DeleteInhibition(ctx context.Context, id uint64) error {
	// 清理源标签绑定记录
	if err := a.labelBindingApp.DeleteByTarget(ctx, labelentity.LabelTargetAlertInhibitionSource, id); err != nil {
		return err
	}
	// 清理目标标签绑定记录
	if err := a.labelBindingApp.DeleteByTarget(ctx, labelentity.LabelTargetAlertInhibitionTarget, id); err != nil {
		return err
	}
	return a.GetRepo().DeleteById(ctx, id)
}

// IsInhibited 检查目标事件是否被源事件抑制
// 逻辑：遍历所有启用的抑制规则，检查是否满足抑制条件
func (a *alertInhibitionAppImpl) IsInhibited(sourceEvent *entity.AlertEvent, targetEvent *entity.AlertEvent) bool {
	inhibitions, err := a.GetRepo().ListEnabled()
	if err != nil || len(inhibitions) == 0 {
		return false
	}

	sourceLabels := parseRuleLabels(sourceEvent.Labels)
	targetLabels := parseRuleLabels(targetEvent.Labels)

	for _, inh := range inhibitions {
		if checkInhibition(inh, sourceLabels, targetLabels) {
			return true
		}
	}
	return false
}

// checkInhibition 检查单条抑制规则是否满足
func checkInhibition(inh *entity.AlertInhibition, sourceLabels, targetLabels map[string]string) bool {
	// 解析源匹配标签
	var sourceMatch map[string]string
	if err := json.Unmarshal([]byte(inh.SourceMatch), &sourceMatch); err != nil {
		return false
	}
	// 解析目标匹配标签
	var targetMatch map[string]string
	if err := json.Unmarshal([]byte(inh.TargetMatch), &targetMatch); err != nil {
		return false
	}
	// 解析相等标签
	var equal []string
	if inh.Equal != "" {
		if err := json.Unmarshal([]byte(inh.Equal), &equal); err != nil {
			equal = nil
		}
	}

	// 检查源事件是否匹配源匹配器
	if !matchLabels(sourceMatch, sourceLabels) {
		return false
	}
	// 检查目标事件是否匹配目标匹配器
	if !matchLabels(targetMatch, targetLabels) {
		return false
	}
	// 检查相等标签
	for _, key := range equal {
		if sourceLabels[key] != targetLabels[key] {
			return false
		}
	}

	return true
}

// matchLabels 检查 labels 是否匹配 matchers（所有 matcher 都必须满足）
func matchLabels(matchers, labels map[string]string) bool {
	if len(matchers) == 0 {
		return false
	}
	for k, v := range matchers {
		if labels[k] != v {
			return false
		}
	}
	return true
}
