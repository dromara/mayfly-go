package machinetool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mayfly-go/internal/ai/imsg"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/i18n"
	"strings"

	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// CommandExecParam 命令执行参数
type CommandExecParam struct {
	AuthCertName string `json:"authCertName" jsonschema_description:"授权凭证名称"`
	Command      string `json:"command" jsonschema_description:"要执行的命令" jsonschema:"required"`
	Remark       string `json:"remark" jsonschema_description:"命令作用说明，简要描述该命令的用途或目的（供用户审批与事后审计理解）" jsonschema:"required"`
}

// CommandExecOutput 命令执行输出
type CommandExecOutput struct {
	AuthCertName string `json:"authCertName" jsonschema_description:"授权凭证名称"`
	MachineId    uint64 `json:"machineId" jsonschema_description:"机器ID"`
	MachineName  string `json:"machineName" jsonschema_description:"机器名称"`
	MachineIp    string `json:"machineIp" jsonschema_description:"机器IP地址"`
	MachinePort  int    `json:"machinePort" jsonschema_description:"机器端口"`
	Username     string `json:"username" jsonschema_description:"连接用户名"`
	Output       string `json:"output" jsonschema_description:"命令执行输出"`
	Success      bool   `json:"success" jsonschema_description:"是否执行成功"`
}

// GetCommandExec 获取命令执行工具
// init 在包加载时声明入参形状。
//
// 不能放在 GetCommandExec 里：下发受控执行许可（allowedToolsForAction 的参数校验）发生在
// 工具首次构造之前，那时 schema 表还是空的，校验会走「未声明则跳过」分支静默放行——
// 端到端实测到坏参数（machineId/cmd）一路漏到执行期让模型自己猜
func init() {
	tools.DeclareParamSchema(tools.ToolNameMachineCommandExec, CommandExecParam{})
	// 凭证解析器注册：解析需账号可见资源清单，本包 import application/resource 取之（正向依赖）后
	// 把实现注册进 tools 父包接缝，父包（NormalizeActionParams 所在）因此无需依赖 application。
	// 消费点有两个：本包工具执行时的兜底纠正、以及提案登记时的参数规范化。
	// 必须在 init 里注册而不是在 GetCommandExec 里：消费点可能早于任何一次工具构造
	tools.DeclareAuthCertResolver(resolveUniqueAuthCert)
	// 内容级能力：本工具既能跑只读探查（top/ps/df）又能跑破坏命令，按工具名一刀切成 Dangerous
	// 会让只读诊断档连 top 都执行不了。这里按命令内容下探——命中只读白名单算 Read（只读档放行、
	// 且工具体内 isWhitelistCommand 也免审批），其余仍 Dangerous（走审批/提案）。
	// 判定复用同一个 isWhitelistCommand（mcm.CmdAnalyzer），与工具体审批白名单严格同源，杜绝口径分叉
	tools.DeclareContentCapability(tools.ToolNameMachineCommandExec, func(argsJSON string) tools.Capability {
		if isWhitelistCommand(commandOfArgs(argsJSON)) {
			return tools.CapabilityRead
		}
		return tools.CapabilityDangerous
	})
	// 结果内容级判定：本工具在命令退出码非零时仍返回正常结果（为让模型看到输出），
	// 失败事实只写在载荷的 success:false 里。不把这一层声出来，下游「变更有没有发生」就会判成功
	// （实测：命令 false 执行失败、提案却落 executed、工单落处理成功）
	tools.DeclareResultJudge(tools.ToolNameMachineCommandExec, judgeCommandExecResult)
}

