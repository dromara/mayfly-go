package monitor

import (
	"context"
	"encoding/json"
	"testing"

	"mayfly-go/internal/ai/application/resource"
	alertservice "mayfly-go/internal/alert/domain/service"
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/model"

	"github.com/cloudwego/eino/components/tool"
	"github.com/stretchr/testify/require"
)

// fakeEvaluator 单测用评估器：仅提供指标元信息与固定取值，验证工具层容错契约。
//
// 挂在 machine 类型上——授权面要求资源类型必须在 consts 登记键名，未登记的类型根本进不到取值环节。
// missingResource=true 时模拟资源不存在（验证资源级容错分支）
type fakeEvaluator struct {
	missingResource bool
	// shortItems 模拟评估器违约少返回条目；noEval 模拟整体未评估。
	// 两者都是「缺口不得被静默填 0」的回归样本
	shortItems bool
	noEval     bool
}

func (fakeEvaluator) ResourceType() int8 { return consts.ResourceTypeMachine }

func (fakeEvaluator) Metrics() []alertservice.MetricDefinition {
	return []alertservice.MetricDefinition{
		{Key: "cpu_rate", Unit: "%"},
		{Key: "mem_rate", Unit: "%"},
	}
}

func (e fakeEvaluator) Evaluate(_ context.Context, param *alertservice.EvalParam) (*alertservice.EvalResult, error) {
	res := &alertservice.EvalResult{Evaluated: !e.noEval}
	if e.noEval {
		return res, nil
	}
	for _, item := range param.Rule.Condition.Items {
		res.Items = append(res.Items, alertservice.ConditionEval{Metric: item.Metric, CurrentValue: 42.5})
		if e.shortItems {
			break // 只回第一条，其余项成为缺口
		}
	}
	return res, nil
}
func (fakeEvaluator) ResourceName(uint64) string           { return "fake" }
func (e fakeEvaluator) ResourceExists(uint64) bool         { return !e.missingResource }
func (fakeEvaluator) ResourceIdsByCodes([]string) []uint64 { return nil }

// fakeProvider 测试用资源可见面提供者（授权判定的数据源，与 resourcetool 的测试桩同一形态）
type fakeProvider struct {
	typ string
	ids []string
}

func (p *fakeProvider) Type() string { return p.typ }

func (p *fakeProvider) List(context.Context, uint64) ([]*resource.Resource, error) {
	out := make([]*resource.Resource, 0, len(p.ids))
	for _, id := range p.ids {
		out = append(out, &resource.Resource{Type: p.typ, Id: id, Name: "r" + id})
	}
	return out, nil
}

// withTestVisibility 装配授权前提：账号 1 在 machine 类型下仅可见 id 1/2，并注册 fake 评估器。
//
// 注册表与 loader 都是包级可变状态，必须成对还原，否则会污染同包后续用例
func withTestVisibility(t *testing.T) {
	t.Helper()

	app := resource.NewApp()
	app.RegisterProvider(&fakeProvider{typ: resource.TypeMachine, ids: []string{"1", "2"}})
	SetResourceAppLoader(func() resource.App { return app })
	t.Cleanup(func() { SetResourceAppLoader(resource.GetApp) })

	alertservice.RegisterEvaluator(fakeEvaluator{})
	t.Cleanup(func() { alertservice.UnregisterEvaluator(consts.ResourceTypeMachine) })
}

// authCtx 带运行身份（owner=1）的 ctx：工具授权判定的唯一依据
func authCtx() context.Context {
	return contextx.WithLoginAccount(context.Background(), &model.LoginAccount{Id: 1, Username: "admin"})
}

func invokableTool(t *testing.T, created tool.BaseTool) tool.InvokableTool {
	t.Helper()
	invokable, ok := created.(tool.InvokableTool)
	require.True(t, ok, "tool must be invokable")
	return invokable
}

