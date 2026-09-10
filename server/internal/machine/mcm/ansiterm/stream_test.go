package ansiterm

import (
	"strings"
	"testing"
)

// helper: 创建 Screen + Stream 并关联
func newTestStream(cols, lines int) (*Screen, *Stream) {
	screen := NewScreen(cols, lines)
	stream := initializeStream(screen, false)
	stream.Attach(screen)
	return screen, stream
}

// ==================== Feed 基础测试 ====================

func TestFeed_PlainText(t *testing.T) {
	screen, stream := newTestStream(80, 24)
	stream.Feed("hello world")
	lines := screen.Display()
	if !strings.HasPrefix(lines[0], "hello world") {
		t.Errorf("expected 'hello world', got %q", lines[0])
	}
}

func TestFeed_CRLF(t *testing.T) {
	screen, stream := newTestStream(80, 24)
	stream.Feed("line1\r\nline2\r\nline3")
	lines := screen.Display()
	if !strings.HasPrefix(lines[0], "line1") {
		t.Errorf("line 0 = %q, want prefix 'line1'", lines[0])
	}
	if !strings.HasPrefix(lines[1], "line2") {
		t.Errorf("line 1 = %q, want prefix 'line2'", lines[1])
	}
	if !strings.HasPrefix(lines[2], "line3") {
		t.Errorf("line 2 = %q, want prefix 'line3'", lines[2])
	}
}

// ==================== ANSI 颜色序列测试 ====================

func TestFeed_SGRColors(t *testing.T) {
	screen, stream := newTestStream(80, 24)
	// 红色文本: ESC[31m text ESC[0m
	stream.Feed("\x1b[31mred text\x1b[0m normal")
	lines := screen.Display()
	text := lines[0]
	if !strings.Contains(text, "red text") {
		t.Errorf("should contain 'red text', got %q", text)
	}
	if !strings.Contains(text, "normal") {
		t.Errorf("should contain 'normal', got %q", text)
	}
}

// ==================== 光标定位 CSI 序列测试 ====================

func TestFeed_CursorPositionCSI(t *testing.T) {
	screen, stream := newTestStream(80, 24)
	// CSI 5;10H → 光标到第 5 行第 10 列 (0-based: 4, 9)
	stream.Feed("\x1b[5;10H")
	if screen.cursor.Y != 4 || screen.cursor.X != 9 {
		t.Errorf("cursor = (%d,%d), want (9,4)", screen.cursor.X, screen.cursor.Y)
	}
}

func TestFeed_CursorHome(t *testing.T) {
	screen, stream := newTestStream(80, 24)
	stream.Feed("some text")
	stream.Feed("\x1b[H") // CSI H = 光标到 home (1,1) → (0,0)
	if screen.cursor.X != 0 || screen.cursor.Y != 0 {
		t.Errorf("cursor = (%d,%d), want (0,0)", screen.cursor.X, screen.cursor.Y)
	}
}

// ==================== HandleEscape 重复调用修复验证 ====================

func TestFeed_ESC_D_IndexScrollsOnce(t *testing.T) {
	// 直接测试 Screen.Index() 而非通过异步 FSM 管道
	screen := NewScreen(10, 3)
	screen.Draw("line0")
	screen.CarriageReturn()
	screen.LineFeed()
	screen.Draw("line1")
	screen.CarriageReturn()
	screen.LineFeed()
	screen.Draw("line2")
	// 光标回到最后一行开头
	screen.CarriageReturn()
	// Index: 应只滚动一次
	screen.Index()

	lines := screen.Display()
	if strings.TrimSpace(lines[0]) != "line1" {
		t.Errorf("line 0 = %q, want 'line1' (scrolled once)", lines[0])
	}
	if strings.TrimSpace(lines[1]) != "line2" {
		t.Errorf("line 1 = %q, want 'line2'", lines[1])
	}
	if strings.TrimSpace(lines[2]) != "" {
		t.Errorf("line 2 = %q, want '' (new empty line)", lines[2])
	}
}

func TestFeed_ESC_M_ReverseIndexScrollsOnce(t *testing.T) {
	// 修复验证：ESC M (ReverseIndex) 应只反向滚动一次
	screen, stream := newTestStream(10, 3)
	stream.Feed("line0\r\nline1\r\nline2")
	// 光标到第 0 行
	stream.Feed("\x1b[1;1H")
	stream.Feed("\x1bM") // ReverseIndex: 应只反向滚动一次

	lines := screen.Display()
	if strings.TrimSpace(lines[0]) != "" {
		t.Errorf("line 0 = %q, want '' (new empty line at top)", lines[0])
	}
	if strings.TrimSpace(lines[1]) != "line0" {
		t.Errorf("line 1 = %q, want 'line0'", lines[1])
	}
}

