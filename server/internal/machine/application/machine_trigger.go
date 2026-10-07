package application

import (
	"context"
	"errors"
	flowapp "mayfly-go/internal/flow/application"
	flowdto "mayfly-go/internal/flow/application/dto"
	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/trigger"
	"mayfly-go/internal/machine/imsg"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/internal/pkg/consts"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/ioc"
	"strconv"
	"strings"
)

const (
	// MachineRunCmdFlowBizType 机器命令执行场景。
	//
	// 该场景没有审批通过后的执行回调（命令不像 SQL 那样有工单落库与批后重放），
	// 因此可配置级别由触发引擎推导为「仅提醒 / 禁止执行」两态
	MachineRunCmdFlowBizType = "machine_run_cmd_flow"
)

// 触发策略字段 key
const (
	machineFieldCmdText     = "cmdText"
	machineFieldCmd         = "cmd"
	machineFieldCmdNames    = "cmdNames"
	machineFieldBlacklisted = "blacklisted"
	machineFieldUnsafe      = "unsafePattern"
)

// 内置检查项 key
const (
	machineCheckBlacklisted = "machine.cmd-blacklisted"
	machineCheckUnsafe      = "machine.cmd-unsafe-pattern"
)

// init 注册机器命令执行场景的触发策略能力。
//
// 命令黑名单本身仍是「命令配置」那份按标签维护的数据，这里只把它接成策略可引用的事实，
// 处置级别（提醒/禁止）改由流程定义的触发策略决定，不再散落在各执行入口
func init() {
	flowapp.RegisterTriggerBiz(trigger.BizMeta{
		BizType: MachineRunCmdFlowBizType,
		// 治理的资源类型：保存校验据此判断兜底级别在这些资源上落不落得了地（机器命令无审批通道）
		GovernPaths:    [][]int8{{consts.ResourceTypeMachine}},
		Fields:         machineTriggerFields(),
		Checks:         machineTriggerChecks(),
		Presets:        machineTriggerPresets(),
		SimulateFields: machineSimulateInputs(),
		Simulate:       simulateMachineTriggerInput,
	})
}

func machineTriggerFields() []trigger.TriggerField {
	return []trigger.TriggerField{
		{Key: machineFieldCmdText, TitleKey: "flow.field.cmdText", Group: "command", Type: trigger.TypeString},
		{
			Key: machineFieldCmd, TitleKey: "flow.field.cmd", Group: "command", Type: trigger.TypeString,
			Resolve: func(tc *trigger.Context) (any, error) {
				return firstMachineCmd(machineRawText(tc)), nil
			},
		},
		{
			Key: machineFieldCmdNames, TitleKey: "flow.field.cmdNames", Group: "command", Type: trigger.TypeStringList,
			Resolve: func(tc *trigger.Context) (any, error) {
				return mcm.ExtractCommandNames(machineRawText(tc)), nil
			},
		},
		{
			Key: machineFieldUnsafe, TitleKey: "flow.field.unsafePattern", Group: "risk", Type: trigger.TypeBool,
			Resolve: func(tc *trigger.Context) (any, error) {
				return mcm.HasUnsafePattern(machineRawText(tc)), nil
			},
		},
		{
			Key: machineFieldBlacklisted, TitleKey: "flow.field.blacklisted", Group: "risk", Type: trigger.TypeBool,
			Resolve: func(tc *trigger.Context) (any, error) {
				app, err := machineCmdConfApp()
				if err != nil {
					return nil, err
				}
				matched, _ := app.MatchCmdRule(tc.GoContext(), machineRawText(tc), tc.CodePaths...)
				return matched, nil
			},
		},
	}
}

