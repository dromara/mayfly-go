package machinetool

import (
	"context"
	"fmt"
	"mayfly-go/internal/ai/imsg"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// CommandExecParam 命令执行参数
type CommandExecParam struct {
	AuthCertName string `json:"authCertName" jsonschema_description:"授权凭证名称"`
	Command      string `json:"command" jsonschema_description:"要执行的命令"`
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
func GetCommandExec() (tool.InvokableTool, error) {
	return utils.InferTool("MachineCommandExec",
		i18n.T(imsg.MachineCommandExecToolInfo),
		func(ctx context.Context, param *CommandExecParam) (*CommandExecOutput, error) {
			toolDesc := i18n.TC(ctx, imsg.MachineCommandExecToolDesc)

			tools.TryApplyResumedParams(ctx, param)
			// 检查必要参数，触发参数完善
			if param.AuthCertName == "" {
				if err := tools.InterruptOrResumeParamCompletion(ctx, toolDesc, param, i18n.TC(ctx, imsg.MachineInfoIncomplete), "machine", []tools.CompletionParamInfo{
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
				return nil, tools.NewToolError(err, tools.RecoverRetry)
			}

			// 判断命令是否需要审批：
			// 1. 白名单检测：不在白名单中的命令需要审批
			// 2. MachineCmdConf 检查：即使白名单命令，若匹配管理员配置的安全策略也需审批
			needApproval := false

			if !isWhitelistCommand(param.Command) {
				needApproval = true
			}

			// MachineCmdConf 策略检查：管理员可按机器标签配置命令过滤规则（正则匹配），
			// 所有入口（AI Agent、Web 终端、API）共享此策略，确保一致的安全控制
			if !needApproval {
				cmdConfs := application.GetMachineCmdConfApp().GetCmdConfsByMachineTags(ctx, cli.Info.CodePath...)
				if len(cmdConfs) > 0 {
					// 将 application.MachineCmd 转换为 mcm.CmdFilterRule 供共享分析器使用
					filters := make([]*mcm.CmdFilterRule, 0, len(cmdConfs))
					for _, mc := range cmdConfs {
						filters = append(filters, &mcm.CmdFilterRule{CmdRegexp: mc.CmdRegexp, Strategy: mc.Stratege})
					}
					if matched := mcm.MatchCmdFilters(param.Command, filters); matched != nil {
						logx.InfofContext(ctx, "[MachineCmdConf] command matched cmdConf rule, cmd=%s, strategy=%s", param.Command, matched.Strategy)
						needApproval = true
					}
				}
			}

			if needApproval {
				// 触发审批中断
				if err := tools.InterruptOrResumeApproval(ctx, toolDesc, param, i18n.TC(ctx, imsg.CommandExecApprovalReason)); err != nil {
					return nil, err
				}
			}

			// 执行命令
			output, err := cli.Run(param.Command)
			success := err == nil

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
// 仅包含无 shell 逃逸能力的只读查询命令。
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
		{Command: "uname"},    // 系统信息
		{Command: "hostname"}, // 主机名
		{Command: "whoami"},   // 当前用户
		{Command: "id"},       // 用户信息
		{Command: "pwd"},      // 当前目录
		{Command: "date"},     // 日期时间
		{Command: "cal"},      // 日历
		{Command: "uptime"},   // 运行时间
		{Command: "w"},        // 登录用户
		{Command: "who"},      // 登录用户
		{Command: "last"},     // 登录历史
		{Command: "lastlog"},  // 最后登录

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
		{Command: "mpstat"},  // CPU统计
		{Command: "netstat"}, // 网络统计
		{Command: "ss"},      // 网络统计

		// 文件和目录查看（注意：find 因 -exec 逃逸能力已移除）
		{Command: "ls"},      // 列出文件
		{Command: "dir"},     // 列出文件
		{Command: "locate"},  // 查找文件
		{Command: "which"},   // 查找命令
		{Command: "whereis"}, // 查找程序
		{Command: "tree"},    // 目录树
		{Command: "file"},    // 文件类型
		{Command: "stat"},    // 文件状态
		{Command: "wc"},      // 统计行数
		{Command: "sort"},    // 排序
		{Command: "uniq"},    // 去重

		// 文件内容查看
		{Command: "cat"},   // 查看文件
		{Command: "tac"},   // 反向查看
		{Command: "less"},  // 分页查看（非交互模式下无 shell 逃逸风险）
		{Command: "more"},  // 分页查看
		{Command: "head"},  // 查看前几行
		{Command: "tail"},  // 查看后几行
		{Command: "grep"},  // 搜索文本
		{Command: "egrep"}, // 扩展搜索
		{Command: "fgrep"}, // 固定搜索
		{Command: "zcat"},  // 查看压缩文件
		{Command: "zless"}, // 分页查看压缩文件
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

		// 网络查询
		{Command: "ping"},     // 网络连通性
		{Command: "nslookup"}, // DNS查询
		{Command: "dig"},      // DNS查询
		{Command: "host"},     // DNS查询
		{Command: "curl"},     // HTTP请求（GET）
		{Command: "wget"},     // 下载文件
		{Command: "ifconfig"}, // 网络配置
		{Command: "ip"},       // 网络配置

		// 进程查看
		{Command: "ps"},     // 进程状态
		{Command: "pgrep"},  // 查找进程
		{Command: "pstree"}, // 进程树

		// 包管理查询（仅允许子命令型包管理工具，allowedArgs 约束第一个非选项参数即子命令）
		// 注意：rpm/dpkg 不在白名单——它们的查询/安装/卸载操作都通过标志区分（如 -q/-e），
		// 当前 allowedArgs 仅约束子命令，无法有效区分标志类参数，故需审批。
		{Command: "yum", AllowedArgs: []string{"list", "info", "search"}}, // 查询包
		{Command: "apt", AllowedArgs: []string{"list", "show", "search"}}, // 查询包
	}
}

// isWhitelistCommand 判断命令是否完全在白名单中，可以自动执行。
// 使用共享命令分析器（mcm.CmdAnalyzer）进行词法分析和安全检测，
// 确保与所有入口（终端、API）使用一致的命令解析逻辑。
func isWhitelistCommand(cmd string) bool {
	return mcm.IsWhitelistCommand(cmd, getWhitelistRules())
}
