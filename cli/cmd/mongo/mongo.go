package mongo

import (
	"encoding/json"
	"mayfly-go/cli/cmd/shared"
	"mayfly-go/cli/i18n"
	"mayfly-go/cli/terminal"

	"github.com/spf13/cobra"
)

// Cmd MongoDB 操作父命令
var Cmd = &cobra.Command{
	Use:     "mongo",
	Aliases: []string{"mg"},
	Short:   i18n.T(i18n.MsgMongoShort),
	Long:    i18n.T(i18n.MsgMongoLong),
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: i18n.T(i18n.MsgMongoListShort),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiClient := shared.NewClient(cmd)
		result, err := apiClient.ListMongo()
		if err := shared.HandleError(err, shared.GetServer(cmd)); err != nil {
			return shared.Fail(i18n.MsgCmdExecFailed, err)
		}

		if result == nil || result.Total == 0 {
			if shared.JsonOutput {
				shared.PrintJSONSuccess(map[string]interface{}{"total": 0, "list": []interface{}{}})
				return nil
			}
			terminal.PrintWarning(i18n.T(i18n.MsgMongoNoResource))
			return nil
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		var rows []terminal.TableRow
		for i, m := range result.List {
			rows = append(rows, terminal.TableRow{
				Values: []interface{}{i + 1, m["name"], shared.FormatAddress(m["host"], m["port"])},
				Color:  terminal.Green,
			})
		}

		terminal.PrintTable(terminal.TableConfig{
			Headers: []string{i18n.T(i18n.MsgIndex), i18n.T(i18n.MsgName), i18n.T(i18n.MsgAddress)},
			Widths:  []int{6, 26, 36},
			Tip:     i18n.T(i18n.MsgMongoTableTip),
		}, rows)
		return nil
	},
}

var databasesCmd = &cobra.Command{
	Use:     "databases",
	Aliases: []string{"dbs"},
	Short:   i18n.T(i18n.MsgMongoDbShort),
	RunE: func(cmd *cobra.Command, args []string) error {
		mongoId, _ := cmd.Flags().GetUint64("id")
		apiClient := shared.NewClient(cmd)
		result, err := apiClient.GetMongoDatabases(mongoId)
		if err != nil {
			return shared.Fail(i18n.MsgCmdExecFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		shared.PrintResult(result)
		return nil
	},
}

var collectionsCmd = &cobra.Command{
	Use:     "collections",
	Aliases: []string{"cols"},
	Short:   i18n.T(i18n.MsgMongoColsShort),
	RunE: func(cmd *cobra.Command, args []string) error {
		mongoId, _ := cmd.Flags().GetUint64("id")
		database, _ := cmd.Flags().GetString("db")
		apiClient := shared.NewClient(cmd)
		result, err := apiClient.GetMongoCollections(mongoId, database)
		if err != nil {
			return shared.Fail(i18n.MsgCmdExecFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		shared.PrintResult(result)
		return nil
	},
}

var findCmd = &cobra.Command{
	Use:   "find",
	Short: i18n.T(i18n.MsgMongoFindShort),
	Long:  i18n.T(i18n.MsgMongoFindLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		mongoId, _ := cmd.Flags().GetUint64("id")
		database, _ := cmd.Flags().GetString("db")
		collection, _ := cmd.Flags().GetString("collection")
		filterStr, _ := cmd.Flags().GetString("filter")
		limit, _ := cmd.Flags().GetInt64("limit")
		skip, _ := cmd.Flags().GetInt64("skip")

		filter := make(map[string]interface{})
		if filterStr != "" {
			if err := json.Unmarshal([]byte(filterStr), &filter); err != nil {
				return shared.Fail(i18n.MsgMongoInvalidFilter, err)
			}
		}
		if limit <= 0 {
			limit = 20
		}

		apiClient := shared.NewClient(cmd)
		result, err := apiClient.MongoFind(mongoId, database, collection, filter, limit, skip)
		if err != nil {
			return shared.Fail(i18n.MsgCmdExecFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		shared.PrintResult(result)
		return nil
	},
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: i18n.T(i18n.MsgMongoRunShort),
	Long:  i18n.T(i18n.MsgMongoRunLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		mongoId, _ := cmd.Flags().GetUint64("id")
		database, _ := cmd.Flags().GetString("db")
		commandStr, _ := cmd.Flags().GetString("command")

		command := make(map[string]interface{})
		if err := json.Unmarshal([]byte(commandStr), &command); err != nil {
			return shared.Fail(i18n.MsgMongoInvalidCommand, err)
		}

		apiClient := shared.NewClient(cmd)
		result, err := apiClient.MongoRunCommand(mongoId, database, command)
		if err != nil {
			return shared.Fail(i18n.MsgCmdExecFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		shared.PrintResult(result)
		return nil
	},
}

func init() {
	Cmd.GroupID = "resource"
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(databasesCmd)
	Cmd.AddCommand(collectionsCmd)
	Cmd.AddCommand(findCmd)
	Cmd.AddCommand(runCmd)

	// mongo databases
	databasesCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgMongoFlagId))
	databasesCmd.MarkFlagRequired("id")

	// mongo collections
	collectionsCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgMongoFlagId))
	collectionsCmd.Flags().String("db", "", i18n.T(i18n.MsgMongoFlagDb))
	collectionsCmd.MarkFlagRequired("id")
	collectionsCmd.MarkFlagRequired("db")

	// mongo find
	findCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgMongoFlagId))
	findCmd.Flags().String("db", "", i18n.T(i18n.MsgMongoFlagDb))
	findCmd.Flags().String("collection", "", i18n.T(i18n.MsgMongoFlagCollection))
	findCmd.Flags().String("filter", "", i18n.T(i18n.MsgMongoFlagFilter))
	findCmd.Flags().Int64("limit", 20, i18n.T(i18n.MsgMongoFlagLimit))
	findCmd.Flags().Int64("skip", 0, i18n.T(i18n.MsgMongoFlagSkip))
	findCmd.MarkFlagRequired("id")
	findCmd.MarkFlagRequired("db")
	findCmd.MarkFlagRequired("collection")

	// mongo run
	runCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgMongoFlagId))
	runCmd.Flags().String("db", "", i18n.T(i18n.MsgMongoFlagDb))
	runCmd.Flags().String("command", "", i18n.T(i18n.MsgMongoFlagCommand))
	runCmd.MarkFlagRequired("id")
	runCmd.MarkFlagRequired("db")
	runCmd.MarkFlagRequired("command")

	// 动态补全
	shared.RegisterIdCompletion(databasesCmd, "mongo")
	shared.RegisterIdCompletion(collectionsCmd, "mongo")
	shared.RegisterIdCompletion(findCmd, "mongo")
	shared.RegisterIdCompletion(runCmd, "mongo")
}
