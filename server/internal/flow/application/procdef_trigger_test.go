package application

import (
	"context"
	"sync"
	"testing"

	"mayfly-go/internal/flow/application/dto"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/trigger"

	"github.com/stretchr/testify/require"
)

// simulatePathBiz 本文件专用的场景：字段带 Resolve，用来观察求值上下文里到底有什么
const simulatePathBiz = "test_simulate_context_flow"

// simulateSeenCodePaths 记录派生字段被求值时看到的资源路径
var simulateSeenCodePaths []string

var registerSimulateScenarioOnce sync.Once

func registerSimulateScenario() {
	registerSimulateScenarioOnce.Do(func() {
		RegisterTriggerBiz(trigger.BizMeta{
			BizType:    simulatePathBiz,
			Approvable: true,
			Fields: []trigger.TriggerField{
				{Key: "cmdText", TitleKey: "flow.field.cmdText", Group: "risk", Type: trigger.TypeString},
				{
					Key: "blacklisted", TitleKey: "flow.field.blacklisted", Group: "risk", Type: trigger.TypeBool,
					Resolve: func(tc *trigger.Context) (any, error) {
						simulateSeenCodePaths = tc.CodePaths
						return false, nil
					},
				},
			},
			SimulateFields: []trigger.SimulateInput{{Key: "machineId", TitleKey: "flow.simulate.machineId", Type: trigger.TypeNumber, Required: true}},
			Simulate: func(ctx context.Context, raw map[string]string) (*trigger.SimulatedFacts, error) {
				return &trigger.SimulatedFacts{
					Raw:       map[string]string{"cmdText": "rm -rf x"},
					CodePaths: []string{"test/test1/222/1|UMaq3B2KHkKg/"},
				}, nil
			},
		})
	})
}

// TestSimulateCarriesResourcePaths 试算必须把试算目标的资源路径带进求值上下文。
//
// 按资源定位的派生事实（比如这台机器的命令黑名单）只认上下文里的路径：缺了它，试算恒判
// 「不命中」而真实执行照样被拦。管理员拿一个专门用来预判结论的工具，得到的却是与线上相反
// 的答案，比没有这个工具更危险
func TestSimulateCarriesResourcePaths(t *testing.T) {
	registerSimulateScenario()
	simulateSeenCodePaths = nil

	res, err := (&procdefAppImpl{}).SimulateTrigger(context.Background(), &dto.SimulateTrigger{
		BizType: simulatePathBiz,
		Policy:  &entity.TriggerPolicy{Version: entity.TriggerPolicyVersion, DefaultSeverity: entity.SeverityDisabled},
		Raw:     map[string]string{"machineId": "1"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"test/test1/222/1|UMaq3B2KHkKg/"}, simulateSeenCodePaths, "试算解析出的资源路径必须进入求值上下文")
	// 解析出的原始输入同样要进上下文，否则条件与派生字段都看不到管理员粘贴的那条命令
	require.Equal(t, "rm -rf x", res.Fields["cmdText"])
}

// TestSimulateRejectsUnregisteredScenario 未注册的场景必须直接拒绝，而不是当成「无属性」判个放行。
//
// 吞掉这层错误会给出「没有规则命中，直接放行」的结论，而真实执行侧根本不存在这个场景
func TestSimulateRejectsUnregisteredScenario(t *testing.T) {
	_, err := (&procdefAppImpl{}).SimulateTrigger(context.Background(), &dto.SimulateTrigger{
		BizType: "ghost_scenario_flow",
		Policy:  &entity.TriggerPolicy{Version: entity.TriggerPolicyVersion, DefaultSeverity: entity.SeverityDisabled},
	})
	require.Error(t, err)
}

// schemaFactsBiz 一个不带审批通道的场景，用来验「类型事实」是否全部由后端一次算清
const schemaFactsBiz = "test_schema_type_facts_flow"

var registerSchemaFactsOnce sync.Once

func registerSchemaFactsScenario() {
	registerSchemaFactsOnce.Do(func() {
		RegisterTriggerBiz(trigger.BizMeta{
			BizType: schemaFactsBiz,
			Fields: []trigger.TriggerField{
				{Key: "size", TitleKey: "flow.checkParam.maxKb", Group: "risk", Type: trigger.TypeNumber},
				{Key: "name", TitleKey: "flow.field.cmdText", Group: "risk", Type: trigger.TypeString},
			},
			SimulateFields: []trigger.SimulateInput{
				{Key: "raw", TitleKey: "flow.simulate.cmdText", Type: trigger.TypeString, Required: true, Multiline: true},
				{Key: "ref", TitleKey: "flow.simulate.machineId", Type: trigger.TypeNumber, EditorKey: "machine-select", NameKey: "name"},
				{Key: "name", TitleKey: "flow.field.cmdText", Type: trigger.TypeString},
			},
			Simulate: func(context.Context, map[string]string) (*trigger.SimulatedFacts, error) {
				return &trigger.SimulatedFacts{Raw: map[string]string{}}, nil
			},
		})
	})
}

// TestPolicySchemaCarriesTypeFacts 界面需要的类型事实必须随 schema 一起下发。
//
// 数值性、值编辑器标识、场景最严可落地级别这三样，只要有一样没下发，前端就得自己按类型名推一遍：
// 那是第二份真源，注册一个新字段类型就会「界面全绿、保存被后端拒」或「兜底级别被前端显示错」
func TestPolicySchemaCarriesTypeFacts(t *testing.T) {
	registerSchemaFactsScenario()

	schema := (&procdefAppImpl{}).PolicySchema(context.Background())
	var scenario *dto.PolicyScenario
	for _, item := range schema.Scenarios {
		if item.BizType == schemaFactsBiz {
			scenario = item
		}
	}
	require.NotNil(t, scenario, "registered scenario must appear in the policy schema")

	require.NotContains(t, scenario.Severities, entity.SeverityRequired, "no approval channel means no approvable severity")
	require.Equal(t, entity.SeverityForbidden, scenario.FailClosedSeverity, "the strictest honourable severity must come from the engine")

	fields := map[string]*dto.PolicyField{}
	for _, field := range scenario.Fields {
		fields[field.Key] = field
	}
	require.True(t, fields["size"].Numeric, "a number field must be declared numeric by the backend")
	require.False(t, fields["name"].Numeric, "a string field must not be numeric")
	for key, field := range fields {
		require.NotEmpty(t, field.EditorKey, "every field must carry its editor identity: %s", key)
	}
	require.True(t, scenario.SimulateFields[0].Multiline, "multiline must be declared by the scenario, not guessed from the type")

	inputs := map[string]*dto.PolicySimulateInput{}
	for _, item := range scenario.SimulateFields {
		inputs[item.Key] = item
	}
	require.Equal(t, "machine-select", inputs["ref"].EditorKey, "a declared editor key must be passed through verbatim, not derived from the type")
	require.Equal(t, "name", inputs["ref"].NameKey, "the resource-name refill target must travel with the schema")
	require.Equal(t, "input", inputs["raw"].EditorKey, "an undeclared editor key falls back to the type registry")
}
