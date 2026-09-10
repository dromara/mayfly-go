package ansiterm

import "testing"

// ==================== StaticDefaultDict 测试 ====================

func TestStaticDefaultDict_GetDefault(t *testing.T) {
	sd := NewStaticDefaultDict[int, string]("default")
	got := sd.Get(42)
	if got != "default" {
		t.Errorf("Get(42) = %q, want 'default'", got)
	}
}

func TestStaticDefaultDict_SetAndGet(t *testing.T) {
	sd := NewStaticDefaultDict[int, string]("default")
	sd.Set(1, "hello")
	sd.Set(2, "world")

	if got := sd.Get(1); got != "hello" {
		t.Errorf("Get(1) = %q, want 'hello'", got)
	}
	if got := sd.Get(2); got != "world" {
		t.Errorf("Get(2) = %q, want 'world'", got)
	}
	// 未设置的键返回默认值
	if got := sd.Get(99); got != "default" {
		t.Errorf("Get(99) = %q, want 'default'", got)
	}
}

func TestStaticDefaultDict_Del(t *testing.T) {
	sd := NewStaticDefaultDict[int, string]("default")
	sd.Set(1, "hello")
	sd.Del(1)
	if got := sd.Get(1); got != "default" {
		t.Errorf("Get(1) after Del = %q, want 'default'", got)
	}
}

func TestStaticDefaultDict_Overwrite(t *testing.T) {
	sd := NewStaticDefaultDict[int, string]("default")
	sd.Set(1, "first")
	sd.Set(1, "second")
	if got := sd.Get(1); got != "second" {
		t.Errorf("Get(1) after overwrite = %q, want 'second'", got)
	}
}

// ==================== ScreenBuffer 测试 ====================

// Get 对不存在的行自动创建，这是 ScreenBuffer 的核心行为
func TestScreenBuffer_Get_CreatesOnDemand(t *testing.T) {
	sb := NewScreenBuffer(Char{Data: "", Fg: "default", Bg: "default"})
	line := sb.Get(5)
	if line == nil {
		t.Fatal("Get should create line on demand")
	}
	// 创建后 HasKey 应为 true
	if !sb.HasKey(5) {
		t.Error("HasKey(5) should be true after Get")
	}
}

// ==================== Char.Update 测试（switch/case 分支逻辑） ====================

func TestChar_Update(t *testing.T) {
	c := Char{Data: "A", Fg: "default"}
	c.Update(map[string]any{
		"data": "B",
		"fg":   "green",
		"bold": true,
	})
	if c.Data != "B" {
		t.Errorf("Data = %q, want 'B'", c.Data)
	}
	if c.Fg != "green" {
		t.Errorf("Fg = %q, want 'green'", c.Fg)
	}
	if !c.Bold {
		t.Error("Bold should be true")
	}
}

// 测试 Update 的所有属性分支
func TestChar_Update_AllFields(t *testing.T) {
	c := Char{}
	c.Update(map[string]any{
		"italics":       true,
		"underscore":    true,
		"strikethrough": true,
		"reverse":       true,
		"blink":         true,
		"bg":            "blue",
	})
	if !c.Italics {
		t.Error("Italics should be true")
	}
	if !c.Underscore {
		t.Error("Underscore should be true")
	}
	if !c.Strikethrough {
		t.Error("Strikethrough should be true")
	}
	if !c.Reverse {
		t.Error("Reverse should be true")
	}
	if !c.Blink {
		t.Error("Blink should be true")
	}
	if c.Bg != "blue" {
		t.Errorf("Bg = %q, want 'blue'", c.Bg)
	}
}
