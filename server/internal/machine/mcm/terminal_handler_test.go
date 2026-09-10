package mcm

import (
	"strings"
	"sync"
	"testing"
)

// ==================== extractCmdAfterPrompt 测试 ====================

func TestExtractCmdAfterPrompt(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// 基本提示符
		{"root prompt", "root@host:~# ls -la", "ls -la"},
		{"dollar prompt", "user@host:~$ ls -la", "ls -la"},
		{"simple hash", "host# ls", "ls"},
		{"simple dollar", "$ ls", "ls"},
		{"simple gt", "> ls", "ls"},

		// 动态 PS1（含时间戳）—— 核心修复场景
		{
			"timestamp in PS1",
			"root@iZ2zeg7yvz4arzq052ys6qZ:~# 2026-09-10 10:13:49 ls",
			"2026-09-10 10:13:49 ls",
		},
		{
			"PS1 with embedded time",
			"[10:13:49] root@host:~# whoami",
			"whoami",
		},

		// 空命令
		{"empty cmd hash", "root@host:~# ", ""},
		{"empty cmd dollar", "user@host:~$ ", ""},
		{"empty cmd only", "root@host:~#", ""},

		// 复杂路径
		{"deep path", "root@host:/usr/local/bin# ls", "ls"},
		{"home tilde", "user@host:~/projects/go# make build", "make build"},

		// 多行内容
		{
			"multiline with prev output",
			"file1.txt\nroot@host:~# cat file.txt",
			"cat file.txt",
		},

		// 无提示符
		{"no prompt", "just some text", "just some text"},
		{"random text", "hello world", "hello world"},

		// 命令含特殊字符
		{"pipe", "root@host:~# cat /etc/passwd | grep root", "cat /etc/passwd | grep root"},
		{"redirect", "root@host:~# echo hello > file.txt", "echo hello > file.txt"},
		{"semicolon", "root@host:~# cd /tmp && ls -la", "cd /tmp && ls -la"},
		{"dollar in cmd", "root@host:~# echo $HOME", "echo $HOME"},
		{"hash in cmd arg", "root@host:~# echo 'hello # world'", "echo 'hello # world'"},

		// 前后空白
		{"leading spaces", "  root@host:~# ls  ", "ls"},
		{"trailing newline", "root@host:~# ls\n", "ls"},

		// 不同用户名
		{"admin user", "admin@server:~# systemctl status nginx", "systemctl status nginx"},
		{"ubuntu user", "ubuntu@ip-172-31-0-1:~$ docker ps", "docker ps"},

		// 长主机名（用户实际场景）
		{
			"long hostname",
			"root@iZ2zeg7yvz4arzq052ys6qZ:~# ls -la",
			"ls -la",
		},
		{
			"long hostname with path",
			"root@iZ2zeg7yvz4arzq052ys6qZ:/var/log# tail -f syslog",
			"tail -f syslog",
		},

		// PS2 风格提示符
		{
			"continuation prompt",
			"> echo hello",
			"echo hello",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCmdAfterPrompt(tt.input)
			if got != tt.want {
				t.Errorf("extractCmdAfterPrompt() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ==================== Parser.GetCmd() 测试（含 ANSI 终端模拟） ====================

func TestParserGetCmd(t *testing.T) {
	tests := []struct {
		name       string
		outputData string // 模拟终端输出
		want       string
	}{
		{
			name:       "simple prompt with command",
			outputData: "root@host:~# ls -la",
			want:       "ls -la",
		},
		{
			name:       "dollar prompt",
			outputData: "user@host:~$ whoami",
			want:       "whoami",
		},
		{
			name:       "prompt with colored ANSI",
			outputData: "\033[01;32mroot@host\033[0m:\033[01;34m~\033[0m# ls",
			want:       "ls",
		},
		{
			name:       "multiline output last line has command",
			outputData: "total 48\r\n-rw-r--r-- 1 root root 1234 Jan 1 00:00 file.txt\r\nroot@host:~# cat file.txt",
			want:       "cat file.txt",
		},
		{
			name:       "empty command after prompt",
			outputData: "root@host:~# ",
			want:       "",
		},
		{
			name:       "long hostname with command",
			outputData: "root@iZ2zeg7yvz4arzq052ys6qZ:~# docker ps -a",
			want:       "docker ps -a",
		},
		{
			name:       "deep path prompt",
			outputData: "root@server:/var/log/nginx# tail -f error.log",
			want:       "tail -f error.log",
		},
		{
			name:       "output with CRLF line endings",
			outputData: "line1\r\nline2\r\nroot@host:~# pwd",
			want:       "pwd",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(120, 40)
			p.OutputData = []byte(tt.outputData)
			got := p.GetCmd()
			if got != tt.want {
				t.Errorf("GetCmd() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ==================== 并发安全测试 ====================

func TestParserConcurrentAccess(t *testing.T) {
	p := NewParser(120, 40)
	p.OutputData = []byte("root@host:~# ")

	var wg sync.WaitGroup
	done := make(chan struct{})

	// 并发写入输出数据（模拟 readFromTerminal → HandleRead）
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			p.AppendOutData([]byte("some terminal output\r\n"))
		}
	}()

	// 并发读取命令（模拟 receiveWsMsg → PreWriteHandle）
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			_ = p.GetCmd()
		}
		close(done)
	}()

	// 并发获取输出
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			_ = p.GetOutput()
		}
	}()

	<-done
	wg.Wait()
}

// ==================== TerminalHandler.PreWriteHandle 端到端测试 ====================

func TestPreWriteHandle_BasicCommand(t *testing.T) {
	handler := &TerminalHandler{
		Parser: NewParser(120, 40),
	}
	// 模拟终端输出：提示符
	handler.Parser.OutputData = []byte("root@host:~# ")

	// 输入 'l'
	err := handler.PreWriteHandle([]byte{'l'})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handler.ExecutedCmds) != 0 {
		t.Fatalf("expected 0 commands, got %d", len(handler.ExecutedCmds))
	}

	// 输入 's'
	err = handler.PreWriteHandle([]byte{'s'})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handler.ExecutedCmds) != 0 {
		t.Fatalf("expected 0 commands, got %d", len(handler.ExecutedCmds))
	}

	// 输入回车 CR，同时终端已回显 "ls"
	// 重置 ANSI 终端状态后设置新的输出（模拟生产环境中 OutputData 增量累积的行为）
	handler.Parser.Output.Listener.Reset()
	handler.Parser.OutputData = []byte("root@host:~# ls")
	err = handler.PreWriteHandle([]byte{CR})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handler.ExecutedCmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(handler.ExecutedCmds))
	}
	if handler.ExecutedCmds[0].Cmd != "ls" {
		t.Errorf("expected command 'ls', got %q", handler.ExecutedCmds[0].Cmd)
	}
	if handler.ExecutedCmds[0].Time == 0 {
		t.Error("expected non-zero timestamp")
	}
}

