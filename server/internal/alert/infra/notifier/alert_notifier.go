package notifier

import (
	"context"
	"fmt"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/repository"
	"mayfly-go/internal/alert/domain/service"
	"mayfly-go/internal/alert/imsg"
	msgapp "mayfly-go/internal/msg/application"
	msgdto "mayfly-go/internal/msg/application/dto"
	msgentity "mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/msgx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	datetimeFmt = "2006-01-02 15:04:05"

	// tmplCodeAlertNotify 告警通知模板编码
	tmplCodeAlertNotify = "alert_notify"
	// tmplCodeAlertRecover 告警恢复通知模板编码。
	// 必须与告警模板分离：恢复场景没有 currentValue/threshold/triggerCount 等语义，
	// 复用告警模板会让 text/template 把缺失字段渲染成 <no value>。
	tmplCodeAlertRecover = "alert_recover"

	// tmplCacheTTL 模板缓存时长，支持不重启修改模板
	tmplCacheTTL = 5 * time.Minute
)

type alertNotifierImpl struct {
	msgTmplApp    msgapp.MsgTmpl            `inject:"T"`
	msgChannelApp msgapp.MsgChannel         `inject:"T"`
	notifyLogRepo repository.AlertNotifyLog `inject:"T"`

	// 模板缓存：code -> 缓存条目
	tmplMu    sync.RWMutex
	tmplCache map[string]tmplCacheEntry
}

type tmplCacheEntry struct {
	tmpl     *msgentity.MsgTmpl
	cachedAt time.Time
}

var _ service.AlertNotifier = (*alertNotifierImpl)(nil)

func NewAlertNotifier() service.AlertNotifier {
	return &alertNotifierImpl{
		tmplCache: make(map[string]tmplCacheEntry),
	}
}

// NotifyWithTarget 使用指定通知目标发送告警
func (n *alertNotifierImpl) NotifyWithTarget(ctx context.Context, event *entity.AlertEvent, rule *entity.AlertRule, target service.NotifyTarget) (int, error) {
	channels, err := n.loadChannelsByIds(target.ChannelIds)
	if err != nil {
		return 0, err
	}
	if len(channels) == 0 {
		logx.Warnf("[alert] rule[%d] has no available channel, skip alert notify", rule.Id)
		return 0, nil
	}

	tmpl, err := n.findTmpl(tmplCodeAlertNotify, defaultAlertNotifyTmpl)
	if err != nil {
		return 0, err
	}
	sendErr := n.doSend(ctx, tmpl, channels, alertNotifyParams(event, rule), target.ReceiverIds)
	n.recordNotifyLogs(event.Id, 0, channels, sendErr)
	return len(channels), sendErr
}

// NotifyRecoverWithTarget 使用指定通知目标发送恢复通知
func (n *alertNotifierImpl) NotifyRecoverWithTarget(ctx context.Context, event *entity.AlertEvent, rule *entity.AlertRule, target service.NotifyTarget) (int, error) {
	channels, err := n.loadChannelsByIds(target.ChannelIds)
	if err != nil {
		return 0, err
	}
	if len(channels) == 0 {
		logx.Warnf("[alert] rule[%d] has no available channel, skip recover notify", rule.Id)
		return 0, nil
	}

	tmpl, err := n.findTmpl(tmplCodeAlertRecover, defaultAlertRecoverTmpl)
	if err != nil {
		return 0, err
	}
	sendErr := n.doSend(ctx, tmpl, channels, alertRecoverParams(event, rule), target.ReceiverIds)
	n.recordNotifyLogs(event.Id, 0, channels, sendErr)
	return len(channels), sendErr
}

// alertNotifyParams 告警通知模板参数。
// 时间与数值在此处完成格式化，避免 time.Time/float64 直接渲染出 RFC3339Nano 与超长小数。
func alertNotifyParams(event *entity.AlertEvent, rule *entity.AlertRule) map[string]any {
	end := time.Now()
	params := baseParams(event, rule, end)
	params["currentValue"] = formatMetricValue(event)
	params["threshold"] = formatThreshold(event)
	params["lastTime"] = formatTime(event.LastTriggerTime)
	params["triggerCount"] = event.TriggerCount
	return params
}

// alertRecoverParams 恢复通知模板参数，与恢复模板占位符严格对应
func alertRecoverParams(event *entity.AlertEvent, rule *entity.AlertRule) map[string]any {
	end := time.Now()
	if event.RecoverTime != nil {
		end = *event.RecoverTime
	}
	params := baseParams(event, rule, end)
	params["recoverTime"] = formatTime(end)
	params["durationText"] = formatDuration(event.FirstTriggerTime, end)
	params["triggerCount"] = event.TriggerCount
	params["notifyCount"] = event.NotifyCount
	return params
}