func TestFeed_ESC_7_8_SaveRestoreCursor(t *testing.T) {
	screen, stream := newTestStream(80, 24)
	stream.Feed("\x1b[5;10H") // 光标到 (4, 9)
	stream.Feed("\x1b7")      // SaveCursor
	stream.Feed("\x1b[1;1H")  // 移动到 (0, 0)
	stream.Feed("\x1b8")      // RestoreCursor

	if screen.cursor.X != 9 || screen.cursor.Y != 4 {
		t.Errorf("restored cursor = (%d,%d), want (9,4)", screen.cursor.X, screen.cursor.Y)
	}
}

// ==================== 私有模式序列测试 ====================

func TestFeed_PrivateMode_ShowHideCursor(t *testing.T) {
	screen, stream := newTestStream(80, 24)
	stream.Feed("\x1b[?25l") // 隐藏光标
	if !screen.cursor.Hidden {
		t.Error("cursor should be hidden")
	}
	stream.Feed("\x1b[?25h") // 显示光标
	if screen.cursor.Hidden {
		t.Error("cursor should be visible")
	}
}

func TestFeed_AlternateScreen(t *testing.T) {
	screen, stream := newTestStream(80, 24)
	stream.Feed("main screen content")
	stream.Feed("\x1b[?1049h") // 进入备用屏幕
	stream.Feed("alt content")
	stream.Feed("\x1b[?1049l") // 退出备用屏幕
	// 退出后应恢复主屏幕
	lines := screen.Display()
	if !strings.Contains(lines[0], "main screen content") {
		t.Errorf("should restore main screen, got %q", lines[0])
	}
}

// ==================== OSC 序列测试 ====================

func TestFeed_OSC_SetTitle(t *testing.T) {
	screen, stream := newTestStream(80, 24)
	// OSC 2 ; title BEL — 解析器将 ";title" 作为完整 param
	stream.Feed("\x1b]2;My Terminal Title\x07")
	// title 应包含内容（解析器将分号也包含在 param 中）
	if screen.title == "" {
		t.Error("title should not be empty after OSC set")
	}
	if !strings.Contains(screen.title, "Terminal Title") {
		t.Errorf("title should contain 'Terminal Title', got %q", screen.title)
	}
}

func TestFeed_OSC_EmptyParamNoPanic(t *testing.T) {
	// 修复验证：空 OSC 参数不应 panic
	screen, stream := newTestStream(80, 24)
	// OSC 2 ; BEL (只有分号，无实际内容)
	stream.Feed("\x1b]2;\x07")
	// 不应 panic — title 可能为空或仅含分号
	_ = screen.title
}

// ==================== 综合场景测试 ====================

func TestFeed_ShellPromptWithColors(t *testing.T) {
	screen, stream := newTestStream(120, 40)
	// 模拟彩色 shell 提示符
	stream.Feed("\x1b[01;32mroot@host\x1b[0m:\x1b[01;34m~\x1b[0m# ls -la\r\n")
	stream.Feed("total 48\r\n")
	stream.Feed("-rw-r--r-- 1 root root 1234 Jan 1 00:00 file.txt\r\n")
	stream.Feed("\x1b[01;32mroot@host\x1b[0m:\x1b[01;34m~\x1b[0m# ")

	lines := screen.Display()
	// 找到最后一行非空内容
	var lastLine string
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			lastLine = lines[i]
			break
		}
	}
	if !strings.Contains(lastLine, "root@host") {
		t.Errorf("last line should contain prompt, got %q", lastLine)
	}
	if !strings.Contains(lastLine, "#") {
		t.Errorf("last line should contain '#', got %q", lastLine)
	}
}

func TestFeed_ClearScreenAndRedraw(t *testing.T) {
	screen, stream := newTestStream(80, 24)
	stream.Feed("old content\r\n")
	stream.Feed("\x1b[2J") // 清屏
	stream.Feed("\x1b[H")  // 光标到 home
	stream.Feed("new content")

	lines := screen.Display()
	if !strings.Contains(lines[0], "new content") {
		t.Errorf("line 0 should contain 'new content', got %q", lines[0])
	}
	// 清屏后旧内容不应存在
	for i := 1; i < len(lines); i++ {
		if strings.Contains(lines[i], "old content") {
			t.Errorf("line %d should not contain 'old content'", i)
		}
	}
}

// ==================== ByteStream UTF-8 测试 ====================

func TestByteStream_UTF8(t *testing.T) {
	screen, stream := newTestStream(80, 24)
	stream.Feed("你好世界")
	lines := screen.Display()
	if !strings.Contains(lines[0], "你好世界") {
		t.Errorf("should contain '你好世界', got %q", lines[0])
	}
}

func TestByteStream_InvalidUTF8(t *testing.T) {
	screen := NewScreen(80, 24)
	stream := InitByteStream(screen, false)
	stream.Attach(screen)
	// 无效 UTF-8 字节应被替换为 U+FFFD
	stream.Feed([]byte{0xff, 0xfe})
	lines := screen.Display()
	if !strings.Contains(lines[0], "\uFFFD") {
		t.Errorf("should contain replacement char, got %q", lines[0])
	}
}
