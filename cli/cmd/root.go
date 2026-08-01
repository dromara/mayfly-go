package cmd

import (
	"context"
	"fmt"
	"mayfly-go/cli/cmd/db"
	"mayfly-go/cli/cmd/mongo"
	"mayfly-go/cli/cmd/redis"
	"mayfly-go/cli/cmd/shared"
	"mayfly-go/cli/cmd/ssh"
	"mayfly-go/cli/cmd/system"
	"mayfly-go/cli/config"
	"mayfly-go/cli/i18n"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	profile string

	// Version 构建版本号，可通过 -ldflags "-X mayfly-go/cli/cmd.Version=x.y.z" 注入
	Version = "dev"

	rootCmd = &cobra.Command{
		Use:           "mayfly-cli",
		Short:         i18n.T(i18n.MsgRootShort),
		Long:          i18n.T(i18n.MsgRootLong),
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// 记录当前命令（用于 Agent 模式 JSON 元数据）
			shared.CurrentCommand = cmd.CommandPath()

			// Agent 模式自动启用 JSON 输出
			if shared.AgentMode {
				shared.JsonOutput = true
			}

			if skipAuthCommands[cmd.Name()] {
				return nil
			}
			if shared.GetToken(cmd) == "" {
				server := shared.GetServer(cmd)
				if shared.JsonOutput {
					shared.PrintJSONError("not logged in, run: mayfly-cli login -s "+server, nil)
					return shared.ErrSilent
				}
				fmt.Fprintf(os.Stderr, "Error: not logged in\n")
				fmt.Fprintf(os.Stderr, "  Run: mayfly-cli login -s %s -u <username> -p <password>\n", server)
				return shared.ErrSilent
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return showDashboard(cmd)
		},
	}
)

// skipAuthCommands 无需 token 即可执行的命令
var skipAuthCommands = map[string]bool{
	"config":     true,
	"login":      true,
	"logout":     true,
	"whoami":     true,
	"commands":   true,
	"ping":       true,
	"help":       true,
	"completion": true,
	"version":    true,
}

func Execute() error {
	// 设置信号处理，支持 Ctrl+C 取消请求
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	shared.GlobalCtx = ctx

	return rootCmd.ExecuteContext(ctx)
}

// ExitWithError 供 main 调用的错误退出入口
func ExitWithError(err error) {
	shared.ExitWithError(err)
}

func init() {
	rootCmd.Version = Version

	cobra.OnInitialize(initConfig)

	// 命令分组
	rootCmd.AddGroup(
		&cobra.Group{ID: "resource", Title: "Resource Commands:"},
		&cobra.Group{ID: "system", Title: "System Commands:"},
	)

	// 注册子模块命令
	rootCmd.AddCommand(db.Cmd)
	rootCmd.AddCommand(ssh.Cmd)
	rootCmd.AddCommand(redis.Cmd)
	rootCmd.AddCommand(mongo.Cmd)
	rootCmd.AddCommand(system.PingCmd)
	rootCmd.AddCommand(system.ConfigCmd)
	rootCmd.AddCommand(system.LoginCmd)
	rootCmd.AddCommand(system.LogoutCmd)
	rootCmd.AddCommand(system.WhoamiCmd)
	rootCmd.AddCommand(system.CommandsCmd)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", i18n.T(i18n.MsgConfigFlagPath))
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", i18n.T(i18n.MsgFlagProfile))
	rootCmd.PersistentFlags().StringP("server", "s", "http://localhost:8888", "mayfly-go server address")
	rootCmd.PersistentFlags().StringP("token", "t", "", "access token")
	rootCmd.PersistentFlags().BoolVarP(&shared.Verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&shared.JsonOutput, "json", false, "output results in JSON format (for agents and scripts)")
	rootCmd.PersistentFlags().BoolVarP(&shared.Quiet, "quiet", "q", false, "quiet mode, output result data only")
	rootCmd.PersistentFlags().BoolVar(&shared.AgentMode, "agent", false, i18n.T(i18n.MsgFlagAgent))
	rootCmd.PersistentFlags().BoolVar(&shared.DryRun, "dry-run", false, i18n.T(i18n.MsgFlagDryRun))
	rootCmd.PersistentFlags().String("lang", "", "language (zh-cn/en, default: auto-detect)")
	rootCmd.PersistentFlags().IntVar(&shared.Timeout, "timeout", 30, i18n.T(i18n.MsgFlagTimeout))
	rootCmd.PersistentFlags().StringVar((*string)(&shared.OutputFmt), "output", "table", i18n.T(i18n.MsgFlagOutput))
}

func initConfig() {
	// 初始化语言
	if lang, _ := rootCmd.Flags().GetString("lang"); lang != "" {
		i18n.SetLang(lang)
	} else {
		i18n.InitLang()
	}

	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		if !shared.Quiet {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
		return
	}

	// 如果指定了 --profile，使用对应 profile 的配置
	if profile != "" {
		if p := cfg.Profiles[profile]; p != nil {
			if !rootCmd.Flags().Changed("server") && p.Server != "" {
				rootCmd.Flags().Set("server", p.Server)
			}
			if !rootCmd.Flags().Changed("token") && p.Token != "" {
				rootCmd.Flags().Set("token", p.Token)
			}
			shared.LoadedRefreshToken = p.RefreshToken
			return
		}
	}

	// 使用默认配置或当前活动 profile
	if activeProfile := cfg.GetActiveProfile(); activeProfile != nil {
		if !rootCmd.Flags().Changed("server") && activeProfile.Server != "" {
			rootCmd.Flags().Set("server", activeProfile.Server)
		}
		if !rootCmd.Flags().Changed("token") && activeProfile.Token != "" {
			rootCmd.Flags().Set("token", activeProfile.Token)
		}
		shared.LoadedRefreshToken = activeProfile.RefreshToken
		return
	}

	// 回退到主配置
	if !rootCmd.Flags().Changed("server") && cfg.Server != "" {
		rootCmd.Flags().Set("server", cfg.Server)
	}

	if !rootCmd.Flags().Changed("token") && cfg.Token != "" {
		rootCmd.Flags().Set("token", cfg.Token)
	}

	shared.LoadedRefreshToken = cfg.RefreshToken
}
