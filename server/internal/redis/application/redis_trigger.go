package application

import (
	"context"
	flowapp "mayfly-go/internal/flow/application"
	flowdto "mayfly-go/internal/flow/application/dto"
	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/trigger"
	flowimsg "mayfly-go/internal/flow/imsg"
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/internal/redis/imsg"
	"mayfly-go/internal/redis/rdm"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/utils/anyx"
	"strings"
)

// checkCmdTrigger 按已绑定流程定义的触发策略判定该命令能否直接执行
func (r *redisAppImpl) checkCmdTrigger(ctx context.Context, procdef *flowentity.Procdef, cmdName, cmdText string, ack WarnAck) error {
	decision, err := r.procdefApp.CheckProcdefTrigger(ctx, procdef, &flowdto.TriggerRequest{
		BizType: RedisRunCmdFlowBizType,
		Raw:     map[string]string{redisFieldCmd: cmdName, redisFieldCmdText: cmdText},
	})
	if err != nil {
		return err
	}
	// 与数据库入口共用同一套阻断语义：命中原因、需提单错误码都不各写一遍
	return flowapp.NewBlockError(ctx, decision, cmdSpeech(ack), ack.Acknowledged)
}

// cmdSpeech 入口处置能力 → 阻断话术。映射集中在这里，才能被单测直接断言：
// 「能不能提单」决定了确认与需审批两套话术，写散到调用点就会各说各话
func cmdSpeech(ack WarnAck) flowapp.BlockSpeech {
	speech := flowapp.BlockSpeech{
		Forbidden:      flowimsg.ErrOperationForbidden,
		Approval:       imsg.ErrSubmitFlowRunCmd,
		RequireWarnAck: ack.askable(),
	}
	if ack.Mode == WarnAskDirect {
		// 面板的结构化操作没有提单表单：两处话术都要改口指路，
		// 否则是给不出出口的指令（确认框里写「或提交工单」、需审批时只说「请提单」）
		speech.Approval = imsg.ErrRunCmdFromConsole
		speech.Warn = flowimsg.ErrNeedWarnConfirmDirect
	}
	return speech
}

// redisCmdText 还原命令原文供文本型条件使用
func redisCmdText(cmdArgs []any) string {
	parts := make([]string, 0, len(cmdArgs))
	for _, arg := range cmdArgs {
		parts = append(parts, anyx.ToString(arg))
	}
	return strings.Join(parts, " ")
}

// 触发策略字段 key
const (
	redisFieldCmd       = "cmd"
	redisFieldCmdText   = "cmdText"
	redisFieldCmdType   = "cmdType"
	redisFieldDangerous = "dangerous"
)

// 内置检查项 key
const (
	redisCheckWriteCmd     = "redis.write-cmd"
	redisCheckDangerousCmd = "redis.dangerous-cmd"
	redisCheckCmdIn        = "redis.cmd-in"
)

const (
	redisCmdTypeRead  = "read"
	redisCmdTypeWrite = "write"
)

// init 注册 Redis 执行命令场景的触发策略能力。
//
// 命令的读写与高危属性由命令名纯函数判定，因此调用方只需提交命令名，无需自行拼装派生字段
func init() {
	flowapp.RegisterTriggerBiz(trigger.BizMeta{
		BizType: RedisRunCmdFlowBizType,
		// 治理的资源类型：保存校验据此判断兜底级别在这些资源上落不落得了地
		GovernPaths:    [][]int8{{consts.ResourceTypeRedis}},
		Approvable:     true,
		Fields:         redisTriggerFields(),
		Checks:         redisTriggerChecks(),
		Presets:        redisTriggerPresets(),
		SimulateFields: redisSimulateInputs(),
		Simulate:       simulateRedisTriggerInput,
	})
}

// redisSimulateInputs 命令的读写与高危属性由命令名纯函数派生，试算不需要连接实例
func redisSimulateInputs() []trigger.SimulateInput {
	return []trigger.SimulateInput{
		{Key: redisFieldCmdText, TitleKey: "flow.simulate.cmdText", Type: trigger.TypeString, PlaceholderKey: "flow.simulate.cmdTextPlaceholder", Required: true, Multiline: true},
	}
}

// redisTriggerPresets 命令场景的预置组合
func redisTriggerPresets() []trigger.PolicyPreset {
	return []trigger.PolicyPreset{
		{Key: "standard", TitleKey: "flow.policy.presetStandard", CheckKeys: []string{redisCheckWriteCmd}},
		{Key: "strict", TitleKey: "flow.policy.presetStrict", CheckKeys: []string{redisCheckWriteCmd, redisCheckDangerousCmd}},
		{Key: "loose", TitleKey: "flow.policy.presetLoose", CheckKeys: []string{redisCheckDangerousCmd}},
	}
}

