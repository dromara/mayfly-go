package terminal

import (
	"fmt"
	"mayfly-go/cli/i18n"
)

// DashboardItem 仪表板项目
type DashboardItem struct {
	Name   string // 资产名称
	Icon   string // 图标
	Total  int    // 总数
	Online int    // 在线/可用数量
	Color  string // 颜色
}

// T 翻译消息的快捷方法
func T(msgId string, attrs ...any) string {
	return i18n.T(msgId, attrs...)
}

// PrintDashboard 打印资产概览仪表板
func PrintDashboard(items []DashboardItem, server string, isLoggedIn bool) {
	fmt.Println()

	// 打印标题
	fmt.Printf("%s%s╔══════════════════════════════════════════════════════════╗%s\n", Cyan, Bold, Reset)
	fmt.Printf("%s%s║          %s                ║%s\n", Cyan, Bold, T(i18n.MsgDashboardTitle), Reset)
	fmt.Printf("%s%s╚══════════════════════════════════════════════════════════╝%s\n\n", Cyan, Bold, Reset)

	// 显示服务器信息
	if server != "" {
		fmt.Printf("  %s%s%s %s\n\n", Bold, T(i18n.MsgDashboardServer), Reset, server)
	}

	// 显示登录状态
	if !isLoggedIn {
		fmt.Printf("  %s⚠️  %s%s %s%s - %s\n", Yellow, T(i18n.MsgDashboardStatus), Reset, Yellow, T(i18n.MsgDashboardNotLogged), Reset)
		fmt.Printf("    %s$ mayfly-cli config%s\n\n", Green, Reset)
	} else {
		fmt.Printf("  %s✅ %s%s %s%s%s\n\n", Green, T(i18n.MsgDashboardStatus), Reset, Green, T(i18n.MsgDashboardLoggedIn), Reset)
	}

	// 打印资产统计
	if len(items) > 0 {
		fmt.Printf("  %s%s📊 %s%s\n", Bold, Cyan, T(i18n.MsgDashboardAssets), Reset)
		fmt.Printf("  %s%s\n", Cyan, "────────────────────────────────────────────────────────")
		fmt.Printf("  %-8s %-6s %-8s %-8s %s\n", T(i18n.MsgDashboardType), T(i18n.MsgDashboardIcon), T(i18n.MsgDashboardTotalCol), T(i18n.MsgDashboardAvailCol), T(i18n.MsgDashboardDesc))
		fmt.Printf("  %s\n", "────────────────────────────────────────────────────────")

		for _, item := range items {
			available := item.Online
			if available == 0 {
				available = item.Total
			}

			availableStr := fmt.Sprintf("%d", available)
			if available == 0 {
				availableStr = fmt.Sprintf("%s0%s", Red, Reset)
			}

			fmt.Printf("  %-8s %-6s %-8d %-8s %s\n",
				item.Name, item.Icon, item.Total, availableStr, item.Color)
		}

		fmt.Printf("  %s\n\n", "────────────────────────────────────────────────────────")

		// 打印快速操作提示
		fmt.Printf("  %s%s%s%s\n", Bold, Cyan, T(i18n.MsgDashboardQuickStart), Reset)
		fmt.Printf("    %s%s%s  mayfly-cli db list%s\n", Blue, T(i18n.MsgDashboardDb), Reset, Reset)
		fmt.Printf("    %s%s%s  mayfly-cli ssh list%s\n", Green, T(i18n.MsgDashboardMachine), Reset, Reset)
		fmt.Printf("    %s%s%s  mayfly-cli redis list%s\n", Purple, T(i18n.MsgDashboardRedis), Reset, Reset)
		fmt.Printf("    %s%s%s  mayfly-cli config%s\n\n", Yellow, T(i18n.MsgDashboardConfig), Reset, Reset)
	}
}

// PrintLoginPrompt 打印登录提示
func PrintLoginPrompt(server string) {
	fmt.Println()
	fmt.Printf("%s%s╔══════════════════════════════════════════════════════════╗%s\n", Yellow, Bold, Reset)
	fmt.Printf("%s%s║              %s                ║%s\n", Yellow, Bold, T(i18n.MsgLoginPromptTitle), Reset)
	fmt.Printf("%s%s╚══════════════════════════════════════════════════════════╝%s\n\n", Yellow, Bold, Reset)

	fmt.Printf("  %s%s%s %s\n\n", Bold, T(i18n.MsgLoginPromptServer), Reset, server)
	fmt.Printf("  %s%s%s\n", Bold, T(i18n.MsgLoginPromptQuick), Reset)
	fmt.Printf("    %s$ mayfly-cli config%s\n\n", Green, Reset)

	fmt.Printf("  %s%s%s\n", Bold, T(i18n.MsgLoginPromptManual), Reset)
	fmt.Printf("    %s$ mayfly-cli -s %s -t <token> db list%s\n\n", Green, server, Reset)
}
