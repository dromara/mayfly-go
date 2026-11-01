// Package backupenc 备份产物的落盘加密。
//
// 目标只有一个：**磁盘上的备份文件是密文**。备份文件里是整库明文数据，一旦落到
// 共享盘、对象存储或运维的下载目录，访问控制就只剩「谁能拿到文件」这一道；
// 落盘加密把「拿到文件」与「看懂文件」拆成两件事。
//
// 为什么不是「压缩后整段 AES-GCM」：备份可能有几十 GB，整段加密要求先把全部密文
// 读进内存才能校验 tag。这里用分块 GCM（STREAM 构造）：每块 64KB，nonce 由
// 8 字节随机前缀 + 4 字节块计数组成，块与块之间用 aad 绑定序号，因此
// 重排、拼接、截断都会在解密时被发现。
//
// 兼容性边界：文件头是 8 字节 magic。没有 magic 的文件按**旧明文备份**原样读取——
// 这不是「兼容旧代码」，而是「已经存在的备份数据必须还能恢复」；新产物一律加密。
package backupenc

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"

	"mayfly-go/internal/pkg/config"
	"mayfly-go/pkg/errorx"
)

// magic 文件头：让「密文」与「历史明文备份」在读取时可判定
var magic = []byte("MAYFLYDB1")

// chunkSize 单块明文大小。64KB 是「一次 Seal 的额外开销可忽略」与
// 「单块损坏最多丢 64KB」之间的折中
const chunkSize = 64 << 10

// noncePrefixLen 随机前缀长度；剩余 4 字节留给块计数，因此单文件最多 2^32 块（256TB）
const noncePrefixLen = 8

// ErrNotEncrypted 读取端：该流没有 magic（历史明文备份）
var ErrNotEncrypted = errors.New("not an encrypted backup")

// deriveKey 由平台 aes.key 派生备份专用子密钥。
//
// 派生而不是直接用 aes.key：同一把钥匙同时用于「实例口令字段加密」与「备份文件加密」，
// 任一处的使用模式出问题会牵连另一处；加域分隔串之后两把钥匙在密码学上无关。
// 同时挡住未配置 aes.key 的部署：宁可拒绝生成备份，也不落一份「看起来加密了」的密文。
func deriveKey() ([]byte, error) {
	if config.Conf == nil || config.Conf.Aes.Key == "" {
		return nil, ErrNoKey
	}
	return DeriveKey(config.Conf.Aes.Key), nil
}

// ErrNoKey 未配置平台密钥：拒绝写，也不接受「先写明文再补加密」
var ErrNoKey = errorx.NewBiz("backup encryption requires aes.key to be configured (refusing to write a backup without encryption)")

// DeriveKey 由平台密钥派生备份子密钥（纯函数：便于用例与将来的多密钥轮换直接引用）
func DeriveKey(platformKey string) []byte {
	sum := sha256.Sum256([]byte(platformKey + "|mayfly-db-backup|v1"))
	return sum[:]
}

// Writer 分块加密写入器。Close 必须调用，否则最后一块与文件尾不会落盘
type Writer struct {
	raw     io.Writer
	aead    cipher.AEAD
	prefix  [noncePrefixLen]byte
	counter uint32

	buf      []byte // 未满一块的暂存
	nonce    []byte
	writeErr error
}

// NewWriter 在 raw 上写 magic + nonce 前缀，返回分块加密写入器
func NewWriter(raw io.Writer) (*Writer, error) {
	key, err := deriveKey()
	if err != nil {
		return nil, err
	}
	return newWriterWithKey(raw, key)
}

// NewWriterWithKey 显式给出平台密钥的写入器：与 NewWriter 同一实现，
// 供集成测试在不覆写全局配置的前提下驱动真实产物
func NewWriterWithKey(raw io.Writer, platformKey string) (*Writer, error) {
	if platformKey == "" {
		return nil, ErrNoKey
	}
	return newWriterWithKey(raw, DeriveKey(platformKey))
}

func newWriterWithKey(raw io.Writer, key []byte) (*Writer, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	w := &Writer{raw: raw, aead: aead, buf: make([]byte, 0, chunkSize), nonce: make([]byte, aead.NonceSize())}
	if _, err := raw.Write(magic); err != nil {
		return nil, err
	}
	if _, err := io.ReadFull(rand.Reader, w.prefix[:]); err != nil {
		return nil, err
	}
	if _, err := raw.Write(w.prefix[:]); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *Writer) Write(p []byte) (int, error) {
	if w.writeErr != nil {
		return 0, w.writeErr
	}
	written := len(p)
	for len(p) > 0 {
		space := chunkSize - len(w.buf)
		take := len(p)
		if take > space {
			take = space
		}
		w.buf = append(w.buf, p[:take]...)
		p = p[take:]
		if len(w.buf) == chunkSize {
			if err := w.flushChunk(); err != nil {
				w.writeErr = err
				return 0, err
			}
		}
	}
	return written, nil
}

