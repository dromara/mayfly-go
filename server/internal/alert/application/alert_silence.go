package application

import (
	"context"
	"encoding/json"
	"fmt"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/repository"
	"mayfly-go/internal/alert/imsg"
	labelapp "mayfly-go/internal/label/application"
	labelentity "mayfly-go/internal/label/domain/entity"
	"mayfly-go/pkg/base"
	global_cache "mayfly-go/pkg/cache"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"strings"
	"time"
)

const (
	// silenceCacheKey 版本号:ruleId:resourceId。
	// 版本号参与 key 组成，使静默规则变更后旧缓存立即失效（无需等待 TTL）
	silenceCacheKey    = "alert:silence:%d:%d:%d"
	silenceCacheVerKey = "alert:silence:ver"
	silenceCacheTTL    = 30 * time.Second
)

type AlertSilence interface {
	base.App[*entity.AlertSilence]

	SaveAlertSilence(ctx context.Context, silence *entity.AlertSilence) error
	GetAlertSilenceList(condition *entity.AlertSilenceQuery, orderBy ...string) (*model.PageResult[*entity.AlertSilence], error)
	ChangeStatus(ctx context.Context, id uint64, status int8) error
	DeleteSilence(ctx context.Context, id uint64) error
	IsSilenced(ctx context.Context, rule *entity.AlertRule, resourceId uint64) bool
	InvalidateSilenceCache()
}

type alertSilenceAppImpl struct {
	base.AppImpl[*entity.AlertSilence, repository.AlertSilence]

	labelBindingApp labelapp.LabelBinding `inject:"T"`
}

var _ AlertSilence = (*alertSilenceAppImpl)(nil)

func (a *alertSilenceAppImpl) SaveAlertSilence(ctx context.Context, silence *entity.AlertSilence) error {
	if err := validateAlertSilence(ctx, silence); err != nil {
		return err
	}

	// 提取 matchLabels 用于保存标签绑定，然后清空实体中的瞬态字段
	matchLabelsJSON := silence.MatchLabels
	silence.MatchLabels = ""

	var err error
	if silence.Id == 0 {
		err = a.Insert(ctx, silence)
	} else {
		err = a.UpdateById(ctx, silence)
	}
	if err != nil {
		return err
	}

	// 保存标签绑定到 t_label_binding 表
	if matchLabelsJSON != "" && matchLabelsJSON != "{}" {
		var labels map[string]string
		if err := json.Unmarshal([]byte(matchLabelsJSON), &labels); err == nil && len(labels) > 0 {
			if err := a.labelBindingApp.SaveBindingsByLabels(ctx, labelentity.LabelTargetAlertSilence, silence.Id, labels); err != nil {
				return err
			}
		}
	} else {
		if err := a.labelBindingApp.DeleteByTarget(ctx, labelentity.LabelTargetAlertSilence, silence.Id); err != nil {
			return err
		}
	}

	// 回填 MatchLabels 以便 API 响应包含标签信息
	silence.MatchLabels = matchLabelsJSON
	a.InvalidateSilenceCache()
	return nil
}

func (a *alertSilenceAppImpl) GetAlertSilenceList(condition *entity.AlertSilenceQuery, orderBy ...string) (*model.PageResult[*entity.AlertSilence], error) {
	return a.GetRepo().GetAlertSilenceList(condition, orderBy...)
}

func (a *alertSilenceAppImpl) ChangeStatus(ctx context.Context, id uint64, status int8) error {
	silence, err := a.GetById(id)
	if err != nil {
		return errorx.NewBizI(ctx, imsg.ErrSilenceNotFound, "id", id)
	}
	if status != entity.AlertSilenceStatusEnable && status != entity.AlertSilenceStatusDisable {
		return errorx.NewBizI(ctx, imsg.ErrRuleStatusInvalid)
	}
	silence.Status = status
	err = a.UpdateById(ctx, silence)
	if err == nil {
		a.InvalidateSilenceCache()
	}
	return err
}

