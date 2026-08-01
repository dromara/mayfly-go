package system

import (
	"mayfly-go/cli/cmd/shared"
	"mayfly-go/cli/i18n"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// CommandInfo 命令信息（机器可读）
type CommandInfo struct {
	Name        string        `json:"name"`
	Use         string        `json:"use"`
	Short       string        `json:"short"`
	Aliases     []string      `json:"aliases,omitempty"`
	Flags       []FlagInfo    `json:"flags,omitempty"`
	SubCommands []CommandInfo `json:"sub_commands,omitempty"`
	Examples    []string      `json:"examples,omitempty"`
}

// FlagInfo 标志信息
type FlagInfo struct {
	Name        string `json:"name"`
	Shorthand   string `json:"shorthand,omitempty"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
}

// CommandsCmd 列出所有命令（机器可读）
var CommandsCmd = &cobra.Command{
	Use:   "commands",
	Short: i18n.T(i18n.MsgAgentCommandsShort),
	Long:  i18n.T(i18n.MsgAgentCommandsLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := cmd.Root()
		commands := collectCommands(root)

		result := map[string]interface{}{
			"version":  root.Version,
			"commands": commands,
		}

		shared.PrintJSONSuccess(result)
		return nil
	},
}

// collectCommands 递归收集命令信息
func collectCommands(cmd *cobra.Command) []CommandInfo {
	var commands []CommandInfo

	for _, c := range cmd.Commands() {
		// 跳过隐藏命令和 help 命令
		if c.Hidden || c.Name() == "help" || c.Name() == "completion" {
			continue
		}

		info := CommandInfo{
			Name:    c.CommandPath(),
			Use:     c.Use,
			Short:   c.Short,
			Aliases: c.Aliases,
		}

		// 收集标志
		c.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Hidden {
				return
			}
			flagInfo := FlagInfo{
				Name:        f.Name,
				Shorthand:   f.Shorthand,
				Type:        f.Value.Type(),
				Default:     f.DefValue,
				Description: f.Usage,
			}
			// 检查是否必需
			if annotations := c.Flags().Lookup(f.Name).Annotations; annotations != nil {
				if _, ok := annotations[cobra.BashCompOneRequiredFlag]; ok {
					flagInfo.Required = true
				}
			}
			info.Flags = append(info.Flags, flagInfo)
		})

		// 收集持久标志（继承自父命令）
		c.InheritedFlags().VisitAll(func(f *pflag.Flag) {
			if f.Hidden {
				return
			}
			info.Flags = append(info.Flags, FlagInfo{
				Name:        f.Name,
				Shorthand:   f.Shorthand,
				Type:        f.Value.Type(),
				Default:     f.DefValue,
				Description: f.Usage + " (global)",
			})
		})

		// 递归收集子命令
		if c.HasSubCommands() {
			info.SubCommands = collectCommands(c)
		}

		commands = append(commands, info)
	}

	return commands
}

func init() {
	CommandsCmd.GroupID = "system"
}
