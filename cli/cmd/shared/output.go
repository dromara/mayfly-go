package shared

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"mayfly-go/cli/i18n"
	"mayfly-go/cli/terminal"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cast"
	"gopkg.in/yaml.v3"
)

// --- JSON 输出工具函数 ---

// OutputFormat 输出格式类型
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatCSV   OutputFormat = "csv"
	FormatYAML  OutputFormat = "yaml"
)

// OutputFmt 全局输出格式（由 root 命令初始化）
var OutputFmt OutputFormat = FormatTable

// CurrentCommand 当前执行的命令（用于 JSON 元数据）
var CurrentCommand string

// OperationResult 操作结果（统一格式）
type OperationResult struct {
	Operation string                 `json:"operation"`
	Target    map[string]interface{} `json:"target,omitempty"`
	Result    string                 `json:"result"` // created, updated, deleted, executed
	Data      interface{}            `json:"data,omitempty"`
}

// PrintOperationResult 输出统一格式的操作结果
func PrintOperationResult(operation string, target map[string]interface{}, result string, data interface{}) {
	if JsonOutput || AgentMode {
		res := OperationResult{
			Operation: operation,
			Target:    target,
			Result:    result,
			Data:      data,
		}
		PrintJSONSuccess(res)
	} else {
		if data != nil {
			fmt.Printf("%s: %s\n", operation, result)
		} else {
			fmt.Printf("%s: %s\n", operation, result)
		}
	}
}

// PrintJSON 输出 JSON 格式数据
func PrintJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

// PrintJSONError 输出 JSON 格式错误
func PrintJSONError(msg string, err error) {
	PrintJSONErrorWithCode(msg, err, 1)
}

// PrintJSONErrorWithCode 输出带错误码的 JSON 格式错误（供 Agent 程序化处理）
func PrintJSONErrorWithCode(msg string, err error, code int) {
	result := map[string]interface{}{
		"success": false,
		"code":    code,
		"message": msg,
	}
	if err != nil {
		result["error"] = err.Error()
	}
	// Agent 模式添加恢复指引
	if AgentMode {
		result["recovery"] = getRecoveryGuidance(code)
	}
	PrintJSON(result)
}

// RecoveryInfo 错误恢复指引
type RecoveryInfo struct {
	Action  string `json:"action"`
	Command string `json:"command,omitempty"`
	Hint    string `json:"hint,omitempty"`
}

// getRecoveryGuidance 根据错误码返回恢复指引
func getRecoveryGuidance(code int) *RecoveryInfo {
	switch code {
	case ErrCodeAuth:
		return &RecoveryInfo{
			Action:  "login",
			Command: "mayfly-cli login -s <server> -u <username> -p <password>",
			Hint:    "请先登录或检查 token 是否过期",
		}
	case ErrCodeConnection:
		return &RecoveryInfo{
			Action:  "check_server",
			Command: "mayfly-cli ping",
			Hint:    "请检查服务器地址是否正确，服务是否运行",
		}
	case ErrCodeValidation:
		return &RecoveryInfo{
			Action:  "check_params",
			Command: "mayfly-cli <command> --help",
			Hint:    "请检查参数是否正确",
		}
	case ErrCodeNotFound:
		return &RecoveryInfo{
			Action:  "list_resources",
			Command: "mayfly-cli <resource> list",
			Hint:    "资源不存在，请先列出可用资源",
		}
	case ErrCodePermission:
		return &RecoveryInfo{
			Action: "check_permission",
			Hint:   "权限不足，请联系管理员",
		}
	default:
		return &RecoveryInfo{
			Action: "retry_or_report",
			Hint:   "请重试或联系管理员",
		}
	}
}

// PrintJSONSuccess 输出 JSON 格式成功结果
func PrintJSONSuccess(data interface{}) {
	result := map[string]interface{}{
		"success": true,
		"data":    data,
	}
	// Agent 模式添加元数据
	if AgentMode {
		result["timestamp"] = time.Now().UTC().Format(time.RFC3339)
		if CurrentCommand != "" {
			result["command"] = CurrentCommand
		}
	}
	PrintJSON(result)
}

// PrintResult 通用结果输出（自动识别结构类型：columns/rows、数组、map 等）
func PrintResult(result interface{}) {
	if result == nil {
		fmt.Printf("\n%s\n", i18n.T(i18n.MsgDbQueryOk))
		return
	}

	if resultMap, ok := result.(map[string]interface{}); ok {
		printStructuredResult(resultMap)
	} else if rows, ok := result.([]interface{}); ok {
		printRowsResult(rows)
	} else {
		fmt.Printf("\n%+v\n\n", result)
	}
}

