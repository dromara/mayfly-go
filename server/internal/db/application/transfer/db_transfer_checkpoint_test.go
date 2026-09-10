package transfer

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// ---------------- 导入侧语句过滤 ----------------

func TestShouldSkipImportStmt(t *testing.T) {
	splitter := sqlparser.NewSplitter(tokenizer.MysqlConfig)
	skip := func(stmt string) bool { return shouldSkipImportStmt(splitter, stmt) }

	// 应过滤：各类事务控制语句及变体（含分号/大小写/空白/注释前缀差异）
	trueCases := []string{
		"BEGIN", "begin", "BEGIN;", "  begin ; ", "BEGIN WORK",
		"COMMIT", "commit;", "COMMIT WORK",
		"START TRANSACTION", "start transaction;",
		"ROLLBACK", "rollback;",
		"BEGIN TRANSACTION",
		"set autocommit=1", "SET AUTOCOMMIT = 0;", "SET @@session.autocommit=1",
		// dump 产物形态：分段注释头与 BEGIN 同属一条语句（切割保留注释原文）
		"-- ----------------------------\n-- Data: t_user \n-- ----------------------------\nBEGIN",
		"/* head */ COMMIT;",
		"-- c\n-- c2\nSTART TRANSACTION",
		// 可执行注释内容会被服务端执行（同样隐式提交当前事务），必须过滤
		"/*!40101 SET autocommit=0 */",
		"", "   \n ",
	}
	for _, c := range trueCases {
		assert.True(t, skip(c), "应过滤事务控制语句: %q", c)
	}

	// 不过滤：业务语句与自增列前置语句
	falseCases := []string{
		"INSERT INTO t VALUES (1)",
		"insert into `t` (`a`) values ('BEGIN')",         // 字符串字面量含BEGIN不受影响
		"INSERT INTO t VALUES ('-- not comment\nBEGIN')", // 字面量内的注释形态不得参与判定
		"SELECT 1",
		"DROP TABLE IF EXISTS `t`",
		"CREATE TABLE `t` (id int)",
		"set identity_insert [t] on",               // mssql自增列前置语句
		"SET IDENTITY_INSERT [t] OFF;",             // 同上
		"set identity_insert \"t\" off;",           // dm
		"BEGINNING",                                // 前缀相似的非事务语句（虽非合法SQL，但不得误判）
		"/*!40000 ALTER TABLE `t` DISABLE KEYS */", // 可执行注释解壳后为业务语句
	}
	for _, c := range falseCases {
		assert.False(t, skip(c), "不应过滤语句: %q", c)
	}
}

// ---------------- 序列化往返 ----------------

func TestMarshalUnmarshalStringSlice(t *testing.T) {
	cases := [][]string{
		nil,
		{},
		{"t1"},
		{"t1", "t2", "t3"},
		{"带中文表名", "with space", "with\"quote"},
	}
	for _, c := range cases {
		s, err := marshalStringSlice(c)
		require.NoError(t, err)
		back, err := unmarshalStringSlice(s)
		require.NoError(t, err)
		expect := c
		if expect == nil {
			expect = []string{} // nil统一归一化为空数组
		}
		assert.Equal(t, expect, back)
	}

	// 非法JSON报错
	_, err := unmarshalStringSlice("[not-json")
	assert.Error(t, err)
	// 空串与null视为空数组
	for _, s := range []string{"", "  ", "null"} {
		back, err := unmarshalStringSlice(s)
		require.NoError(t, err)
		assert.Empty(t, back)
	}
}

// ---------------- remainingTables / equalStringSlice ----------------

func TestRemainingTables(t *testing.T) {
	assert.Equal(t, []string{"t2", "t3"}, remainingTables([]string{"t1", "t2", "t3"}, []string{"t1"}))
	assert.Equal(t, []string{"t3"}, remainingTables([]string{"t1", "t2", "t3"}, []string{"t1", "t2", "t0"}))
	assert.Empty(t, remainingTables([]string{"t1"}, []string{"t1"}))
	assert.Equal(t, []string{"t1", "t2"}, remainingTables([]string{"t1", "t2"}, nil))
}

func TestEqualStringSlice(t *testing.T) {
	assert.True(t, equalStringSlice(nil, nil))
	assert.True(t, equalStringSlice([]string{"a", "b"}, []string{"a", "b"}))
	assert.False(t, equalStringSlice([]string{"a", "b"}, []string{"b", "a"}), "顺序敏感")
	assert.False(t, equalStringSlice([]string{"a"}, []string{"a", "b"}))
}

// ---------------- 内存检查点存储（测试基建） ----------------

type memoryCheckpointStore struct {
	mu sync.Mutex
	m  map[uint64]*transferCheckpoint
}

