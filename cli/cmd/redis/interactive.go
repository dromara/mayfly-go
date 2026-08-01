package redis

import (
	"bufio"
	"fmt"
	"mayfly-go/cli/cmd/shared"
	"mayfly-go/cli/i18n"
	"mayfly-go/cli/terminal"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

// interactiveRedisMenu 交互式 Redis 选择菜单
func interactiveRedisMenu(cmd *cobra.Command, redisList []map[string]interface{}) error {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("\n%s%s%s", terminal.Bold, i18n.T(i18n.MsgRedisMenuSelect), terminal.Reset)

		input, ok := shared.ReadLine(reader)
		if !ok {
			return nil
		}

		if input == "q" || input == "Q" {
			fmt.Println(i18n.T(i18n.MsgQuit))
			return nil
		}

		idx, err := strconv.Atoi(input)
		if err != nil || idx < 1 || idx > len(redisList) {
			fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgRedisMenuInvalid, "max", len(redisList)), terminal.Reset)
			continue
		}

		redis := redisList[idx-1]
		name := redis["name"]
		mode := redis["mode"]

		fmt.Printf("\n%s%s%s\n", terminal.Green, i18n.T(i18n.MsgRedisSelected, "name", name, "mode", mode), terminal.Reset)
		showRedisOperationMenu(cmd, redis, reader)
	}
}

func showRedisOperationMenu(cmd *cobra.Command, redis map[string]interface{}, reader *bufio.Reader) {
	name := redis["name"]

	for {
		fmt.Printf("\n%s╔════════════════════════════════════════╗%s\n", terminal.Cyan, terminal.Reset)
		fmt.Printf("%s║  %s  ║%s\n", terminal.Cyan, i18n.T(i18n.MsgRedisMenuTitle, "name", name), terminal.Reset)
		fmt.Printf("%s╠════════════════════════════════════════╣%s\n", terminal.Cyan, terminal.Reset)
		fmt.Printf("%s║  1. %-34s ║%s\n", terminal.Cyan, i18n.T(i18n.MsgRedisMenuExec), terminal.Reset)
		fmt.Printf("%s║  2. %-34s ║%s\n", terminal.Cyan, i18n.T(i18n.MsgRedisMenuInfo), terminal.Reset)
		fmt.Printf("%s║  3. %-34s ║%s\n", terminal.Cyan, i18n.T(i18n.MsgRedisMenuMem), terminal.Reset)
		fmt.Printf("%s║  4. %-34s ║%s\n", terminal.Cyan, i18n.T(i18n.MsgRedisMenuKeyspace), terminal.Reset)
		fmt.Printf("%s║  5. %-34s ║%s\n", terminal.Cyan, i18n.T(i18n.MsgRedisMenuBack), terminal.Reset)
		fmt.Printf("%s║  6. %-34s ║%s\n", terminal.Cyan, i18n.T(i18n.MsgRedisMenuExit), terminal.Reset)
		fmt.Printf("%s╚════════════════════════════════════════╝%s\n", terminal.Cyan, terminal.Reset)

		fmt.Printf("\n%s%s%s", terminal.Bold, i18n.T(i18n.MsgSelectOp, "max", 6), terminal.Reset)

		input, ok := shared.ReadLine(reader)
		if !ok {
			return
		}

		switch input {
		case "1":
			executeRedisCommand(cmd, redis, reader)
		case "2":
			showRedisInfo(cmd, redis)
		case "3":
			showRedisMemory(cmd, redis)
		case "4":
			showRedisKeyspace(cmd, redis)
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

func executeRedisCommand(cmd *cobra.Command, redis map[string]interface{}, reader *bufio.Reader) {
	redisId := cast.ToUint64(redis["id"])
	apiClient := shared.NewClient(cmd)

	if !shared.Quiet {
		fmt.Printf("\n%s%s%s\n", terminal.Green, i18n.T(i18n.MsgRedisExecPrompt), terminal.Reset)
		fmt.Printf("%s%s%s\n\n", terminal.Yellow, i18n.T(i18n.MsgRedisExecHint), terminal.Reset)
	}

	for {
		fmt.Printf("%s%s> %s", terminal.Yellow, redis["name"], terminal.Reset)
		input, ok := shared.ReadLine(reader)
		if !ok {
			return
		}

		if input == "back" {
			return
		}
		if input == "quit" {
			fmt.Println(i18n.T(i18n.MsgQuit))
			os.Exit(0)
		}
		if input == "" {
			continue
		}

		parts := strings.SplitN(input, " ", 2)
		if len(parts) < 2 {
			fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgRedisExecFmtErr), terminal.Reset)
			continue
		}

		dbNum, err := strconv.Atoi(parts[0])
		if err != nil {
			fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgRedisExecDbErr, "db", parts[0]), terminal.Reset)
			continue
		}

		command := parts[1]

		result, err := apiClient.ExecRedisCmd(redisId, dbNum, command)
		if err != nil {
			fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgRedisExecFail, "err", err.Error()), terminal.Reset)
			continue
		}

		printRedisResult(result)
		fmt.Println()
	}
}

