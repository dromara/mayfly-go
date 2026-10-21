package tools

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCrossLayerExtraKeysHaveSingleSource 跨层 Extra 键必须只用归属层常量，禁止再出现裸字面量。
//
// resumeInfo / interrupt / actionId / turnId / toolStatus 这些键由 agent/tools 写入、被 tools/application/
// session 多层读取。此前每层各以字面量书写（resumeInfo 曾散在两包 6 处），任一处改名其余静默错配——
// 表现为「恢复决策读不到、中断卡片历史不回填」这类无报错失效。真源已收进 session.ExtraKey* 与
// protocol.ExtraKeyInterrupt，这里扫整棵 ai 树的**代码**（跳过注释、常量定义文件与测试）断言不再有裸字面量。
//
// 与协议字符串守卫（protocol_consts_test）同源思路；锚定的是 map 索引 / 取串函数的字面量实参这一
// 代码形态，而非散文措辞，所以改注释不会误红
func TestCrossLayerExtraKeysHaveSingleSource(t *testing.T) {
	// 非法形态：键字面量被用作 map 索引 [ 或 取串/写值函数（GetStr/Unmarshal/Set/…）的实参
	illegal := regexp.MustCompile(`(?:\[|GetStr\(|Unmarshal\(|Set\(|,\s*)"(resumeInfo|interrupt|actionId|turnId|toolStatus)"`)
	sourceFiles := map[string]bool{"extra_keys.go": true, "interrupt_info.go": true} // 常量定义处

	var offenders []string
	err := filepath.Walk("..", func(path string, info fs.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return walkErr
		}
		if strings.HasSuffix(path, "_test.go") || sourceFiles[filepath.Base(path)] {
			return nil
		}
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(src), "\n") {
			t := strings.TrimSpace(line)
			if strings.HasPrefix(t, "//") || strings.HasPrefix(t, "*") {
				continue // 注释里描述存储形状（如 {"interrupt": ...}）不算代码
			}
			if illegal.MatchString(line) {
				offenders = append(offenders, fmt.Sprintf("%s:%d :: %s", path, i+1, t))
			}
		}
		return nil
	})
	require.NoError(t, err)
	require.Empty(t, offenders, "跨层 Extra 键必须引用 session.ExtraKey* / protocol.ExtraKeyInterrupt 常量")
}
