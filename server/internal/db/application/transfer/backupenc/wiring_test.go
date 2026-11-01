package backupenc

// 加密接线的守卫：加密包本身测得再细，只要有一个写入/读取入口忘了包一层，
// 备份就会以明文落盘（写入侧）或把密文当 SQL 灌进目标库（读取侧），
// 两者都不报错、都不可从日志看出来。

import (
	"os"
	"strings"
	"testing"
)

func TestBackupEncryptionWiring(t *testing.T) {
	// 写入侧：加密器必须在压缩器之前套上（先压缩后加密，磁盘上才是密文）
	writeSrc, err := os.ReadFile("../db_transfer.go")
	if err != nil {
		t.Fatalf("read transfer writer: %v", err)
	}
	writeBody := string(writeSrc)
	newWriterPos := strings.Index(writeBody, "backupenc.NewWriter(")
	zipPos := strings.Index(writeBody, "zip.NewWriter(")
	if newWriterPos == -1 {
		t.Fatal("备份写盘必须包一层落盘加密")
	}
	if zipPos != -1 && newWriterPos > zipPos {
		t.Fatal("加密必须包在压缩之前：反过来会把压缩后的明文写在密文外层")
	}
	if !strings.Contains(writeBody, "encWriter.Close()") {
		t.Fatal("加密器必须显式 Close：尾块不写等于文件截断")
	}

	// 读取侧：下载与恢复两条路径都要透明解密
	readSrc, err := os.ReadFile("../../../api/db_transfer.go")
	if err != nil {
		t.Fatalf("read transfer api: %v", err)
	}
	readBody := string(readSrc)
	if count := strings.Count(readBody, "backupenc.WrapReader("); count < 2 {
		t.Fatalf("下载与恢复必须各自解密，当前只找到 %d 处 WrapReader", count)
	}
	for _, handler := range []string{"func (d *DbTransferTask) FileContent(", "func (d *DbTransferTask) FileRun("} {
		start := strings.Index(readBody, handler)
		if start == -1 {
			t.Fatalf("handler %q 应存在", handler)
		}
		fn := readBody[start:]
		if end := strings.Index(fn[1:], "\nfunc "); end != -1 {
			fn = fn[:end+1]
		}
		if !strings.Contains(fn, "backupenc.WrapReader") {
			t.Fatalf("%s 必须解密后再使用文件内容", handler)
		}
	}
}
