package shared

import (
	"errors"
	"fmt"
	"mayfly-go/cli/client"
	"mayfly-go/cli/i18n"
	"mayfly-go/cli/terminal"
	"os"
	"strings"
)

// ErrSilent 表示错误信息已输出完毕，上层只需以非零退出码结束，无需重复打印
var ErrSilent = errors.New("silent error")

// 错误码定义（供 Agent 程序化处理）
const (
	ErrCodeSuccess    = 0  // 成功
	ErrCodeUnknown    = 1  // 未知错误
	ErrCodeAuth       = 10 // 认证错误（未登录/token过期）
	ErrCodeConnection = 11 // 连接错误（服务器不可达）
	ErrCodeValidation = 20 // 参数验证错误
	ErrCodeNotFound   = 21 // 资源未找到
	ErrCodeBusiness   = 30 // 业务逻辑错误
	ErrCodePermission = 31 // 权限不足
)

// ExitCode 全局退出码（由 Fail 设置，供 main 读取）
var ExitCode = ErrCodeUnknown

// AgentMode Agent 模式（禁用交互，确保机器可读输出）
var AgentMode bool

// Fail 统一失败处理：根据输出模式渲染错误，并保证退出码非零。
func Fail(msgID string, err error) error {
	return FailWithCode(msgID, err, ErrCodeUnknown)
}

// FailWithCode 带错误码的失败处理
func FailWithCode(msgID string, err error, code int) error {
	if errors.Is(err, ErrSilent) {
		return err
	}

	ExitCode = code

	var msg string
	if err != nil {
		msg = i18n.T(msgID, "err", err.Error())
		if !strings.Contains(msg, err.Error()) {
			msg = msg + ": " + err.Error()
		}
	} else {
		msg = i18n.T(msgID)
	}

	if JsonOutput || AgentMode {
		PrintJSONErrorWithCode(msg, err, code)
		return ErrSilent
	}
	return errors.New(msg)
}

// HandleError 统一处理连接/认证类错误
func HandleError(err error, server string) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, client.ErrTokenExpired) {
		ExitCode = ErrCodeAuth
		if JsonOutput || AgentMode {
			PrintJSONErrorWithCode(i18n.T(i18n.MsgTokenExpired), err, ErrCodeAuth)
			return ErrSilent
		}
		fmt.Fprintf(os.Stderr, "%s\n", i18n.T(i18n.MsgTokenExpired))
		terminal.PrintLoginPrompt(server)
		return ErrSilent
	}

	// 使用结构化错误类型判断
	var connErr *client.ConnectionError
	if errors.As(err, &connErr) {
		ExitCode = ErrCodeAuth
		if JsonOutput || AgentMode {
			PrintJSONErrorWithCode(i18n.T(i18n.MsgHelpersAuthRequired), err, ErrCodeAuth)
			return ErrSilent
		}
		terminal.PrintLoginPrompt(server)
		return ErrSilent
	}

	// 兖底：字符串匹配（兼容旧路径）
	if IsConnectionError(err.Error()) {
		ExitCode = ErrCodeConnection
		if JsonOutput || AgentMode {
			PrintJSONErrorWithCode(i18n.T(i18n.MsgHelpersAuthRequired), err, ErrCodeConnection)
			return ErrSilent
		}
		terminal.PrintLoginPrompt(server)
		return ErrSilent
	}

	return err
}

// IsConnectionError 判断是否为连接/认证错误（兜底字符串匹配）
func IsConnectionError(errMsg string) bool {
	return strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "code=501") ||
		strings.Contains(errMsg, "code=502")
}

// ExitWithError 输出最终错误并以非零退出码结束（供 main 调用）
func ExitWithError(err error) {
	if !errors.Is(err, ErrSilent) {
		if JsonOutput || AgentMode || ArgsHasJSONFlag() {
			PrintJSONErrorWithCode(i18n.T(i18n.MsgCmdExecFailed), err, ExitCode)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
	}
	os.Exit(ExitCode)
}

// ArgsHasJSONFlag 在标志解析失败时兼容检测 --json
func ArgsHasJSONFlag() bool {
	for _, arg := range os.Args[1:] {
		if arg == "--json" {
			return true
		}
	}
	return false
}