// baseParams 告警/恢复通知共用的模板参数
func baseParams(event *entity.AlertEvent, rule *entity.AlertRule, end time.Time) map[string]any {
	return map[string]any{
		"ruleName":     rule.Name,
		"resourceName": displayName(event),
		"metric":       event.Metric,
		"priority":     event.Priority.Code(),
		"firstTime":    formatTime(event.FirstTriggerTime),
		"endTime":      formatTime(end),
	}
}

// displayName 资源展示名，历史事件可能缺失 resourceName，回退为资源标识避免通知内容空白
func displayName(event *entity.AlertEvent) string {
	if event.ResourceName != "" {
		return event.ResourceName
	}
	return fmt.Sprintf("resource#%d", event.ResourceId)
}

// formatMetricValue 按指标语义格式化当前值：百分比指标带 % 后缀并保留两位小数
func formatMetricValue(event *entity.AlertEvent) string {
	if def, ok := service.FindMetric(event.ResourceType, event.Metric); ok {
		if def.Unit != "" {
			return fmt.Sprintf("%.2f%s", event.CurrentValue, def.Unit)
		}
	}
	return strconv.FormatFloat(event.CurrentValue, 'f', 2, 64)
}

// formatThreshold 将阈值描述格式化为人类可读文本。
// 输入格式："cpu_rate gt 1.00" → 输出："CPU 使用率 > 1.00%"
func formatThreshold(event *entity.AlertEvent) string {
	if event.Threshold == "" {
		return "-"
	}
	// 解析 "metric operator value" 格式
	parts := strings.SplitN(event.Threshold, " ", 3)
	if len(parts) < 3 {
		return event.Threshold
	}
	metric := parts[0]
	operator := parts[1]
	value := parts[2]

	// 获取指标定义（用于单位等元信息）
	def, ok := service.FindMetric(event.ResourceType, metric)

	// 指标名：优先使用注册表中的通知标签（人类可读 + i18n），未知指标回退为原始 key
	metricLabel := metric
	if msgId, ok := metricLabelMap[metric]; ok {
		metricLabel = i18n.T(msgId)
	}

	// 操作符映射：技术符号 → 人类可读符号
	operatorSymbol := operatorSymbolMap[operator]
	if operatorSymbol == "" {
		operatorSymbol = operator
	}

	// 格式化数值：整数不补小数，浮点数保留两位
	var formattedValue string
	if v, err := strconv.ParseFloat(value, 64); err == nil {
		if v == float64(int64(v)) {
			formattedValue = strconv.FormatInt(int64(v), 10)
		} else {
			formattedValue = strconv.FormatFloat(v, 'f', 2, 64)
		}
	} else {
		formattedValue = value
	}

	// 单位：从指标注册表获取
	unit := ""
	if ok {
		unit = def.Unit
	}

	return fmt.Sprintf("%s %s %s%s", metricLabel, operatorSymbol, formattedValue, unit)
}

// operatorSymbolMap 比较操作符：技术符号 → 人类可读符号
// 参考 Grafana/Prometheus Alertmanager 的展示方式
var operatorSymbolMap = map[string]string{
	"gt":  ">",
	"gte": "≥",
	"lt":  "<",
	"lte": "≤",
	"eq":  "=",
	"neq": "≠",
}

// metricLabelMap 指标名 → 通知用人类可读标签（i18n）。
// 新增指标只需在此追加一行，无需修改 formatThreshold 等现有逻辑。
// 未知指标回退为原始 key（如 "custom_metric"），保证向前兼容。
var metricLabelMap = map[string]i18n.MsgId{
	"cpu_rate":   imsg.MetricCpuRate,
	"mem_rate":   imsg.MetricMemRate,
	"disk_usage": imsg.MetricDiskUsage,
	"status":     imsg.MetricStatus,
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format(datetimeFmt)
}

// formatDuration 格式化告警持续时长（中文语义由模板承载，此处只给可读时长）
func formatDuration(start, end time.Time) string {
	if start.IsZero() {
		return "-"
	}
	d := end.Sub(start)
	if d < 0 {
		d = 0
	}
	return d.Round(time.Second).String()
}

// loadChannelsByIds 按渠道ID列表加载渠道，并过滤掉已禁用渠道
func (n *alertNotifierImpl) loadChannelsByIds(channelIds []uint64) ([]*msgentity.MsgChannel, error) {
	if len(channelIds) == 0 {
		return nil, nil
	}
	channels, err := n.msgChannelApp.GetByIds(channelIds)
	if err != nil {
		return nil, err
	}
	if len(channels) != len(channelIds) {
		logx.Warnf("[alert] configured %d channels but got %d (deleted?)", len(channelIds), len(channels))
	}
	enabled := make([]*msgentity.MsgChannel, 0, len(channels))
	for _, channel := range channels {
		if channel.Status != msgentity.ChannelStatusEnable {
			logx.Warnf("[alert] channel[%d:%s] is disabled, skip", channel.Id, channel.Code)
			continue
		}
		enabled = append(enabled, channel)
	}
	return enabled, nil
}

