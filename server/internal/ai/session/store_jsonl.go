package session

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/filex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// numLockShards is the fixed number of mutexes used to serialize
	// per-session access. Using a sharded array instead of a map keeps
	// memory bounded regardless of how many sessions are created over
	// the lifetime of the process — important for a long-running daemon.
	numLockShards = 64

	// maxLineSize is the maximum size of a single JSON line in a .jsonl
	// file. Tool results (read_file, web search, etc.) can be large, so
	// we set a generous limit. The scanner starts at 64 KB and grows
	// only as needed up to this cap.
	maxLineSize = 10 * 1024 * 1024 // 10 MB
)

// JSONLStore implements Store using append-only JSONL files.
//
// Each session is stored as two files:
//
//	{sanitized_key}.jsonl      — one JSON-encoded message per line, append-only
//	{sanitized_key}.meta.json  — session metadata (summary, logical truncation offset)
//
// Messages are never physically deleted from the JSONL file. Instead,
// TruncateHistory records a "skip" offset in the metadata file and
// GetHistory ignores lines before that offset. This keeps all writes
// append-only, which is both fast and crash-safe.
type JSONLStore struct {
	dir   string
	locks [numLockShards]sync.Mutex
}

var _ Store = (*JSONLStore)(nil)

// NewJSONLStore creates a new JSONL-backed store rooted at dir.
func NewStoreJSONL(dir string) (*JSONLStore, error) {
	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		return nil, fmt.Errorf("jsonl: create directory: %w", err)
	}
	return &JSONLStore{dir: dir}, nil
}

// sessionLock returns a mutex for the given session key.
// Keys are mapped to a fixed pool of shards via FNV hash, so
// memory usage is O(1) regardless of total session count.
func (s *JSONLStore) sessionLock(key string) *sync.Mutex {
	h := fnv.New32a()
	h.Write([]byte(key))
	return &s.locks[h.Sum32()%numLockShards]
}

func (s *JSONLStore) jsonlPath(key string) string {
	return filepath.Join(s.dir, sanitizeKey(key)+".jsonl")
}

func (s *JSONLStore) metaPath(key string) string {
	return filepath.Join(s.dir, sanitizeKey(key)+".meta.json")
}

// sanitizeKey converts a session key to a safe filename component.
// Mirrors pkg/session.sanitizeFilename so that migration paths match.
// Replaces ':' with '_' (session key separator) and '/' and '\' with '_'
// so composite IDs (e.g. Telegram forum "chatID/threadID", Slack "channel/thread_ts")
// do not create subdirectories or break on Windows.
func sanitizeKey(key string) string {
	s := strings.ReplaceAll(key, ":", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	return s
}

// readMeta loads the metadata file for a session.
// Returns a zero-value sessionMeta if the file does not exist.
func (s *JSONLStore) readMeta(key string) (SessionMeta, error) {
	data, err := os.ReadFile(s.metaPath(key))
	if os.IsNotExist(err) {
		return SessionMeta{Key: key}, nil
	}
	if err != nil {
		return SessionMeta{}, fmt.Errorf("jsonl: read meta: %w", err)
	}
	var meta SessionMeta
	err = json.Unmarshal(data, &meta)
	if err != nil {
		return SessionMeta{}, fmt.Errorf("jsonl: decode meta: %w", err)
	}
	return meta, nil
}

// writeMeta atomically writes the metadata file using the project's
// standard WriteFileAtomic (temp + fsync + rename).
func (s *JSONLStore) writeMeta(key string, meta SessionMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("jsonl: encode meta: %w", err)
	}
	return filex.WriteFileAtomic(s.metaPath(key), data, 0o644)
}

