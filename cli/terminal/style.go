package terminal

import (
	"fmt"
	"mayfly-go/cli/i18n"
	"os"
	"strings"
	"unicode"

	"golang.org/x/term"
)

// ANSI 颜色代码
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	White  = "\033[37m"
	Bold   = "\033[1m"
)

// TableConfig 表格配置
type TableConfig struct {
	Title   string   // 标题（支持 Emoji）
	Headers []string // 表头
	Widths  []int    // 列宽
	Colors  []string // 列颜色（可选）
	Total   int      // 总数
	Tip     string   // 底部提示
}

// TableRow 表格行数据
type TableRow struct {
	Values []interface{} // 单元格值
	Color  string        // 行颜色（可选）
}

// PrintTitle 打印带边框的标题
func PrintTitle(title string, total int) {
	width := 58
	titleWidth := displayWidth(title)
	padding := (width - titleWidth - 10) / 2 // 10 是 Emoji 和括号的空间

	if padding < 2 {
		padding = 2
	}

	leftPad := strings.Repeat(" ", padding)
	rightPad := strings.Repeat(" ", width-titleWidth-10-padding)

	fmt.Printf("\n%s%s╔%s╗%s\n", Cyan, Bold, strings.Repeat("═", width), Reset)
	fmt.Printf("%s%s║%s  %s  %s║%s\n", Cyan, Bold, leftPad, title, rightPad, Reset)
	if total > 0 {
		fmt.Printf("%s%s║%s  (%s %d)  %s║%s\n", Cyan, Bold, strings.Repeat(" ", 8), T(i18n.MsgTotal), total, strings.Repeat(" ", 10), Reset)
	}
	fmt.Printf("%s%s╚%s╝%s\n\n", Cyan, Bold, strings.Repeat("═", width), Reset)
}

// displayWidth 计算字符串的显示宽度（CJK 字符占 2 列）
func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hangul, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hiragana, r) {
			width += 2
		} else {
			width++
		}
	}
	return width
}

// PrintTable 打印表格
func PrintTable(config TableConfig, rows []TableRow) {
	totalWidth := 0
	for _, width := range config.Widths {
		totalWidth += width
	}

	// 顶部框线
	fmt.Printf("  %s┌%s┐%s\n", Cyan, strings.Repeat("─", totalWidth), Reset)

	// 打印表头
	fmt.Printf("  %s│%s", Cyan, Bold)
	for i, header := range config.Headers {
		if i < len(config.Widths) {
			fmt.Printf("%-*s", config.Widths[i], header)
		} else {
			fmt.Printf("%s", header)
		}
	}
	fmt.Printf("%s%s│%s\n", Reset, Cyan, Reset)

	// 表头分隔线
	fmt.Printf("  %s├%s┤%s\n", Cyan, strings.Repeat("─", totalWidth), Reset)

	// 打印数据行
	for _, row := range rows {
		fmt.Printf("  %s│%s", Cyan, Reset)
		for i, value := range row.Values {
			if i < len(config.Widths) {
				cellColor := row.Color
				if i < len(config.Colors) && config.Colors[i] != "" {
					cellColor = config.Colors[i]
				}
				valStr := fmt.Sprintf("%v", value)
				fmt.Printf("%s%-*s%s", cellColor, config.Widths[i], valStr, Reset)
			} else {
				fmt.Printf("%v", value)
			}
		}
		fmt.Printf("%s│%s\n", Cyan, Reset)
	}

	// 底部框线
	fmt.Printf("  %s└%s┘%s\n", Cyan, strings.Repeat("─", totalWidth), Reset)

	// 打印提示
	if config.Tip != "" {
		fmt.Printf("\n%s%s %s%s\n\n", Cyan, T(i18n.MsgTermTip), config.Tip, Reset)
	}
}

// PrintWarning 打印警告信息
func PrintWarning(message string) {
	fmt.Printf("%s⚠️  %s%s\n", Yellow, message, Reset)
}

// PrintError 打印错误信息
func PrintError(message string) {
	fmt.Printf("%s❌ %s%s\n", Red, message, Reset)
}

// PrintSuccess 打印成功信息
func PrintSuccess(message string) {
	fmt.Printf("%s✅ %s%s\n", Green, message, Reset)
}

// PrintInfo 打印信息
func PrintInfo(message string) {
	fmt.Printf("%sℹ️  %s%s\n", Cyan, message, Reset)
}

// Colorize 给文本添加颜色
func Colorize(text string, color string) string {
	return fmt.Sprintf("%s%s%s", color, text, Reset)
}

// BoldText 加粗文本
func BoldText(text string) string {
	return fmt.Sprintf("%s%s%s", Bold, text, Reset)
}

// StatusText 状态文本（在线/离线）
func StatusText(isOnline bool) string {
	if isOnline {
		return fmt.Sprintf("%s%s%s", Green, T(i18n.MsgOnline), Reset)
	}
	return fmt.Sprintf("%s%s%s", Red, T(i18n.MsgOffline), Reset)
}

// DatabaseTypeColor 根据数据库类型返回颜色
func DatabaseTypeColor(dbType string) string {
	switch dbType {
	case "mysql":
		return Blue
	case "postgres", "postgresql":
		return Purple
	case "sqlite":
		return Yellow
	case "dm":
		return Cyan
	default:
		return Green
	}
}

// RedisModeColor 根据 Redis 模式返回颜色
func RedisModeColor(mode string) string {
	switch mode {
	case "cluster":
		return Purple
	case "sentinel":
		return Yellow
	default:
		return Green
	}
}

// ProgressBar 简单的进度条
func ProgressBar(current, total int, width int) string {
	if total == 0 {
		return ""
	}

	percent := float64(current) / float64(total)
	filled := int(percent * float64(width))
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return fmt.Sprintf("%s%s %d/%d (%.1f%%)%s", Green, bar, current, total, percent*100, Reset)
}

// ClearLine 清除当前行
func ClearLine() {
	fmt.Print("\033[2K\r")
}

// CursorUp 光标上移 n 行
func CursorUp(n int) {
	if n > 0 {
		fmt.Printf("\033[%dA", n)
	}
}

// CursorDown 光标下移 n 行
func CursorDown(n int) {
	if n > 0 {
		fmt.Printf("\033[%dB", n)
	}
}

// ReadPassword 读取密码输入（隐藏输入内容）
func ReadPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // 换行
	if err != nil {
		return "", err
	}
	return string(password), nil
}