func (a *alertSilenceAppImpl) DeleteSilence(ctx context.Context, id uint64) error {
	if _, err := a.GetById(id); err != nil {
		return errorx.NewBizI(ctx, imsg.ErrSilenceNotFound, "id", id)
	}
	// 清理标签绑定记录
	if err := a.labelBindingApp.DeleteByTarget(ctx, labelentity.LabelTargetAlertSilence, id); err != nil {
		return err
	}
	err := a.DeleteById(ctx, id)
	if err == nil {
		a.InvalidateSilenceCache()
	}
	return err
}

// IsSilenced 判断规则在某资源上的告警是否处于静默中，结果按 版本号+规则+资源 缓存
func (a *alertSilenceAppImpl) IsSilenced(ctx context.Context, rule *entity.AlertRule, resourceId uint64) bool {
	cacheKey := fmt.Sprintf(silenceCacheKey, silenceCacheVersion(), rule.Id, resourceId)
	if cached := global_cache.GetStr(cacheKey); cached != "" {
		return cached == "1"
	}
	result := a.checkSilenced(ctx, rule, resourceId)
	cacheVal := "0"
	if result {
		cacheVal = "1"
	}
	_ = global_cache.Set(cacheKey, cacheVal, silenceCacheTTL)
	return result
}

func (a *alertSilenceAppImpl) checkSilenced(ctx context.Context, rule *entity.AlertRule, resourceId uint64) bool {
	silences, err := a.GetRepo().ListActive()
	if err != nil {
		// 静默规则查询失败时放行告警：漏报比误报危险
		logx.Errorf("[alert] list active silences error: %s", err.Error())
		return false
	}
	if len(silences) == 0 {
		return false
	}

	// 从 t_label_binding 批量加载所有活跃静默规则的标签
	silenceIds := make([]uint64, len(silences))
	for i, s := range silences {
		silenceIds[i] = s.Id
	}
	labelsMap, err := a.labelBindingApp.ListByTargets(ctx, labelentity.LabelTargetAlertSilence, silenceIds)
	if err != nil {
		logx.Errorf("[alert] list silence labels error: %s", err.Error())
		return false
	}

	ruleLabels := parseRuleLabels(rule.Labels)
	for _, s := range silences {
		silenceLabels := bindingsToMap(labelsMap[s.Id])
		if matchRuleLabelsMap(silenceLabels, ruleLabels) {
			logx.Infof("[alert] silence[%d:%s] matched rule[%d] resource[%d]", s.Id, s.Name, rule.Id, resourceId)
			return true
		}
	}
	return false
}

// InvalidateSilenceCache 自增静默缓存版本号，使全部已缓存的静默判定结果立即失效。
// 多实例共享 Redis 时对所有实例同时生效；本地缓存模式下作用于当前进程
func (a *alertSilenceAppImpl) InvalidateSilenceCache() {
	if _, err := global_cache.Incr(silenceCacheVerKey); err != nil {
		logx.Warnf("[alert] invalidate silence cache error: %s", err.Error())
	}
}

func silenceCacheVersion() int64 {
	return int64(global_cache.GetInt(silenceCacheVerKey))
}

// validateAlertSilence 静默规则保存校验。
//
// 静默是"会让告警消失"的操作，配置错的后果是告警凭空丢失且无任何提示，
// 因此时间范围和匹配标签都必须显式校验。
func validateAlertSilence(ctx context.Context, silence *entity.AlertSilence) error {
	if strings.TrimSpace(silence.Name) == "" {
		return errorx.NewBizI(ctx, imsg.ErrSilenceNameRequired)
	}
	if silence.Status != entity.AlertSilenceStatusEnable && silence.Status != entity.AlertSilenceStatusDisable {
		return errorx.NewBizI(ctx, imsg.ErrRuleStatusInvalid)
	}
	if silence.StartTime.IsZero() || silence.EndTime.IsZero() {
		return errorx.NewBizI(ctx, imsg.ErrSilenceTimeRequired)
	}
	if !silence.EndTime.Time.After(silence.StartTime.Time) {
		return errorx.NewBizI(ctx, imsg.ErrSilenceTimeInvalid)
	}

	if silence.MatchLabels == "" {
		silence.MatchLabels = "{}"
	}
	var labels map[string]string
	if err := json.Unmarshal([]byte(silence.MatchLabels), &labels); err != nil {
		return errorx.NewBizI(ctx, imsg.ErrSilenceMatchLabelsInvalid)
	}

	return nil
}
