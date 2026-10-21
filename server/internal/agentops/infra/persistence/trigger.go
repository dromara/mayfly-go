package persistence

import (
	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type triggerRepoImpl struct {
	base.RepoImpl[*entity.Trigger]
}

var _ repository.Trigger = (*triggerRepoImpl)(nil)

func newTriggerRepo() repository.Trigger {
	return &triggerRepoImpl{}
}

// GetPageList 分页查询触发器（列名 camelCase 由基座统一蛇形化）
func (t *triggerRepoImpl) GetPageList(condition *entity.Trigger, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.Trigger], error) {
	qd := model.NewCond().
		Like("name", condition.Name).
		Eq("sourceType", condition.SourceType).
		Eq("enabled", condition.Enabled).
		Eq("ownerAccountId", condition.OwnerAccountId).
		OrderBy(orderBy...)
	return t.PageByCond(qd, pageParam)
}

// ListEnabledBySource 列出某来源类型下全部启用的触发器。
// sourceType/enabled 恒为确定值，用 EqStrict 精确匹配（避免零值跳过语义干扰）；
// 软删过滤由基座 QueryCond 自动附加。
func (t *triggerRepoImpl) ListEnabledBySource(sourceType string) ([]*entity.Trigger, error) {
	return t.SelectByCond(model.NewCond().
		EqStrict("sourceType", sourceType).
		EqStrict("enabled", entity.TriggerEnabled))
}

// GetByWebhookToken 按 webhook 令牌精确定位触发器（令牌恒非空，用 EqStrict）
func (t *triggerRepoImpl) GetByWebhookToken(token string) (*entity.Trigger, error) {
	trigger := new(entity.Trigger)
	if err := t.GetByCond(model.NewCond().EqStrict("webhookToken", token).Dest(trigger)); err != nil {
		return nil, err
	}
	return trigger, nil
}