func TestPreWriteHandle_EmptyCommand(t *testing.T) {
	handler := &TerminalHandler{
		Parser: NewParser(120, 40),
	}
	// 终端只有提示符，用户直接按回车
	handler.Parser.OutputData = []byte("root@host:~# ")
	handler.Parser.AppendInputData([]byte{CR})
	handler.Parser.Output.Listener.Reset()
	handler.Parser.OutputData = []byte("root@host:~# ")

	err := handler.PreWriteHandle([]byte{CR})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handler.ExecutedCmds) != 0 {
		t.Fatalf("empty command should not be recorded, got %d commands", len(handler.ExecutedCmds))
	}
}

func TestPreWriteHandle_CmdFilter(t *testing.T) {
	handler := &TerminalHandler{
		Parser: NewParser(120, 40),
		Filters: []CmdFilterFunc{
			func(cmd string) error {
				if strings.Contains(cmd, "rm -rf") {
					return &filterError{msg: "dangerous command blocked"}
				}
				return nil
			},
		},
	}
	handler.Parser.OutputData = []byte("root@host:~# ")
	handler.Parser.AppendInputData([]byte("rm -rf /"))
	handler.Parser.Output.Listener.Reset()
	handler.Parser.OutputData = []byte("root@host:~# rm -rf /")

	err := handler.PreWriteHandle([]byte{CR})
	if err == nil {
		t.Fatal("expected error for dangerous command")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("expected 'blocked' in error, got: %v", err)
	}
	if len(handler.ExecutedCmds) != 0 {
		t.Fatalf("filtered command should not be recorded, got %d", len(handler.ExecutedCmds))
	}
}