func redisTriggerFields() []trigger.TriggerField {
	return []trigger.TriggerField{
		{Key: redisFieldCmd, TitleKey: "flow.field.cmd", Group: "command", Type: trigger.TypeString},
		{Key: redisFieldCmdText, TitleKey: "flow.field.cmdText", Group: "command", Type: trigger.TypeString},
		{
			Key: redisFieldCmdType, TitleKey: "flow.field.cmdType", Group: "command", Type: trigger.TypeEnum,
			Options: []trigger.FieldOption{{Value: redisCmdTypeRead}, {Value: redisCmdTypeWrite}},
			Resolve: func(tc *trigger.Context) (any, error) {
				cmd, _ := tc.RawValue(redisFieldCmd)
				return redisCmdType(cmd), nil
			},
		},
		{
			Key: redisFieldDangerous, TitleKey: "flow.field.dangerous", Group: "risk", Type: trigger.TypeBool,
			Resolve: func(tc *trigger.Context) (any, error) {
				cmd, _ := tc.RawValue(redisFieldCmd)
				return rdm.IsDangerousCmd(cmd), nil
			},
		},
	}
}

func redisTriggerChecks() []trigger.CheckDef {
	return []trigger.CheckDef{
		{
			Key: redisCheckWriteCmd, TitleKey: "flow.check.writeCmd", Summary: imsg.TriggerReasonWriteCmd,
			BizTypes: []string{RedisRunCmdFlowBizType}, Default: flowentity.SeverityRequired,
			Evaluate: func(ctx context.Context, tc *trigger.Context, params map[string]any) (bool, error) {
				cmd, _ := tc.RawValue(redisFieldCmd)
				return redisCmdType(cmd) == redisCmdTypeWrite, nil
			},
		},
		{
			Key: redisCheckDangerousCmd, TitleKey: "flow.check.dangerousCmd", Summary: imsg.TriggerReasonDangerousCmd,
			DescriptionKey: "flow.checkDesc.dangerousCmd",
			BizTypes:       []string{RedisRunCmdFlowBizType}, Default: flowentity.SeverityRequired,
			Evaluate: func(ctx context.Context, tc *trigger.Context, params map[string]any) (bool, error) {
				cmd, _ := tc.RawValue(redisFieldCmd)
				return rdm.IsDangerousCmd(cmd), nil
			},
		},
		{
			Key: redisCheckCmdIn, TitleKey: "flow.check.cmdIn", Summary: imsg.TriggerReasonCmdIn,
			DescriptionKey: "flow.checkDesc.cmdIn",
			BizTypes:       []string{RedisRunCmdFlowBizType}, Default: flowentity.SeverityRequired,
			Params: []trigger.CheckParam{{
				Key: "cmds", TitleKey: "flow.checkParam.cmds", Type: trigger.TypeStringList, Required: true,
			}},
			Evaluate: func(ctx context.Context, tc *trigger.Context, params map[string]any) (bool, error) {
				cmd, _ := tc.RawValue(redisFieldCmd)
				return trigger.OneOf(cmd, trigger.ParamStrings(params, "cmds")), nil
			},
		},
	}
}

// redisCmdType 命令的读写属性判定统一收敛到这里，与权限校验共用同一份命令分类表，
// 避免两处分别维护导致「有写权限但被判成读」的分叉
func redisCmdType(cmd string) string {
	if rdm.IsWriteCmd(cmd) {
		return redisCmdTypeWrite
	}
	return redisCmdTypeRead
}

// simulateRedisTriggerInput 从管理员粘贴的完整命令中提取命令名，用于策略试算
func simulateRedisTriggerInput(ctx context.Context, raw map[string]string) (*trigger.SimulatedFacts, error) {
	cmdText := strings.TrimSpace(raw[redisFieldCmdText])
	if cmdText == "" {
		return nil, errorx.NewBizf("a redis command is required to simulate the trigger policy")
	}
	// 多条命令以分号分隔，取首条的命令名参与判定，与执行侧逐条校验的口径一致
	firstStmt := cmdText
	if semicolon := strings.Index(firstStmt, ";"); semicolon >= 0 {
		firstStmt = firstStmt[:semicolon]
	}
	fields := strings.Fields(firstStmt)
	if len(fields) == 0 {
		return nil, errorx.NewBizf("a redis command is required to simulate the trigger policy")
	}
	return &trigger.SimulatedFacts{Raw: map[string]string{redisFieldCmd: fields[0], redisFieldCmdText: cmdText}}, nil
}
