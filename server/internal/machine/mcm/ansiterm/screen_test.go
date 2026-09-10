package ansiterm

import (
	"strings"
	"testing"
)

// ==================== Draw 测试 ====================

func TestScreenDraw_WideChar(t *testing.T) {
	s := NewScreen(80, 24)
	// CJK 字符占 2 列
	s.Draw("你好")
	if s.cursor.X != 4 {
		t.Errorf("cursor.X = %d, want 4 (two CJK chars = 4 columns)", s.cursor.X)
	}
}

func TestScreenDraw_LineWrap(t *testing.T) {
	s := NewScreen(5, 3)
	s.Draw("abcdefgh") // 超过 5 列应自动换行
	lines := s.Display()
	if lines[0] != "abcde" {
		t.Errorf("line 0 = %q, want 'abcde'", lines[0])
	}
	if !strings.HasPrefix(lines[1], "fgh") {
		t.Errorf("line 1 = %q, want prefix 'fgh'", lines[1])
	}
}

func TestScreenDraw_CarriageReturnOverwrite(t *testing.T) {
	s := NewScreen(80, 24)
	s.Draw("hello")
	s.CarriageReturn()
	s.Draw("HELLO")
	lines := s.Display()
	if !strings.HasPrefix(lines[0], "HELLO") {
		t.Errorf("expected 'HELLO' overwrite, got %q", lines[0])
	}
}

// ==================== CursorPosition 测试（修复：1-based → 0-based） ====================

func TestCursorPosition_TopLeft(t *testing.T) {
	s := NewScreen(80, 24)
	// ANSI CUP: CSI 1;1H → 光标应到 (0, 0)
	s.CursorPosition(1, 1)
	if s.cursor.X != 0 || s.cursor.Y != 0 {
		t.Errorf("CursorPosition(1,1) = (%d,%d), want (0,0)", s.cursor.X, s.cursor.Y)
	}
}

func TestCursorPosition_ZeroMeansOne(t *testing.T) {
	s := NewScreen(80, 24)
	// ANSI: 0 等同于 1
	s.CursorPosition(0, 0)
	if s.cursor.X != 0 || s.cursor.Y != 0 {
		t.Errorf("CursorPosition(0,0) = (%d,%d), want (0,0)", s.cursor.X, s.cursor.Y)
	}
}

func TestCursorPosition_MiddleOfScreen(t *testing.T) {
	s := NewScreen(80, 24)
	// ANSI: 第 5 行第 10 列 → 内部 (9, 4)
	s.CursorPosition(5, 10)
	if s.cursor.X != 9 || s.cursor.Y != 4 {
		t.Errorf("CursorPosition(5,10) = (%d,%d), want (9,4)", s.cursor.X, s.cursor.Y)
	}
}

func TestCursorPosition_ConsistentWithSetCursorPosition(t *testing.T) {
	s1 := NewScreen(80, 24)
	s2 := NewScreen(80, 24)

	s1.CursorPosition(3, 7)
	s2.SetCursorPosition(3, 7)

	if s1.cursor.X != s2.cursor.X || s1.cursor.Y != s2.cursor.Y {
		t.Errorf("CursorPosition(3,7) = (%d,%d) but SetCursorPosition(3,7) = (%d,%d)",
			s1.cursor.X, s1.cursor.Y, s2.cursor.X, s2.cursor.Y)
	}
}

func TestCursorPosition_ClampToBounds(t *testing.T) {
	s := NewScreen(80, 24)
	// 超出屏幕范围应被钳制
	s.CursorPosition(100, 200)
	if s.cursor.X != 79 || s.cursor.Y != 23 {
		t.Errorf("CursorPosition(100,200) = (%d,%d), want (79,23)", s.cursor.X, s.cursor.Y)
	}
}

// ==================== 光标移动边界测试 ====================

