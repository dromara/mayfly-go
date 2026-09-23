//go:build it

package sync

import (
	"os"
	"testing"

	"mayfly-go/internal/db/ititest/scratchclean"
)

// TestMain 使用进程独占临时库，测试结束无条件清理，清理失败使测试失败。
func TestMain(m *testing.M) {
	os.Exit(scratchclean.Run(m.Run))
}