// judgeCommandExecResult 从命令执行输出里判定真实成败。
//
// 判据只认结构化字段，不靠输出文本猜测：`success` 缺失或输出不是 JSON 时一律当作成功
// （未声明形态就不猜，与「未声明参数形状则跳过校验」同一口径），只有明确 success:false 才算失败，
// 并把命令输出拼成原因——这串会随 ToolErrors 一路写入运行 Extra，成为提案判失败的依据
func judgeCommandExecResult(output string) (bool, string) {
	var res struct {
		Success *bool  `json:"success"`
		Output  string `json:"output"`
	}
	if err := json.Unmarshal([]byte(output), &res); err != nil || res.Success == nil {
		return true, ""
	}
	if *res.Success {
		return true, ""
	}
	reason := strings.TrimSpace(res.Output)
	if reason == "" {
		// 机器可读标记而不是英文句子：这串会经 ToolErrors 进提案 err、最终展示在中文界面上，
		// 在工具层拼话术等于把不可翻译的句子塞进用户可见区（docs/server/i18n.md）
		reason = "exit status != 0"
	}
	// 与 ToolErrors 的截断口径一致：工具输出可能很长（含整个 stderr），只需留可定位的一段
	if runes := []rune(reason); len(runes) > 300 {
		reason = string(runes[:300]) + "...(截断)"
	}
	return false, reason
}

// commandOfArgs 从工具入参 JSON 取 command 字段；解析失败或无该字段返回空串
// （空命令 isWhitelistCommand 判 false → Dangerous，fail-closed，绝不因解析不了就放行）
func commandOfArgs(argsJSON string) string {
	var p CommandExecParam
	if err := json.Unmarshal([]byte(argsJSON), &p); err != nil {
		return ""
	}
	return p.Command
}

// agentCommandExecTimeout Agent 执行机器命令的硬超时：白名单里的只读命令也可能永久阻塞
// （tail -f、top -b 不带 -bn、大目录 du），不设上限会挂死 SSH 会话与整轮 turn。诊断类命令秒级返回，30s 足够
const agentCommandExecTimeout = 30 * time.Second

