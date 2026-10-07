package application

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/trigger"
	flowimsg "mayfly-go/internal/flow/imsg"
	"mayfly-go/internal/redis/imsg"
	"mayfly-go/pkg/i18n"

	"github.com/stretchr/testify/require"
)

// defaultRedisPolicy 迁移为存量流程定义写入的默认规则包（Redis 部分）
func defaultRedisPolicy() *flowentity.TriggerPolicy {
	return &flowentity.TriggerPolicy{
		Version:         flowentity.TriggerPolicyVersion,
		DefaultSeverity: flowentity.SeverityRequired,
		Checks: []*flowentity.CheckConfig{
			{Key: redisCheckWriteCmd, BizType: RedisRunCmdFlowBizType, Severity: flowentity.SeverityRequired},
			{Key: redisCheckDangerousCmd, BizType: RedisRunCmdFlowBizType, Severity: flowentity.SeverityRequired},
		},
	}
}

func TestDefaultRedisRulePackPassesValidation(t *testing.T) {
	if err := trigger.ValidatePolicy(defaultRedisPolicy()); err != nil {
		t.Fatalf("the default redis rule pack must be valid, got %v", err)
	}
}

// 读写与高危属性由命令名派生，命令名大小写不敏感，否则 `hset` 会绕过写操作审批
func TestRedisTriggerChecks(t *testing.T) {
	cases := []struct {
		cmd  string
		want flowentity.Severity
	}{
		{"GET", flowentity.SeverityDisabled},
		{"get", flowentity.SeverityDisabled},
		{"HGETALL", flowentity.SeverityDisabled},
		{"SET", flowentity.SeverityRequired},
		{"hset", flowentity.SeverityRequired},
		{"DEL", flowentity.SeverityRequired},
		{"FLUSHALL", flowentity.SeverityRequired},
		{"keys", flowentity.SeverityRequired},
	}

	for _, tc := range cases {
		t.Run(tc.cmd, func(t *testing.T) {
			ctx := trigger.NewContext(context.Background(), RedisRunCmdFlowBizType, 1, map[string]string{redisFieldCmd: tc.cmd, redisFieldCmdText: tc.cmd}, nil)
			decision := trigger.Evaluate(context.Background(), defaultRedisPolicy(), ctx)
			if decision.Severity != tc.want {
				t.Fatalf("expected severity %v for %q but got %v (findings=%+v)", tc.want, tc.cmd, decision.Severity, decision.Findings)
			}
		})
	}
}

// 命令派生字段必须与权限校验共用同一份分类表，避免两处口径分叉
func TestRedisDerivedFieldsMatchPermissionClassification(t *testing.T) {
	for _, cmd := range []string{"SET", "set", "HSET", "DEL", "FLUSHDB", "GET", "TTL"} {
		ctx := trigger.NewContext(context.Background(), RedisRunCmdFlowBizType, 1, map[string]string{redisFieldCmd: cmd}, nil)
		field, ok := trigger.FieldOf(RedisRunCmdFlowBizType, redisFieldCmdType)
		if !ok {
			t.Fatalf("the field %q must be registered", redisFieldCmdType)
		}
		value, available, resolveErr := ctx.FieldValue(field)
		if resolveErr != nil || !available {
			t.Fatalf("the command type must be derivable from %q", cmd)
		}
		if want := redisCmdType(cmd); value != want {
			t.Fatalf("%q must derive the command type %q, got %v", cmd, want, value)
		}
	}
}

