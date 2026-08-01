package db

import (
	"bufio"
	"errors"
	"fmt"
	"mayfly-go/cli/cmd/shared"
	"mayfly-go/cli/i18n"
	"mayfly-go/cli/terminal"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

// 交互流程控制哨兵错误
var (
	errBack = errors.New("back")
	errQuit = errors.New("quit")
)

// Cmd 数据库操作父命令
var Cmd = &cobra.Command{
	Use:   "db",
	Short: i18n.T(i18n.MsgDbShort),
	Long:  i18n.T(i18n.MsgDbLong),
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: i18n.T(i18n.MsgDbListShort),
	Long:  i18n.T(i18n.MsgDbListLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiClient := shared.NewClient(cmd)
		result, err := apiClient.ListDbs()

		if err := shared.HandleError(err, shared.GetServer(cmd)); err != nil {
			return shared.Fail(i18n.MsgDbListFailed, err)
		}

		if result == nil || result.Total == 0 {
			if shared.JsonOutput {
				shared.PrintJSONSuccess(map[string]interface{}{"total": 0, "list": []interface{}{}})
				return nil
			}
			terminal.PrintWarning(i18n.T(i18n.MsgDbNoResource))
			return nil
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		printDatabaseTable(result)
		return interactiveDatabaseMenu(cmd, result.List)
	},
}

var execCmd = &cobra.Command{
	Use:     "exec",
	Aliases: []string{"query", "q"},
	Short:   i18n.T(i18n.MsgDbExecShort),
	Long:    i18n.T(i18n.MsgDbExecLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbId, _ := cmd.Flags().GetUint64("id")
		dbName, _ := cmd.Flags().GetString("db")
		sqlStr, _ := cmd.Flags().GetString("sql")
		sqlFile, _ := cmd.Flags().GetString("file")

		if sqlFile != "" {
			data, err := os.ReadFile(sqlFile)
			if err != nil {
				return shared.Fail(i18n.MsgDbExecFailed, fmt.Errorf("read sql file: %w", err))
			}
			sqlStr = string(data)
		}

		if sqlStr == "" {
			return shared.Fail(i18n.MsgDbExecRequired, nil)
		}

		apiClient := shared.NewClient(cmd)
		result, err := apiClient.ExecSql(dbId, dbName, sqlStr)
		if err != nil {
			return shared.Fail(i18n.MsgDbExecFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		shared.PrintResult(result)
		return nil
	},
}

var sqlCmd = &cobra.Command{
	Use:   i18n.T(i18n.MsgDbSqlUse),
	Short: i18n.T(i18n.MsgDbSqlShort),
	Long:  i18n.T(i18n.MsgDbSqlLong),
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("id") {
			return dbSqlNonInteractive(cmd)
		}

		apiClient := shared.NewClient(cmd)
		result, err := apiClient.ListDbs()

		if err := shared.HandleError(err, shared.GetServer(cmd)); err != nil {
			return shared.Fail(i18n.MsgDbSqlGetListFail, err)
		}

		if result == nil || result.Total == 0 {
			if shared.JsonOutput {
				shared.PrintJSONSuccess(map[string]interface{}{"total": 0, "list": []interface{}{}})
				return nil
			}
			terminal.PrintWarning(i18n.T(i18n.MsgDbSqlNoRes))
			return nil
		}

		if len(args) > 0 {
			idx, err := strconv.Atoi(args[0])
			if err != nil || idx < 1 || idx > len(result.List) {
				return fmt.Errorf("%s", i18n.T(i18n.MsgDbSqlInvalidDbIdx, "arg", args[0], "max", len(result.List)))
			}

			err = enterInstance(cmd, apiClient, result.List[idx-1], bufio.NewReader(os.Stdin))
			if errors.Is(err, errBack) || errors.Is(err, errQuit) {
				return nil
			}
			return err
		}

		printDatabaseTable(result)
		return interactiveDatabaseMenu(cmd, result.List)
	},
}

var tablesCmd = &cobra.Command{
	Use:   "tables",
	Short: i18n.T(i18n.MsgDbTablesShort),
	Long:  i18n.T(i18n.MsgDbTablesLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbId, _ := cmd.Flags().GetUint64("id")
		dbName, _ := cmd.Flags().GetString("db")

		apiClient := shared.NewClient(cmd)
		result, err := apiClient.GetTableInfos(dbId, dbName)
		if err != nil {
			return shared.Fail(i18n.MsgDbTablesFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		shared.PrintResult(result)
		return nil
	},
}

var descCmd = &cobra.Command{
	Use:   "desc",
	Short: i18n.T(i18n.MsgDbDescShort),
	Long:  i18n.T(i18n.MsgDbDescLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbId, _ := cmd.Flags().GetUint64("id")
		dbName, _ := cmd.Flags().GetString("db")
		tableName, _ := cmd.Flags().GetString("table")
		dbType, _ := cmd.Flags().GetString("type")

		descSQL := buildDescSQL(dbType, tableName)
		if descSQL == "" {
			descSQL = fmt.Sprintf("DESCRIBE %s", tableName)
		}

		apiClient := shared.NewClient(cmd)
		result, err := apiClient.ExecSql(dbId, dbName, descSQL)
		if err != nil {
			return shared.Fail(i18n.MsgDbDescFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		shared.PrintResult(result)
		return nil
	},
}

var columnsCmd = &cobra.Command{
	Use:   "columns",
	Short: i18n.T(i18n.MsgDbColumnsShort),
	Long:  i18n.T(i18n.MsgDbColumnsLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbId, _ := cmd.Flags().GetUint64("id")
		dbName, _ := cmd.Flags().GetString("db")
		tableName, _ := cmd.Flags().GetString("table")

		apiClient := shared.NewClient(cmd)
		result, err := apiClient.GetColumnMetadata(dbId, dbName, tableName)
		if err != nil {
			return shared.Fail(i18n.MsgDbColumnsFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		shared.PrintResult(result)
		return nil
	},
}

var ddlCmd = &cobra.Command{
	Use:   "ddl",
	Short: i18n.T(i18n.MsgDbDdlShort),
	Long:  i18n.T(i18n.MsgDbDdlLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbId, _ := cmd.Flags().GetUint64("id")
		dbName, _ := cmd.Flags().GetString("db")
		tableName, _ := cmd.Flags().GetString("table")

		apiClient := shared.NewClient(cmd)
		result, err := apiClient.GetTableDDL(dbId, dbName, tableName)
		if err != nil {
			return shared.Fail(i18n.MsgDbDdlFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		if ddl, ok := result.(string); ok {
			fmt.Println(ddl)
		} else {
			shared.PrintResult(result)
		}
		return nil
	},
}

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: i18n.T(i18n.MsgDbIndexShort),
	Long:  i18n.T(i18n.MsgDbIndexLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbId, _ := cmd.Flags().GetUint64("id")
		dbName, _ := cmd.Flags().GetString("db")
		tableName, _ := cmd.Flags().GetString("table")

		apiClient := shared.NewClient(cmd)
		result, err := apiClient.GetTableIndex(dbId, dbName, tableName)
		if err != nil {
			return shared.Fail(i18n.MsgDbIndexFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		shared.PrintResult(result)
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: i18n.T(i18n.MsgDbVersionShort),
	Long:  i18n.T(i18n.MsgDbVersionLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbId, _ := cmd.Flags().GetUint64("id")
		dbName, _ := cmd.Flags().GetString("db")
		apiClient := shared.NewClient(cmd)
		result, err := apiClient.GetDbVersion(dbId, dbName)
		if err != nil {
			return shared.Fail(i18n.MsgDbExecFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		fmt.Printf("\n  %s\n\n", i18n.T(i18n.MsgDbVersionResult, "id", dbId, "db", dbName, "version", result))
		return nil
	},
}

var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: i18n.T(i18n.MsgDbDumpShort),
	Long:  i18n.T(i18n.MsgDbDumpLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbId, _ := cmd.Flags().GetUint64("id")
		dbName, _ := cmd.Flags().GetString("db")
		dumpType, _ := cmd.Flags().GetString("type")
		tables, _ := cmd.Flags().GetString("tables")
		output, _ := cmd.Flags().GetString("output")

		apiClient := shared.NewClient(cmd)
		data, err := apiClient.DumpDb(dbId, dbName, dumpType, tables)
		if err != nil {
			return shared.Fail(i18n.MsgDbDumpFailed, err)
		}

		if output != "" {
			if err := os.WriteFile(output, data, 0644); err != nil {
				return shared.Fail(i18n.MsgDbDumpFailed, err)
			}
			if shared.JsonOutput {
				shared.PrintJSONSuccess(map[string]interface{}{"file": output, "size": len(data)})
				return nil
			}
			fmt.Printf("%s %s (%d bytes)\n", i18n.T(i18n.MsgDbDumpOk), output, len(data))
			return nil
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(map[string]interface{}{"sql": string(data), "size": len(data)})
			return nil
		}
		fmt.Print(string(data))
		return nil
	},
}

func init() {
	Cmd.GroupID = "resource"
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(execCmd)
	Cmd.AddCommand(sqlCmd)
	Cmd.AddCommand(tablesCmd)
	Cmd.AddCommand(descCmd)
	Cmd.AddCommand(columnsCmd)
	Cmd.AddCommand(ddlCmd)
	Cmd.AddCommand(indexCmd)
	Cmd.AddCommand(dumpCmd)
	Cmd.AddCommand(versionCmd)

	// db exec 标志
	execCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgDbFlagExecId))
	execCmd.Flags().String("db", "", i18n.T(i18n.MsgDbFlagExecDb))
	execCmd.Flags().String("sql", "", i18n.T(i18n.MsgDbFlagExecSql))
	execCmd.Flags().String("file", "", "SQL file to execute (alternative to --sql)")
	execCmd.MarkFlagRequired("id")
	execCmd.MarkFlagRequired("db")

	// db version 标志
	versionCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgDbFlagVersionId))
	versionCmd.Flags().String("db", "", i18n.T(i18n.MsgDbFlagVersionDb))
	versionCmd.MarkFlagRequired("id")
	versionCmd.MarkFlagRequired("db")

	// db sql 标志
	sqlCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgDbFlagSqlId))
	sqlCmd.Flags().String("db", "", i18n.T(i18n.MsgDbFlagSqlDb))
	sqlCmd.Flags().String("sql", "", i18n.T(i18n.MsgDbFlagSqlSql))

	// db tables 标志
	tablesCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgDbFlagTablesId))
	tablesCmd.Flags().String("db", "", i18n.T(i18n.MsgDbFlagTablesDb))
	tablesCmd.MarkFlagRequired("id")
	tablesCmd.MarkFlagRequired("db")

	// db desc 标志
	descCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgDbFlagDescId))
	descCmd.Flags().String("db", "", i18n.T(i18n.MsgDbFlagDescDb))
	descCmd.Flags().String("table", "", i18n.T(i18n.MsgDbFlagDescTable))
	descCmd.Flags().String("type", "mysql", i18n.T(i18n.MsgDbFlagDescType))
	descCmd.MarkFlagRequired("id")
	descCmd.MarkFlagRequired("db")
	descCmd.MarkFlagRequired("table")

	// db columns 标志
	columnsCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgDbFlagColumnsId))
	columnsCmd.Flags().String("db", "", i18n.T(i18n.MsgDbFlagColumnsDb))
	columnsCmd.Flags().String("table", "", i18n.T(i18n.MsgDbFlagColumnsTable))
	columnsCmd.MarkFlagRequired("id")
	columnsCmd.MarkFlagRequired("db")
	columnsCmd.MarkFlagRequired("table")

	// db ddl 标志
	ddlCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgDbFlagDdlId))
	ddlCmd.Flags().String("db", "", i18n.T(i18n.MsgDbFlagDdlDb))
	ddlCmd.Flags().String("table", "", i18n.T(i18n.MsgDbFlagDdlTable))
	ddlCmd.MarkFlagRequired("id")
	ddlCmd.MarkFlagRequired("db")
	ddlCmd.MarkFlagRequired("table")

	// db index 标志
	indexCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgDbFlagIndexId))
	indexCmd.Flags().String("db", "", i18n.T(i18n.MsgDbFlagIndexDb))
	indexCmd.Flags().String("table", "", i18n.T(i18n.MsgDbFlagIndexTable))
	indexCmd.MarkFlagRequired("id")
	indexCmd.MarkFlagRequired("db")
	indexCmd.MarkFlagRequired("table")

	// db dump 标志
	dumpCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgDbFlagDumpId))
	dumpCmd.Flags().String("db", "", i18n.T(i18n.MsgDbFlagDumpDb))
	dumpCmd.Flags().String("type", "3", i18n.T(i18n.MsgDbFlagDumpType))
	dumpCmd.Flags().String("tables", "", i18n.T(i18n.MsgDbFlagDumpTables))
	dumpCmd.Flags().StringP("output", "o", "", i18n.T(i18n.MsgDbFlagDumpOutput))
	dumpCmd.MarkFlagRequired("id")
	dumpCmd.MarkFlagRequired("db")
	dumpCmd.MarkFlagRequired("type")

	// 动态补全
	shared.RegisterIdCompletion(execCmd, "db")
	shared.RegisterIdCompletion(versionCmd, "db")
	shared.RegisterIdCompletion(tablesCmd, "db")
	shared.RegisterIdCompletion(descCmd, "db")
	shared.RegisterIdCompletion(columnsCmd, "db")
	shared.RegisterIdCompletion(ddlCmd, "db")
	shared.RegisterIdCompletion(indexCmd, "db")
	shared.RegisterIdCompletion(dumpCmd, "db")
}
