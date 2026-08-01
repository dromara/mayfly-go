package cmd

import (
	"fmt"
	"mayfly-go/cli/cmd/shared"
	"mayfly-go/cli/i18n"
	"mayfly-go/cli/terminal"

	"github.com/spf13/cobra"
)

func showDashboard(cmd *cobra.Command) error {
	server := shared.GetServer(cmd)

	apiClient := shared.NewClient(cmd)
	isLoggedIn := apiClient.CheckAuth()

	if shared.JsonOutput {
		status := map[string]interface{}{
			"server":   server,
			"loggedIn": isLoggedIn,
		}

		if isLoggedIn {
			stats, err := apiClient.GetAssetStats()
			if err == nil {
				status["stats"] = stats
			}
		}

		shared.PrintJSONSuccess(status)
		return nil
	}

	if !isLoggedIn {
		terminal.PrintLoginPrompt(server)
		return nil
	}

	stats, err := apiClient.GetAssetStats()
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T(i18n.MsgAssetStatsFailed), err)
	}

	var items []terminal.DashboardItem

	if dbs, ok := stats["dbs"]; ok {
		items = append(items, terminal.DashboardItem{
			Name:  i18n.T("dashboard.db_name"),
			Icon:  "📊",
			Total: dbs,
			Color: terminal.Colorize(i18n.T(i18n.MsgDashboardDbDesc), terminal.Cyan),
		})
	}

	if machines, ok := stats["machines"]; ok {
		online := stats["machines_online"]
		items = append(items, terminal.DashboardItem{
			Name:   i18n.T("dashboard.machine_name"),
			Icon:   "🖥️",
			Total:  machines,
			Online: online,
			Color:  terminal.Colorize(fmt.Sprintf("%s", i18n.T(i18n.MsgDashboardOnlineFmt, "count", online)), terminal.Green),
		})
	}

	if redis, ok := stats["redis"]; ok {
		items = append(items, terminal.DashboardItem{
			Name:  "Redis",
			Icon:  "🔴",
			Total: redis,
			Color: terminal.Colorize(i18n.T(i18n.MsgDashboardRedisDesc), terminal.Purple),
		})
	}

	terminal.PrintDashboard(items, server, true)

	return nil
}