func TestRedisSimulateExtractsCommandName(t *testing.T) {
	facts, err := simulateRedisTriggerInput(context.Background(), map[string]string{redisFieldCmdText: "  DEL 'a' 'b'; HSET h f v"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if facts.Raw[redisFieldCmd] != "DEL" {
		t.Fatalf("the first statement command name must be extracted, got %q", facts.Raw[redisFieldCmd])
	}
	if len(facts.Attributes) != 0 {
		t.Fatalf("the redis scenario derives its fields from the command name, got %v", facts.Attributes)
	}

	if _, err := simulateRedisTriggerInput(context.Background(), map[string]string{}); err == nil {
		t.Fatalf("an empty simulated command must be rejected")
	}
}

// 豁免条件优先于检查项：给常规写命令开的口子不能被写命令检查覆盖
func TestRedisUnlessExemptsWriteCmd(t *testing.T) {
	policy := &flowentity.TriggerPolicy{
		Version: flowentity.TriggerPolicyVersion,
		Checks:  []*flowentity.CheckConfig{{Key: redisCheckWriteCmd, BizType: RedisRunCmdFlowBizType, Severity: flowentity.SeverityForbidden}},
		Customs: []*flowentity.CustomCondition{{
			BizType:  RedisRunCmdFlowBizType,
			Severity: flowentity.SeverityRequired,
			Unless: &flowentity.RuleNode{
				Kind: flowentity.NodeKindCondition, Field: redisFieldCmd, Op: trigger.OpNotIn,
				Value: []any{"FLUSHALL", "FLUSHDB"},
			},
		}},
	}

	exempted := trigger.Evaluate(context.Background(), policy, newRedisContext("SET"))
	if !exempted.Exempted || exempted.Severity != flowentity.SeverityDisabled {
		t.Fatalf("a write command outside the dangerous set must be exempted, got %v %v", exempted.Exempted, exempted.Severity)
	}

	// 豁免条件不成立时，检查项的禁止结论仍然生效
	stillForbidden := trigger.Evaluate(context.Background(), policy, newRedisContext("FLUSHALL"))
	if stillForbidden.Exempted || stillForbidden.Severity != flowentity.SeverityForbidden {
		t.Fatalf("FLUSHALL must stay forbidden, got %v %v", stillForbidden.Exempted, stillForbidden.Severity)
	}
}

func newRedisContext(cmd string) *trigger.Context {
	return trigger.NewContext(context.Background(), RedisRunCmdFlowBizType, 1, map[string]string{redisFieldCmd: cmd, redisFieldCmdText: cmd}, nil)
}

// TestBatchCmdResolvesProcdefOnce 批量命令只解析一次流程定义。
//
// key 面板一次操作会下发多条命令（改 TTL、删字段、重命名等），流程定义只跟连接走：
// 放在循环里就是每条命令两趟库查询。这里用接线断言钉住「解析在循环外」，
// 因为把它挪回循环内不会改变任何判定结果，行为测试发现不了
func TestBatchCmdResolvesProcdefOnce(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(".", "key_value.go"))
	require.NoError(t, err)
	source := string(content)

	resolve := strings.Index(source, "k.redisApp.TriggerProcdefOf(ctx, conn)")
	loop := strings.Index(source, "for _, args := range cmds {")
	require.NotEqual(t, -1, resolve, "批量命令应改为解析一次流程定义")
	require.NotEqual(t, -1, loop)
	require.Less(t, resolve, loop, "流程定义解析必须在循环外，否则又变成每条命令查两趟库")
	require.NotContains(t, source[loop:], "CheckCmdFlow(", "循环体内不能再走带解析的入口")
	require.Contains(t, source[loop:], "CheckCmdTrigger(ctx, procdef,")
}

// TestCmdSpeechPerAckMode 入口能不能提单，决定两套话术必须同时改口。
//
// key 面板的类型化操作没有提单表单：确认框若还写「或提交工单审批后执行」、
// 需审批时若还只说「请提交工单」，都是在指挥用户去找一个界面上不存在的按钮
func TestCmdSpeechPerAckMode(t *testing.T) {
	ticketable := cmdSpeech(WarnAckOf(false))
	require.Equal(t, i18n.MsgId(imsg.ErrSubmitFlowRunCmd), ticketable.Approval)
	require.Zero(t, ticketable.Warn, "可转审批的入口用默认确认话术")
	require.True(t, ticketable.RequireWarnAck)

	direct := cmdSpeech(WarnAckDirectOf(false))
	require.Equal(t, i18n.MsgId(imsg.ErrRunCmdFromConsole), direct.Approval, "需审批时必须指路到命令控制台")
	require.Equal(t, i18n.MsgId(flowimsg.ErrNeedWarnConfirmDirect), direct.Warn, "确认话术不能再提转审批")
	require.True(t, direct.RequireWarnAck)

	// 不能弹确认的入口（结果被广播给所有客户端）：问了也没人答，提醒只能不阻断地回显
	batch := cmdSpeech(warnNoAsk)
	require.False(t, batch.RequireWarnAck)
	require.Equal(t, i18n.MsgId(imsg.ErrSubmitFlowRunCmd), batch.Approval, "需审批的拦截与能否弹确认无关")
}

// TestTypedOpsDeclareDirectAck 面板的写操作都要声明「只能直接执行」。
// 映射本身有上一条测试守着，这里守接线：漏掉哪个入口，哪个入口就会退回带转审批措辞的提示
func TestTypedOpsDeclareDirectAck(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "api", "key_value.go"))
	require.NoError(t, err)
	source := string(content)
	// 6 = 成员写入、视角操作、设置 TTL、重命名、复制、删除 key
	require.Equal(t, 6, strings.Count(source, "application.WarnAckDirectOf("), "面板写操作数量变化时同步这条断言")
	// 只有「读内容」例外：它能就地给出「申请查看」提单入口，所以按可提单口径出话术
	// （它落在哪个处理函数上由 TestContentReadingIsCheckedBeforeLoading 钉住）
	require.Equal(t, 1, strings.Count(source, "application.WarnAckOf("), "可提单的面板入口数量变了，同步此处")
}
