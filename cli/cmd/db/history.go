package db

import (
	"fmt"
	"mayfly-go/cli/i18n"
	"mayfly-go/cli/terminal"
	"os"
	"path/filepath"
	"strings"
)

var sqlHistory []string
var historyFile string

func init() {
	if homeDir, err := os.UserHomeDir(); err == nil {
		historyFile = filepath.Join(homeDir, ".mayfly-cli-history")
	}
}

func loadHistory() {
	if historyFile == "" {
		return
	}
	data, err := os.ReadFile(historyFile)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			sqlHistory = append(sqlHistory, line)
		}
	}
}

func saveHistory() {
	if historyFile == "" {
		return
	}
	content := strings.Join(sqlHistory, "\n")
	os.WriteFile(historyFile, []byte(content), 0644)
}

func addToHistory(sql string) {
	if len(sqlHistory) > 0 && sqlHistory[len(sqlHistory)-1] == sql {
		return
	}
	sqlHistory = append(sqlHistory, sql)
	if len(sqlHistory) > 100 {
		sqlHistory = sqlHistory[len(sqlHistory)-100:]
	}
	saveHistory()
}

func showHistory() {
	if len(sqlHistory) == 0 {
		fmt.Printf("%s%s%s\n", terminal.Yellow, i18n.T(i18n.MsgDbHistoryNone), terminal.Reset)
		return
	}
	fmt.Printf("\n%s%s%s\n", terminal.Cyan, i18n.T(i18n.MsgDbHistoryTitle, "count", len(sqlHistory)), terminal.Reset)
	for i, sql := range sqlHistory {
		fmt.Printf("  %s%3d%s: %s\n", terminal.Bold, i+1, terminal.Reset, sql)
	}
	fmt.Println()
}

func clearHistory() {
	sqlHistory = nil
	saveHistory()
	fmt.Printf("%s%s%s\n", terminal.Green, i18n.T(i18n.MsgDbHistoryCleared), terminal.Reset)
}

func showSQLHelp() {
	fmt.Printf("\n%s%s%s\n", terminal.Cyan, i18n.T(i18n.MsgDbHelpTitle), terminal.Reset)
	fmt.Printf("%s─────────────────────────────────────────────────────────────%s\n", terminal.Cyan, terminal.Reset)
	fmt.Printf("\n%s%s%s\n", terminal.Bold, i18n.T(i18n.MsgDbHelpQuery), terminal.Reset)
	fmt.Printf("  SELECT * FROM table_name LIMIT 10;\n")
	fmt.Printf("  SELECT COUNT(*) FROM table_name;\n")
	fmt.Printf("\n%s%s%s\n", terminal.Bold, i18n.T(i18n.MsgDbHelpStruct), terminal.Reset)
	fmt.Printf("  %s\n", i18n.T(i18n.MsgDbHelpTables))
	fmt.Printf("  %s\n", i18n.T(i18n.MsgDbHelpDesc))
	fmt.Printf("\n%s%s%s\n", terminal.Bold, i18n.T(i18n.MsgDbHelpShortcut), terminal.Reset)
	fmt.Printf("  %s\n", i18n.T(i18n.MsgDbHelpHelp))
	fmt.Printf("  %s\n", i18n.T(i18n.MsgDbHelpHistory))
	fmt.Printf("  %s\n", i18n.T(i18n.MsgDbHelpTablesList))
	fmt.Printf("  %s\n", i18n.T(i18n.MsgDbHelpDescTable))
	fmt.Printf("  %s\n", i18n.T(i18n.MsgDbHelpBack))
	fmt.Printf("  %s\n", i18n.T(i18n.MsgDbHelpQuit))
	fmt.Printf("\n%s%s%s\n", terminal.Yellow, i18n.T(i18n.MsgDbHelpAgent), terminal.Reset)
	fmt.Printf("  %s\n", i18n.T(i18n.MsgDbHelpAgentExec))
	fmt.Printf("  %s\n", i18n.T(i18n.MsgDbHelpAgentTables))
	fmt.Printf("  %s\n", i18n.T(i18n.MsgDbHelpAgentDesc))
	fmt.Println()
}
