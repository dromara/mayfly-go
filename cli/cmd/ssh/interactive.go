package ssh

import (
	"bufio"
	"fmt"
	"mayfly-go/cli/client"
	"mayfly-go/cli/cmd/shared"
	"mayfly-go/cli/i18n"
	"mayfly-go/cli/terminal"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

// 进程列表展示行数上限
const (
	maxProcessLines = 25
	maxProcessRows  = 20
)

// renderMachineStats 渲染机器监控指标
func renderMachineStats(stats map[string]interface{}) {
	if stats == nil {
		return
	}
	if cpu, ok := stats["cpuUsage"].(float64); ok {
		fmt.Printf("  %s%s%s %.1f%%\n", terminal.Bold, i18n.T(i18n.MsgSshSysInfoCpu), terminal.Reset, cpu)
	}
	if mem, ok := stats["memUsage"].(float64); ok {
		fmt.Printf("  %s%s%s %.1f%%\n", terminal.Bold, i18n.T(i18n.MsgSshSysInfoMem), terminal.Reset, mem)
	}
	if disk, ok := stats["diskUsage"].(float64); ok {
		fmt.Printf("  %s%s%s %.1f%%\n", terminal.Bold, i18n.T(i18n.MsgSshSysInfoDisk), terminal.Reset, disk)
	}
	if load, ok := stats["load1"].(float64); ok {
		fmt.Printf("  %s%s%s %.2f\n", terminal.Bold, i18n.T(i18n.MsgSshSysInfoLoad), terminal.Reset, load)
	}
}

// renderProcessList 渲染进程列表
func renderProcessList(processes interface{}) {
	switch v := processes.(type) {
	case string:
		lines := strings.Split(v, "\n")
		showCount := maxProcessLines
		if len(lines) < showCount {
			showCount = len(lines)
		}
		for i := 0; i < showCount; i++ {
			fmt.Printf("  %s\n", lines[i])
		}
		if len(lines) > maxProcessLines {
			fmt.Printf("\n%s\n", i18n.T(i18n.MsgSshProcessMore, "count", len(lines)))
		}
	case []interface{}:
		if len(v) == 0 {
			fmt.Printf("%s%s%s\n", terminal.Yellow, i18n.T(i18n.MsgSshProcessNoData2), terminal.Reset)
			return
		}
		showCount := maxProcessRows
		if len(v) < showCount {
			showCount = len(v)
		}
		headers := strings.Split(i18n.T(i18n.MsgSshProcessHeader), ",")
		if len(headers) >= 5 {
			fmt.Printf("  %-10s %-8s %-6s %-6s %s\n", headers[0], headers[1], headers[2], headers[3], headers[4])
		}
		fmt.Printf("  %s\n", strings.Repeat("─", 70))
		for i := 0; i < showCount; i++ {
			if procMap, ok := v[i].(map[string]interface{}); ok {
				cmdStr := cast.ToString(procMap["command"])
				if len(cmdStr) > 40 {
					cmdStr = cmdStr[:40] + "..."
				}
				fmt.Printf("  %-10v %-8v %-6v %-6v %s\n", procMap["pid"], procMap["user"], procMap["cpu"], procMap["mem"], cmdStr)
			}
		}
		if len(v) > maxProcessRows {
			fmt.Printf("\n%s\n", i18n.T(i18n.MsgSshProcessTotal, "count", len(v)))
		}
	default:
		fmt.Printf("  %v\n", processes)
	}
}

// renderUserList 渲染机器用户列表
func renderUserList(users []interface{}) {
	headers := strings.Split(i18n.T(i18n.MsgSshUserListHeader), ",")
	if len(headers) >= 3 {
		fmt.Printf("  %-30s %-20s %s\n", headers[0], headers[1], headers[2])
	}
	fmt.Printf("  %s\n", strings.Repeat("─", 70))

	for _, user := range users {
		if userMap, ok := user.(map[string]interface{}); ok {
			fmt.Printf("  %-30v %-20v %v\n", userMap["username"], userMap["groups"], userMap["shell"])
		}
	}

	fmt.Printf("\n  %s\n", i18n.T(i18n.MsgSshUserListCount, "count", len(users)))
	fmt.Println()
}

// interactiveMachineMenu 交互式机器选择菜单
func interactiveMachineMenu(cmd *cobra.Command, machines []map[string]interface{}) error {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("\n%s%s%s", terminal.Bold, i18n.T(i18n.MsgSshMenuSelect), terminal.Reset)

		input, ok := shared.ReadLine(reader)
		if !ok {
			return nil
		}

		if input == "q" || input == "Q" {
			fmt.Println(i18n.T(i18n.MsgQuit))
			return nil
		}

		idx, err := strconv.Atoi(input)
		if err != nil || idx < 1 || idx > len(machines) {
			fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgSshMenuInvalid, "max", len(machines)), terminal.Reset)
			continue
		}

		machine := machines[idx-1]
		name := machine["name"]
		status := machine["status"]

		isOnline := status == float64(1) || status == 1

		if !isOnline {
			fmt.Printf("%s%s%s\n", terminal.Yellow, i18n.T(i18n.MsgSshOffline, "name", name), terminal.Reset)
			continue
		}

		fmt.Printf("\n%s%s%s\n", terminal.Green, i18n.T(i18n.MsgSshSelected, "name", name), terminal.Reset)
		showMachineOperationMenu(cmd, machine)
	}
}