func machineTriggerChecks() []trigger.CheckDef {
	return []trigger.CheckDef{
		{
			Key: machineCheckBlacklisted, TitleKey: "flow.check.cmdBlacklisted", Summary: imsg.TriggerReasonCmdBlacklisted,
			DescriptionKey: "flow.checkDesc.cmdBlacklisted",
			BizTypes:       []string{MachineRunCmdFlowBizType}, Default: flowentity.SeverityForbidden,
			Evaluate: func(ctx context.Context, tc *trigger.Context, params map[string]any) (bool, error) {
				app, err := machineCmdConfApp()
				if err != nil {
					return false, err
				}
				matched, _ := app.MatchCmdRule(ctx, machineRawText(tc), tc.CodePaths...)
				return matched, nil
			},
		},
		{
			Key: machineCheckUnsafe, TitleKey: "flow.check.cmdUnsafePattern", Summary: imsg.TriggerReasonCmdUnsafe,
			DescriptionKey: "flow.checkDesc.cmdUnsafePattern",
			BizTypes:       []string{MachineRunCmdFlowBizType}, Default: flowentity.SeverityForbidden,
			Evaluate: func(ctx context.Context, tc *trigger.Context, params map[string]any) (bool, error) {
				return mcm.HasUnsafePattern(machineRawText(tc)), nil
			},
		},
	}
}

func machineTriggerPresets() []trigger.PolicyPreset {
	return []trigger.PolicyPreset{
		{Key: "strict", TitleKey: "flow.policy.presetStrict", CheckKeys: []string{machineCheckBlacklisted, machineCheckUnsafe}},
		{Key: "loose", TitleKey: "flow.policy.presetLoose", CheckKeys: []string{machineCheckBlacklisted}},
	}
}

func machineSimulateInputs() []trigger.SimulateInput {
	return []trigger.SimulateInput{
		// 目标机器用资源树选择：机器 id 是内部主键，命令黑名单按所选机器的标签取配置
		{Key: "machineId", TitleKey: "flow.simulate.machineId", Type: trigger.TypeNumber, EditorKey: "machine-select", Required: true},
		{Key: machineFieldCmdText, TitleKey: "flow.simulate.cmdText", Type: trigger.TypeString, PlaceholderKey: "flow.simulate.machineCmdPlaceholder", Required: true, Multiline: true},
	}
}

// CheckMachineCmd 机器命令执行的统一前置校验，Web 终端、非交互命令执行与 AI Agent 三个入口共用。
//
// 收敛前三个入口各自实现：终端路径命中正则一律拒绝、API 与 AI 路径才看策略且策略从未参与判定，
// 同一台机器上同一条命令会得到不同结论
func CheckMachineCmd(ctx context.Context, codePaths []string, cmd string) (notice string, err error) {
	if strings.TrimSpace(cmd) == "" {
		return "", nil
	}

	procdefApp := flowapp.GetProcdefApp()
	// 先定位流程定义再针对它求值：命令过滤规则的基线判定需要知道「这台机器被哪个策略管着」，
	// 而 CheckTrigger 定位后不回传流程定义
	procdef := procdefApp.GetProcdefByCodePath(ctx, codePaths...)
	decision, err := procdefApp.CheckProcdefTrigger(ctx, procdef, &flowdto.TriggerRequest{
		BizType:   MachineRunCmdFlowBizType,
		CodePaths: codePaths,
		Raw:       map[string]string{machineFieldCmdText: cmd},
	})
	if err != nil {
		return "", err
	}
	// 机器命令没有审批通道：Approval 传 0，出现「需审批」结论按禁止执行拒绝，绝不静默放行
	if err := flowapp.NewBlockError(ctx, decision, flowapp.BlockSpeech{Forbidden: imsg.TerminalCmdDisable}, true); err != nil {
		return "", err
	}
	if err := rejectUngovernedCmdRule(ctx, procdef, codePaths, cmd); err != nil {
		return "", err
	}
	// 提醒不要求确认（确认了也无处可转），但命中文案要交回调用方写进终端，
	// 否则管理员配的「仅提醒」在机器侧只剩服务端日志一处出口
	return flowapp.WarnNotice(ctx, decision), nil
}

// rejectUngovernedCmdRule 兜住「命令过滤规则」这条早于触发策略存在的硬管控面。
//
// 策略负责的是命中之后按什么级别处置，不该决定要不要查黑名单：该检查项没被策略显式纳管时
// （未绑定流程定义，或绑了但没勾这条），仍按原语义直接拒绝。否则管理员在「命令过滤规则」里
// 配的黑名单会在接入策略后静默失效 —— 这正是收敛前的老行为（终端命中正则即拒绝，与策略无关）
//
// 反过来，一旦策略里配了这条（哪怕是「仅提醒」或「不处置」），就以管理员的显式选择为准，
// 基线不再越权，降级路径也因此在策略编辑器里可见可改
func rejectUngovernedCmdRule(ctx context.Context, procdef *flowentity.Procdef, codePaths []string, cmd string) error {
	var policy *flowentity.TriggerPolicy
	if procdef != nil {
		policy = procdef.TriggerPolicy
	}
	if policy.CheckConfigured(MachineRunCmdFlowBizType, machineCheckBlacklisted) {
		return nil
	}

	app, err := machineCmdConfApp()
	if err != nil {
		return err
	}
	if matched, _ := app.MatchCmdRule(ctx, cmd, codePaths...); matched {
		return errorx.NewBizI(ctx, imsg.TerminalCmdDisable)
	}
	return nil
}