// readMessages reads valid JSON lines from a .jsonl file, skipping
// the first `skip` lines without unmarshaling them. This avoids the
// cost of json.Unmarshal on logically truncated messages.
// Malformed trailing lines (e.g. from a crash) are silently skipped.
func readMessages(path string, skip int) ([]*Message, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return []*Message{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("jsonl: open jsonl: %w", err)
	}
	defer f.Close()

	var msgs []*Message
	scanner := bufio.NewScanner(f)
	// Allow large lines for tool results (read_file, web search, etc.).
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineSize)

	lineNum := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		lineNum++
		if lineNum <= skip {
			continue
		}
		var msg Message
		if err := json.Unmarshal(line, &msg); err != nil {
			// Corrupt line — likely a partial write from a crash.
			// Log so operators know data was skipped, but don't
			// fail the entire read; this is the standard JSONL
			// recovery pattern.
			log.Printf("jsonl: skipping corrupt line %d in %s: %v",
				lineNum, filepath.Base(path), err)
			continue
		}
		msgs = append(msgs, &msg)
	}
	if scanner.Err() != nil {
		return nil, fmt.Errorf("jsonl: scan jsonl: %w", scanner.Err())
	}

	if msgs == nil {
		msgs = []*Message{}
	}
	return msgs, nil
}

// countLines counts the total number of non-empty lines in a .jsonl file.
// Used by TruncateHistory to reconcile a stale meta.Count without
// the overhead of unmarshaling every message.
func countLines(path string) (int, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("jsonl: open jsonl: %w", err)
	}
	defer f.Close()

	n := 0
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineSize)
	for scanner.Scan() {
		if len(scanner.Bytes()) > 0 {
			n++
		}
	}
	return n, scanner.Err()
}

func (s *JSONLStore) AppendMsgs(
	_ context.Context, sessionKey string, msgs ...*Message) error {
	return s.addMsg(sessionKey, msgs...)
}

// addMsg appends one or more messages to the session's JSONL file in a single write operation.
// Using batch writes reduces disk I/O overhead compared to writing each message individually.
// Note: This method only appends messages to storage, it does NOT update metadata.
// Metadata updates should be handled by the Manager layer.
func (s *JSONLStore) addMsg(sessionKey string, msgs ...*Message) error {
	l := s.sessionLock(sessionKey)
	l.Lock()
	defer l.Unlock()

	if len(msgs) == 0 {
		return nil
	}

	// Marshal all messages into JSON lines
	var buf bytes.Buffer
	for i, msg := range msgs {
		line, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("jsonl: marshal message %d: %w", i, err)
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}

	// Open file for appending
	f, err := os.OpenFile(
		s.jsonlPath(sessionKey),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0o644,
	)
	if err != nil {
		return fmt.Errorf("jsonl: open jsonl for append: %w", err)
	}

	// Write all messages in one operation
	_, writeErr := f.Write(buf.Bytes())
	if writeErr != nil {
		f.Close()
		return fmt.Errorf("jsonl: append messages: %w", writeErr)
	}

	// Flush to physical storage before closing. This matches the
	// durability guarantee of writeMeta and rewriteJSONL (which use
	// WriteFileAtomic with fsync). Without Sync, a power loss could
	// leave the append in the kernel page cache only — lost on reboot.
	if syncErr := f.Sync(); syncErr != nil {
		f.Close()
		return fmt.Errorf("jsonl: sync jsonl: %w", syncErr)
	}

	if closeErr := f.Close(); closeErr != nil {
		return fmt.Errorf("jsonl: close jsonl: %w", closeErr)
	}

	return nil
}

func (s *JSONLStore) GetHistory(
	_ context.Context, sessionKey string, limit int,
) ([]*Message, error) {
	l := s.sessionLock(sessionKey)
	l.Lock()
	defer l.Unlock()

	// 读取元数据获取 Skip 偏移量，跳过已摘要的历史消息，减少 I/O
	meta, err := s.readMeta(sessionKey)
	if err != nil {
		return nil, err
	}

	msgs, err := readMessages(s.jsonlPath(sessionKey), meta.Skip)
	if err != nil {
		return nil, err
	}

	logx.DebugfContext(context.Background(), "[Store] GetHistory: loaded %d messages, skip=%d, limit=%d", len(msgs), meta.Skip, limit)

	// Apply limit if specified and positive
	if limit > 0 && len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
		logx.DebugfContext(context.Background(), "[Store] GetHistory: after limit, returning %d messages", len(msgs))
	}

	return msgs, nil
}

// CountUnsummarized 未摘要消息条数：JSONL 后端本就是「读 skip 之后的全部消息」，
// 与 GetHistory 同一取数口径（两个后端必须同一语义，见 Store 接口注释）
func (s *JSONLStore) CountUnsummarized(_ context.Context, sessionKey string) (int, error) {
	l := s.sessionLock(sessionKey)
	l.Lock()
	defer l.Unlock()

	meta, err := s.readMeta(sessionKey)
	if err != nil {
		return 0, err
	}
	msgs, err := readMessages(s.jsonlPath(sessionKey), meta.Skip)
	if err != nil {
		return 0, err
	}
	return len(msgs), nil
}

