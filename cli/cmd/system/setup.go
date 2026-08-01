package system

import (
	"bufio"
	"fmt"
	"mayfly-go/cli/client"
	"mayfly-go/cli/cmd/shared"
	"mayfly-go/cli/config"
	"mayfly-go/cli/i18n"
	"mayfly-go/cli/terminal"
	"os"

	"github.com/spf13/cobra"
)

// ConfigCmd 配置向导命令
var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: i18n.T(i18n.MsgConfigShort),
	Long:  i18n.T(i18n.MsgConfigLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		server, _ := cmd.Flags().GetString("server")
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")

		if cmd.Flags().Changed("username") && cmd.Flags().Changed("password") {
			return doLogin(cmd, server, username, password)
		}

		reader := bufio.NewReader(os.Stdin)

		fmt.Println("====================================")
		fmt.Printf("  %s\n", i18n.T(i18n.MsgConfigWizardTitle))
		fmt.Println("====================================")
		fmt.Println()

		fmt.Printf("%s", i18n.T(i18n.MsgConfigServerHint, "server", server))
		serverInput, ok := shared.ReadLine(reader)
		if !ok {
			return shared.Fail(i18n.MsgConfigLong, nil)
		}
		if serverInput != "" {
			server = serverInput
		}

		fmt.Print(i18n.T(i18n.MsgConfigUserHint))
		usernameInput, ok := shared.ReadLine(reader)
		if !ok {
			return shared.Fail(i18n.MsgConfigLong, nil)
		}
		if usernameInput != "" {
			username = usernameInput
		} else {
			username = "admin"
		}

		fmt.Print(i18n.T(i18n.MsgConfigPassHint))
		passwordInput, err := terminal.ReadPassword("")
		if err != nil {
			// 回退到普通输入
			passwordInput, ok = shared.ReadLine(reader)
			if !ok {
				return shared.Fail(i18n.MsgConfigLong, nil)
			}
		}
		if passwordInput != "" {
			password = passwordInput
		} else if envPwd := os.Getenv("MAYFLY_PASSWORD"); envPwd != "" {
			password = envPwd
		} else {
			return shared.Fail(i18n.MsgLoginNoPassword, nil)
		}

		return doLogin(cmd, server, username, password)
	},
}

// LoginCmd 独立登录命令
var LoginCmd = &cobra.Command{
	Use:   "login",
	Short: i18n.T(i18n.MsgLoginShort),
	Long:  i18n.T(i18n.MsgLoginLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		server, _ := cmd.Flags().GetString("server")
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")

		if password == "" {
			password = os.Getenv("MAYFLY_PASSWORD")
		}
		if password == "" {
			return shared.Fail(i18n.MsgLoginNoPassword, nil)
		}

		return doLogin(cmd, server, username, password)
	},
}

func doLogin(cmd *cobra.Command, server, username, password string) error {
	if !shared.Quiet && !shared.JsonOutput {
		fmt.Println(i18n.T(i18n.MsgLoginLogging))
	}

	// 创建无 token 的客户端用于登录
	apiClient := client.NewApiClient(server, "").WithContext(shared.GlobalCtx)

	if !shared.Quiet && !shared.JsonOutput {
		fmt.Print(i18n.T(i18n.MsgLoginStep1))
	}
	publicKey, err := apiClient.GetPublicKey()
	if err != nil {
		return shared.Fail(i18n.MsgLoginGetPubKey, err)
	}
	if !shared.Quiet && !shared.JsonOutput {
		fmt.Println("✓")
	}

	if !shared.Quiet && !shared.JsonOutput {
		fmt.Print(i18n.T(i18n.MsgLoginStep2))
	}
	encryptedPassword, err := client.EncryptPassword(password, publicKey)
	if err != nil {
		return shared.Fail(i18n.MsgLoginEncrypt, err)
	}
	if !shared.Quiet && !shared.JsonOutput {
		fmt.Println("✓")
	}

	if !shared.Quiet && !shared.JsonOutput {
		fmt.Print(i18n.T(i18n.MsgLoginStep3))
	}
	token, refreshToken, err := apiClient.Login(username, encryptedPassword)
	if err != nil {
		return shared.Fail(i18n.MsgLoginFailed, err)
	}
	if !shared.Quiet && !shared.JsonOutput {
		fmt.Println("✓")
	}

	cfg := &config.Config{
		Server:       server,
		Token:        token,
		RefreshToken: refreshToken,
	}

	configPath, _ := cmd.Flags().GetString("config")
	if err := config.SaveConfig(cfg, configPath); err != nil {
		return shared.Fail(i18n.MsgLoginSaveCfg, err)
	}

	if shared.JsonOutput {
		shared.PrintJSONSuccess(map[string]interface{}{
			"server":   server,
			"username": username,
			"token":    token,
			"config":   getConfigPath(configPath),
		})
	} else {
		fmt.Println()
		fmt.Printf("✅ %s\n", i18n.T(i18n.MsgConfigSaved))
		fmt.Println(i18n.T(i18n.MsgConfigPath, "path", getConfigPath(configPath)))
		fmt.Println()
		fmt.Println(i18n.T(i18n.MsgReadyToUse))
		fmt.Println("  mayfly-cli db list     # list databases")
		fmt.Println("  mayfly-cli ssh list    # list machines")
		fmt.Println("  mayfly-cli redis list  # list Redis")
		fmt.Println()
	}

	return nil
}