// flushChunk 加密并写出当前缓冲
func (w *Writer) flushChunk() error {
	w.setNonce(w.counter)
	// aad 绑定块序号：块被重排或删除时解密端立刻失败
	var aad [4]byte
	binary.BigEndian.PutUint32(aad[:], w.counter)

	sealed := w.aead.Seal(nil, w.nonce, w.buf, aad[:])
	var head [4]byte
	binary.BigEndian.PutUint32(head[:], uint32(len(w.buf)))
	if _, err := w.raw.Write(head[:]); err != nil {
		return err
	}
	if _, err := w.raw.Write(sealed); err != nil {
		return err
	}
	w.counter++
	w.buf = w.buf[:0]
	return nil
}

// Close 写出尾块（含 0 长度终止块）并结束
func (w *Writer) Close() error {
	if w.writeErr != nil {
		return w.writeErr
	}
	if len(w.buf) > 0 {
		if err := w.flushChunk(); err != nil {
			w.writeErr = err
			return err
		}
	}
	// 显式终止块：不写它也能读完，但「截断」与「正常结束」就无法区分
	if err := w.flushChunk(); err != nil {
		w.writeErr = err
		return err
	}
	return nil
}

func (w *Writer) setNonce(counter uint32) {
	copy(w.nonce, w.prefix[:])
	binary.BigEndian.PutUint32(w.nonce[noncePrefixLen:], counter)
}

// Reader 分块解密读取器
type Reader struct {
	raw       io.Reader
	aead      cipher.AEAD
	prefix    [noncePrefixLen]byte
	counter   uint32
	remaining []byte // 当前已解密但未读走的数据
	done      bool
}

// WrapReader 判定 magic：命中返回解密读取器，未命中把**原始字节原样**交回。
//
// 用 bufio.Peek 而不是先 Read 8 字节再回退：恢复侧的源流来自文件存储，可能是不可 Seek
// 的对象存储流；直接读走 8 字节再当明文解析，会把历史备份的第一个 token 吃掉，
// 表现为「恢复出来的数据少了开头一段」这种最难发现的损坏。
func WrapReader(raw io.Reader) (io.Reader, error) {
	buffered := bufio.NewReaderSize(raw, chunkSize+64)
	head, err := buffered.Peek(len(magic))
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			// 短于 magic 的文件不可能是密文：按明文交回，由下游解析器报真实错误
			return buffered, nil
		}
		return nil, err
	}
	if !bytes.Equal(head, magic) {
		return buffered, nil
	}
	if _, err := buffered.Discard(len(magic)); err != nil {
		return nil, err
	}
	key, err := deriveKey()
	if err != nil {
		return nil, err
	}
	return wrapReaderWithKey(buffered, key)
}

// WrapReaderWithKey 显式给出平台密钥的读取器（测试与密钥轮换用）
func WrapReaderWithKey(raw io.Reader, platformKey string) (io.Reader, error) {
	buffered := bufio.NewReaderSize(raw, chunkSize+64)
	head, err := buffered.Peek(len(magic))
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return buffered, nil
		}
		return nil, err
	}
	if !bytes.Equal(head, magic) {
		return buffered, nil
	}
	if _, err := buffered.Discard(len(magic)); err != nil {
		return nil, err
	}
	if platformKey == "" {
		return nil, ErrNoKey
	}
	return wrapReaderWithKey(buffered, DeriveKey(platformKey))
}

func wrapReaderWithKey(buffered *bufio.Reader, key []byte) (io.Reader, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	r := &Reader{raw: buffered, aead: aead}
	if _, err := io.ReadFull(buffered, r.prefix[:]); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Reader) Read(p []byte) (int, error) {
	for len(r.remaining) == 0 {
		if r.done {
			return 0, io.EOF
		}
		if err := r.readChunk(); err != nil {
			return 0, err
		}
	}
	n := copy(p, r.remaining)
	r.remaining = r.remaining[n:]
	return n, nil
}

// readChunk 读入并解密一块
func (r *Reader) readChunk() error {
	var head [4]byte
	if _, err := io.ReadFull(r.raw, head[:]); err != nil {
		return err
	}
	size := binary.BigEndian.Uint32(head[:])
	if size > chunkSize {
		return errorx.NewBiz("corrupted backup: chunk size exceeds limit")
	}
	ciphertext := make([]byte, size+uint32(r.aead.Overhead()))
	if _, err := io.ReadFull(r.raw, ciphertext); err != nil {
		return err
	}
	nonce := make([]byte, r.aead.NonceSize())
	copy(nonce, r.prefix[:])
	binary.BigEndian.PutUint32(nonce[noncePrefixLen:], r.counter)

	var aad [4]byte
	binary.BigEndian.PutUint32(aad[:], r.counter)
	plain, err := r.aead.Open(nil, nonce, ciphertext, aad[:])
	if err != nil {
		return errorx.NewBiz("failed to decrypt the backup file: it is corrupted or the encryption key has changed")
	}
	r.counter++
	r.remaining = plain
	// 终止块（size==0）也要走完解密：它的 tag 是最后一个可校验点，
	// 提前 return 会让「文件尾部被改写/截断到块边界」静默通过校验
	if size == 0 {
		r.done = true
	}
	return nil
}
