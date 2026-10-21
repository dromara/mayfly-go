package repository

import (
	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

// Trigger 事件驱动触发器仓储
type Trigger interface {
	base.Repo[*entity.Trigger]

	// GetPageList 分页查询触发器（按名称模糊、来源类型、启用态、归属账号过滤）
	GetPageList(condition *entity.Trigger, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.Trigger], error)

	// ListEnabledBySource 列出某来源类型下全部启用的触发器（事件到达时做匹配派发用）。
	// 软删过滤由基座 QueryCond 自动附加，禁用的触发器不返回。
	ListEnabledBySource(sourceType string) ([]*entity.Trigger, error)

	// GetByWebhookToken 按 webhook 令牌定位触发器（webhook 入口校验用）；不存在返回 model.ErrNotFound
	GetByWebhookToken(token string) (*entity.Trigger, error)
}