func GetCommandExec() (tool.InvokableTool, error) {

	return utils.InferTool(tools.ToolNameMachineCommandExec,
		i18n.T(imsg.MachineCommandExecToolInfo),
		func(ctx context.Context, param *CommandExecParam) (*CommandExecOutput, error) {

			tools.TryApplyResumedParams(ctx, param)
			// 检查必要参数，触发参数完善
			if param.AuthCertName == "" {
				if err := tools.InterruptOrResumeParamCompletion(ctx, tools.ToolNameMachineCommandExec, param, i18n.TC(ctx, imsg.MachineInfoIncomplete), "machine", []tools.CompletionParamInfo{
					{Param: "authCertName", Name: "授权凭证名称"},
				}, queryMachineOptions(ctx)); err != nil {
					return nil, err
				}
			}

			// 检查命令是否为空
			if param.Command == "" {
				return nil, tools.NewToolError(fmt.Errorf("%s", i18n.TC(ctx, imsg.MissingRequiredParams)), tools.RecoverRetry)
			}

			// 执行目的必填：供用户审批与事后审计理解（缺失时要求模型重试补充）
			if param.Remark == "" {
				return nil, tools.NewToolError(fmt.Errorf("remark parameter is required: describe the purpose of this command"), tools.RecoverRetry)
			}

			// 获取机器客户端（提前获取，用于 MachineCmdConf 检查）
			cli, err := application.GetMachineApp().GetCliByAc(ctx, param.AuthCertName)
			if err != nil {
				// 模型首轮不查清单、直接拿机器名/ip 当凭证名传（实测固定造成一次 `ac not found` 空转）。
				// 该机器只有唯一可操作凭证时不存在歧义，确定性纠正；多凭证不猜，保持原错交给参数补全由人选
				if cert, ok := resolveUniqueAuthCert(ctx, param.AuthCertName); ok {
					if retried, retryErr := application.GetMachineApp().GetCliByAc(ctx, cert); retryErr == nil {
						param.AuthCertName, cli, err = cert, retried, nil
					}
				}
				if err != nil {
					return nil, tools.NewToolError(fmt.Errorf("auth cert %q: %w", param.AuthCertName, err), tools.RecoverRetry)
				}
			}

			// 管理员配置的命令安全策略：与 Web 终端、非交互 API 共用同一个前置校验入口。
			// 放在向用户请求审批之前——被禁止的命令不该先弹一次「是否执行」再失败
			notice, err := application.CheckMachineCmd(ctx, cli.Info.CodePath, param.Command)
			if err != nil {
				// 这是管理员策略的结论而不是可恢复的调用失败：给模型重试机会等于让它换个写法绕过治理
				// （改大小写、拆成多条、套个管道都是常见走法），因此按不可重试上抛
				return nil, tools.NewToolError(err, tools.RecoverNone)
			}

			// Agent 侧独立关卡：白名单外的命令仍需用户确认，与管理员策略是两回事
			needApproval := !isWhitelistCommand(param.Command)

			if needApproval {
				// 触发审批中断
				if err := tools.InterruptOrResumeApproval(ctx, tools.ToolNameMachineCommandExec, param, i18n.TC(ctx, imsg.CommandExecApprovalReason)); err != nil {
					return nil, err
				}
			}

			// 执行命令：带硬超时，防只读命令阻塞挂死会话（cli.Run 无超时，tail -f/top 会永久挂住）
			output, err := cli.RunWithTimeout(agentCommandExecTimeout, param.Command)
			success := err == nil
			if errors.Is(err, mcm.ErrRunTimeout) {
				output = fmt.Sprintf("%s\n[command execution timed out after %s, session closed]", output, agentCommandExecTimeout)
			}
			if notice != "" {
				// 提醒随输出一起给回模型：Agent 场景没有终端可写，模型也不该不知道自己踩了管理员的提醒规则
				output = notice + "\n" + output
			}

			// 从 CLI 中获取机器信息
			machineInfo := cli.Info

			// 即使执行失败也返回输出，让AI能看到错误信息
			return &CommandExecOutput{
				AuthCertName: param.AuthCertName,
				MachineId:    machineInfo.Id,
				MachineName:  machineInfo.Name,
				MachineIp:    machineInfo.Ip,
				MachinePort:  machineInfo.Port,
				Username:     machineInfo.Username,
				Output:       output,
				Success:      success,
			}, nil
		},
	)
}

// ============================================================================
// AI Agent 白名单规则
//
// 仅包含「无 shell 逃逸能力」且「无对外交互 / 无写文件 / 无改配置副作用」的纯只读查询命令。
// 判定同时驱动两处且必须同源：只读诊断档的内容级能力（DeclareContentCapability）与工具体审批
// （needApproval）。以下命令曾因「看着只读实则有副作用」被移出免审批、改为需审批：
//   - curl/wget：对外网络交互 / 下载写盘
//   - ping/nslookup/dig/host：对外网络交互
//   - ip/ifconfig：可改网络配置（ip route add、ifconfig eth0 down）
//   - sort：`sort -o file` 写文件，且 -o 不含重定向符会逃过 HasUnsafePattern
//   - yum/apt：查询子命令仍会联网访问仓库，且与安装/卸载同工具难区分
//   - hostname/date：`hostname X` 改主机名、`date -s` 改系统时间（破坏证书/Kerberos/日志时序）
//   - uniq/tree/mpstat：GNU uniq 第二参数是输出文件、tree -o / mpstat -o 写文件（AllowedArgs 只锁子命令、拦不住这些 flag）
//   - less/more/zless：分页器 `!cmd`/`+!cmd` 可起 shell，即便非交互也保守移出，用 cat/head/tail 替代
// 注意：以下命令已从白名单移除（因其内建 shell 逃逸能力，可绕过审批执行任意命令）：
//   - awk：可通过 system() 执行任意命令
//   - find：可通过 -exec/-execdir 执行任意命令
//   - sed：GNU sed 的 e 命令可执行 shell 命令
//
// 这些命令在 AI Agent 场景中使用时会触发审批，用户秒级审批后放行。
// ============================================================================