func showRedisInfo(cmd *cobra.Command, redis map[string]interface{}) {
	redisId := cast.ToUint64(redis["id"])
	apiClient := shared.NewClient(cmd)

	fmt.Printf("\n%s╔════════════════════════════════════════╗%s\n", terminal.Cyan, terminal.Reset)
	fmt.Printf("%s║  📊  %s Server Info                ║%s\n", terminal.Cyan, redis["name"], terminal.Reset)
	fmt.Printf("%s╚════════════════════════════════════════╝%s\n\n", terminal.Cyan, terminal.Reset)

	result, err := apiClient.ExecRedisCmd(redisId, 0, "INFO server")
	if err != nil {
		fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgRedisInfoQueryFail, "err", err.Error()), terminal.Reset)
		return
	}

	renderRedisInfoFields(redisInfoString(result), redisServerFields)
	fmt.Println()
}

func showRedisMemory(cmd *cobra.Command, redis map[string]interface{}) {
	redisId := cast.ToUint64(redis["id"])
	apiClient := shared.NewClient(cmd)

	fmt.Printf("\n%s╔════════════════════════════════════════╗%s\n", terminal.Cyan, terminal.Reset)
	fmt.Printf("%s║  💾  %s Memory Usage              ║%s\n", terminal.Cyan, redis["name"], terminal.Reset)
	fmt.Printf("%s╚════════════════════════════════════════╝%s\n\n", terminal.Cyan, terminal.Reset)

	result, err := apiClient.ExecRedisCmd(redisId, 0, "INFO memory")
	if err != nil {
		fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgRedisMemQueryFail, "err", err.Error()), terminal.Reset)
		return
	}

	renderRedisInfoFields(redisInfoString(result), redisMemoryFields)
	fmt.Println()
}

func showRedisKeyspace(cmd *cobra.Command, redis map[string]interface{}) {
	redisId := cast.ToUint64(redis["id"])
	apiClient := shared.NewClient(cmd)

	fmt.Printf("\n%s╔════════════════════════════════════════╗%s\n", terminal.Cyan, terminal.Reset)
	fmt.Printf("%s║  🔑  %s Keyspace Stats              ║%s\n", terminal.Cyan, redis["name"], terminal.Reset)
	fmt.Printf("%s╚════════════════════════════════════════╝%s\n\n", terminal.Cyan, terminal.Reset)

	result, err := apiClient.ExecRedisCmd(redisId, 0, "INFO keyspace")
	if err != nil {
		fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgRedisKeyspaceQueryFail, "err", err.Error()), terminal.Reset)
		return
	}

	info := redisInfoString(result)
	if strings.TrimSpace(info) == "" {
		fmt.Printf("  %s%s%s\n", terminal.Yellow, i18n.T(i18n.MsgRedisKeyspaceNoData), terminal.Reset)
	} else {
		headers := strings.Split(i18n.T(i18n.MsgRedisKeyspaceHeader), ",")
		if len(headers) >= 3 {
			fmt.Printf("  %-12s %-10s %-10s\n", headers[0], headers[1], headers[2])
		}
		fmt.Printf("  %s\n", strings.Repeat("─", 32))

		for _, line := range strings.Split(info, "\r\n") {
			if strings.HasPrefix(line, "db") {
				fmt.Printf("  %s\n", line)
			}
		}
	}

	fmt.Println()
}
