package machinetool

import (
	"context"
	"fmt"
	"mayfly-go/internal/ai/imsg"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/internal/machine/application"
	"mayfly-go/pkg/i18n"
	"strings"

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

			// 白名单命令检测：不在白名单中的命令需要审批
			if !isWhitelistCommand(param.Command) {
				// 触发审批中断
				if err := tools.InterruptOrResumeApproval(ctx, toolDesc, param, i18n.TC(ctx, imsg.CommandExecApprovalReason)); err != nil {
					return nil, err
				}
			}

			// 获取机器客户端
			cli, err := application.GetMachineApp().GetCliByAc(ctx, param.AuthCertName)
			if err != nil {
				return nil, tools.NewToolError(err, tools.RecoverRetry)
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

// isWhitelistCommand 判断命令是否在白名单中，可以自动执行
// 白名单包含：查询、统计、查看类命令，不包含修改系统资源的命令
func isWhitelistCommand(cmd string) bool {
	if cmd == "" {
		return false
	}

	// 定义白名单命令规则
	type whitelistRule struct {
		command     string   // 命令名
		allowedArgs []string // 允许的参数模式（空表示所有参数都允许）
	}

	whitelistRules := []whitelistRule{
		// 系统信息查询
		{command: "uname", allowedArgs: nil},    // 系统信息
		{command: "hostname", allowedArgs: nil}, // 主机名
		{command: "whoami", allowedArgs: nil},   // 当前用户
		{command: "id", allowedArgs: nil},       // 用户信息
		{command: "pwd", allowedArgs: nil},      // 当前目录
		{command: "date", allowedArgs: nil},     // 日期时间
		{command: "cal", allowedArgs: nil},      // 日历
		{command: "uptime", allowedArgs: nil},   // 运行时间
		{command: "w", allowedArgs: nil},        // 登录用户
		{command: "who", allowedArgs: nil},      // 登录用户
		{command: "last", allowedArgs: nil},     // 登录历史
		{command: "lastlog", allowedArgs: nil},  // 最后登录

		// 硬件和系统状态
		{command: "lscpu", allowedArgs: nil},   // CPU信息
		{command: "lsblk", allowedArgs: nil},   // 块设备
		{command: "lspci", allowedArgs: nil},   // PCI设备
		{command: "lsusb", allowedArgs: nil},   // USB设备
		{command: "free", allowedArgs: nil},    // 内存使用
		{command: "df", allowedArgs: nil},      // 磁盘使用
		{command: "du", allowedArgs: nil},      // 目录大小
		{command: "top", allowedArgs: nil},     // 进程状态
		{command: "htop", allowedArgs: nil},    // 进程状态
		{command: "vmstat", allowedArgs: nil},  // 虚拟内存
		{command: "iostat", allowedArgs: nil},  // IO统计
		{command: "mpstat", allowedArgs: nil},  // CPU统计
		{command: "netstat", allowedArgs: nil}, // 网络统计
		{command: "ss", allowedArgs: nil},      // 网络统计
		{command: "uptime", allowedArgs: nil},  // 运行时间

		// 文件和目录查看
		{command: "ls", allowedArgs: nil},      // 列出文件
		{command: "dir", allowedArgs: nil},     // 列出文件
		{command: "find", allowedArgs: nil},    // 查找文件（只读）
		{command: "locate", allowedArgs: nil},  // 查找文件
		{command: "which", allowedArgs: nil},   // 查找命令
		{command: "whereis", allowedArgs: nil}, // 查找程序
		{command: "tree", allowedArgs: nil},    // 目录树
		{command: "file", allowedArgs: nil},    // 文件类型
		{command: "stat", allowedArgs: nil},    // 文件状态
		{command: "wc", allowedArgs: nil},      // 统计行数
		{command: "sort", allowedArgs: nil},    // 排序
		{command: "uniq", allowedArgs: nil},    // 去重

		// 文件内容查看
		{command: "cat", allowedArgs: nil},   // 查看文件
		{command: "tac", allowedArgs: nil},   // 反向查看
		{command: "less", allowedArgs: nil},  // 分页查看
		{command: "more", allowedArgs: nil},  // 分页查看
		{command: "head", allowedArgs: nil},  // 查看前几行
		{command: "tail", allowedArgs: nil},  // 查看后几行
		{command: "grep", allowedArgs: nil},  // 搜索文本
		{command: "egrep", allowedArgs: nil}, // 扩展搜索
		{command: "fgrep", allowedArgs: nil}, // 固定搜索
		{command: "zcat", allowedArgs: nil},  // 查看压缩文件
		{command: "zless", allowedArgs: nil}, // 分页查看压缩文件
		{command: "zgrep", allowedArgs: nil}, // 搜索压缩文件

		// 文本处理
		{command: "awk", allowedArgs: nil},       // 文本处理
		{command: "sed", allowedArgs: nil},       // 流编辑器（只读使用）
		{command: "cut", allowedArgs: nil},       // 截取文本
		{command: "paste", allowedArgs: nil},     // 合并文本
		{command: "tr", allowedArgs: nil},        // 转换字符
		{command: "diff", allowedArgs: nil},      // 比较文件
		{command: "cmp", allowedArgs: nil},       // 比较文件
		{command: "md5sum", allowedArgs: nil},    // MD5校验
		{command: "sha256sum", allowedArgs: nil}, // SHA256校验

		// 输出和打印
		{command: "echo", allowedArgs: nil},   // 输出文本
		{command: "printf", allowedArgs: nil}, // 格式化输出

		// 网络查询
		{command: "ping", allowedArgs: nil},     // 网络连通性
		{command: "nslookup", allowedArgs: nil}, // DNS查询
		{command: "dig", allowedArgs: nil},      // DNS查询
		{command: "host", allowedArgs: nil},     // DNS查询
		{command: "curl", allowedArgs: nil},     // HTTP请求（GET）
		{command: "wget", allowedArgs: nil},     // 下载文件
		{command: "ifconfig", allowedArgs: nil}, // 网络配置
		{command: "ip", allowedArgs: nil},       // 网络配置

		// 进程查看
		{command: "ps", allowedArgs: nil},     // 进程状态
		{command: "pgrep", allowedArgs: nil},  // 查找进程
		{command: "pstree", allowedArgs: nil}, // 进程树

		// 包管理查询
		{command: "rpm", allowedArgs: []string{"-q", "-qa", "-qi", "-ql"}}, // 查询包
		{command: "dpkg", allowedArgs: []string{"-l", "-s"}},               // 查询包
		{command: "yum", allowedArgs: []string{"list", "info", "search"}},  // 查询包
		{command: "apt", allowedArgs: []string{"list", "show", "search"}},  // 查询包
	}

	// 词法分析命令
	tokens := tokenize(cmd)

	// 遍历 tokens，查找命令及其参数
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

		// 跳过操作符和引号内容
		if isOperator(token) || isQuotedString(token) {
			continue
		}

		// 提取命令名（去除路径）
		cmdName := token
		if idx := strings.LastIndex(token, "/"); idx >= 0 {
			cmdName = token[idx+1:]
		}

		// 检查是否匹配白名单命令
		for _, rule := range whitelistRules {
			if cmdName != rule.command {
				continue
			}

			// 如果规则没有限制参数，该命令所有参数都允许
			if len(rule.allowedArgs) == 0 {
				return true
			}

			// 检查参数是否在允许列表中
			args := extractArgs(tokens[i+1:])
			for _, arg := range args {
				// 如果参数不在允许列表中，需要审批
				if !isArgAllowed(arg, rule.allowedArgs) {
					return false
				}
			}
			return true
		}
	}

	// 不在白名单中的命令需要审批
	return false
}

// extractArgs 从 tokens 中提取参数（排除操作符）
func extractArgs(tokens []string) []string {
	var args []string
	for _, token := range tokens {
		if !isOperator(token) && !isQuotedString(token) {
			args = append(args, token)
		}
	}
	return args
}

// isArgAllowed 检查参数是否在允许列表中
func isArgAllowed(arg string, allowedArgs []string) bool {
	// 跳过选项标志（- 开头的）
	if strings.HasPrefix(arg, "-") {
		return true
	}

	// 检查是否在允许列表中
	for _, allowed := range allowedArgs {
		if arg == allowed {
			return true
		}
	}
	return false
}

// tokenize 将命令字符串分割成 tokens
func tokenize(cmd string) []string {
	var tokens []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for i := 0; i < len(cmd); i++ {
		ch := cmd[i]

		if escaped {
			current.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' && !inSingleQuote {
			current.WriteByte(ch)
			escaped = true
			continue
		}

		if ch == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			current.WriteByte(ch)
			continue
		}

		if ch == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			current.WriteByte(ch)
			continue
		}

		if inSingleQuote || inDoubleQuote {
			current.WriteByte(ch)
			continue
		}

		if isOperatorChar(ch) {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}

			operator := string(ch)
			if i+1 < len(cmd) && isOperatorChar(cmd[i+1]) {
				nextCh := cmd[i+1]
				if ch == nextCh {
					operator += string(nextCh)
					i++
				}
			}

			tokens = append(tokens, operator)
			continue
		}

		if ch == ' ' || ch == '\t' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteByte(ch)
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// isOperatorChar 判断字符是否为操作符字符
func isOperatorChar(ch byte) bool {
	return ch == '|' || ch == '&' || ch == ';' || ch == '>' || ch == '<'
}

// isOperator 判断 token 是否为操作符
func isOperator(token string) bool {
	if len(token) == 0 {
		return false
	}
	return isOperatorChar(token[0])
}

// isQuotedString 判断是否为引号字符串
func isQuotedString(token string) bool {
	if len(token) < 2 {
		return false
	}
	// 单引号或双引号包裹
	return (token[0] == '\'' && token[len(token)-1] == '\'') ||
		(token[0] == '"' && token[len(token)-1] == '"')
}
