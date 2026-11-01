package backupenc

// 备份落盘加密的三条硬约束：
//   1. 密文→明文可还原（跨块边界、含恰好整块与空文件）；
//   2. 历史明文备份必须原样读出（少一个字节都等于数据损坏）；
//   3. 密文被篡改/截断必须报错，而不是给出「看起来正常」的半份数据。

import (
	"bytes"
	"io"
	"testing"

	"mayfly-go/internal/pkg/config"
	"mayfly-go/pkg/global"
)

func withAesKey(t *testing.T) {
	t.Helper()
	previous := config.Conf
	config.Conf = &config.Config{Aes: config.Aes{Key: "0123456789abcdef0123456789abcdef"}}
	t.Cleanup(func() { config.Conf = previous })
}

func encryptTo(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := NewWriter(&buf)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	// 分几次写，覆盖跨块与不足一块的尾块
	for _, part := range [][]byte{payload[:min(len(payload), 1000)], payload[min(len(payload), 1000):]} {
		if _, err := w.Write(part); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return buf.Bytes()
}

func TestBackupEncryptRoundTrip(t *testing.T) {
	withAesKey(t)
	payload := bytes.Repeat([]byte("mayfly-backup-payload;"), 20000) // ~420KB，跨多块
	ciphertext := encryptTo(t, payload)

	if bytes.Contains(ciphertext, []byte("mayfly-backup-payload")) {
		t.Fatal("ciphertext must not contain the plaintext payload")
	}
	if !bytes.HasPrefix(ciphertext, magic) {
		t.Fatal("ciphertext must start with the magic header")
	}

	reader, err := WrapReader(bytes.NewReader(ciphertext))
	if err != nil {
		t.Fatalf("WrapReader: %v", err)
	}
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("round trip mismatch: got %d bytes, want %d", len(got), len(payload))
	}
}

func TestBackupEncryptEmptyPayload(t *testing.T) {
	withAesKey(t)
	ciphertext := encryptTo(t, nil)
	reader, err := WrapReader(bytes.NewReader(ciphertext))
	if err != nil {
		t.Fatalf("WrapReader: %v", err)
	}
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("empty payload must stay empty, got %d bytes", len(got))
	}
}

// 历史明文备份：没有 magic，必须原样读出（含首字节），不能吃掉开头
func TestBackupEncryptPassesThroughLegacyPlaintext(t *testing.T) {
	withAesKey(t)
	plain := []byte("-- legacy plain backup\nCREATE TABLE t (id int);\n")
	reader, err := WrapReader(bytes.NewReader(plain))
	if err != nil {
		t.Fatalf("WrapReader: %v", err)
	}
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("legacy plaintext must be returned byte-for-byte, got %q", string(got))
	}

	// 短于 magic 的碎片同样原样返回（由下游解析器报真实错误）
	short := []byte("ab")
	reader, err = WrapReader(bytes.NewReader(short))
	if err != nil {
		t.Fatalf("WrapReader: %v", err)
	}
	got, _ = io.ReadAll(reader)
	if !bytes.Equal(got, short) {
		t.Fatalf("short input must be returned as-is, got %q", string(got))
	}
}

func TestBackupEncryptDetectsTamper(t *testing.T) {
	withAesKey(t)
	ciphertext := encryptTo(t, bytes.Repeat([]byte("x"), 70000))

	tampered := append([]byte(nil), ciphertext...)
	tampered[len(tampered)-1] ^= 0xff // 破坏最后一块的 tag
	reader, err := WrapReader(bytes.NewReader(tampered))
	if err != nil {
		t.Fatalf("WrapReader: %v", err)
	}
	if _, err := io.ReadAll(reader); err == nil {
		t.Fatal("a tampered ciphertext must fail to decrypt")
	}

	// 截断成只剩头 + 半块：同样必须报错而不是静默返回部分数据
	truncated := ciphertext[:len(ciphertext)/2]
	reader, err = WrapReader(bytes.NewReader(truncated))
	if err != nil {
		t.Fatalf("WrapReader: %v", err)
	}
	if _, err := io.ReadAll(reader); err == nil {
		t.Fatal("a truncated ciphertext must fail instead of returning partial data")
	}
}

// 未配置 aes.key 时拒绝写：宁可不产出备份，也不落一份「看起来加密了」的明文
func TestBackupEncryptRefusesWithoutKey(t *testing.T) {
	previous := config.Conf
	config.Conf = nil
	t.Cleanup(func() { config.Conf = previous })
	_ = global.Db

	var buf bytes.Buffer
	if _, err := NewWriter(&buf); err == nil {
		t.Fatal("NewWriter must fail when aes.key is not configured")
	}
	if buf.Len() != 0 {
		t.Fatalf("nothing must be written on refusal, got %d bytes", buf.Len())
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestCloseBeforeWriteLosesAllData 复现「加密器在数据写入前就被 Close」的后果。
//
// 这正是 transfer2File 的历史缺陷形态：加密器在外层函数 defer Close，而写数据在 gox.Go
// 协程里；外层函数返回时先写出 0 长度终止块，随后写入的数据排在终止块之后（或被拒），
// 恢复端读到 size==0 立刻视为 EOF —— 备份"能读出来"，但内容是空的。
//
// 本用例锁的是「收尾顺序必须晚于全部数据写入」这条不变量；transfer 侧的源码守卫
// （TestEncryptedFileWriterLivesInsideWorker）锁的是"加密器在协程内创建"。
func TestCloseBeforeWriteLosesAllData(t *testing.T) {
	withAesKey(t)

	var buf bytes.Buffer
	w, err := NewWriter(&buf)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// 终止块之后再写：等价于协程晚于外层 defer 才开始写
	_, _ = w.Write([]byte("late payload written after the terminator"))

	reader, err := WrapReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("WrapReader: %v", err)
	}
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("terminator-first ordering must recover empty payload, got %d bytes", len(got))
	}
}