func printStructuredResult(result map[string]interface{}) {
	if columns, hasColumns := result["columns"]; hasColumns {
		if rows, hasRows := result["rows"]; hasRows {
			printSQLResult(columns, rows)
			return
		}
	}

	// 对 map key 排序后输出，保证输出顺序稳定
	keys := make([]string, 0, len(result))
	for key := range result {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	fmt.Println()
	for _, key := range keys {
		fmt.Printf("  %s%-15s%s: %v\n", terminal.Bold, key, terminal.Reset, result[key])
	}
	fmt.Println()
}

func printRowsResult(rows []interface{}) {
	if len(rows) == 0 {
		fmt.Printf("\n%s\n", i18n.T(i18n.MsgDbQueryOk))
		return
	}

	if firstRow, ok := rows[0].(map[string]interface{}); ok {
		printMapRowsResult(rows, firstRow)
	} else {
		fmt.Printf("\n%s%s, %s %d %s\n\n", terminal.Green, i18n.T(i18n.MsgDbQueryOk), i18n.T(i18n.MsgDbQueryRecords), len(rows), terminal.Reset)
		for i, row := range rows {
			fmt.Printf("  [%d] %v\n", i+1, row)
		}
		fmt.Println()
	}
}

func printMapRowsResult(rows []interface{}, firstRow map[string]interface{}) {
	columns := make([]string, 0, len(firstRow))
	for col := range firstRow {
		columns = append(columns, col)
	}
	sort.Strings(columns)
	RenderRowsTable(columns, rows)
}

func printSQLResult(columns interface{}, rows interface{}) {
	colSlice, ok := columns.([]interface{})
	if !ok {
		fmt.Printf("\n%+v\n\n", columns)
		return
	}

	rowSlice, ok := rows.([]interface{})
	if !ok {
		fmt.Printf("\n%+v\n\n", rows)
		return
	}

	if len(rowSlice) == 0 {
		fmt.Print(i18n.T(i18n.MsgDbSqlResultOk))
		return
	}

	columnNames := make([]string, len(colSlice))
	for i, col := range colSlice {
		if colMap, ok := col.(map[string]interface{}); ok {
			if name, exists := colMap["name"]; exists {
				columnNames[i] = cast.ToString(name)
			}
		} else {
			columnNames[i] = cast.ToString(col)
		}
	}

	RenderRowsTable(columnNames, rowSlice)
}

// maxDisplayRows 终端表格最多展示的行数
const maxDisplayRows = 100

// RenderRowsTable 渲染表格行
func RenderRowsTable(columnNames []string, rows []interface{}) {
	widths := make([]int, len(columnNames))
	for i, name := range columnNames {
		widths[i] = len(name) + 2
	}

	for _, row := range rows {
		if rowMap, ok := row.(map[string]interface{}); ok {
			for i, name := range columnNames {
				valLen := len(cast.ToString(rowMap[name]))
				if valLen+2 > widths[i] {
					widths[i] = valLen + 2
				}
			}
		}
	}

	fmt.Printf("\n%s", terminal.Bold)
	for i, name := range columnNames {
		fmt.Printf("%-*s", widths[i], name)
	}
	fmt.Printf("%s\n", terminal.Reset)

	totalWidth := 0
	for _, width := range widths {
		totalWidth += width
	}
	fmt.Printf("%s%s%s\n", terminal.Cyan, strings.Repeat("-", totalWidth), terminal.Reset)

	for i, row := range rows {
		if i >= maxDisplayRows {
			fmt.Printf("\n%s%s%s\n", terminal.Yellow, i18n.T(i18n.MsgDbQueryShowFirst, "count", len(rows)), terminal.Reset)
			break
		}
		if rowMap, ok := row.(map[string]interface{}); ok {
			for j, name := range columnNames {
				fmt.Printf("%-*s", widths[j], cast.ToString(rowMap[name]))
			}
			fmt.Println()
		}
	}

	fmt.Printf("\n%s%s%s\n\n", terminal.Green, i18n.T(i18n.MsgDbQueryRecords, "count", len(rows)), terminal.Reset)
}

// PrintYAML 输出 YAML 格式数据
func PrintYAML(v interface{}) {
	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)
	defer enc.Close()
	enc.Encode(v)
}

// PrintCSV 输出 CSV 格式数据
func PrintCSV(columnNames []string, rows []interface{}) {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	// 写入表头
	w.Write(columnNames)

	// 写入数据行
	for _, row := range rows {
		if rowMap, ok := row.(map[string]interface{}); ok {
			record := make([]string, len(columnNames))
			for i, name := range columnNames {
				record[i] = cast.ToString(rowMap[name])
			}
			w.Write(record)
		}
	}
}

// PrintFormatted 根据全局输出格式输出数据
func PrintFormatted(data interface{}, columnNames []string, rows []interface{}) {
	switch OutputFmt {
	case FormatJSON:
		PrintJSONSuccess(data)
	case FormatYAML:
		PrintYAML(data)
	case FormatCSV:
		if columnNames != nil && rows != nil {
			PrintCSV(columnNames, rows)
		} else {
			PrintYAML(data)
		}
	default:
		// table 格式由调用方处理
	}
}
