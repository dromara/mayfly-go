package persistence

import (
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
)

type alertNotifyPolicyRepoImpl struct {
	base.RepoImpl[*entity.AlertNotifyPolicy]
}

var _ repository.AlertNotifyPolicy = (*alertNotifyPolicyRepoImpl)(nil)

func newAlertNotifyPolicyRepo() repository.AlertNotifyPolicy {
	return &alertNotifyPolicyRepoImpl{}
}

func (r *alertNotifyPolicyRepoImpl) GetAlertNotifyPolicyList(condition *entity.AlertNotifyPolicyQuery, orderBy ...string) (*model.PageResult[*entity.AlertNotifyPolicy], error) {
	qd := model.NewCond().
		Like("name", condition.Name).
		Eq("status", condition.Status)

	keyword := condition.Keyword
	if keyword != "" {
		keyword = "%" + keyword + "%"
		qd.And("name like ? or remark like ?", keyword, keyword)
	}
	return r.PageByCond(qd, condition.PageParam)
}

func (r *alertNotifyPolicyRepoImpl) ListEnabled() ([]*entity.AlertNotifyPolicy, error) {
	return r.SelectByCond(model.NewCond().Eq("status", entity.AlertNotifyPolicyStatusEnable))
}

func (r *alertNotifyPolicyRepoImpl) GetChannelDistribution() (map[string]int64, error) {
	type policyChannel struct {
		ChannelName string
		Cnt         int64
	}
	var rows []policyChannel
	err := global.Db.Table("t_alert_notify_policy p").
		Select("c.name as channel_name, COUNT(DISTINCT p.id) as cnt").
		Joins("JOIN t_alert_notify_policy_channel pc ON pc.policy_id = p.id").
		Joins("JOIN t_msg_channel c ON c.id = pc.channel_id").
		Where("p.status = 1 AND p.is_deleted = 0").
		Group("c.name").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(rows))
	for _, row := range rows {
		result[row.ChannelName] = row.Cnt
	}
	return result, nil
}