func showMachineOperationMenu(cmd *cobra.Command, machine map[string]interface{}) {
	name := machine["name"]
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("\n%s╔════════════════════════════════════════╗%s\n", terminal.Cyan, terminal.Reset)
		fmt.Printf("%s║  %s  ║%s\n", terminal.Cyan, i18n.T(i18n.MsgSshMenuTitle, "name", name), terminal.Reset)
		fmt.Printf("%s╠════════════════════════════════════════╣%s\n", terminal.Cyan, terminal.Reset)
		fmt.Printf("%s║  1. %-34s ║%s\n", terminal.Cyan, i18n.T(i18n.MsgSshMenuTerminal), terminal.Reset)
		fmt.Printf("%s║  2. %-34s ║%s\n", terminal.Cyan, i18n.T(i18n.MsgSshMenuSysInfo), terminal.Reset)
		fmt.Printf("%s║  3. %-34s ║%s\n", terminal.Cyan, i18n.T(i18n.MsgSshMenuProcess), terminal.Reset)
		fmt.Printf("%s║  4. %-34s ║%s\n", terminal.Cyan, i18n.T(i18n.MsgSshMenuUsers), terminal.Reset)
		fmt.Printf("%s║  5. %-34s ║%s\n", terminal.Cyan, i18n.T(i18n.MsgSshMenuBack), terminal.Reset)
		fmt.Printf("%s║  6. %-34s ║%s\n", terminal.Cyan, i18n.T(i18n.MsgSshMenuExit), terminal.Reset)
		fmt.Printf("%s╚════════════════════════════════════════╝%s\n", terminal.Cyan, terminal.Reset)

		fmt.Printf("\n%s%s%s", terminal.Bold, i18n.T(i18n.MsgSelectOp, "max", 6), terminal.Reset)

		input, ok := shared.ReadLine(reader)
		if !ok {
			return
		}

		switch input {
		case "1":
			startSSHTerminal(cmd, machine)
		case "2":
			showSystemInfo(cmd, machine)
		case "3":
			showProcessList(cmd, machine)
		case "4":
			showUserList(cmd, machine)
		case "5":
			return
		case "6":
			fmt.Println(i18n.T(i18n.MsgQuit))
			os.Exit(0)
		default:
			fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgInvalidOp, "max", 6), terminal.Reset)
		}
	}
}