func (n *alertNotifierImpl) doSend(ctx context.Context, tmpl *msgentity.MsgTmpl, channels []*msgentity.MsgChannel, params map[string]any, receiverIds []int64) error {
	receiverUints := make([]uint64, 0, len(receiverIds))
	for _, id := range receiverIds {
		if id > 0 {
			receiverUints = append(receiverUints, uint64(id))
		}
	}

	// 渠道发送为异步 fire-and-forget，此处返回 nil 仅代表已投递给发送器
	return n.msgTmplApp.SendMsg(ctx, &msgdto.MsgTmplSend{
		Tmpl:        tmpl,
		Channels:    channels,
		Params:      params,
		ReceiverIds: receiverUints,
	})
}

// recordNotifyLogs 记录通知发送日志到 t_alert_notify_log
// 由于底层 SendMsg 为 fire-and-forget，此处记录的是"投递"状态：
// SendMsg 返回 nil 视为投递成功（status=1），否则视为投递失败（status=2）
func (n *alertNotifierImpl) recordNotifyLogs(eventId, policyId uint64, channels []*msgentity.MsgChannel, sendErr error) {
	if n.notifyLogRepo == nil || len(channels) == 0 {
		return
	}
	now := time.Now()
	status := int8(1) // 投递成功
	errMsg := ""
	if sendErr != nil {
		status = 2 // 投递失败
		errMsg = sendErr.Error()
		if len(errMsg) > 500 {
			errMsg = errMsg[:500]
		}
	}
	for _, ch := range channels {
		log := &entity.AlertNotifyLog{
			EventId:     eventId,
			PolicyId:    policyId,
			ChannelId:   ch.Id,
			ChannelName: ch.Name,
			Status:      status,
			ErrorMsg:    errMsg,
			SendTime:    now,
		}
		if createErr := n.notifyLogRepo.Create(log); createErr != nil {
			logx.Warnf("[alert] record notify log failed: event=%d channel=%d err=%v", eventId, ch.Id, createErr)
		}
	}
}

// findTmpl 按编码查找消息模板（带 TTL 缓存）；模板不存在时回退到内置默认模板
func (n *alertNotifierImpl) findTmpl(code string, fallback func() *msgentity.MsgTmpl) (*msgentity.MsgTmpl, error) {
	n.tmplMu.RLock()
	if entry, ok := n.tmplCache[code]; ok && entry.tmpl != nil && time.Since(entry.cachedAt) < tmplCacheTTL {
		tmpl := entry.tmpl
		n.tmplMu.RUnlock()
		return tmpl, nil
	}
	n.tmplMu.RUnlock()

	tmpls, err := n.msgTmplApp.ListByCond(model.NewCond().Eq("code", code))
	if err != nil {
		return nil, err
	}
	result := fallback()
	if len(tmpls) > 0 {
		result = tmpls[0]
	}

	n.tmplMu.Lock()
	n.tmplCache[code] = tmplCacheEntry{tmpl: result, cachedAt: time.Now()}
	n.tmplMu.Unlock()

	return result, nil
}

func defaultAlertNotifyTmpl() *msgentity.MsgTmpl {
	return &msgentity.MsgTmpl{
		Code:    tmplCodeAlertNotify,
		Name:    i18n.T(imsg.TmplAlertNotifyName),
		Title:   i18n.T(imsg.TmplAlertNotifyTitle),
		MsgType: msgx.MsgTypeText,
		Status:  msgentity.TmplStatusEnable,
		Tmpl: "【告警通知】\n规则: {{.ruleName}}\n资源: {{.resourceName}}\n指标: {{.metric}}\n" +
			"当前值: {{.currentValue}}\n阈值: {{.threshold}}\n优先级: {{.priority}}\n" +
			"首次触发: {{.firstTime}}\n最近触发: {{.lastTime}}\n累计触发: {{.triggerCount}}次",
	}
}

func defaultAlertRecoverTmpl() *msgentity.MsgTmpl {
	return &msgentity.MsgTmpl{
		Code:    tmplCodeAlertRecover,
		Name:    i18n.T(imsg.TmplAlertRecoverName),
		Title:   i18n.T(imsg.TmplAlertRecoverTitle),
		MsgType: msgx.MsgTypeText,
		Status:  msgentity.TmplStatusEnable,
		Tmpl: "【告警恢复】\n规则: {{.ruleName}}\n资源: {{.resourceName}}\n指标: {{.metric}}\n优先级: {{.priority}}\n" +
			"首次触发: {{.firstTime}}\n恢复时间: {{.recoverTime}}\n持续时长: {{.durationText}}\n" +
			"累计触发: {{.triggerCount}}次\n通知次数: {{.notifyCount}}次",
	}
}