// GetMessage 根据查询条件获取单条消息
// GetMessage 显式失败：JSONL 是单文件历史，而 MessageQuery 不含 sessionKey，
// 无法定位到具体会话。旧实现用空 key 读历史后返回空切片——调用方把「不支持」
// 当成「查不到」，审批/补全的 resume 信息落不了库却查不出原因。
// 生产走 DB 后端（application/application.go 装配），此路径只服务兜底后端
func (s *JSONLStore) GetMessage(_ context.Context, _ *MessageQuery) ([]*Message, error) {
	return nil, fmt.Errorf("%w: JSONLStore.GetMessage (no session key in MessageQuery)", ErrStoreUnsupported)
}

// UpdateMessage 更新单条消息
func (s *JSONLStore) UpdateMessage(ctx context.Context, msg *Message) error {
	if msg == nil {
		return nil
	}
	// JSONL 是追加写的，需要读取所有消息，找到对应的消息，更新后重写整个文件
	// 这里需要根据 ActionId 查找对应的消息
	// 由于 JSONL 不支持直接更新，需要完整的 sessionKey 才能操作
	// 这里返回不支持的错误
	return fmt.Errorf("%w: JSONLStore.UpdateMessage", ErrStoreUnsupported)
}

// ClearHistory 清空会话历史消息（保留元数据）
func (s *JSONLStore) ClearHistory(ctx context.Context, sessionKey string) error {
	l := s.sessionLock(sessionKey)
	l.Lock()
	defer l.Unlock()

	// 清空 JSONL 文件
	jsonlPath := s.jsonlPath(sessionKey)
	err := filex.WriteFileAtomic(jsonlPath, []byte{}, 0o644)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("jsonl: clear history: %w", err)
	}

	// 更新元数据
	meta, err := s.readMeta(sessionKey)
	if err != nil {
		return err
	}

	meta.Count = 0
	meta.Skip = 0
	meta.UpdatedAt = time.Now()

	return s.writeMeta(sessionKey, meta)
}

func (s *JSONLStore) GetSummary(
	_ context.Context, sessionKey string,
) (string, error) {
	l := s.sessionLock(sessionKey)
	l.Lock()
	defer l.Unlock()

	meta, err := s.readMeta(sessionKey)
	if err != nil {
		return "", err
	}
	return meta.Summary, nil
}

func (s *JSONLStore) UpdateSummary(
	_ context.Context, sessionKey, summary string,
) error {
	l := s.sessionLock(sessionKey)
	l.Lock()
	defer l.Unlock()

	meta, err := s.readMeta(sessionKey)
	if err != nil {
		return err
	}
	now := time.Now()
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = now
	}
	meta.Summary = summary
	meta.UpdatedAt = now

	return s.writeMeta(sessionKey, meta)
}

func (s *JSONLStore) TruncateHistory(
	_ context.Context, sessionKey string, keepLast int,
) error {
	l := s.sessionLock(sessionKey)
	l.Lock()
	defer l.Unlock()

	meta, err := s.readMeta(sessionKey)
	if err != nil {
		return err
	}

	// Always reconcile meta.Count with the actual line count on disk.
	// A crash between the JSONL append and the meta update in addMsg
	// leaves meta.Count stale (e.g. file has 101 lines but meta says
	// 100). Counting lines is cheap — no unmarshal, just a scan — and
	// TruncateHistory is not a hot path, so always re-count.
	n, countErr := countLines(s.jsonlPath(sessionKey))
	if countErr != nil {
		return countErr
	}
	meta.Count = n

	if keepLast <= 0 {
		meta.Skip = meta.Count
	} else {
		effective := meta.Count - meta.Skip
		if keepLast < effective {
			meta.Skip = meta.Count - keepLast
		}
	}
	meta.UpdatedAt = time.Now()

	return s.writeMeta(sessionKey, meta)
}

