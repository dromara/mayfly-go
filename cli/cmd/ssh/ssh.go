package ssh

import (
	"fmt"
	"mayfly-go/cli/client"
	"mayfly-go/cli/cmd/shared"
	"mayfly-go/cli/i18n"
	"mayfly-go/cli/terminal"

	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

// Cmd SSH 连接管理父命令
var Cmd = &cobra.Command{
	Use:   "ssh",
	Short: i18n.T(i18n.MsgSshShort),
	Long:  i18n.T(i18n.MsgSshLong),
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: i18n.T(i18n.MsgSshListShort),
	Long:  i18n.T(i18n.MsgSshListLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiClient := shared.NewClient(cmd)
		result, err := apiClient.ListMachines()
		if err := shared.HandleError(err, shared.GetServer(cmd)); err != nil {
			return shared.Fail(i18n.MsgSshListFailed, err)
		}

		if result == nil || result.Total == 0 {
			if shared.JsonOutput {
				shared.PrintJSONSuccess(map[string]interface{}{"total": 0, "list": []interface{}{}})
				return nil
			}
			terminal.PrintWarning(i18n.T(i18n.MsgSshNoResource))
			return nil
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		var rows []terminal.TableRow
		for i, machine := range result.List {
			name := machine["name"]
			host := machine["host"]
			port := machine["port"]
			status := machine["status"]

			address := shared.FormatAddress(host, port)
			isOnline := status == float64(1) || status == 1
			statusText := terminal.StatusText(isOnline)

			rows = append(rows, terminal.TableRow{
				Values: []interface{}{i + 1, name, address, statusText},
			})
		}

		terminal.PrintTable(terminal.TableConfig{
			Headers: []string{i18n.T(i18n.MsgIndex), i18n.T(i18n.MsgName), i18n.T(i18n.MsgAddress), i18n.T(i18n.MsgStatus)},
			Widths:  []int{8, 26, 32, 12},
			Tip:     i18n.T(i18n.MsgSshTableTip),
		}, rows)

		return interactiveMachineMenu(cmd, result.List)
	},
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: i18n.T(i18n.MsgSshStatsShort),
	Long:  i18n.T(i18n.MsgSshStatsLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineId, _ := cmd.Flags().GetUint64("id")
		ids, _ := shared.GetIdsFromFlags(cmd)

		// 批量模式：--ids 1,2,3
		if len(ids) > 0 {
			apiClient := shared.NewClient(cmd)
			results := make([]map[string]interface{}, 0, len(ids))
			for _, id := range ids {
				stats, err := apiClient.GetMachineStats(id)
				if err != nil {
					results = append(results, map[string]interface{}{
						"machine_id": id,
						"error":      err.Error(),
					})
					continue
				}
				stats["machine_id"] = id
				results = append(results, stats)
			}
			if shared.JsonOutput {
				shared.PrintJSONSuccess(results)
			} else {
				for _, r := range results {
					if errMsg, ok := r["error"]; ok {
						fmt.Printf("Machine %v: Error - %v\n", r["machine_id"], errMsg)
					} else {
						fmt.Printf("\n%sMachine %v%s\n", terminal.Cyan, r["machine_id"], terminal.Reset)
						renderMachineStats(r)
					}
				}
			}
			return nil
		}

		// 单个模式：--id 1
		if machineId == 0 {
			return shared.Fail(i18n.MsgSshStatsRequired, nil)
		}

		apiClient := shared.NewClient(cmd)
		stats, err := apiClient.GetMachineStats(machineId)
		if err != nil {
			return shared.Fail(i18n.MsgSshStatsFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(stats)
			return nil
		}

		fmt.Printf("\n%s%s%s\n", terminal.Cyan, i18n.T(i18n.MsgSshStatsTitle, "id", machineId), terminal.Reset)
		fmt.Printf("%s─────────────────────────────────────────────────────────────%s\n", terminal.Cyan, terminal.Reset)
		renderMachineStats(stats)
		fmt.Println()
		return nil
	},
}

var processCmd = &cobra.Command{
	Use:   "process",
	Short: i18n.T(i18n.MsgSshProcessShort),
	Long:  i18n.T(i18n.MsgSshProcessLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineId, _ := cmd.Flags().GetUint64("id")
		apiClient := shared.NewClient(cmd)
		processes, err := apiClient.GetMachineProcesses(machineId)
		if err != nil {
			return shared.Fail(i18n.MsgSshProcessFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(processes)
			return nil
		}

		fmt.Printf("\n%s%s%s\n", terminal.Cyan, i18n.T(i18n.MsgSshProcessTitle, "id", machineId), terminal.Reset)
		fmt.Printf("%s─────────────────────────────────────────────────────────────%s\n", terminal.Cyan, terminal.Reset)
		renderProcessList(processes)
		fmt.Println()
		return nil
	},
}

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: i18n.T(i18n.MsgSshUsersShort),
	Long:  i18n.T(i18n.MsgSshUsersLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineId, _ := cmd.Flags().GetUint64("id")
		apiClient := shared.NewClient(cmd)
		users, err := apiClient.GetMachineUsers(machineId)
		if err != nil {
			return shared.Fail(i18n.MsgSshUsersFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(users)
			return nil
		}

		if len(users) == 0 {
			fmt.Printf("%s%s%s\n", terminal.Yellow, i18n.T(i18n.MsgSshUsersNoData), terminal.Reset)
			return nil
		}

		fmt.Printf("\n%s%s%s\n", terminal.Cyan, i18n.T(i18n.MsgSshUsersTitle, "id", machineId), terminal.Reset)
		renderUserList(users)
		return nil
	},
}

var execCmd = &cobra.Command{
	Use:     "exec",
	Aliases: []string{"run"},
	Short:   i18n.T(i18n.MsgSshExecShort),
	Long:    i18n.T(i18n.MsgSshExecLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineId, _ := cmd.Flags().GetUint64("id")
		command, _ := cmd.Flags().GetString("cmd")
		authCert, _ := cmd.Flags().GetString("auth-cert")

		apiClient := shared.NewClient(cmd)

		authCert, err := resolveAuthCert(apiClient, machineId, authCert)
		if err != nil {
			return shared.Fail(i18n.MsgSshExecNotFound, err)
		}

		res, err := apiClient.RunMachineCmd(machineId, authCert, command)
		if err != nil {
			return shared.Fail(i18n.MsgSshExecFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(res)
			return nil
		}

		if output, ok := res["output"].(string); ok {
			fmt.Printf("\n%s%s%s\n", terminal.Cyan, i18n.T(i18n.MsgSshExecTitle, "id", machineId), terminal.Reset)
			fmt.Printf("%s─────────────────────────────────────────────────────────────%s\n", terminal.Cyan, terminal.Reset)
			fmt.Print(output)
		}
		if errMsg, ok := res["error"].(string); ok && errMsg != "" {
			fmt.Printf("\n%s%s%s\n", terminal.Yellow, i18n.T(i18n.MsgSshExecErrLabel, "err", errMsg), terminal.Reset)
		}
		fmt.Println()
		return nil
	},
}

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: i18n.T(i18n.MsgSshConnectShort),
	Long:  i18n.T(i18n.MsgSshConnectLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineId, _ := cmd.Flags().GetUint64("id")
		authCert, _ := cmd.Flags().GetString("auth-cert")

		server := shared.GetServer(cmd)
		token := shared.GetToken(cmd)
		apiClient := shared.NewClient(cmd)

		authCert, err := resolveAuthCert(apiClient, machineId, authCert)
		if err != nil {
			return shared.Fail(i18n.MsgSshConnectNotFound, err)
		}

		if !shared.JsonOutput {
			fmt.Printf("\n%s%s%s\n", terminal.Green, i18n.T(i18n.MsgSshConnecting, "id", machineId), terminal.Reset)
		}

		session := client.NewSSHSession(server, token, authCert, machineId)
		if err := session.Start(); err != nil {
			return shared.Fail(i18n.MsgSshConnectFailed, err)
		}

		if !shared.JsonOutput {
			fmt.Printf("\n%s%s%s\n", terminal.Green, i18n.T(i18n.MsgSshConnectClosed), terminal.Reset)
		}
		return nil
	},
}

func init() {
	Cmd.GroupID = "resource"
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(statsCmd)
	Cmd.AddCommand(processCmd)
	Cmd.AddCommand(usersCmd)
	Cmd.AddCommand(execCmd)
	Cmd.AddCommand(connectCmd)

	// ssh stats/process/users 标志
	statsCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgSshFlagStatsId))
	statsCmd.Flags().String("ids", "", i18n.T(i18n.MsgFlagIds))
	processCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgSshFlagProcessId))
	processCmd.MarkFlagRequired("id")
	usersCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgSshFlagUsersId))
	usersCmd.MarkFlagRequired("id")
	// ssh exec 标志
	execCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgSshFlagExecId))
	execCmd.Flags().String("cmd", "", i18n.T(i18n.MsgSshFlagExecCmd))
	execCmd.Flags().String("auth-cert", "", i18n.T(i18n.MsgSshFlagExecCert))
	execCmd.MarkFlagRequired("id")
	execCmd.MarkFlagRequired("cmd")
	// ssh connect 标志
	connectCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgSshFlagConnectId))
	connectCmd.Flags().String("auth-cert", "", i18n.T(i18n.MsgSshFlagConnectCert))
	connectCmd.MarkFlagRequired("id")

	// 动态补全
	shared.RegisterIdCompletion(statsCmd, "machine")
	shared.RegisterIdCompletion(processCmd, "machine")
	shared.RegisterIdCompletion(usersCmd, "machine")
	shared.RegisterIdCompletion(execCmd, "machine")
	shared.RegisterIdCompletion(connectCmd, "machine")
}

// resolveAuthCert 解析机器的授权凭证名
func resolveAuthCert(apiClient *client.ApiClient, machineId uint64, authCert string) (string, error) {
	if authCert != "" {
		return authCert, nil
	}

	result, err := apiClient.ListMachines()
	if err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T(i18n.MsgSshListFailed), err)
	}
	for _, m := range result.List {
		if cast.ToUint64(m["id"]) == machineId {
			if cert := client.GetAuthCertName(m); cert != "" {
				return cert, nil
			}
			break
		}
	}
	return "", fmt.Errorf("id=%d", machineId)
}
