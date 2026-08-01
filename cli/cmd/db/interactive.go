package db

import (
	"bufio"
	"errors"
	"fmt"
	"mayfly-go/cli/client"
	"mayfly-go/cli/cmd/shared"
	"mayfly-go/cli/i18n"
	"mayfly-go/cli/terminal"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

// interactiveDatabaseMenu 交互式数据库选择菜单
func interactiveDatabaseMenu(cmd *cobra.Command, databases []map[string]interface{}) error {
	reader := bufio.NewReader(os.Stdin)
	apiClient := shared.NewClient(cmd)

	for {
		fmt.Printf("\n%s%s%s", terminal.Bold, i18n.T(i18n.MsgDbMenuSelect), terminal.Reset)

		input, ok := shared.ReadLine(reader)
		if !ok {
			return nil
		}

		if input == "q" || input == "Q" {
			fmt.Println(i18n.T(i18n.MsgQuit))
			return nil
		}

		idx, err := strconv.Atoi(input)
		if err != nil || idx < 1 || idx > len(databases) {
			fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgInvalid, "max", len(databases)), terminal.Reset)
			continue
		}

		err = enterInstance(cmd, apiClient, databases[idx-1], reader)
		switch {
		case errors.Is(err, errQuit):
			fmt.Println(i18n.T(i18n.MsgQuit))
			return nil
		case errors.Is(err, errBack):
			continue
		case err != nil:
			fmt.Printf("%s%s%s\n", terminal.Red, err.Error(), terminal.Reset)
			fmt.Printf("%s%s%s\n\n", terminal.Yellow, i18n.T(i18n.MsgDbMenuRetry), terminal.Reset)
		}
	}
}

// enterInstance 进入数据库实例，选择具体数据库并执行 SQL
func enterInstance(cmd *cobra.Command, apiClient *client.ApiClient, dbInstance map[string]interface{}, reader *bufio.Reader) error {
	instanceName := cast.ToString(dbInstance["name"])
	authCertName := cast.ToString(dbInstance["authCertName"])

	if !shared.Quiet && !shared.JsonOutput {
		fmt.Printf("\n%s%s%s\n", terminal.Cyan, i18n.T(i18n.MsgDbSqlGetDbInfo, "name", instanceName), terminal.Reset)
	}
	dbList, err := apiClient.GetInstanceDatabases(authCertName)
	if err != nil {
		return shared.Fail(i18n.MsgDbSqlGetInfoFail, err)
	}

	if len(dbList) == 0 {
		return shared.Fail(i18n.MsgDbSqlNoDb, nil)
	}

	if shared.JsonOutput {
		shared.PrintJSONSuccess(map[string]interface{}{
			"instance":  instanceName,
			"databases": dbList,
		})
		return nil
	}

	fmt.Printf("\n%s%s%s\n", terminal.Cyan, i18n.T(i18n.MsgDbSqlDbList, "name", instanceName), terminal.Reset)
	for i, dbName := range dbList {
		fmt.Printf("  %s%2d%s: %s\n", terminal.Bold, i+1, terminal.Reset, dbName)
	}

	fmt.Printf("\n%s%s%s", terminal.Bold, i18n.T(i18n.MsgDbMenuSelectDb), terminal.Reset)
	dbInput, ok := shared.ReadLine(reader)
	if !ok {
		return errQuit
	}

	if dbInput == "b" || dbInput == "B" {
		return errBack
	}
	if dbInput == "q" || dbInput == "Q" {
		return errQuit
	}

	dbIdx, err := strconv.Atoi(dbInput)
	if err != nil || dbIdx < 1 || dbIdx > len(dbList) {
		return errors.New(i18n.T(i18n.MsgInvalid, "max", len(dbList)))
	}

	selectedDb := dbList[dbIdx-1]
	dbId := cast.ToUint64(dbInstance["id"])

	db := map[string]interface{}{
		"id":   dbId,
		"name": selectedDb,
		"type": dbInstance["type"],
	}

	if !shared.Quiet {
		fmt.Printf("\n%s%s%s\n", terminal.Cyan, i18n.T(i18n.MsgDbSqlConnecting), terminal.Reset)
	}
	if err := apiClient.TestDbConnection(dbId, selectedDb); err != nil {
		return shared.Fail(i18n.MsgDbSqlConnFail, err)
	}

	if !shared.Quiet {
		fmt.Printf("%s%s%s\n", terminal.Green, i18n.T(i18n.MsgDbSqlConnected, "db", selectedDb, "type", dbInstance["type"]), terminal.Reset)
	}
	executeSQLQuery(cmd, db, reader)
	return nil
}