func (s *JSONLStore) SetHistory(
	_ context.Context,
	sessionKey string,
	history []*Message,
) error {
	l := s.sessionLock(sessionKey)
	l.Lock()
	defer l.Unlock()

	meta, err := s.readMeta(sessionKey)
	if err != nil {
		return err
	}
	now := time.Now()
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = now
	}
	meta.Skip = 0
	meta.Count = len(history)
	meta.UpdatedAt = now

	// Write meta BEFORE rewriting the JSONL file. If we crash between
	// the two writes, meta has Skip=0 and the old file is still intact,
	// so GetHistory reads from line 1 — returning "too many" messages
	// rather than losing data. The next SetHistory call corrects this.
	err = s.writeMeta(sessionKey, meta)
	if err != nil {
		return err
	}

	return s.rewriteJSONL(sessionKey, history)
}

// Compact physically rewrites the JSONL file, dropping all logically
// skipped lines. This reclaims disk space that accumulates after
// repeated TruncateHistory calls.
//
// It is safe to call at any time; if there is nothing to compact
// (skip == 0) the method returns immediately.
func (s *JSONLStore) Compact(
	_ context.Context, sessionKey string,
) error {
	l := s.sessionLock(sessionKey)
	l.Lock()
	defer l.Unlock()

	meta, err := s.readMeta(sessionKey)
	if err != nil {
		return err
	}
	if meta.Skip == 0 {
		return nil
	}

	// Read only the active messages, skipping truncated lines
	// without unmarshaling them.
	active, err := readMessages(s.jsonlPath(sessionKey), meta.Skip)
	if err != nil {
		return err
	}

	// Write meta BEFORE rewriting the JSONL file. If the process
	// crashes between the two writes, meta has Skip=0 and the old
	// (uncompacted) file is still intact, so GetHistory reads from
	// line 1 — returning previously-truncated messages rather than
	// losing data. The next Compact or TruncateHistory corrects this.
	meta.Skip = 0
	meta.Count = len(active)
	meta.UpdatedAt = time.Now()

	err = s.writeMeta(sessionKey, meta)
	if err != nil {
		return err
	}

	return s.rewriteJSONL(sessionKey, active)
}

// rewriteJSONL atomically replaces the JSONL file with the given messages
// using the project's standard WriteFileAtomic (temp + fsync + rename).
func (s *JSONLStore) rewriteJSONL(
	sessionKey string, msgs []*Message,
) error {
	var buf bytes.Buffer
	for i, msg := range msgs {
		line, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("jsonl: marshal message %d: %w", i, err)
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	return filex.WriteFileAtomic(s.jsonlPath(sessionKey), buf.Bytes(), 0o644)
}

func (s *JSONLStore) GetSession(
	_ context.Context, sessionKey string,
) (*SessionMeta, error) {
	l := s.sessionLock(sessionKey)
	l.Lock()
	defer l.Unlock()

	meta, err := s.readMeta(sessionKey)
	if err != nil {
		return nil, err
	}

	// If the meta file doesn't exist, check if the JSONL file exists
	// to provide a minimal response with just the key
	if meta.Key == "" || meta.CreatedAt.IsZero() {
		// Check if JSONL file exists
		jsonlPath := s.jsonlPath(sessionKey)
		if _, err := os.Stat(jsonlPath); err == nil {
			// JSONL exists but meta doesn't - reconstruct basic meta
			count, countErr := countLines(jsonlPath)
			if countErr != nil {
				return nil, countErr
			}
			meta.Key = sessionKey
			meta.Count = count
			meta.CreatedAt = time.Now()
			meta.UpdatedAt = time.Now()
		} else {
			// Neither file exists - return zero value with key set
			return &SessionMeta{Key: sessionKey}, nil
		}
	}

	// Return a copy to prevent race conditions
	metaCopy := meta
	return &metaCopy, nil
}

func (s *JSONLStore) SaveSession(_ context.Context, meta *SessionMeta) error {
	l := s.sessionLock(meta.Key)
	l.Lock()
	defer l.Unlock()

	_, err := s.readMeta(meta.Key)
	if err != nil {
		return err
	}

	return s.writeMeta(meta.Key, *meta)
}

// DeleteHistory 删除会话历史消息（清空 JSONL 文件）
func (s *JSONLStore) DeleteHistory(ctx context.Context, sessionKey string) error {
	l := s.sessionLock(sessionKey)
	l.Lock()
	defer l.Unlock()

	// 清空 JSONL 文件
	jsonlPath := s.jsonlPath(sessionKey)
	err := filex.WriteFileAtomic(jsonlPath, []byte{}, 0o644)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("jsonl: truncate jsonl file: %w", err)
	}

	// 更新元数据
	meta, err := s.readMeta(sessionKey)
	if err != nil {
		return err
	}

	meta.Count = 0
	meta.Skip = 0
	meta.UpdatedAt = time.Now()

	return s.writeMeta(sessionKey, meta)
}