func TestCursorMovement_Boundaries(t *testing.T) {
	// CursorUp: 正常移动 + 边界钳制
	s := NewScreen(80, 24)
	s.CursorPosition(10, 1) // cursor.Y = 9
	s.CursorUp(3)
	if s.cursor.Y != 6 {
		t.Errorf("CursorUp: cursor.Y = %d, want 6", s.cursor.Y)
	}
	s.CursorUp(100) // 不应超出顶部
	if s.cursor.Y != 0 {
		t.Errorf("CursorUp bound: cursor.Y = %d, want 0", s.cursor.Y)
	}

	// CursorDown: 正常移动 + 边界钳制
	s = NewScreen(80, 24)
	s.CursorDown(5)
	if s.cursor.Y != 5 {
		t.Errorf("CursorDown: cursor.Y = %d, want 5", s.cursor.Y)
	}
	s.CursorDown(100) // 不应超出底部
	if s.cursor.Y != 23 {
		t.Errorf("CursorDown bound: cursor.Y = %d, want 23", s.cursor.Y)
	}

	// CursorForward: 正常移动 + 边界钳制
	s = NewScreen(80, 24)
	s.CursorForward(10)
	if s.cursor.X != 10 {
		t.Errorf("CursorForward: cursor.X = %d, want 10", s.cursor.X)
	}
	s.CursorForward(200) // 不应超出右边界
	if s.cursor.X != 79 {
		t.Errorf("CursorForward bound: cursor.X = %d, want 79", s.cursor.X)
	}

	// CursorBack: 正常移动 + 边界钳制
	s = NewScreen(80, 24)
	s.CursorPosition(1, 20) // cursor.X = 19
	s.CursorBack(5)
	if s.cursor.X != 14 {
		t.Errorf("CursorBack: cursor.X = %d, want 14", s.cursor.X)
	}
	s.CursorBack(100) // 不应超出左边界
	if s.cursor.X != 0 {
		t.Errorf("CursorBack bound: cursor.X = %d, want 0", s.cursor.X)
	}
}

// ==================== EraseInDisplay 测试 ====================

func TestEraseInDisplay_All(t *testing.T) {
	s := NewScreen(80, 24)
	s.Draw("hello world")
	s.EraseInDisplay(2)
	lines := s.Display()
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			t.Errorf("line %d should be empty after ED(2), got %q", i, line)
		}
	}
}

func TestEraseInDisplay_BelowCursor(t *testing.T) {
	s := NewScreen(10, 5)
	s.Draw("line0")
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("line1")
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("line2")
	// 光标回到第 3 行开头
	s.CarriageReturn()
	// 此时光标在 (0, 2)，EraseInDisplay(0) 擦除光标所在行及以下
	s.EraseInDisplay(0)
	lines := s.Display()
	if strings.TrimSpace(lines[0]) != "line0" {
		t.Errorf("line 0 should be preserved, got %q", lines[0])
	}
	if strings.TrimSpace(lines[1]) != "line1" {
		t.Errorf("line 1 should be preserved, got %q", lines[1])
	}
	if strings.TrimSpace(lines[2]) != "" {
		t.Errorf("line 2 should be erased, got %q", lines[2])
	}
}

// ==================== EraseInLine 测试 ====================

func TestEraseInLine_FromCursorToEnd(t *testing.T) {
	s := NewScreen(20, 3)
	s.Draw("hello world")
	s.CarriageReturn()
	s.CursorForward(5)
	s.EraseInLine(0, false) // 从光标到行尾
	lines := s.Display()
	if !strings.HasPrefix(lines[0], "hello") {
		t.Errorf("expected 'hello' prefix, got %q", lines[0])
	}
	if strings.TrimSpace(lines[0]) != "hello" {
		t.Errorf("expected only 'hello' after erase, got %q", lines[0])
	}
}

// ==================== InsertLines 测试（修复：循环条件反向） ====================

