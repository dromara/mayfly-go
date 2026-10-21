package entity

import (
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// bindQuery 走 Gin 的真实查询绑定路径（与 rc.BindQueryAndPage 同一条链：Ctx.BindQuery → gin.BindQuery）。
//
// 为什么必须走真实路径而不是自己反射读 tag：本组用例钉的是「界面上的搜索条件到底能不能绑进
// 查询实体」。Gin 在字段缺 form tag 时会**静默退回字段名**（大小写敏感），自己按 tag 校验会把
// 「压根没写 tag」与「tag 大小写写错」都判成通过——本轮正是这类缺陷漏到真机：搜索框还在、
// 请求照发、接口照返 200，但过滤条件被丢弃、列表恒返回全量，用户只会以为「没有匹配的数据」
func bindQuery[T any](t *testing.T, rawQuery string) *T {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?"+rawQuery, nil)
	data := new(T)
	require.NoError(t, c.ShouldBindQuery(data), "查询串 %q 绑定失败", rawQuery)
	return data
}

// 下面四个用例登记的查询维度，真源是各仓储 GetPageList 里实际拼出的谓词：
//
//	proposalRepoImpl.GetPageList  → title(Like)/status/risk/action/ownerAccountId/runId/procInstId
//	runRepoImpl.GetPageList       → triggerType/triggerRef/status/profileId/ownerAccountId/conversationId/parentRunId/feedback
//	agentTaskRepoImpl.GetPageList → name(Like)/status/profileId/ownerAccountId
//	triggerRepoImpl.GetPageList   → name(Like)/sourceType/enabled/ownerAccountId
//
// 本包被 persistence 反向依赖，不能 import 它取真源，故在此镜像登记：仓储新增过滤条件时
// 必须同步补一行。漏补不会红（少了断言而已），所以宁可把清单写全、写成一眼能对着仓储核的样子。
// 断言值全部取非零：缺 tag 时字段停在零值，非零期望值才拦得住

func TestProposalQueryDimensionsBindable(t *testing.T) {
	got := bindQuery[Proposal](t,
		"title=os-release&status=executed&risk=high&action=MachineCommandExec&ownerAccountId=7&runId=86&procInstId=42")

	require.Equal(t, "os-release", got.Title, "标题搜索框形同虚设：填了关键词却返回全部提案")
	require.Equal(t, ProposalStatusExecuted, got.Status, "按状态筛提案失效，界面下拉选什么都返回全量")
	require.Equal(t, ProposalRiskHigh, got.Risk, "按风险级筛提案失效，高危提案无法单独捞出")
	require.Equal(t, "MachineCommandExec", got.Action, "按变更动作筛提案失效")
	require.Equal(t, uint64(7), got.OwnerAccountId, "按归属账号筛提案失效，越权收敛只能靠 API 层兜底")
	require.Equal(t, uint64(86), got.RunId, "按来源运行筛提案失效，从运行回查它产出的提案断链")
	require.Equal(t, uint64(42), got.ProcInstId, "按审批工单筛提案失效，工单侧无法定位对应提案")
}

func TestRunQueryDimensionsBindable(t *testing.T) {
	got := bindQuery[Run](t,
		"triggerType=cron&triggerRef=12&status=succeeded&profileId=diagnostician&ownerAccountId=7&conversationId=5&parentRunId=3&feedback=down")

	require.Equal(t, TriggerTypeCron, got.TriggerType, "按触发类型筛运行失效")
	require.Equal(t, "12", got.TriggerRef, "按触发源筛运行失效，任务列表「执行记录」跳转会退化成全量")
	require.Equal(t, RunStatusSucceeded, got.Status, "按状态筛运行失效")
	require.Equal(t, "diagnostician", got.ProfileId, "按装配档筛运行失效")
	require.Equal(t, uint64(7), got.OwnerAccountId, "按运行身份筛运行失效")
	require.Equal(t, uint64(5), got.ConversationId, "按会话筛运行失效，无法列出某个会话产生过的运行")
	require.Equal(t, uint64(3), got.ParentRunId, "按父运行筛子运行失效，编排树只能拿到全量运行自己拼")
	require.Equal(t, FeedbackDown, got.Feedback, "按人类反馈筛运行失效，无法把点踩过的运行单独捞出复盘")
}

func TestAgentTaskQueryDimensionsBindable(t *testing.T) {
	got := bindQuery[AgentTask](t, "name=disk&status=-1&profileId=diagnostician&ownerAccountId=7")

	require.Equal(t, "disk", got.Name, "任务名搜索框形同虚设")
	require.Equal(t, AgentTaskStatusDisable, got.Status, "按启停状态筛任务失效，停用的任务无法单独捞出")
	require.Equal(t, "diagnostician", got.ProfileId, "按装配档筛任务失效")
	require.Equal(t, uint64(7), got.OwnerAccountId, "按归属账号筛任务失效")
}

func TestTriggerQueryDimensionsBindable(t *testing.T) {
	got := bindQuery[Trigger](t, "name=cpu&sourceType=alert&enabled=-1&ownerAccountId=7")

	require.Equal(t, "cpu", got.Name, "触发器名搜索框形同虚设")
	require.Equal(t, "alert", got.SourceType, "按来源类型筛触发器失效")
	require.Equal(t, TriggerDisabled, got.Enabled, "按启停筛触发器失效，已禁用的触发器无法单独捞出")
	require.Equal(t, uint64(7), got.OwnerAccountId, "按归属账号筛触发器失效")
}

// 带 form tag 的字段必须与 json tag 同名。
//
// 前端一律按接口返回的 json 名回填搜索表单、再按同名发查询参数（见各 *List.vue 的 state.params），
// 两者不同名等于给这个查询维度另起了一个只有后端知道的名字：界面上照样填、参数照样发、
// 后端照样收不到，症状与「缺 form tag」完全一致，但读 tag 的自检会放过它
func TestFormTagsMatchJsonNames(t *testing.T) {
	for _, tc := range []struct {
		name  string
		model any
	}{
		{"Proposal", new(Proposal)},
		{"Run", new(Run)},
		{"AgentTask", new(AgentTask)},
		{"Trigger", new(Trigger)},
	} {
		walkFields(reflect.TypeOf(tc.model).Elem(), func(field reflect.StructField) {
			form := field.Tag.Get("form")
			if form == "" || form == "-" {
				return
			}
			jsonName := strings.Split(field.Tag.Get("json"), ",")[0]
			require.Equal(t, jsonName, form,
				"%s.%s 的 form tag 与 json tag 不同名，前端按 json 名发的查询参数后端收不到", tc.name, field.Name)
		})
	}
}

// walkFields 深度遍历结构体字段，含匿名嵌入的基座模型（model.Model/ModelNLD/ExtraData）：
// 查询维度也可能落在嵌入结构体上，只看外层字段会漏
func walkFields(typ reflect.Type, visit func(reflect.StructField)) {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Anonymous {
			embedded := field.Type
			if embedded.Kind() == reflect.Pointer {
				embedded = embedded.Elem()
			}
			if embedded.Kind() == reflect.Struct {
				walkFields(embedded, visit)
				continue
			}
		}
		visit(field)
	}
}
