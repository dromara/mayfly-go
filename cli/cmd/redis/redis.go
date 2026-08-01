package redis

import (
	"fmt"
	"mayfly-go/cli/cmd/shared"
	"mayfly-go/cli/i18n"
	"mayfly-go/cli/terminal"
	"strings"

	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

// Cmd Redis 操作父命令
var Cmd = &cobra.Command{
	Use:   "redis",
	Short: i18n.T(i18n.MsgRedisShort),
	Long:  i18n.T(i18n.MsgRedisLong),
}

// infoField Redis INFO 输出中需要展示的字段
type infoField struct {
	prefix string
	label  string
	suffix string
}

var (
	redisServerFields = []infoField{
		{"redis_version:", i18n.MsgRedisInfoVersion, ""},
		{"os:", i18n.MsgRedisInfoOs, ""},
		{"tcp_port:", i18n.MsgRedisInfoPort, ""},
		{"process_id:", i18n.MsgRedisInfoPid, ""},
		{"uptime_in_seconds:", i18n.MsgRedisInfoUptime, " seconds"},
	}
	redisMemoryFields = []infoField{
		{"used_memory_human:", i18n.MsgRedisMemUsed, ""},
		{"maxmemory_human:", i18n.MsgRedisMemMax, ""},
		{"used_memory_peak_human:", i18n.MsgRedisMemPeak, ""},
		{"used_memory_rss_human:", i18n.MsgRedisMemRss, ""},
	}
)

func redisInfoString(result interface{}) string {
	switch v := result.(type) {
	case string:
		return v
	case map[string]interface{}:
		if data, exists := v["result"]; exists {
			return cast.ToString(data)
		}
		return ""
	default:
		return cast.ToString(result)
	}
}

func renderRedisInfoFields(infoStr string, fields []infoField) {
	for _, line := range strings.Split(infoStr, "\r\n") {
		for _, f := range fields {
			if strings.HasPrefix(line, f.prefix) {
				fmt.Printf("  %s%s%s %s%s\n", terminal.Bold, i18n.T(f.label), terminal.Reset, strings.TrimPrefix(line, f.prefix), f.suffix)
				break
			}
		}
	}
}

func printRedisResult(result interface{}) {
	if resultMap, ok := result.(map[string]interface{}); ok {
		if data, exists := resultMap["result"]; exists {
			fmt.Printf("%+v\n", data)
			return
		}
		fmt.Printf("%+v\n", resultMap)
		return
	}
	fmt.Printf("%+v\n", result)
}

func parseRedisInfo(result interface{}) map[string]string {
	infoMap := make(map[string]string)
	for _, line := range strings.Split(redisInfoString(result), "\r\n") {
		if idx := strings.Index(line, ":"); idx > 0 {
			infoMap[line[:idx]] = line[idx+1:]
		}
	}
	return infoMap
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: i18n.T(i18n.MsgRedisListShort),
	Long:  i18n.T(i18n.MsgRedisListLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiClient := shared.NewClient(cmd)
		result, err := apiClient.ListRedis()
		if err := shared.HandleError(err, shared.GetServer(cmd)); err != nil {
			return shared.Fail(i18n.MsgRedisListFailed, err)
		}

		if result == nil || result.Total == 0 {
			if shared.JsonOutput {
				shared.PrintJSONSuccess(map[string]interface{}{"total": 0, "list": []interface{}{}})
				return nil
			}
			terminal.PrintWarning(i18n.T(i18n.MsgRedisNoResource))
			return nil
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		var rows []terminal.TableRow
		for i, redis := range result.List {
			name := redis["name"]
			host := redis["host"]
			mode := redis["mode"]

			modeColor := terminal.RedisModeColor(cast.ToString(mode))

			rows = append(rows, terminal.TableRow{
				Values: []interface{}{i + 1, name, host, mode},
				Color:  modeColor,
			})
		}

		terminal.PrintTable(terminal.TableConfig{
			Headers: []string{i18n.T(i18n.MsgIndex), i18n.T(i18n.MsgName), i18n.T(i18n.MsgAddress), i18n.T(i18n.MsgMode)},
			Widths:  []int{8, 26, 32, 16},
			Tip:     i18n.T(i18n.MsgRedisTableTip),
		}, rows)

		return interactiveRedisMenu(cmd, result.List)
	},
}

var execCmd = &cobra.Command{
	Use:     "exec",
	Aliases: []string{"run"},
	Short:   i18n.T(i18n.MsgRedisExecShort),
	Long:    i18n.T(i18n.MsgRedisExecLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		redisId, _ := cmd.Flags().GetUint64("id")
		dbNum, _ := cmd.Flags().GetInt("db")
		command, _ := cmd.Flags().GetString("cmd")

		apiClient := shared.NewClient(cmd)
		result, err := apiClient.ExecRedisCmd(redisId, dbNum, command)
		if err != nil {
			return shared.Fail(i18n.MsgRedisExecFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		printRedisResult(result)
		return nil
	},
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: i18n.T(i18n.MsgRedisInfoShort),
	Long:  i18n.T(i18n.MsgRedisInfoLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		redisId, _ := cmd.Flags().GetUint64("id")
		apiClient := shared.NewClient(cmd)
		result, err := apiClient.ExecRedisCmd(redisId, 0, "INFO server")
		if err != nil {
			return shared.Fail(i18n.MsgRedisInfoFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(parseRedisInfo(result))
			return nil
		}

		fmt.Printf("\n%s%s%s\n", terminal.Cyan, i18n.T(i18n.MsgRedisInfoTitle, "id", redisId), terminal.Reset)
		fmt.Printf("%s─────────────────────────────────────────────────────────────%s\n", terminal.Cyan, terminal.Reset)
		renderRedisInfoFields(redisInfoString(result), redisServerFields)
		fmt.Println()
		return nil
	},
}

var memCmd = &cobra.Command{
	Use:   "mem",
	Short: i18n.T(i18n.MsgRedisMemShort),
	Long:  i18n.T(i18n.MsgRedisMemLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		redisId, _ := cmd.Flags().GetUint64("id")
		apiClient := shared.NewClient(cmd)
		result, err := apiClient.ExecRedisCmd(redisId, 0, "INFO memory")
		if err != nil {
			return shared.Fail(i18n.MsgRedisMemFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(parseRedisInfo(result))
			return nil
		}

		fmt.Printf("\n%s%s%s\n", terminal.Cyan, i18n.T(i18n.MsgRedisMemTitle, "id", redisId), terminal.Reset)
		fmt.Printf("%s─────────────────────────────────────────────────────────────%s\n", terminal.Cyan, terminal.Reset)
		renderRedisInfoFields(redisInfoString(result), redisMemoryFields)
		fmt.Println()
		return nil
	},
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: i18n.T(i18n.MsgRedisScanShort),
	Long:  i18n.T(i18n.MsgRedisScanLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		redisId, _ := cmd.Flags().GetUint64("id")
		db, _ := cmd.Flags().GetInt("db")
		match, _ := cmd.Flags().GetString("match")
		count, _ := cmd.Flags().GetInt64("count")

		if match == "" {
			match = "*"
		}
		if count <= 0 {
			count = 100
		}

		apiClient := shared.NewClient(cmd)
		result, err := apiClient.ScanRedisKeys(redisId, db, match, count, nil)
		if err != nil {
			return shared.Fail(i18n.MsgRedisScanFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		shared.PrintResult(result)
		return nil
	},
}

var keyInfoCmd = &cobra.Command{
	Use:   "key-info",
	Short: i18n.T(i18n.MsgRedisKeyInfoShort),
	Long:  i18n.T(i18n.MsgRedisKeyInfoLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		redisId, _ := cmd.Flags().GetUint64("id")
		db, _ := cmd.Flags().GetInt("db")
		key, _ := cmd.Flags().GetString("key")

		apiClient := shared.NewClient(cmd)
		result, err := apiClient.GetKeyInfo(redisId, db, key)
		if err != nil {
			return shared.Fail(i18n.MsgRedisExecFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		shared.PrintResult(result)
		return nil
	},
}

var keyTtlCmd = &cobra.Command{
	Use:   "key-ttl",
	Short: i18n.T(i18n.MsgRedisKeyTtlShort),
	RunE: func(cmd *cobra.Command, args []string) error {
		redisId, _ := cmd.Flags().GetUint64("id")
		db, _ := cmd.Flags().GetInt("db")
		key, _ := cmd.Flags().GetString("key")

		apiClient := shared.NewClient(cmd)
		result, err := apiClient.GetKeyTTL(redisId, db, key)
		if err != nil {
			return shared.Fail(i18n.MsgRedisExecFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		fmt.Println(i18n.T(i18n.MsgRedisKeyTtlResult, "ttl", result))
		return nil
	},
}

var keyMemCmd = &cobra.Command{
	Use:   "key-mem",
	Short: i18n.T(i18n.MsgRedisKeyMemShort),
	RunE: func(cmd *cobra.Command, args []string) error {
		redisId, _ := cmd.Flags().GetUint64("id")
		db, _ := cmd.Flags().GetInt("db")
		key, _ := cmd.Flags().GetString("key")

		apiClient := shared.NewClient(cmd)
		result, err := apiClient.GetKeyMemoryUsage(redisId, db, key)
		if err != nil {
			return shared.Fail(i18n.MsgRedisExecFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		fmt.Println(i18n.T(i18n.MsgRedisKeyMemResult, "mem", result))
		return nil
	},
}

func init() {
	Cmd.GroupID = "resource"
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(execCmd)
	Cmd.AddCommand(infoCmd)
	Cmd.AddCommand(memCmd)
	Cmd.AddCommand(scanCmd)
	Cmd.AddCommand(keyInfoCmd)
	Cmd.AddCommand(keyTtlCmd)
	Cmd.AddCommand(keyMemCmd)

	// redis exec 标志
	execCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgRedisFlagExecId))
	execCmd.Flags().Int("db", 0, i18n.T(i18n.MsgRedisFlagExecDb))
	execCmd.Flags().String("cmd", "", i18n.T(i18n.MsgRedisFlagExecCmd))
	execCmd.MarkFlagRequired("id")
	execCmd.MarkFlagRequired("cmd")

	// redis info/mem 标志
	infoCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgRedisFlagInfoId))
	infoCmd.MarkFlagRequired("id")
	memCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgRedisFlagMemId))
	memCmd.MarkFlagRequired("id")

	// redis scan 标志
	scanCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgRedisFlagScanId))
	scanCmd.Flags().Int("db", 0, i18n.T(i18n.MsgRedisFlagScanDb))
	scanCmd.Flags().String("match", "*", i18n.T(i18n.MsgRedisFlagScanMatch))
	scanCmd.Flags().Int64("count", 100, i18n.T(i18n.MsgRedisFlagScanCount))
	scanCmd.MarkFlagRequired("id")

	// redis key-info/key-ttl/key-mem 标志
	for _, c := range []*cobra.Command{keyInfoCmd, keyTtlCmd, keyMemCmd} {
		c.Flags().Uint64("id", 0, i18n.T(i18n.MsgRedisFlagKeyId))
		c.Flags().Int("db", 0, i18n.T(i18n.MsgRedisFlagKeyDb))
		c.Flags().String("key", "", i18n.T(i18n.MsgRedisFlagKeyName))
		c.MarkFlagRequired("id")
		c.MarkFlagRequired("key")
	}

	// 动态补全
	shared.RegisterIdCompletion(execCmd, "redis")
	shared.RegisterIdCompletion(infoCmd, "redis")
	shared.RegisterIdCompletion(memCmd, "redis")
	shared.RegisterIdCompletion(scanCmd, "redis")
	shared.RegisterIdCompletion(keyInfoCmd, "redis")
	shared.RegisterIdCompletion(keyTtlCmd, "redis")
	shared.RegisterIdCompletion(keyMemCmd, "redis")
}