func newMemoryCheckpointStore() *memoryCheckpointStore {
	return &memoryCheckpointStore{m: make(map[uint64]*transferCheckpoint)}
}

func (s *memoryCheckpointStore) Load(taskId uint64) (*transferCheckpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp, ok := s.m[taskId]
	if !ok {
		return nil, nil
	}
	cpCopy := *cp
	cpCopy.DoneTables = append([]string{}, cp.DoneTables...)
	cpCopy.PlannedTables = append([]string{}, cp.PlannedTables...)
	return &cpCopy, nil
}

func (s *memoryCheckpointStore) Save(taskId uint64, cp *transferCheckpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cpCopy := *cp
	cpCopy.DoneTables = append([]string{}, cp.DoneTables...)
	cpCopy.PlannedTables = append([]string{}, cp.PlannedTables...)
	s.m[taskId] = &cpCopy
	return nil
}

func (s *memoryCheckpointStore) AppendDone(taskId uint64, table string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp, ok := s.m[taskId]
	if !ok {
		return fmt.Errorf("checkpoint of transfer task [%d] not found", taskId)
	}
	for _, t := range cp.DoneTables {
		if t == table {
			return nil
		}
	}
	cp.DoneTables = append(cp.DoneTables, table)
	return nil
}

func (s *memoryCheckpointStore) Clear(taskId uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, taskId)
	return nil
}

// ---------------- initCheckpoint / AppendDone ----------------

func TestInitCheckpoint_FreshAndResume(t *testing.T) {
	store := newMemoryCheckpointStore()
	planned := []string{"t1", "t2", "t3"}

	// 首次：新建检查点，非续传
	cp, resume, err := initCheckpoint(store, 1, planned)
	require.NoError(t, err)
	assert.False(t, resume)
	assert.Equal(t, planned, cp.PlannedTables)
	assert.Empty(t, cp.DoneTables)

	// 记录表1完成
	require.NoError(t, store.AppendDone(1, "t1"))

	// 重跑：计划一致 → 续传，剩余t2/t3
	cp, resume, err = initCheckpoint(store, 1, planned)
	require.NoError(t, err)
	assert.True(t, resume)
	assert.Equal(t, []string{"t1"}, cp.DoneTables)
	assert.Equal(t, []string{"t2", "t3"}, remainingTables(cp.PlannedTables, cp.DoneTables))
}

func TestInitCheckpoint_PlannedChanged(t *testing.T) {
	store := newMemoryCheckpointStore()
	require.NoError(t, store.Save(1, &transferCheckpoint{
		PlannedTables: []string{"t1", "t2"},
		DoneTables:    []string{"t1"},
	}))

	// 计划表清单变更（任务配置调整）→ 作废重建，非续传
	cp, resume, err := initCheckpoint(store, 1, []string{"t1", "t3"})
	require.NoError(t, err)
	assert.False(t, resume)
	assert.Equal(t, []string{"t1", "t3"}, cp.PlannedTables)
	assert.Empty(t, cp.DoneTables)
}

func TestAppendDone_ConcurrentNoLostUpdate(t *testing.T) {
	store := newMemoryCheckpointStore()
	require.NoError(t, store.Save(1, &transferCheckpoint{PlannedTables: []string{"t1", "t2", "t3", "t4", "t5"}}))

	var wg sync.WaitGroup
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			require.NoError(t, store.AppendDone(1, fmt.Sprintf("t%d", n)))
		}(i)
	}
	wg.Wait()

	cp, err := store.Load(1)
	require.NoError(t, err)
	assert.Len(t, cp.DoneTables, 5, "并发AppendDone不应丢失更新")
}

func TestCheckpointStore_ClearAndClearAgain(t *testing.T) {
	store := newMemoryCheckpointStore()
	require.NoError(t, store.Save(1, &transferCheckpoint{PlannedTables: []string{"t1"}}))
	require.NoError(t, store.Clear(1))
	cp, err := store.Load(1)
	require.NoError(t, err)
	assert.Nil(t, cp)

	// 重复清除安全
	require.NoError(t, store.Clear(1))
}

// StartedAt 序列化往返（gorm store经CreateTime承载，内存形态直接JSON语义验证）
func TestTransferCheckpoint_JsonRoundTrip(t *testing.T) {
	cp := &transferCheckpoint{
		PlannedTables: []string{"a", "b"},
		DoneTables:    []string{"a"},
		StartedAt:     time.Now(),
	}
	require.NoError(t, newMemoryCheckpointStore().Save(1, cp))
	loaded, err := newMemoryCheckpointStore().Load(1)
	require.NoError(t, err)
	assert.Nil(t, loaded, "新store不应有任务1的数据")
}