func TestInsertLines_Basic(t *testing.T) {
	s := NewScreen(10, 5)
	s.Draw("line0")
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("line1")
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("line2")
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("line3")
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("line4")

	// 光标回到第 1 行
	s.CursorPosition(1, 1)
	s.InsertLines(2) // 在第 0 行插入 2 个空行

	lines := s.Display()
	// 前 2 行应为空
	if strings.TrimSpace(lines[0]) != "" {
		t.Errorf("line 0 should be empty after insert, got %q", lines[0])
	}
	if strings.TrimSpace(lines[1]) != "" {
		t.Errorf("line 1 should be empty after insert, got %q", lines[1])
	}
	// 原来的 line0, line1, line2 应下移到 2, 3, 4
	if strings.TrimSpace(lines[2]) != "line0" {
		t.Errorf("line 2 should be 'line0', got %q", lines[2])
	}
	if strings.TrimSpace(lines[3]) != "line1" {
		t.Errorf("line 3 should be 'line1', got %q", lines[3])
	}
	if strings.TrimSpace(lines[4]) != "line2" {
		t.Errorf("line 4 should be 'line2', got %q", lines[4])
	}
}

func TestInsertLines_InMiddle(t *testing.T) {
	s := NewScreen(10, 5)
	s.Draw("AAA")
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("BBB")
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("CCC")
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("DDD")
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("EEE")

	// 光标到第 3 行 (index=2)
	s.CursorPosition(3, 1)
	s.InsertLines(1) // 在第 2 行插入 1 个空行

	lines := s.Display()
	if strings.TrimSpace(lines[0]) != "AAA" {
		t.Errorf("line 0 should be 'AAA', got %q", lines[0])
	}
	if strings.TrimSpace(lines[1]) != "BBB" {
		t.Errorf("line 1 should be 'BBB', got %q", lines[1])
	}
	if strings.TrimSpace(lines[2]) != "" {
		t.Errorf("line 2 should be empty (inserted), got %q", lines[2])
	}
	if strings.TrimSpace(lines[3]) != "CCC" {
		t.Errorf("line 3 should be 'CCC', got %q", lines[3])
	}
}

// ==================== DeleteLines 测试 ====================

func TestDeleteLines_Basic(t *testing.T) {
	s := NewScreen(10, 5)
	s.Draw("line0")
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("line1")
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("line2")
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("line3")
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("line4")

	s.CursorPosition(1, 1)
	s.DeleteLines(2) // 删除前 2 行

	lines := s.Display()
	if strings.TrimSpace(lines[0]) != "line2" {
		t.Errorf("line 0 should be 'line2', got %q", lines[0])
	}
	if strings.TrimSpace(lines[1]) != "line3" {
		t.Errorf("line 1 should be 'line3', got %q", lines[1])
	}
}

// ==================== Index / ReverseIndex / Scroll 测试 ====================

func TestIndex_ScrollsWhenAtBottom(t *testing.T) {
	s := NewScreen(10, 3)
	s.Draw("line0")
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("line1")
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("line2")
	// 光标在最后一行
	s.Index() // 应向上滚动

	lines := s.Display()
	if strings.TrimSpace(lines[0]) != "line1" {
		t.Errorf("line 0 should be 'line1' after scroll, got %q", lines[0])
	}
	if strings.TrimSpace(lines[1]) != "line2" {
		t.Errorf("line 1 should be 'line2' after scroll, got %q", lines[1])
	}
}

func TestReverseIndex_ScrollsWhenAtTop(t *testing.T) {
	s := NewScreen(10, 3)
	s.Draw("line0")
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("line1")
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("line2")

	// 光标回到第 0 行
	s.CursorPosition(1, 1)
	s.ReverseIndex() // 应向下滚动

	lines := s.Display()
	if strings.TrimSpace(lines[0]) != "" {
		t.Errorf("line 0 should be empty after reverse scroll, got %q", lines[0])
	}
	if strings.TrimSpace(lines[1]) != "line0" {
		t.Errorf("line 1 should be 'line0', got %q", lines[1])
	}
}

// ==================== SaveCursor / RestoreCursor 测试 ====================