func getConfigPath(configPath string) string {
	if configPath != "" {
		return configPath
	}
	if path, err := config.DefaultPath(); err == nil {
		return path
	}
	return "~/" + config.DefaultFileName
}

// LogoutCmd 登出命令
var LogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: i18n.T(i18n.MsgLogoutShort),
	Long:  i18n.T(i18n.MsgLogoutLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		if configPath == "" {
			configPath, _ = config.DefaultPath()
		}

		// 清空配置中的 token
		cfg := &config.Config{}
		if err := config.SaveConfig(cfg, configPath); err != nil {
			return shared.Fail(i18n.MsgLogoutFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(map[string]interface{}{"loggedOut": true})
			return nil
		}

		fmt.Printf("%s%s%s\n", terminal.Green, i18n.T(i18n.MsgLogoutSuccess), terminal.Reset)
		return nil
	},
}

// WhoamiCmd 显示当前登录状态
var WhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: i18n.T(i18n.MsgWhoamiShort),
	Long:  i18n.T(i18n.MsgWhoamiLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		server := shared.GetServer(cmd)
		token := shared.GetToken(cmd)

		status := map[string]interface{}{
			"server":   server,
			"loggedIn": false,
		}

		if token != "" {
			apiClient := shared.NewClient(cmd)
			if apiClient.CheckAuth() {
				status["loggedIn"] = true
			}
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(status)
			return nil
		}

		fmt.Printf("\n%s%s%s: %s\n", terminal.Bold, i18n.T(i18n.MsgWhoamiServer), terminal.Reset, server)
		if status["loggedIn"] == true {
			fmt.Printf("%s%s%s: %s%s%s\n", terminal.Bold, i18n.T(i18n.MsgWhoamiStatus), terminal.Reset, terminal.Green, i18n.T(i18n.MsgWhoamiOnline), terminal.Reset)
		} else {
			fmt.Printf("%s%s%s: %s%s%s\n", terminal.Bold, i18n.T(i18n.MsgWhoamiStatus), terminal.Reset, terminal.Red, i18n.T(i18n.MsgWhoamiOffline), terminal.Reset)
		}
		fmt.Println()
		return nil
	},
}

func init() {
	ConfigCmd.GroupID = "system"
	LoginCmd.GroupID = "system"
	LogoutCmd.GroupID = "system"
	WhoamiCmd.GroupID = "system"

	ConfigCmd.Flags().String("config", "", i18n.T(i18n.MsgConfigFlagPath))
	ConfigCmd.Flags().StringP("username", "u", "", i18n.T(i18n.MsgConfigFlagUser))
	ConfigCmd.Flags().StringP("password", "p", "", i18n.T(i18n.MsgConfigFlagPass))

	LoginCmd.Flags().StringP("username", "u", "admin", i18n.T(i18n.MsgLoginFlagUser))
	LoginCmd.Flags().StringP("password", "p", "", i18n.T(i18n.MsgLoginFlagPass))

	LogoutCmd.Flags().String("config", "", i18n.T(i18n.MsgConfigFlagPath))
}