func (s *JSONLStore) ListSessions(
	_ context.Context,
) ([]*SessionMeta, error) {
	// List all .jsonl files in the directory
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("jsonl: read directory: %w", err)
	}

	var sessions []*SessionMeta
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .jsonl files (not .meta.json files)
		name := entry.Name()
		if !strings.HasSuffix(name, ".jsonl") {
			continue
		}

		// Extract the session key from the filename
		sessionKey := strings.TrimSuffix(name, ".jsonl")
		sessionKey = strings.ReplaceAll(sessionKey, "_", ":")

		// Load the metadata
		meta, err := s.readMeta(sessionKey)
		if err != nil {
			log.Printf("jsonl: failed to read meta for session %s: %v", sessionKey, err)
			continue
		}

		// Ensure the key is set correctly
		if meta.Key == "" {
			meta.Key = sessionKey
		}

		sessions = append(sessions, &meta)
	}

	return sessions, nil
}

func (s *JSONLStore) ListMetas(
	_ context.Context,
) ([]*SessionMeta, error) {
	// List all .jsonl files in the directory
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("jsonl: read directory: %w", err)
	}

	var sessions []*SessionMeta
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .jsonl files (not .meta.json files)
		name := entry.Name()
		if !strings.HasSuffix(name, ".jsonl") {
			continue
		}

		// Extract session key from filename (replace ":" with "_")
		sessionKey := strings.TrimSuffix(name, ".jsonl")
		sessionKey = strings.ReplaceAll(sessionKey, "_", ":")

		// Load metadata for each session
		meta, err := s.GetMeta(context.Background(), sessionKey)
		if err != nil {
			// Skip failed sessions but log warning
			continue
		}

		sessions = append(sessions, meta)
	}

	return sessions, nil
}

func (s *JSONLStore) RemoveSession(
	_ context.Context, sessionKey string,
) error {
	l := s.sessionLock(sessionKey)
	l.Lock()
	defer l.Unlock()

	// Remove JSONL file
	jsonlPath := s.jsonlPath(sessionKey)
	if err := os.Remove(jsonlPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("jsonl: remove jsonl file: %w", err)
	}

	// Remove meta file
	metaPath := s.metaPath(sessionKey)
	if err := os.Remove(metaPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("jsonl: remove meta file: %w", err)
	}

	return nil
}

func (s *JSONLStore) Close() error {
	return nil
}

func (s *JSONLStore) GetMeta(
	_ context.Context, sessionKey string,
) (*SessionMeta, error) {
	l := s.sessionLock(sessionKey)
	l.Lock()
	defer l.Unlock()

	meta, err := s.readMeta(sessionKey)
	if err != nil {
		return nil, err
	}

	// If the meta file doesn't exist, check if the JSONL file exists
	// to provide a minimal response with just the key
	if meta.Key == "" || meta.CreatedAt.IsZero() {
		return nil, nil
	}

	// Return a copy to prevent race conditions
	metaCopy := meta
	return &metaCopy, nil
}

func (s *JSONLStore) SaveMeta(_ context.Context, meta *SessionMeta) error {
	l := s.sessionLock(meta.Key)
	l.Lock()
	defer l.Unlock()

	_, err := s.readMeta(meta.Key)
	if err != nil {
		return err
	}

	return s.writeMeta(meta.Key, *meta)
}

func (s *JSONLStore) DeleteMeta(
	_ context.Context, sessionKey string,
) error {
	l := s.sessionLock(sessionKey)
	l.Lock()
	defer l.Unlock()

	// Remove meta file only
	metaPath := s.metaPath(sessionKey)
	if err := os.Remove(metaPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("jsonl: remove meta file: %w", err)
	}

	return nil
}