// TestQueryMetrics 容错契约：空指标列可用 / 未知指标不报错并标记 / 混合请求仅评估已知项
func TestQueryMetrics(t *testing.T) {
	withTestVisibility(t)
	metrics, err := GetMetricsQuery()
	require.NoError(t, err)
	invokable := invokableTool(t, metrics)
	ctx := authCtx()

	// 1) metrics 为空：列出全部可用指标
	raw, err := invokable.InvokableRun(ctx, `{"resourceType":1,"resourceId":1,"metrics":[]}`)
	require.NoError(t, err)
	// 注意：每个用例必须用全新的 out 接收反序列化结果——json.Unmarshal 会复用既有
	// 切片的背底数组，共享 out 会让上一用例的 unit/err 字段残留到下一用例（曾致
	// 「下标错位回归用例」误报：生产代码正确而测试被残留字段污染）
	var out MetricsQueryOutput
	require.NoError(t, json.Unmarshal([]byte(raw), &out))
	require.Equal(t, []string{"cpu_rate", "mem_rate"}, out.Available)

	// 2) 混合已知/未知指标：未知不报错，仅标记 Err 并附可用列表；已知返回值
	raw, err = invokable.InvokableRun(ctx, `{"resourceType":1,"resourceId":1,"metrics":["cpu_rate","nope"]}`)
	require.NoError(t, err)
	out = MetricsQueryOutput{}
	require.NoError(t, json.Unmarshal([]byte(raw), &out))
	require.Len(t, out.Values, 2)
	require.Equal(t, 42.5, out.Values[0].Value)
	require.Empty(t, out.Values[0].Err)
	require.NotEmpty(t, out.Values[1].Err)
	require.Equal(t, "nope", out.Values[1].Metric)
	require.NotEmpty(t, out.Available)

	// 3) 全部未知：不发起评估，返回错误标记 + 可用列表
	raw, err = invokable.InvokableRun(ctx, `{"resourceType":1,"resourceId":1,"metrics":["x","y"]}`)
	require.NoError(t, err)
	out = MetricsQueryOutput{}
	require.NoError(t, json.Unmarshal([]byte(raw), &out))
	require.Len(t, out.Values, 2)
	for _, v := range out.Values {
		require.NotEmpty(t, v.Err)
	}
	require.NotEmpty(t, out.Available)

	// 4) 未知指标在前（下标错位回归）：已知指标值不得写进未知指标名下，
	// 未知项的 Err 不得被空串覆盖
	raw, err = invokable.InvokableRun(ctx, `{"resourceType":1,"resourceId":1,"metrics":["nope","cpu_rate"]}`)
	require.NoError(t, err)
	out = MetricsQueryOutput{}
	require.NoError(t, json.Unmarshal([]byte(raw), &out))
	require.Len(t, out.Values, 2)
	require.Equal(t, "nope", out.Values[0].Metric)
	require.NotEmpty(t, out.Values[0].Err)
	require.Zero(t, out.Values[0].Value)
	require.Equal(t, "cpu_rate", out.Values[1].Metric)
	require.Empty(t, out.Values[1].Err)
	require.Equal(t, 42.5, out.Values[1].Value)

	// 5) 资源不存在：不报错，返回资源级错误标记 + 可用指标列表（Agent 可自纠后重试）
	alertservice.RegisterEvaluator(fakeEvaluator{missingResource: true})
	raw, err = invokable.InvokableRun(ctx, `{"resourceType":1,"resourceId":1,"metrics":["cpu_rate"]}`)
	require.NoError(t, err)
	out = MetricsQueryOutput{}
	require.NoError(t, json.Unmarshal([]byte(raw), &out))
	require.NotEmpty(t, out.ResourceErr)
	require.Contains(t, out.ResourceErr, "not found")
	require.Empty(t, out.Values)
	require.NotEmpty(t, out.Available)
}

// TestQueryMetricsAuthorization 授权面（fail-closed）：无运行身份、资源不在可见面、
// 类型未登记键名、缺 resourceId 一律拒绝。
//
// 这条链没有权限码可依赖（HTTP 侧靠 alert:event 把关的是人类操作者），不判定的结果就是
// 任意一次派生运行都能读走任意资源的实时指标
func TestQueryMetricsAuthorization(t *testing.T) {
	withTestVisibility(t)
	metrics, err := GetMetricsQuery()
	require.NoError(t, err)
	invokable := invokableTool(t, metrics)

	// 无运行身份：不得猜测为管理员
	_, err = invokable.InvokableRun(context.Background(), `{"resourceType":1,"resourceId":1,"metrics":["cpu_rate"]}`)
	require.ErrorContains(t, err, "no login account")

	// 资源不在该账号可见面内（可见集合只含 1/2）
	_, err = invokable.InvokableRun(authCtx(), `{"resourceType":1,"resourceId":999,"metrics":["cpu_rate"]}`)
	require.ErrorContains(t, err, "not accessible")

	// 未登记键名的类型（库名 22）：不退回数字串去查可见面
	_, err = invokable.InvokableRun(authCtx(), `{"resourceType":22,"resourceId":1,"metrics":["cpu_rate"]}`)
	require.ErrorContains(t, err, "unknown resource type")

	// 缺 resourceId：指标必须落到具体资源
	_, err = invokable.InvokableRun(authCtx(), `{"resourceType":1,"resourceId":0,"metrics":["cpu_rate"]}`)
	require.ErrorContains(t, err, "resourceId is required")

	// 可见资源正常放行（证明上面几条不是「一律拒绝」造成的假绿）
	raw, err := invokable.InvokableRun(authCtx(), `{"resourceType":1,"resourceId":1,"metrics":["cpu_rate"]}`)
	require.NoError(t, err)
	var out MetricsQueryOutput
	require.NoError(t, json.Unmarshal([]byte(raw), &out))
	require.Empty(t, out.ResourceErr)
	require.Len(t, out.Values, 1)
}