func TestSaveRestoreCursor(t *testing.T) {
	s := NewScreen(80, 24)
	s.CursorPosition(5, 10)
	s.SaveCursor()

	s.CursorPosition(1, 1)
	s.RestoreCursor()

	if s.cursor.X != 9 || s.cursor.Y != 4 {
		t.Errorf("restored cursor = (%d,%d), want (9,4)", s.cursor.X, s.cursor.Y)
	}
}

// ==================== Reset 测试 ====================

func TestScreenReset(t *testing.T) {
	s := NewScreen(80, 24)
	s.Draw("hello world")
	s.CursorPosition(10, 20)
	s.Reset()

	if s.cursor.X != 0 || s.cursor.Y != 0 {
		t.Errorf("cursor after reset = (%d,%d), want (0,0)", s.cursor.X, s.cursor.Y)
	}
	lines := s.Display()
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			t.Errorf("line %d should be empty after reset, got %q", i, line)
		}
	}
}

// ==================== SetMode / ResetMode 测试 ====================

func TestSetMode_PrivateCursorVisibility(t *testing.T) {
	s := NewScreen(80, 24)
	// 私有模式 25 = DECTCEM (光标可见)
	s.SetMode([]int{25}, map[string]any{"private": true})
	if s.cursor.Hidden {
		t.Error("cursor should be visible after SetMode(25, private)")
	}
}

func TestResetMode_PrivateCursorVisibility(t *testing.T) {
	s := NewScreen(80, 24)
	s.cursor.Hidden = false
	// 私有模式 25 = DECTCEM
	s.ResetMode([]int{25}, map[string]any{"private": true})
	if !s.cursor.Hidden {
		t.Error("cursor should be hidden after ResetMode(25, private)")
	}
}

func TestSetMode_PrivateAutoWrap(t *testing.T) {
	s := NewScreen(80, 24)
	// 私有模式 7 = DECAWM (自动换行)
	s.SetMode([]int{7}, map[string]any{"private": true})
	if _, ok := s.mode[224]; !ok { // DECAWM = 224
		t.Error("DECAWM should be set after SetMode(7, private)")
	}
}

// ==================== SelectGraphicRendition 测试 ====================

func TestSGR_Reset(t *testing.T) {
	s := NewScreen(80, 24)
	s.SelectGraphicRendition(1) // bold
	s.SelectGraphicRendition(0) // reset
	if s.cursor.Attrs.Bold {
		t.Error("bold should be false after SGR(0)")
	}
}

func TestSGR_ForegroundColor(t *testing.T) {
	s := NewScreen(80, 24)
	s.SelectGraphicRendition(31) // red foreground
	if s.cursor.Attrs.Fg != "red" {
		t.Errorf("fg = %q, want 'red'", s.cursor.Attrs.Fg)
	}
}

func TestSGR_BackgroundColor(t *testing.T) {
	s := NewScreen(80, 24)
	s.SelectGraphicRendition(42) // green background
	if s.cursor.Attrs.Bg != "green" {
		t.Errorf("bg = %q, want 'green'", s.cursor.Attrs.Bg)
	}
}

// ==================== Resize 测试 ====================

func TestResize(t *testing.T) {
	s := NewScreen(80, 24)
	s.Resize(40, 120)

	lines := s.Display()
	if len(lines) != 40 {
		t.Errorf("after Resize(40,120): Display returned %d lines, want 40", len(lines))
	}

	// Resize(0, 132) 模拟 DECCOLM：lines=0 表示不改变行数
	s2 := NewScreen(80, 24)
	s2.Resize(0, 132)
	lines2 := s2.Display()
	if len(lines2) != 24 {
		t.Errorf("after Resize(0,132): Display returned %d lines, want 24 (lines unchanged)", len(lines2))
	}

	// 光标应被钳制到新边界内
	s3 := NewScreen(10, 5)
	s3.CursorPosition(5, 10) // 光标在 (9, 4)
	s3.Resize(3, 5)          // 缩小到 3 行 5 列
	if s3.cursor.Y > 2 {
		t.Errorf("cursor.Y = %d, should be clamped to ≤ 2", s3.cursor.Y)
	}
	if s3.cursor.X > 4 {
		t.Errorf("cursor.X = %d, should be clamped to ≤ 4", s3.cursor.X)
	}
}