func TestPreWriteHandle_CmdFilterAllowsSafeCommand(t *testing.T) {
	handler := &TerminalHandler{
		Parser: NewParser(120, 40),
		Filters: []CmdFilterFunc{
			func(cmd string) error {
				if strings.Contains(cmd, "rm -rf") {
					return &filterError{msg: "dangerous command blocked"}
				}
				return nil
			},
		},
	}
	handler.Parser.OutputData = []byte("root@host:~# ")
	handler.Parser.AppendInputData([]byte("ls -la"))
	handler.Parser.Output.Listener.Reset()
	handler.Parser.OutputData = []byte("root@host:~# ls -la")

	err := handler.PreWriteHandle([]byte{CR})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handler.ExecutedCmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(handler.ExecutedCmds))
	}
	if handler.ExecutedCmds[0].Cmd != "ls -la" {
		t.Errorf("expected 'ls -la', got %q", handler.ExecutedCmds[0].Cmd)
	}
}

func TestPreWriteHandle_MultipleCommands(t *testing.T) {
	handler := &TerminalHandler{
		Parser: NewParser(120, 40),
	}

	// 第一个命令
	handler.Parser.OutputData = []byte("root@host:~# ")
	handler.Parser.AppendInputData([]byte("ls"))
	handler.Parser.Output.Listener.Reset()
	handler.Parser.OutputData = []byte("root@host:~# ls")
	_ = handler.PreWriteHandle([]byte{CR})

	// 第二个命令
	handler.Parser.Output.Listener.Reset()
	handler.Parser.OutputData = []byte("root@host:~# ")
	handler.Parser.AppendInputData([]byte("pwd"))
	handler.Parser.Output.Listener.Reset()
	handler.Parser.OutputData = []byte("root@host:~# pwd")
	_ = handler.PreWriteHandle([]byte{CR})

	// 第三个命令
	handler.Parser.Output.Listener.Reset()
	handler.Parser.OutputData = []byte("root@host:~# ")
	handler.Parser.AppendInputData([]byte("whoami"))
	handler.Parser.Output.Listener.Reset()
	handler.Parser.OutputData = []byte("root@host:~# whoami")
	_ = handler.PreWriteHandle([]byte{CR})

	if len(handler.ExecutedCmds) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(handler.ExecutedCmds))
	}
	expected := []string{"ls", "pwd", "whoami"}
	for i, cmd := range handler.ExecutedCmds {
		if cmd.Cmd != expected[i] {
			t.Errorf("command[%d] = %q, want %q", i, cmd.Cmd, expected[i])
		}
	}
}

func TestPreWriteHandle_DynamicPS1WithTimestamp(t *testing.T) {
	// 核心修复场景：PS1 包含动态时间戳，旧代码 TrimPrefix 会失败
	handler := &TerminalHandler{
		Parser: NewParser(120, 40),
	}

	// 初始提示符（时间戳 T1）
	handler.Parser.OutputData = []byte("root@host:~# ")
	handler.Parser.AppendInputData([]byte{'l'})

	// 终端输出更新，时间戳变为 T2（不同于 T1）
	handler.Parser.Output.Listener.Reset()
	handler.Parser.OutputData = []byte("2026-09-10 10:13:49\r\nroot@host:~# ls")

	err := handler.PreWriteHandle([]byte{CR})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handler.ExecutedCmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(handler.ExecutedCmds))
	}
	cmd := handler.ExecutedCmds[0].Cmd
	if cmd != "ls" {
		t.Errorf("expected 'ls', got %q (should not contain timestamp or prompt)", cmd)
	}
	if strings.Contains(cmd, "2026") {
		t.Errorf("command should not contain timestamp: %q", cmd)
	}
	if strings.Contains(cmd, "root@") {
		t.Errorf("command should not contain prompt: %q", cmd)
	}
}

func TestPreWriteHandle_LongHostnameNoPromptLeak(t *testing.T) {
	handler := &TerminalHandler{
		Parser: NewParser(120, 40),
	}
	handler.Parser.OutputData = []byte("root@iZ2zeg7yvz4arzq052ys6qZ:~# ")
	handler.Parser.AppendInputData([]byte("docker ps"))
	handler.Parser.Output.Listener.Reset()
	handler.Parser.OutputData = []byte("root@iZ2zeg7yvz4arzq052ys6qZ:~# docker ps")

	err := handler.PreWriteHandle([]byte{CR})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handler.ExecutedCmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(handler.ExecutedCmds))
	}
	cmd := handler.ExecutedCmds[0].Cmd
	if cmd != "docker ps" {
		t.Errorf("expected 'docker ps', got %q", cmd)
	}
	if strings.Contains(cmd, "iZ2zeg7yvz4arzq052ys6qZ") {
		t.Errorf("command should not contain hostname: %q", cmd)
	}
}

