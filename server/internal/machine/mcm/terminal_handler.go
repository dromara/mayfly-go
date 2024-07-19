package mcm

import (
	"bytes"
	"fmt"
	"mayfly-go/pkg/errorx"
	"strings"
	"time"

	"github.com/veops/go-ansiterm"
)

// 命令过滤函数，若返回error，则不执行该命令
type CmdFilterFunc func(cmd string) error

const (
	CR  = 0x0d // 单个字节 13 通常表示发送一个 CR（Carriage Return，回车）字符 \r
	EOT = 0x03 // 通过向标准输入发送单个字节 3 通常表示发送一个 EOT（End of Transmission）信号。EOT 是一种控制字符，在通信中用于指示数据传输的结束。发送 EOT 信号可以被用来终止当前的交互或数据传输
)

type ExecutedCmd struct {
	Cmd  string `json:"cmd"`  // 执行的命令
	Time int64  `json:"time"` // 执行时间戳
}

// TerminalHandler 终端处理器
type TerminalHandler struct {
	Filters      []CmdFilterFunc
	ExecutedCmds []*ExecutedCmd // 已执行的命令

	Parser *Parser
}

// PreWriteHandle 写入数据至终端前的处理，可进行过滤等操作
func (tf *TerminalHandler) PreWriteHandle(p []byte) error {
	tf.Parser.AppendInputData(p)

	// 不是回车命令，则表示命令未结束
	if bytes.LastIndex(p, []byte{CR}) != 0 {
		return nil
	}

	// time.Sleep(time.Millisecond * 30)
	command := tf.Parser.GetCmd()
	// 重置终端输入输出
	tf.Parser.Reset()

	if command == "" {
		return nil
	}

	// 执行命令过滤器
	for _, filter := range tf.Filters {
		if err := filter(command); err != nil {
			msg := fmt.Sprintf("\r\n%s%s", tf.Parser.Ps1, GetErrorContent(err.Error()))
			return errorx.NewBiz(msg)
		}
	}

	// 记录执行命令
	tf.ExecutedCmds = append(tf.ExecutedCmds, &ExecutedCmd{
		Cmd:  command,
		Time: time.Now().Unix(),
	})
	return nil
}

// HandleRead 处理从终端读取的数据进行操作
func (tf *TerminalHandler) HandleRead(data []byte) error {
	tf.Parser.AppendOutData(data)
	return nil
}

type Parser struct {
	Output     *ansiterm.ByteStream
	InputData  []byte
	OutputData []byte
	Ps1        string

	vimState     bool
	commandState bool
}

func NewParser(width, height int) *Parser {
	return &Parser{
		Output:       NewParserByteStream(width, height),
		vimState:     false,
		commandState: true,
	}
}

func NewParserByteStream(width, height int) *ansiterm.ByteStream {
	screen := ansiterm.NewScreen(width, height)
	stream := ansiterm.InitByteStream(screen, false)
	stream.Attach(screen)
	return stream
}

var (
	enterMarks = [][]byte{
		[]byte("\x1b[?1049h"), // 从备用屏幕缓冲区恢复屏幕内容
		[]byte("\x1b[?1048h"),
		[]byte("\x1b[?1047h"),
		[]byte("\x1b[?47h"),
		[]byte("\x1b[?25l"), // 隐藏光标
	}

	exitMarks = [][]byte{
		[]byte("\x1b[?1049l"), // 从备用屏幕缓冲区恢复屏幕内容
		[]byte("\x1b[?1048l"),
		[]byte("\x1b[?1047l"),
		[]byte("\x1b[?47l"),
		[]byte("\x1b[?25h"), // 显示光标
	}

	screenMarks = [][]byte{
		{0x1b, 0x5b, 0x4b, 0x0d, 0x0a},
		{0x1b, 0x5b, 0x34, 0x6c},
	}
)

func (p *Parser) AppendInputData(data []byte) {
	if len(p.InputData) == 0 {
		// 如 "root@cloud-s0ervh-hh87:~# " 获取前一段用户名等提示内容
		p.Ps1 = p.GetOutput()
	}
	p.InputData = append(p.InputData, data...)
}

func (p *Parser) AppendOutData(data []byte) {
	// 非编辑等状态，则追加输出内容
	if !p.State(data) {
		p.OutputData = append(p.OutputData, data...)
	}
}

// GetCmd 获取执行的命令
func (p *Parser) GetCmd() string {
	// "root@cloud-s0ervh-hh87:~# ls"
	s := p.GetOutput()
	// Ps1 = "root@cloud-s0ervh-hh87:~# "
	return strings.TrimPrefix(s, p.Ps1)
}

func (p *Parser) Reset() {
	p.Output.Listener.Reset()
	p.OutputData = nil
	p.InputData = nil
}

func (p *Parser) GetOutput() string {
	p.Output.Feed(p.OutputData)

	res := parseOutput(p.Output.Listener.Display())
	if len(res) == 0 {
		return ""
	}
	return res[len(res)-1]
}

func parseOutput(data []string) (output []string) {
	for _, line := range data {
		if strings.TrimSpace(line) != "" {
			output = append(output, line)
		}
	}
	return output
}

func (p *Parser) State(b []byte) bool {
	if !p.vimState && IsEditEnterMode(b) {
		if !isNewScreen(b) {
			p.vimState = true
			p.commandState = false
		}
	}
	if p.vimState && IsEditExitMode(b) {
		// 重置终端输入输出
		p.Reset()
		p.vimState = false
		p.commandState = true
	}
	return p.vimState
}

func isNewScreen(p []byte) bool {
	return matchMark(p, screenMarks)
}

func IsEditEnterMode(p []byte) bool {
	return matchMark(p, enterMarks)
}

func IsEditExitMode(p []byte) bool {
	return matchMark(p, exitMarks)
}

func matchMark(p []byte, marks [][]byte) bool {
	for _, item := range marks {
		if bytes.Contains(p, item) {
			return true
		}
	}
	return false
}

// GetErrorContent 包装返回终端错误消息
func GetErrorContent(msg string) string {
	return fmt.Sprintf("\033[1;31m%s\033[0m", msg)
}

// GetErrorContentRn 包装返回终端错误消息, 并自动回车换行
func GetErrorContentRn(msg string) string {
	return fmt.Sprintf("\r\n%s", GetErrorContent(msg))
}