// TestQueryMetricsMarksMissingValues 评估器少返回条目 / 整体未评估时，缺口必须显式标错：
// 否则该项停在「value 0 + 无 err + 带单位」，与真的测到 0% 完全同形，会被当成事实进结论
func TestQueryMetricsMarksMissingValues(t *testing.T) {
	t.Run("少返回条目", func(t *testing.T) {
		withTestVisibility(t)
		alertservice.RegisterEvaluator(fakeEvaluator{shortItems: true})

		metrics, err := GetMetricsQuery()
		require.NoError(t, err)
		raw, err := invokableTool(t, metrics).InvokableRun(authCtx(), `{"resourceType":1,"resourceId":1,"metrics":["cpu_rate","mem_rate"]}`)
		require.NoError(t, err)
		var out MetricsQueryOutput
		require.NoError(t, json.Unmarshal([]byte(raw), &out))
		require.Len(t, out.Values, 2)
		require.Empty(t, out.Values[0].Err, "真取到值的项不得被误标")
		require.Equal(t, 42.5, out.Values[0].Value)
		require.NotEmpty(t, out.Values[1].Err, "缺口必须标错")
		require.Zero(t, out.Values[1].Value)
	})

	t.Run("整体未评估", func(t *testing.T) {
		withTestVisibility(t)
		alertservice.RegisterEvaluator(fakeEvaluator{noEval: true})

		metrics, err := GetMetricsQuery()
		require.NoError(t, err)
		raw, err := invokableTool(t, metrics).InvokableRun(authCtx(), `{"resourceType":1,"resourceId":1,"metrics":["cpu_rate","mem_rate"]}`)
		require.NoError(t, err)
		var out MetricsQueryOutput
		require.NoError(t, json.Unmarshal([]byte(raw), &out))
		require.Len(t, out.Values, 2)
		for _, v := range out.Values {
			require.Contains(t, v.Err, "did not evaluate", "未评估必须逐项标出原因，不能留零值假读数")
			require.Zero(t, v.Value)
		}
	})
}

// TestQueryAlertHistoryAuthorization 告警历史必须显式声明资源类型才能安全收窄：
// 告警事件的 resource_id 在不同资源类型下会重复，只按 id 集合过滤会跨类型误放行
func TestQueryAlertHistoryAuthorization(t *testing.T) {
	withTestVisibility(t)
	history, err := GetAlertHistory()
	require.NoError(t, err)
	invokable := invokableTool(t, history)

	_, err = invokable.InvokableRun(authCtx(), `{"resourceId":1}`)
	require.ErrorContains(t, err, "resourceType is required")

	_, err = invokable.InvokableRun(context.Background(), `{"resourceType":1}`)
	require.ErrorContains(t, err, "no login account")

	_, err = invokable.InvokableRun(authCtx(), `{"resourceType":22}`)
	require.ErrorContains(t, err, "unknown resource type")
}

// TestResourceTypeCodesDesc 工具说明里的类型编码表必须由 consts 真源生成：
// 此前清单手写在 struct tag 里且已与实际映射漂移（说明列 6 类、映射 8 类）
func TestResourceTypeCodesDesc(t *testing.T) {
	desc := resourceTypeCodesDesc()
	for _, rt := range consts.ResourceTypeWithKey() {
		key, ok := consts.ResourceTypeKey(rt)
		require.True(t, ok)
		require.Contains(t, desc, key, "工具说明必须覆盖每个已登记键名的资源类型")
	}
}
