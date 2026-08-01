package db

import (
	"fmt"
	"mayfly-go/cli/client"
	"mayfly-go/cli/cmd/shared"
	"mayfly-go/cli/i18n"
	"mayfly-go/cli/terminal"
	"strings"

	"github.com/spf13/cast"
)

// printDatabaseTable 打印数据库实例列表表格
func printDatabaseTable(result *client.PageResult) {
	terminal.PrintTitle(i18n.T(i18n.MsgDbMenuTitle), int(result.Total))

	var rows []terminal.TableRow
	for i, db := range result.List {
		address := shared.FormatAddress(db["host"], db["port"])
		typeColor := terminal.DatabaseTypeColor(cast.ToString(db["type"]))

		rows = append(rows, terminal.TableRow{
			Values: []interface{}{i + 1, db["name"], db["type"], address},
			Color:  typeColor,
		})
	}

	terminal.PrintTable(terminal.TableConfig{
		Headers: []string{i18n.T(i18n.MsgIndex), i18n.T(i18n.MsgName), i18n.T(i18n.MsgType), i18n.T(i18n.MsgAddress)},
		Widths:  []int{8, 26, 14, 42},
		Tip:     i18n.T(i18n.MsgDbTableTip),
	}, rows)
}

// buildDescSQL 根据数据库类型构建 DESC 语句
func buildDescSQL(dbType, tableName string) string {
	switch strings.ToLower(dbType) {
	case "mysql":
		return fmt.Sprintf("DESCRIBE %s", tableName)
	case "postgresql", "postgres":
		return fmt.Sprintf("SELECT column_name, data_type, is_nullable, column_default FROM information_schema.columns WHERE table_name = '%s' ORDER BY ordinal_position", tableName)
	case "sqlite":
		return fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	case "dm":
		return fmt.Sprintf("SELECT COLUMN_NAME, DATA_TYPE, NULLABLE, DATA_DEFAULT FROM USER_TAB_COLUMNS WHERE TABLE_NAME = '%s'", tableName)
	default:
		return ""
	}
}

// printSQLError 格式化输出 SQL 执行错误
func printSQLError(err error, sql string) {
	errMsg := err.Error()
	fmt.Printf("\n%s%s%s\n", terminal.Bold, i18n.T(i18n.MsgDbErrTitle), terminal.Reset)
	fmt.Printf("%s─────────────────────────────────────────────────────────────%s\n", terminal.Red, terminal.Reset)

	if strings.Contains(errMsg, "connection refused") {
		fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgDbErrConnRefused), terminal.Reset)
	} else if strings.Contains(errMsg, "Unknown database") {
		fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgDbErrNoDb), terminal.Reset)
	} else if strings.Contains(errMsg, "Access denied") {
		fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgDbErrAccessDenied), terminal.Reset)
	} else if strings.Contains(errMsg, "syntax error") || strings.Contains(errMsg, "SQL syntax") {
		fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgDbErrSyntax), terminal.Reset)
	} else if strings.Contains(errMsg, "Table") && strings.Contains(errMsg, "doesn't exist") {
		fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgDbErrNoTable), terminal.Reset)
	} else if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "Timeout") {
		fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgDbErrTimeout), terminal.Reset)
	} else {
		fmt.Printf("%s%s%s\n", terminal.Red, i18n.T(i18n.MsgDbErrDetail, "err", err.Error()), terminal.Reset)
	}

	fmt.Printf("%s─────────────────────────────────────────────────────────────%s\n\n", terminal.Red, terminal.Reset)
}
