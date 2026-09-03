package memory

import (
	"context"
	"os"
	"testing"
)

// TestJSONLStore_Basic 测试基本的读写操作
func TestJSONLStore_Basic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewJSONLStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	ctx := context.Background()
	userID := "test_user_1"

	// 创建测试记忆（使用新的 Type-Content-Tags 结构）
	item := CreateMemory(userID, "preference", "用户偏好使用 vim 作为代码编辑器", []string{"editor", "vim", "preference"})

	// 保存记忆
	err = store.Save(ctx, []*MemoryItem{item})
	if err != nil {
		t.Fatalf("save memory failed: %v", err)
	}

	// 检索记忆
	items, err := store.GetByUser(ctx, userID, nil)
	if err != nil {
		t.Fatalf("get memory failed: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}

	if items[0].Type != "preference" {
		t.Errorf("expected type 'preference', got '%s'", items[0].Type)
	}

	if items[0].Content != "用户偏好使用 vim 作为代码编辑器" {
		t.Errorf("expected content '用户偏好使用 vim 作为代码编辑器', got '%s'", items[0].Content)
	}

	if len(items[0].Tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(items[0].Tags))
	}

	t.Logf("✅ Basic CRUD test passed")
}

// TestJSONLStore_MultipleItems 测试保存多条记忆
func TestJSONLStore_MultipleItems(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, _ := NewJSONLStore(tempDir)
	ctx := context.Background()
	userID := "test_user_2"

	// 保存多条不同类型的记忆
	items := []*MemoryItem{
		CreateMemory(userID, "preference", "用户喜欢深色主题", []string{"theme", "dark", "ui"}),
		CreateMemory(userID, "fact", "用户的服务器IP地址为 192.168.1.100", []string{"server", "ip", "infrastructure"}),
		CreateMemory(userID, "skill", "用户熟练掌握 Go 语言开发", []string{"go", "programming", "skill"}),
	}

	err = store.Save(ctx, items)
	if err != nil {
		t.Fatalf("batch save failed: %v", err)
	}

	// 验证所有记忆都已保存
	retrieved, _ := store.GetByUser(ctx, userID, nil)
	if len(retrieved) != 3 {
		t.Errorf("expected 3 items, got %d", len(retrieved))
	}

	t.Logf("✅ Multiple items test passed")
}

// TestJSONLStore_FilterByTags 测试标签过滤功能
func TestJSONLStore_FilterByTags(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, _ := NewJSONLStore(tempDir)
	ctx := context.Background()
	userID := "test_user_3"

	// 创建多个记忆
	items := []*MemoryItem{
		CreateMemory(userID, "preference", "用户偏好使用 vim 编辑器", []string{"editor", "vim"}),
		CreateMemory(userID, "fact", "服务器IP是 192.168.1.100", []string{"server", "ip"}),
		CreateMemory(userID, "skill", "用户精通 Python 编程", []string{"python", "programming"}),
		CreateMemory(userID, "experience", "用户正在开发 mayfly-go 项目", []string{"project", "go"}),
	}

	store.Save(ctx, items)

	// 按标签过滤：只获取包含 "editor" 标签的记忆
	filtered, _ := store.GetByUser(ctx, userID, []string{"editor"})
	if len(filtered) != 1 {
		t.Errorf("expected 1 item with 'editor' tag, got %d", len(filtered))
	}

	if filtered[0].Type != "preference" {
		t.Errorf("expected preference type, got %s", filtered[0].Type)
	}

	// 按多个标签过滤：获取包含 "server" 或 "ip" 的记忆
	serverItems, _ := store.GetByUser(ctx, userID, []string{"server", "ip"})
	if len(serverItems) != 1 {
		t.Errorf("expected 1 item with server/ip tags, got %d", len(serverItems))
	}

	t.Logf("✅ Filter by tags test passed")
}

// TestJSONLStore_Delete 测试删除功能
func TestJSONLStore_Delete(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, _ := NewJSONLStore(tempDir)
	ctx := context.Background()
	userID := "test_user_4"

	// 保存多条记忆
	items := []*MemoryItem{
		CreateMemory(userID, "preference", "偏好A", []string{"a"}),
		CreateMemory(userID, "fact", "事实B", []string{"b"}),
		CreateMemory(userID, "skill", "技能C", []string{"c"}),
	}

	store.Save(ctx, items)

	// 验证初始数量并获取生成的ID
	all, _ := store.GetByUser(ctx, userID, nil)
	if len(all) != 3 {
		t.Fatalf("expected 3 items initially, got %d", len(all))
	}

	// 打印调试信息
	t.Logf("Saved items IDs: %v, %v, %v", all[0].ID, all[1].ID, all[2].ID)

	// 删除前两条记忆（通过ID）
	idsToDelete := []string{all[0].ID, all[1].ID}
	err = store.Delete(ctx, userID, idsToDelete)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// 验证剩余数量
	remaining, _ := store.GetByUser(ctx, userID, nil)
	t.Logf("After delete: %d items remaining", len(remaining))

	if len(remaining) != 1 {
		t.Errorf("expected 1 item after delete, got %d", len(remaining))
		return // 避免后续访问空数组导致 panic
	}

	if remaining[0].Type != "skill" {
		t.Errorf("expected 'skill' type remaining, got '%s'", remaining[0].Type)
	}

	t.Logf("✅ Delete test passed")
}

