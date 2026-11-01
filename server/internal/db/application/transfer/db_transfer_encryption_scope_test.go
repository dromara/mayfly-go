package transfer

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEncryptedFileWriterLivesInsideWorker 落盘加密器的创建与收尾必须在写数据的协程内。
//
// 历史缺陷：encWriter 在外层函数创建并 defer Close，而实际写数据在 gox.Go 的协程里。
// 外层函数返回时先执行 Close（写出 0 长度终止块），协程随后写入的数据排在终止块之后，
// 恢复端读到 size==0 即 EOF —— 备份文件恢复出来是空库，任务/文件状态却标成功。
func TestEncryptedFileWriterLivesInsideWorker(t *testing.T) {
	content, err := os.ReadFile("db_transfer.go")
	require.NoError(t, err)
	source := string(content)

	goIdx := strings.Index(source, "gox.Go(func() {")
	encIdx := strings.Index(source, "backupenc.NewWriter(writer)")
	require.Greater(t, goIdx, -1, "未找到写文件协程")
	require.Greater(t, encIdx, -1, "未找到加密器创建")
	require.Greater(t, encIdx, goIdx, "加密器必须在写数据的协程内创建（外层作用域会在数据写入前就 Close）")

	// 协程启动之前不得把 writer 换成加密器：那正是"外层 Close 先于数据写入"的成因
	require.NotContains(t, source[:goIdx], "writer = encWriter", "协程外不得替换 writer")

	// 加密尾块失败必须并入协程 err，不能写进无人读取的局部变量
	require.Contains(t, source, "failed to finalize backup encryption",
		"加密器 Close 错误必须并入任务错误，否则损坏的备份会被标成功")
}

// TestEncryptedFinalizeRunsBeforeFileClose 收尾顺序必须是「zip → 加密终止块 → 关闭保存文件」。
//
// Go 的 defer 是 LIFO：后注册先执行。所以代码里必须是
// closeFunc 的 defer 在前、encWriter.Close 的 defer 在后、zipWriter.Close 的 defer 最后。
// 顺序反了会出现「文件先被关闭保存，加密终止块写不进去」，恢复端读不到结束标记，
// 备份不可恢复（且错误只在 close 失败时暴露，极易被当成成功）。
func TestEncryptedFinalizeRunsBeforeFileClose(t *testing.T) {
	content, err := os.ReadFile("db_transfer.go")
	require.NoError(t, err)
	source := string(content)

	closeFuncIdx := strings.Index(source, "closeFunc(&err)")
	encCloseIdx := strings.Index(source, "encWriter.Close()")
	zipCloseIdx := strings.Index(source, "zipWriter.Close()")
	require.Greater(t, closeFuncIdx, -1)
	require.Greater(t, encCloseIdx, -1)
	require.Greater(t, zipCloseIdx, -1)

	require.Less(t, closeFuncIdx, encCloseIdx,
		"LIFO：加密器 Close 必须比 closeFunc 更晚注册（才会更早执行）")
	require.Less(t, encCloseIdx, zipCloseIdx,
		"zip 收尾必须最先执行（最晚注册），否则压缩目录会丢")
}