func showSystemInfo(cmd *cobra.Command, machine map[string]interface{}) {
	machineId := cast.ToUint64(machine["id"])
	apiClient := shared.NewClient(cmd)

	fmt.Printf("\n%s╔════════════════════════════════════════╗%s\n", terminal.Cyan, terminal.Reset)
	fmt.Printf("%s║  %s  ║%s\n", terminal.Cyan, i18n.T(i18n.MsgSshSysInfoTitle, "name", machine["name"]), terminal.Reset)
	fmt.Printf("%s╚════════════════════════════════════════╝%s\n\n", terminal.Cyan, terminal.Reset)

	stats, err := apiClient.GetMachineStats(machineId)
	if err != nil {
		fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgSshSysInfoGetFail, "err", err), terminal.Reset)
		return
	}

	if stats != nil {
		renderMachineStats(stats)
	}

	fmt.Println()
}

func showProcessList(cmd *cobra.Command, machine map[string]interface{}) {
	machineId := cast.ToUint64(machine["id"])
	apiClient := shared.NewClient(cmd)

	fmt.Printf("\n%s╔════════════════════════════════════════╗%s\n", terminal.Cyan, terminal.Reset)
	fmt.Printf("%s║  %s  ║%s\n", terminal.Cyan, i18n.T(i18n.MsgSshProcessTitle, "id", machineId), terminal.Reset)
	fmt.Printf("%s╚════════════════════════════════════════╝%s\n\n", terminal.Cyan, terminal.Reset)

	processes, err := apiClient.GetMachineProcesses(machineId)
	if err != nil {
		fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgSshProcessGetFail, "err", err), terminal.Reset)
		return
	}

	if processes == nil {
		fmt.Printf("%s%s%s\n", terminal.Yellow, i18n.T(i18n.MsgSshProcessNoData2), terminal.Reset)
		return
	}

	fmt.Printf("  %s\n", strings.Repeat("─", 70))
	renderProcessList(processes)
	fmt.Println()
}

func showUserList(cmd *cobra.Command, machine map[string]interface{}) {
	machineId := cast.ToUint64(machine["id"])
	apiClient := shared.NewClient(cmd)

	fmt.Printf("\n%s╔════════════════════════════════════════╗%s\n", terminal.Cyan, terminal.Reset)
	fmt.Printf("%s║  %s  ║%s\n", terminal.Cyan, i18n.T(i18n.MsgSshUserListTitle, "name", machine["name"]), terminal.Reset)
	fmt.Printf("%s╚════════════════════════════════════════╝%s\n\n", terminal.Cyan, terminal.Reset)

	users, err := apiClient.GetMachineUsers(machineId)
	if err != nil {
		fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgSshUserListGetFail, "err", err), terminal.Reset)
		return
	}

	if users == nil || len(users) == 0 {
		fmt.Printf("%s%s%s\n", terminal.Yellow, i18n.T(i18n.MsgSshUserListNoData), terminal.Reset)
		return
	}

	renderUserList(users)
}

func startSSHTerminal(cmd *cobra.Command, machine map[string]interface{}) {
	server := shared.GetServer(cmd)
	token := shared.GetToken(cmd)
	machineId := cast.ToUint64(machine["id"])

	authCertName := client.GetAuthCertName(machine)
	if authCertName == "" {
		fmt.Printf("%s%s%s\n", terminal.Yellow, i18n.T(i18n.MsgSshNoCert), terminal.Reset)
		return
	}

	fmt.Printf("\n%s🚀 %s...%s\n", terminal.Green, i18n.T(i18n.MsgSshConnecting, "id", machineId), terminal.Reset)

	session := client.NewSSHSession(server, token, authCertName, machineId)

	if err := session.Start(); err != nil {
		fmt.Printf("\n%s%s\n", terminal.Red, i18n.T(i18n.MsgSshTerminalConnFail, "err", err))
		return
	}

	fmt.Printf("\n%s%s%s\n", terminal.Green, i18n.T(i18n.MsgSshTerminalClosed), terminal.Reset)
}