// TestJSONLStore_Search 测试搜索功能（简化实现）
func TestJSONLStore_Search(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, _ := NewJSONLStore(tempDir)
	ctx := context.Background()
	userID := "test_user_5"

	// 保存多条记忆
	items := []*MemoryItem{
		CreateMemory(userID, "preference", "第一条记忆", []string{"tag1"}),
		CreateMemory(userID, "fact", "第二条记忆", []string{"tag2"}),
		CreateMemory(userID, "skill", "第三条记忆", []string{"tag3"}),
	}

	store.Save(ctx, items)

	// 测试搜索（当前实现返回最近N条）
	results, _ := store.Search(ctx, userID, "", 2)
	if len(results) != 2 {
		t.Errorf("expected 2 items from search, got %d", len(results))
	}

	t.Logf("✅ Search test passed")
}

// TestMemoryManager_SaveAndRetrieve 测试管理器的保存和检索
func TestMemoryManager_SaveAndRetrieve(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "memory_test_*")
	defer os.RemoveAll(tempDir)

	store, _ := NewJSONLStore(tempDir)
	manager := NewManager(store)

	ctx := context.Background()
	userID := "test_user_6"

	// 保存单条记忆
	item := CreateMemory(userID, "preference", "用户喜欢使用 Go 语言", []string{"go", "language"})
	err := manager.Save(ctx, item)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// 检索所有记忆
	all, _ := manager.RetrieveAll(ctx, userID)
	if len(all) != 1 {
		t.Errorf("expected 1 item, got %d", len(all))
	}

	t.Logf("✅ Manager save and retrieve test passed")
}

// TestMemoryManager_BatchSave 测试批量保存
func TestMemoryManager_BatchSave(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "memory_test_*")
	defer os.RemoveAll(tempDir)

	store, _ := NewJSONLStore(tempDir)
	manager := NewManager(store)

	ctx := context.Background()
	userID := "test_user_7"

	// 批量保存
	items := []*MemoryItem{
		CreateMemory(userID, "preference", "偏好1", []string{"p1"}),
		CreateMemory(userID, "fact", "事实2", []string{"f2"}),
		CreateMemory(userID, "skill", "技能3", []string{"s3"}),
	}

	err := manager.SaveBatch(ctx, items)
	if err != nil {
		t.Fatalf("batch save failed: %v", err)
	}

	// 验证数量
	all, _ := manager.RetrieveAll(ctx, userID)
	if len(all) != 3 {
		t.Errorf("expected 3 items, got %d", len(all))
	}

	t.Logf("✅ Batch save test passed")
}

// TestMemoryManager_Delete 测试管理器删除
func TestMemoryManager_Delete(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "memory_test_*")
	defer os.RemoveAll(tempDir)

	store, _ := NewJSONLStore(tempDir)
	manager := NewManager(store)

	ctx := context.Background()
	userID := "test_user_8"

	// 保存记忆
	items := []*MemoryItem{
		CreateMemory(userID, "preference", "记忆A", []string{"a"}),
		CreateMemory(userID, "fact", "记忆B", []string{"b"}),
	}

	manager.SaveBatch(ctx, items)

	// 获取所有记忆的ID
	all, _ := manager.RetrieveAll(ctx, userID)
	if len(all) != 2 {
		t.Fatalf("expected 2 items, got %d", len(all))
	}

	// 删除第一条记忆
	err := manager.Delete(ctx, userID, []string{all[0].ID})
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// 验证剩余数量
	remaining, _ := manager.RetrieveAll(ctx, userID)
	if len(remaining) != 1 {
		t.Errorf("expected 1 item after delete, got %d", len(remaining))
	}

	t.Logf("✅ Manager delete test passed")
}

// TestMemoryManager_Config 测试配置禁用功能
func TestMemoryManager_Config(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "memory_test_*")
	defer os.RemoveAll(tempDir)

	store, _ := NewJSONLStore(tempDir)
	manager := NewManager(store)

	// 禁用记忆功能
	manager.WithConfig(&Config{Enabled: false})

	ctx := context.Background()
	userID := "test_user_9"

	// 尝试保存（应该被跳过）
	item := CreateMemory(userID, "preference", "测试记忆", []string{"test"})
	err := manager.Save(ctx, item)
	if err != nil {
		t.Fatalf("save should not fail when disabled: %v", err)
	}

	// 验证没有保存任何内容
	all, _ := manager.RetrieveAll(ctx, userID)
	if len(all) != 0 {
		t.Errorf("expected 0 items when disabled, got %d", len(all))
	}

	t.Logf("✅ Config disable test passed")
}