// ==================== Parser.Reset 测试 ====================

func TestParserReset(t *testing.T) {
	p := NewParser(120, 40)
	p.InputData = []byte("test input")
	p.OutputData = []byte("test output")
	p.Ps1 = "root@host:~# "

	p.Reset()

	if len(p.InputData) != 0 {
		t.Error("InputData should be nil after reset")
	}
	if len(p.OutputData) != 0 {
		t.Error("OutputData should be nil after reset")
	}
	if p.Ps1 != "root@host:~# " {
		t.Errorf("Ps1 should be preserved after reset, got %q", p.Ps1)
	}
}

// ==================== Parser.GetOutput 测试 ====================

func TestParserGetOutput(t *testing.T) {
	p := NewParser(120, 40)
	p.OutputData = []byte("line1\r\nline2\r\nline3")

	got := p.GetOutput()
	if got != "line3" {
		t.Errorf("GetOutput() = %q, want 'line3'", got)
	}
}

func TestParserGetOutput_Empty(t *testing.T) {
	p := NewParser(120, 40)
	got := p.GetOutput()
	if got != "" {
		t.Errorf("GetOutput() = %q, want ''", got)
	}
}

func TestParserGetOutput_SkipsEmptyLines(t *testing.T) {
	p := NewParser(120, 40)
	p.OutputData = []byte("line1\r\n\r\n\r\nline2")

	got := p.GetOutput()
	if got != "line2" {
		t.Errorf("GetOutput() = %q, want 'line2'", got)
	}
}

// ==================== ParseMsg 测试 ====================

func TestParseMsg_DataMsg(t *testing.T) {
	msg := []byte("2|ls -la")
	wsMsg, err := ParseMsg(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wsMsg.Type != MsgTypeData {
		t.Errorf("type = %d, want %d", wsMsg.Type, MsgTypeData)
	}
	if wsMsg.Msg != "ls -la" {
		t.Errorf("msg = %q, want 'ls -la'", wsMsg.Msg)
	}
}

func TestParseMsg_ResizeMsg(t *testing.T) {
	msg := []byte("1|120|40")
	wsMsg, err := ParseMsg(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wsMsg.Type != MsgTypeResize {
		t.Errorf("type = %d, want %d", wsMsg.Type, MsgTypeResize)
	}
	if wsMsg.Rows != 120 {
		t.Errorf("rows = %d, want 120", wsMsg.Rows)
	}
	if wsMsg.Cols != 40 {
		t.Errorf("cols = %d, want 40", wsMsg.Cols)
	}
}

func TestParseMsg_PingMsg(t *testing.T) {
	msg := []byte("3|")
	wsMsg, err := ParseMsg(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wsMsg.Type != MsgTypePing {
		t.Errorf("type = %d, want %d", wsMsg.Type, MsgTypePing)
	}
}

func TestParseMsg_InvalidFormat(t *testing.T) {
	msg := []byte("invalid message without separator")
	_, err := ParseMsg(msg)
	if err == nil {
		t.Fatal("expected error for invalid message format")
	}
}

// ==================== Vim 状态测试 ====================

func TestParserState_VimEnterExit(t *testing.T) {
	p := NewParser(120, 40)

	// 隐藏光标（进入编辑模式且非新屏幕）
	result := p.State([]byte("\x1b[?25l"))
	if !result {
		t.Error("hide cursor should set vimState to true")
	}

	// 显示光标（退出编辑模式）
	result = p.State([]byte("\x1b[?25h"))
	if result {
		t.Error("show cursor should set vimState to false")
	}
}

func TestParserState_AlternateScreenEntersVim(t *testing.T) {
	p := NewParser(120, 40)
	// \x1b[?1049h 匹配 enterMarks 但不匹配 screenMarks，vimState 被设为 true
	result := p.State([]byte("\x1b[?1049h"))
	if !result {
		t.Error("\\x1b[?1049h should enter vim state")
	}
}

// ==================== filterError (test helper) ====================

type filterError struct {
	msg string
}

func (e *filterError) Error() string {
	return e.msg
}
