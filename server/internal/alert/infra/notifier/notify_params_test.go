package notifier

import (
	"regexp"
	"strconv"
	"testing"
	"time"

	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/service"
	"mayfly-go/internal/alert/imsg"
	"mayfly-go/pkg/i18n"

	"github.com/stretchr/testify/require"
)

// 通知正文的「占位符 ↔ 参数键」必须同源。
//
// 本仓已在 agentops 诊断通知上真机踩过一次：模板里写了占位符、params 却用了另一套键名，
// text/template 把缺失字段静默渲染成 `<no value>` 发给用户，而接口、日志、界面全都没有任何异常。
// 那次的解法是把键名与模板收进同一处；这里对告警正文做同等约束：模板多一个占位符就必须同时
// 给参数补一个键，否则本用例红（本轮新增 notifyStage 正是靠它兜住的）

// tmplPlaceholders 取正文里的 `{{.x}}` 字段名集合
func tmplPlaceholders(body string) []string {
	var names []string
	for _, m := range regexp.MustCompile(`\{\{\s*\.([A-Za-z0-9_]+)\s*\}\}`).FindAllStringSubmatch(body, -1) {
		names = append(names, m[1])
	}
	return names
}

func sampleEventAndRule() (*entity.AlertEvent, *entity.AlertRule) {
	now := time.Now()
	return &entity.AlertEvent{
		RuleId: 7, RuleName: "磁盘使用率", ResourceType: 1, ResourceId: 1, ResourceName: "aliyun",
		Priority: entity.AlertPriorityHigh, Metric: "disk_usage", CurrentValue: 91.5, Threshold: ">90",
		FirstTriggerTime: now.Add(-time.Hour), LastTriggerTime: now,
	}, &entity.AlertRule{
		Name: "磁盘使用率", Priority: entity.AlertPriorityHigh,
	}
}

func TestAlertNotifyParamsCoverTemplatePlaceholders(t *testing.T) {
	event, rule := sampleEventAndRule()

	cases := []struct {
		name   string
		body   string
		params map[string]any
	}{
		{"告警通知", i18n.T(imsg.TmplAlertNotifyBody), alertNotifyParams(event, rule, service.NotifyTarget{})},
		{"恢复通知", i18n.T(imsg.TmplAlertRecoverBody), alertRecoverParams(event, rule)},
	}
	for _, tc := range cases {
		names := tmplPlaceholders(tc.body)
		require.NotEmptyf(t, names, "%s 正文里没解析到任何占位符，本判据在空跑", tc.name)
		for _, name := range names {
			_, ok := tc.params[name]
			require.Truef(t, ok, "%s 模板用了 {{.%s}} 但参数里没有该键：会静默渲染成 <no value> 发给用户", tc.name, name)
		}
	}
}

// TestBothLanguagesKeepPlaceholderParity 两份语言的正文必须用同一集占位符。
//
// 上面那条用例走 i18n.T，永远只取默认语言（zh-cn）的正文：下一轮只在 en.go 里加一个占位符
// 时它不会响，英文用户会直接收到 <no value>——正是本注释里举的 agentops 事故形态。两条链要分开钉：
// 参数键覆盖默认语言正文（上一条），中英两份正文不得长出不一致（本条）
func TestBothLanguagesKeepPlaceholderParity(t *testing.T) {
	for _, tmpl := range []struct {
		name string
		key  i18n.MsgId
	}{{"告警通知", imsg.TmplAlertNotifyBody}, {"恢复通知", imsg.TmplAlertRecoverBody}} {
		zh := tmplPlaceholders(i18n.TL(i18n.Zh_CN, tmpl.key))
		en := tmplPlaceholders(i18n.TL(i18n.En, tmpl.key))
		require.NotEmpty(t, zh, tmpl.name+" 中文正文没解析到占位符，本判据在空跑")
		require.ElementsMatch(t, zh, en,
			tmpl.name+" 中英文正文的占位符集不一致：少一侧就会让那边语言的用户收到 <no value>")
	}
}

// TestNotifyStageDistinguishesEscalation 升级策略触发的通知必须带可区分的阶段文案与级别。
//
// 判据取渠道策略归因而非 event.EscalationLvl：升级是先发通知后推进级别，第 1 级通知时该字段
// 仍为 0；拿它判会漏报第一次升级（本用例同时钉住这个 0 值场景）
func TestNotifyStageDistinguishesEscalation(t *testing.T) {
	event, rule := sampleEventAndRule()
	// 0 基级别：第一条升级规则命中时库里仍是 0，界面必须报「第 1 级」而不是「第 0 级」
	event.EscalationLvl = 0

	escalationTarget := service.NotifyTarget{ChannelIds: []uint64{4}}.WithEscalationBinding(11, []uint64{4})
	notifyTarget := service.NotifyTarget{ChannelIds: []uint64{4}}.WithNotifyPolicyBinding(2, []uint64{4})

	escalated := alertNotifyParams(event, rule, escalationTarget)["notifyStage"].(string)
	plain := alertNotifyParams(event, rule, notifyTarget)["notifyStage"].(string)
	unbound := alertNotifyParams(event, rule, service.NotifyTarget{})["notifyStage"].(string)

	require.NotEqual(t, escalated, plain, "升级通知与首报/重复告警的阶段文案必须可区分（相同即本缺陷原样存在）")
	require.Contains(t, escalated, strconv.Itoa(event.EscalationLvl+1), "阶段文案必须带级别，且 0 基索引要对运维显示为第 1 级")
	require.Equal(t, plain, unbound, "无渠道归因时按普通告警处理，不得凭空造出升级语义")
	// 服务端 i18n 的插值语法是 {{.name}}，写成 {name}（前端 vue-i18n 的写法）不会报错，
	// 而是把花括号连同占位符名一起发给用户 —— 本断言守的就是这种跨端语法混用
	require.NotContains(t, escalated, "{{", "升级阶段文案的占位符没被插值，用户会看到花括号原文")
	require.NotContains(t, plain, "{{", "普通告警阶段文案的占位符没被插值")
}

// TestEscalationBindingIsNonEmpty 上一用例的前提：升级归因确实登记进了 target。
//
// 归因 map 为空时 notifyStageText 会静默走「普通告警」分支，上面的 NotEqual 断言就会因为
// 两边都是同一句而红——但红的原因看起来像"文案没区分"。先钉住归因本身，失败时能一眼分清是哪一层
func TestEscalationBindingIsNonEmpty(t *testing.T) {
	target := service.NotifyTarget{ChannelIds: []uint64{4}}.WithEscalationBinding(11, []uint64{4})
	require.NotEmpty(t, target.PolicyByChannel, "升级归因没登记进 target，上面的阶段判据失去输入")
	require.Equal(t, entity.NotifyPolicyTypeEscalation, target.PolicyOfChannel(4).Type)
}
