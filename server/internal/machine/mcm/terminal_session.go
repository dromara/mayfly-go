package mcm

import (
	"context"
	"fmt"
	"io"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"

	"github.com/may-fly/cast"

	"strings"
	"time"
	"unicode/utf8"

	"github.com/gorilla/websocket"
)

const (
	Resize = 1
	Data   = 2
	Ping   = 3

	MsgSplit = "|"
)

type TerminalSession struct {
	ID       string
	wsConn   *websocket.Conn
	terminal *Terminal
	handler  *TerminalHandler
	recorder *Recorder
	ctx      context.Context
	cancel   context.CancelFunc
	dataChan chan rune
	tick     *time.Ticker
}

type CreateTerminalSessionParam struct {
	SessionId      string
	Cli            *Cli
	WsConn         *websocket.Conn
	Rows           int
	Cols           int
	Recorder       *Recorder
	LogCmd         bool            // 是否记录命令
	CmdFilterFuncs []CmdFilterFunc // 命令过滤器
}

func NewTerminalSession(param *CreateTerminalSessionParam) (*TerminalSession, error) {
	sessionId, rows, cols := param.SessionId, param.Rows, param.Cols
	cli, ws, recorder := param.Cli, param.WsConn, param.Recorder

	terminal, err := NewTerminal(cli)
	if err != nil {
		return nil, err
	}
	err = terminal.RequestPty("xterm-256color", rows, cols)
	if err != nil {
		return nil, err
	}
	err = terminal.Shell()
	if err != nil {
		return nil, err
	}

	if recorder != nil {
		recorder.WriteHeader(rows-3, cols)
	}

	var handler *TerminalHandler
	// 记录命令或者存在命令过滤器时，则创建对应的终端处理器
	if param.LogCmd || param.CmdFilterFuncs != nil {
		handler = &TerminalHandler{Parser: NewParser(120, 40), Filters: param.CmdFilterFuncs}
	}

	ctx, cancel := context.WithCancel(context.Background())
	tick := time.NewTicker(time.Millisecond * time.Duration(60))
	ts := &TerminalSession{
		ID:       sessionId,
		wsConn:   ws,
		terminal: terminal,
		handler:  handler,
		recorder: recorder,
		ctx:      ctx,
		cancel:   cancel,
		dataChan: make(chan rune),
		tick:     tick,
	}

	// 清除终端内容
	ts.WriteToWs("\033[2J\033[3J\033[1;1H")
	return ts, nil
}

func (r TerminalSession) Start() {
	go r.readFromTerminal()
	go r.writeToWebsocket()
	r.receiveWsMsg()
}

func (r TerminalSession) Stop() {
	logx.Debug("close machine ssh terminal session")
	r.tick.Stop()
	r.cancel()
	if r.terminal != nil {
		if err := r.terminal.Close(); err != nil {
			if err != io.EOF {
				logx.Errorf("failed to close the machine ssh terminal: %s", err.Error())
			}
		}
	}
}

// 获取终端会话执行的所有命令
func (r TerminalSession) GetExecCmds() []*ExecutedCmd {
	if r.handler != nil {
		return r.handler.ExecutedCmds
	}
	return []*ExecutedCmd{}
}

func (ts TerminalSession) readFromTerminal() {
	for {
		select {
		case <-ts.ctx.Done():
			return
		default:
			rn, size, err := ts.terminal.ReadRune()
			if err != nil {
				if err != io.EOF {
					logx.Error("the machine ssh terminal failed to read the message: ", err)
				}
				return
			}
			if size > 0 {
				ts.dataChan <- rn
			}
		}
	}
}