// dbSqlNonInteractive 非交互式 SQL 执行（使用 --id --db --sql 标志）
func dbSqlNonInteractive(cmd *cobra.Command) error {
	dbId, _ := cmd.Flags().GetUint64("id")
	dbName, _ := cmd.Flags().GetString("db")
	sqlStr, _ := cmd.Flags().GetString("sql")

	apiClient := shared.NewClient(cmd)

	if dbName == "" {
		dbList, err := apiClient.GetInstanceDatabasesById(dbId)
		if err != nil {
			return shared.Fail(i18n.MsgDbNonInteractiveGetListFail, err)
		}
		if shared.JsonOutput {
			shared.PrintJSONSuccess(map[string]interface{}{"databases": dbList})
			return nil
		}
		for i, name := range dbList {
			fmt.Printf("  %d: %s\n", i+1, name)
		}
		return nil
	}

	if sqlStr == "" {
		if err := apiClient.TestDbConnection(dbId, dbName); err != nil {
			return shared.Fail(i18n.MsgDbNonInteractiveConnFail, err)
		}
		if shared.JsonOutput {
			shared.PrintJSONSuccess(map[string]interface{}{
				"dbId":   dbId,
				"db":     dbName,
				"status": "connected",
			})
		} else {
			fmt.Printf("%s\n", i18n.T(i18n.MsgDbNonInteractiveConnOk, "db", dbName, "id", dbId))
		}
		return nil
	}

	result, err := apiClient.ExecSql(dbId, dbName, sqlStr)
	if err != nil {
		return shared.Fail(i18n.MsgDbNonInteractiveExecFail, err)
	}

	if shared.JsonOutput {
		shared.PrintJSONSuccess(result)
		return nil
	}

	shared.PrintResult(result)
	return nil
}

// executeSQLQuery 交互式 SQL 执行循环
func executeSQLQuery(cmd *cobra.Command, db map[string]interface{}, reader *bufio.Reader) {
	dbId := cast.ToUint64(db["id"])
	dbName := cast.ToString(db["name"])
	apiClient := shared.NewClient(cmd)

	loadHistory()

	if !shared.Quiet {
		fmt.Printf("\n%s%s%s\n", terminal.Green, i18n.T(i18n.MsgDbSqlPrompt), terminal.Reset)
		fmt.Printf("%s%s%s\n", terminal.Yellow, i18n.T(i18n.MsgDbSqlHint1), terminal.Reset)
		fmt.Printf("%s%s%s\n", terminal.Yellow, i18n.T(i18n.MsgDbSqlHint2), terminal.Reset)
		fmt.Printf("%s%s%s\n", terminal.Yellow, i18n.T(i18n.MsgDbSqlHint3), terminal.Reset)
		fmt.Printf("%s%s%s\n\n", terminal.Yellow, i18n.T(i18n.MsgDbSqlHint4), terminal.Reset)
	}

	for {
		fmt.Printf("%s%s> %s", terminal.Yellow, dbName, terminal.Reset)
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

		switch input {
		case "history":
			showHistory()
			continue
		case "clear":
			clearHistory()
			continue
		case "help":
			showSQLHelp()
			continue
		case "tables":
			fmt.Printf("\n%s%s%s\n", terminal.Cyan, i18n.T(i18n.MsgDbSqlGetTables), terminal.Reset)
			result, err := apiClient.GetTableInfos(dbId, dbName)
			if err != nil {
				fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgDbTablesFailed, "err", err.Error()), terminal.Reset)
				continue
			}
			shared.PrintResult(result)
			continue
		case "info":
			fmt.Printf("\n%s%s%s\n", terminal.Cyan, i18n.T(i18n.MsgDbSqlGetInfo), terminal.Reset)
			dbInfo, err := apiClient.GetDbInfo(dbId)
			if err != nil {
				fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgDbSqlInfoFailed, "err", err.Error()), terminal.Reset)
				continue
			}
			fmt.Printf("\n%s%s%s\n", terminal.Cyan, i18n.T(i18n.MsgDbSqlInfoTitle), terminal.Reset)
			// 对 map key 排序后输出，保证输出顺序稳定
			keys := make([]string, 0, len(dbInfo))
			for k := range dbInfo {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, key := range keys {
				fmt.Printf("  %s%-15s%s: %v\n", terminal.Bold, key, terminal.Reset, dbInfo[key])
			}
			fmt.Println()
			continue
		}

		if strings.HasPrefix(strings.ToLower(input), "desc ") {
			tableName := strings.TrimSpace(input[5:])
			if tableName == "" {
				fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgDbSqlDescSpecify), terminal.Reset)
				continue
			}
			dbType := cast.ToString(db["type"])
			descSQL := buildDescSQL(dbType, tableName)
			if descSQL == "" {
				fmt.Printf("%s%s%s\n", terminal.Yellow, i18n.T(i18n.MsgDbSqlDescUnsupport, "type", dbType), terminal.Reset)
				continue
			}
			result, err := apiClient.ExecSql(dbId, dbName, descSQL)
			if err != nil {
				fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgDbSqlQueryFailed, "err", err.Error()), terminal.Reset)
				continue
			}
			shared.PrintResult(result)
			continue
		}

		addToHistory(input)

		result, err := apiClient.ExecSql(dbId, dbName, input)
		if err != nil {
			printSQLError(err, input)
			continue
		}

		shared.PrintResult(result)
	}
}
