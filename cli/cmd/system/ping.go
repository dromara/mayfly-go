package system

import (
	"fmt"
	"mayfly-go/cli/cmd/shared"
	"mayfly-go/cli/i18n"
	"mayfly-go/cli/terminal"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// PingCmd 服务器连通性检查
var PingCmd = &cobra.Command{
	Use:   "ping",
	Short: i18n.T(i18n.MsgPingShort),
	Long:  i18n.T(i18n.MsgPingLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		server := shared.GetServer(cmd)
		token := shared.GetToken(cmd)

		start := time.Now()
		apiClient := shared.NewClient(cmd)

		stats, err := apiClient.GetAssetStats()
		latency := time.Since(start)

		if err != nil {
			if shared.JsonOutput {
				shared.PrintJSON(map[string]interface{}{
					"success": false,
					"server":  server,
					"error":   err.Error(),
					"code":    classifyErrorCode(err),
				})
				return shared.ErrSilent
			}
			terminal.PrintError(i18n.T(i18n.MsgPingFailed, "err", err.Error()))
			return shared.ErrSilent
		}

		authOk := token != ""
		authText := i18n.T(i18n.MsgPingAuthOk)
		if !authOk {
			authText = i18n.T(i18n.MsgPingAuthAnon)
		}

		result := map[string]interface{}{
			"success": true,
			"server":  server,
			"auth":    authText,
			"latency": fmt.Sprintf("%dms", latency.Milliseconds()),
			"assets":  stats,
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		fmt.Printf("\n%s%s%s\n", terminal.Bold, "  "+i18n.T(i18n.MsgPingTitle), terminal.Reset)
		fmt.Printf("  %s─────────────────────────────────────────%s\n", terminal.Cyan, terminal.Reset)
		fmt.Printf("  %-12s %s%s%s\n", i18n.T(i18n.MsgPingServer)+":", terminal.Green, server, terminal.Reset)
		fmt.Printf("  %-12s %s%s%s\n", i18n.T(i18n.MsgPingAuth)+":", terminal.Green, authText, terminal.Reset)
		fmt.Printf("  %-12s %dms\n", i18n.T(i18n.MsgPingLatency)+":", latency.Milliseconds())
		fmt.Printf("  %s─────────────────────────────────────────%s\n", terminal.Cyan, terminal.Reset)
		fmt.Printf("  %-12s %d\n", "Databases:", stats["dbs"])
		fmt.Printf("  %-12s %d (%d online)\n", "Machines:", stats["machines"], stats["machines_online"])
		fmt.Printf("  %-12s %d\n", "Redis:", stats["redis"])
		fmt.Println()

		return nil
	},
}

func classifyErrorCode(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case containsAny(msg, "connection refused", "no such host", "timeout", "dial tcp"):
		return "CONNECTION_FAILED"
	case containsAny(msg, "code=501", "code=502", "401", "token"):
		return "AUTH_EXPIRED"
	case containsAny(msg, "403", "permission"):
		return "PERMISSION_DENIED"
	case containsAny(msg, "404", "not found"):
		return "NOT_FOUND"
	default:
		return "UNKNOWN"
	}
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func init() {
	PingCmd.GroupID = "system"
}