func (ts TerminalSession) writeToWebsocket() {
	var buf []byte
	for {
		select {
		case <-ts.ctx.Done():
			return
		case <-ts.tick.C:
			if len(buf) == 0 {
				continue
			}
			if ts.handler != nil {
				ts.handler.HandleRead(buf)
			}

			s := string(buf)
			if err := ts.WriteToWs(s); err != nil {
				logx.Error("the machine ssh endpoint failed to send a message to the websocket: ", err)
				return
			}

			// 如果记录器存在，则记录操作回放信息
			if ts.recorder != nil {
				ts.recorder.Lock()
				ts.recorder.WriteData(OutPutType, s)
				ts.recorder.Unlock()
			}

			buf = []byte{}
		case data := <-ts.dataChan:
			if data != utf8.RuneError {
				p := make([]byte, utf8.RuneLen(data))
				utf8.EncodeRune(p, data)
				buf = append(buf, p...)
			} else {
				buf = append(buf, []byte("@")...)
			}
		}
	}
}

type WsMsg struct {
	Type int    `json:"type"`
	Msg  string `json:"msg"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

// receiveWsMsg 接收客户端ws发送过来的消息，并写入终端会话中。
func (ts *TerminalSession) receiveWsMsg() {
	for {
		select {
		case <-ts.ctx.Done():
			return
		default:
			// read websocket msg
			_, wsData, err := ts.wsConn.ReadMessage()
			if err != nil {
				logx.Debugf("the machine ssh terminal failed to read the websocket message: %s", err.Error())
				return
			}
			// 解析消息
			msgObj, err := parseMsg(wsData)
			if err != nil {
				ts.WriteToWs(GetErrorContentRn("failed to parse the message content..."))
				logx.Error("machine ssh terminal message parsing failed: ", err)
				return
			}

			switch msgObj.Type {
			case Resize:
				if msgObj.Cols > 0 && msgObj.Rows > 0 {
					if err := ts.terminal.WindowChange(msgObj.Rows, msgObj.Cols); err != nil {
						logx.Error("ssh pty change windows size failed")
					}
				}
			case Data:
				data := []byte(msgObj.Msg)
				if ts.handler != nil {
					if err := ts.handler.PreWriteHandle(data); err != nil {
						ts.WriteToWs(err.Error())
						// 发送命令终止指令
						ts.terminal.Write([]byte{EOT})
						continue
					}
				}

				_, err := ts.terminal.Write([]byte(msgObj.Msg))
				if err != nil {
					logx.Errorf("failed to write data to the ssh terminal: %s", err)
					ts.WriteToWs(GetErrorContentRn(fmt.Sprintf("failed to write data to the ssh terminal: %s", err.Error())))
				}
			case Ping:
				_, err := ts.terminal.SshSession.SendRequest("ping", true, nil)
				if err != nil {
					ts.WriteToWs(GetErrorContentRn("the terminal connection has been disconnected..."))
					return
				}
			}
		}
	}
}

// WriteToWs 将消息写入websocket连接
func (ts *TerminalSession) WriteToWs(msg string) error {
	return ts.wsConn.WriteMessage(websocket.TextMessage, []byte(msg))
}

// 解析消息
func parseMsg(msg []byte) (*WsMsg, error) {
	// 消息格式为 msgType|msgContent， 如果msgType为resize则为msgType|rows|cols
	msgStr := string(msg)
	// 查找第一个 "|" 的位置
	index := strings.Index(msgStr, MsgSplit)
	if index == -1 {
		return nil, errorx.NewBiz("the message content does not conform to the specified rules")
	}

	// 获取消息类型, 提取第一个 "|" 之前的内容
	msgType := cast.ToIntD(msgStr[:index], Ping)
	// 其余内容则为消息内容
	msgContent := msgStr[index+1:]

	wsMsg := &WsMsg{Type: msgType, Msg: msgContent}
	if msgType == Resize {
		rowsAndCols := strings.Split(msgContent, MsgSplit)
		wsMsg.Rows = cast.ToIntD(rowsAndCols[0], 80)
		wsMsg.Cols = cast.ToIntD(rowsAndCols[1], 80)
	}
	return wsMsg, nil
}