// ==================== Display 测试 ====================

func TestDisplay_MultipleLines(t *testing.T) {
	s := NewScreen(10, 3)
	s.Draw("AAA")
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("BBB")
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("CCC")

	lines := s.Display()
	if !strings.HasPrefix(lines[0], "AAA") {
		t.Errorf("line 0 = %q, want prefix 'AAA'", lines[0])
	}
	if !strings.HasPrefix(lines[1], "BBB") {
		t.Errorf("line 1 = %q, want prefix 'BBB'", lines[1])
	}
	if !strings.HasPrefix(lines[2], "CCC") {
		t.Errorf("line 2 = %q, want prefix 'CCC'", lines[2])
	}
}

func TestDisplay_WideCharacters(t *testing.T) {
	// 验证 Display() 正确处理宽字符（CJK 等双宽字符）
	s := NewScreen(10, 2)
	s.Draw("中文") // '中' 和 '文' 各占 2 列，共 4 列

	lines := s.Display()
	if len(lines) == 0 {
		t.Fatal("Display returned no lines")
	}

	// '中' 占 2 列，'文' 占 2 列，Display 应正确跳过宽字符的第二列
	line := lines[0]
	if !strings.Contains(line, "中") || !strings.Contains(line, "文") {
		t.Errorf("line = %q, want contains '中文'", line)
	}

	// 验证宽字符后继续绘制正常
	s.CarriageReturn()
	s.LineFeed()
	s.Draw("AB") // 2 个单宽字符
	lines = s.Display()
	if len(lines) < 2 {
		t.Fatalf("Display returned %d lines, want ≥ 2", len(lines))
	}
	if !strings.HasPrefix(lines[1], "AB") {
		t.Errorf("line 1 = %q, want prefix 'AB'", lines[1])
	}
}

// ==================== SetMargins 测试 ====================

func TestSetMargins(t *testing.T) {
	s := NewScreen(80, 24)
	s.SetMargins(5, 20)
	if s.margins == nil {
		t.Fatal("margins should not be nil")
	}
	// 验证 margins 已设置且不 panic
	if s.margins.Top < 0 || s.margins.Bottom >= s.lines {
		t.Errorf("margins out of range: (%d, %d)", s.margins.Top, s.margins.Bottom)
	}
	if s.margins.Top >= s.margins.Bottom {
		t.Errorf("top (%d) should be < bottom (%d)", s.margins.Top, s.margins.Bottom)
	}
}

// ==================== DeleteCharacters / InsertCharacters 测试 ====================

func TestDeleteCharacters(t *testing.T) {
	s := NewScreen(20, 3)
	s.Draw("hello world")
	s.CarriageReturn()
	s.CursorForward(5)
	// 验证 DeleteCharacters 不 panic
	s.DeleteCharacters(1)
	lines := s.Display()
	// 行内容应发生变化
	if lines[0] == "hello world" {
		t.Error("DeleteCharacters should modify line content")
	}
}

func TestInsertCharacters(t *testing.T) {
	s := NewScreen(20, 3)
	s.Draw("hello")
	s.CarriageReturn()
	// 验证 InsertCharacters 不 panic
	s.InsertCharacters(3)
	lines := s.Display()
	// 行内容应存在
	if strings.TrimSpace(lines[0]) == "" {
		t.Error("line should not be empty after InsertCharacters")
	}
}

// ==================== AlignmentDisplay 测试 ====================

func TestAlignmentDisplay(t *testing.T) {
	s := NewScreen(5, 3)
	s.AlignmentDisplay()
	lines := s.Display()
	for i, line := range lines {
		if line != "EEEEE" {
			t.Errorf("line %d = %q, want 'EEEEE'", i, line)
		}
	}
}