// getWhitelistRules 返回 AI Agent 白名单命令规则列表
func getWhitelistRules() []mcm.WhitelistRule {
	return []mcm.WhitelistRule{
		// 系统信息查询
		{Command: "uname"},   // 系统信息
		{Command: "whoami"},  // 当前用户
		{Command: "id"},      // 用户信息
		{Command: "pwd"},     // 当前目录
		{Command: "cal"},     // 日历
		{Command: "uptime"},  // 运行时间
		{Command: "w"},       // 登录用户
		{Command: "who"},     // 登录用户
		{Command: "last"},    // 登录历史
		{Command: "lastlog"}, // 最后登录

		// 硬件和系统状态
		{Command: "lscpu"},   // CPU信息
		{Command: "lsblk"},   // 块设备
		{Command: "lspci"},   // PCI设备
		{Command: "lsusb"},   // USB设备
		{Command: "free"},    // 内存使用
		{Command: "df"},      // 磁盘使用
		{Command: "du"},      // 目录大小
		{Command: "top"},     // 进程状态
		{Command: "htop"},    // 进程状态
		{Command: "vmstat"},  // 虚拟内存
		{Command: "iostat"},  // IO统计
		{Command: "netstat"}, // 网络统计
		{Command: "ss"},      // 网络统计

		// 文件和目录查看（注意：find 因 -exec 逃逸能力已移除）
		{Command: "ls"},      // 列出文件
		{Command: "dir"},     // 列出文件
		{Command: "locate"},  // 查找文件
		{Command: "which"},   // 查找命令
		{Command: "whereis"}, // 查找程序
		{Command: "file"},    // 文件类型
		{Command: "stat"},    // 文件状态
		{Command: "wc"},      // 统计行数

		// 文件内容查看
		{Command: "cat"},   // 查看文件
		{Command: "tac"},   // 反向查看
		{Command: "head"},  // 查看前几行
		{Command: "tail"},  // 查看后几行
		{Command: "grep"},  // 搜索文本
		{Command: "egrep"}, // 扩展搜索
		{Command: "fgrep"}, // 固定搜索
		{Command: "zcat"},  // 查看压缩文件
		{Command: "zgrep"}, // 搜索压缩文件

		// 文本处理（注意：awk、sed 因 shell 逃逸能力已移除）
		{Command: "cut"},       // 截取文本
		{Command: "paste"},     // 合并文本
		{Command: "tr"},        // 转换字符
		{Command: "diff"},      // 比较文件
		{Command: "cmp"},       // 比较文件
		{Command: "md5sum"},    // MD5校验
		{Command: "sha256sum"}, // SHA256校验

		// 输出和打印
		{Command: "echo"},   // 输出文本
		{Command: "printf"}, // 格式化输出

		// 网络状态查询仅保留只读的 ss/netstat（见上）；ping/dig/nslookup/host/curl/wget（对外交互）、
		// ifconfig/ip（可改网络配置）均已移出，改为需审批

		// 进程查看
		{Command: "ps"},     // 进程状态
		{Command: "pgrep"},  // 查找进程
		{Command: "pstree"}, // 进程树

		// 包管理工具（yum/apt/rpm/dpkg）全部移出免审批：查询子命令仍联网访问仓库，
		// 且只读查询与安装/卸载难以靠子命令白名单可靠区分，统一需审批
	}
}

// isWhitelistCommand 判断命令是否完全在白名单中，可以自动执行。
// 使用共享命令分析器（mcm.CmdAnalyzer）进行词法分析和安全检测，
// 确保与所有入口（终端、API）使用一致的命令解析逻辑。
func isWhitelistCommand(cmd string) bool {
	return mcm.IsWhitelistCommand(cmd, getWhitelistRules())
}
