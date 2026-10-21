package trigger

import (
	"context"

	"mayfly-go/internal/agentops/imsg"
	"mayfly-go/internal/pkg/event"
	"mayfly-go/pkg/errorx"
)

// AlertSource 告警触发源。
//
// alert 引擎已在「发布侧」把 AlertEvent 归一化为中立 OpsTriggerEvent（维持 alert 不依赖
// ai/agentops），故此处 Normalize 只做透传校验与 Source 补全；Match 复用通用字段白名单，
// 按资源类型/资源 id/严重级/指标匹配启用触发器。
type AlertSource struct{}

var _ TriggerSource = (*AlertSource)(nil)

// NewAlertSource 创建告警触发源
func NewAlertSource() *AlertSource { return &AlertSource{} }

func (s *AlertSource) Type() string { return SourceTypeAlert }

// Normalize 透传校验：入参须已是发布侧归一化好的 *event.OpsTriggerEvent
func (s *AlertSource) Normalize(ctx context.Context, raw any) (*event.OpsTriggerEvent, error) {
	evt, ok := raw.(*event.OpsTriggerEvent)
	if !ok || evt == nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrAlertPayloadInvalid)
	}
	if evt.Source == "" {
		evt.Source = SourceTypeAlert
	}
	return evt, nil
}

func (s *AlertSource) Match(evt *event.OpsTriggerEvent, cond *MatchCondition) bool {
	return MatchByFields(evt, cond)
}

// InstallAlertSource 向注册表注册告警触发源
func InstallAlertSource(r *Registry) { r.Register(NewAlertSource()) }
