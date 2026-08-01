package shared

import (
	"bufio"
	"context"
	"fmt"
	"mayfly-go/cli/client"
	"mayfly-go/cli/config"
	"mayfly-go/cli/terminal"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// 全局状态（由 root 命令初始化）
var (
	JsonOutput         bool
	Quiet              bool
	Verbose            bool
	Timeout            int
	DryRun             bool // 预演模式（不实际执行破坏性操作）
	LoadedRefreshToken string
	GlobalCtx          context.Context = context.Background()
)

// ReadLine 从 reader 读取一行并去除首尾空白。
// ok 为 false 表示 stdin 已关闭（EOF）或读取失败，交互循环应立即退出。
func ReadLine(reader *bufio.Reader) (input string, ok bool) {
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return "", false
	}
	return strings.TrimSpace(line), true
}

// GetServer 获取服务器地址（优先级：--server flag > MAYFLY_SERVER 环境变量 > 配置文件）
func GetServer(cmd *cobra.Command) string {
	if cmd.Flags().Changed("server") {
		val, _ := cmd.Flags().GetString("server")
		return val
	}
	if env := os.Getenv("MAYFLY_SERVER"); env != "" {
		return env
	}
	val, _ := cmd.Flags().GetString("server")
	return val
}

// GetToken 获取 token（优先级：--token flag > MAYFLY_TOKEN 环境变量 > 配置文件）
func GetToken(cmd *cobra.Command) string {
	if cmd.Flags().Changed("token") {
		val, _ := cmd.Flags().GetString("token")
		return val
	}
	if env := os.Getenv("MAYFLY_TOKEN"); env != "" {
		return env
	}
	val, _ := cmd.Flags().GetString("token")
	return val
}

// NewClient 创建 API 客户端（自动配置 refreshToken 及刷新回调）
func NewClient(cmd *cobra.Command) *client.ApiClient {
	apiClient := client.NewApiClient(GetServer(cmd), GetToken(cmd)).
		WithContext(GlobalCtx).
		WithVerbose(Verbose).
		WithTimeout(Timeout)

	if LoadedRefreshToken != "" {
		configPath, _ := cmd.Flags().GetString("config")
		apiClient.SetRefreshToken(LoadedRefreshToken, func(accessToken, refreshToken string) {
			cfg := &config.Config{
				Server:       GetServer(cmd),
				Token:        accessToken,
				RefreshToken: refreshToken,
			}
			_ = config.SaveConfig(cfg, configPath)
		})
	}

	return apiClient
}

// FormatAddress 格式化地址（处理 nil 值）
func FormatAddress(host, port interface{}) string {
	if host == nil {
		return fmt.Sprintf(":%v", port)
	}
	return fmt.Sprintf("%v:%v", host, port)
}

// PrintLoginPrompt 输出登录提示
func PrintLoginPrompt(server string) {
	terminal.PrintLoginPrompt(server)
}