// machineRawText 取本次求值的命令原文。
// 试算路径与真实执行路径都只保证 cmdText 存在，命令名等派生值一律由字段字典再算，避免两处口径分叉
func machineRawText(tc *trigger.Context) string {
	if text, ok := tc.RawValue(machineFieldCmdText); ok {
		return text
	}
	return ""
}

// firstMachineCmd 取命令串的首个命令名，多条命令以管道/分号等分隔时以首段为准
func firstMachineCmd(cmd string) string {
	names := mcm.ExtractCommandNames(cmd)
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

// machineCmdConfApp 取命令配置应用。
//
// 用 GetBeansByType 而不是 ioc.Get：后者在未注册时会直接 panic，那会让整条命令请求 500；
// 这里返回错误，交给触发引擎按「无法判定」做 fail-closed 处置
func machineCmdConfApp() (MachineCmdConf, error) {
	apps := ioc.GetBeansByType[MachineCmdConf]()
	if len(apps) == 0 {
		return nil, errors.New("the machine command configuration is unavailable")
	}
	return apps[0], nil
}

// machineCodePaths 按机器 id 取其标签路径，用于试算时按资源定位命令配置与流程定义
func machineCodePaths(ctx context.Context, machineId uint64) ([]string, error) {
	machine, err := GetMachineApp().GetById(machineId)
	if err != nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrMachineNotFound)
	}

	tagTreeApp := ioc.Get[tagapp.TagTree]()
	codePaths := tagTreeApp.ListTagPathByTypeAndCode(consts.ResourceTypeMachine, machine.Code)

	// 试算必须与真实执行、Web 终端、AI Agent 用同一道资源鉴权：
	// 试算会把这台机器的黑名单命中结果回显给调用方，漏掉这一步等于让无权限的人
	// 拿着一台别人的机器 id 去试探它被治理成什么样
	if loginAccount := contextx.GetLoginAccount(ctx); loginAccount != nil {
		if err := tagTreeApp.CanAccess(loginAccount.Id, codePaths...); err != nil {
			return nil, err
		}
	} else if len(codePaths) == 0 {
		return nil, errorx.NewBizf("the target machine %d is not bound to any tag", machineId)
	}

	return codePaths, nil
}

// simulateMachineTriggerInput 解析管理员粘贴的命令用于策略试算。
//
// 黑名单命中依赖具体机器的命令配置，因此试算必须指定机器 id：
// 不指定就猜一份规则集，会让「试算说会拦、实际没拦」的判定漂移重新出现
func simulateMachineTriggerInput(ctx context.Context, raw map[string]string) (*trigger.SimulatedFacts, error) {
	cmdText := strings.TrimSpace(raw[machineFieldCmdText])
	if cmdText == "" {
		return nil, errorx.NewBizf("a machine command is required to simulate the trigger policy")
	}
	machineId, err := strconv.ParseUint(raw["machineId"], 10, 64)
	if err != nil || machineId == 0 {
		return nil, errorx.NewBizf("a target machine is required to simulate the machine command trigger policy")
	}

	codePaths, err := machineCodePaths(ctx, machineId)
	if err != nil {
		return nil, err
	}

	// 这里只交回资源路径，不自己算黑名单：黑名单是带 Resolve 的派生字段，
	// 它只认上下文里的资源路径。在此再算一份写进 Raw 会变成第二个真源，
	// 而且派生字段优先于 Raw，写进去的值根本不会被读到（此前正是这样恒判「不命中」）
	return &trigger.SimulatedFacts{
		Raw:       map[string]string{machineFieldCmdText: cmdText},
		CodePaths: codePaths,
	}, nil
}
