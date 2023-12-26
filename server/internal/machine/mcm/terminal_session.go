package mcm

import (
	"context"
	"encoding/json"
	"io"
	"mayfly-go/pkg/logx"
	"time"
	"unicode/utf8"

	"github.com/gorilla/websocket"
)

const (
	Resize = 1
	Data   = 2
	Ping   = 3
)

type TerminalSession struct {
	ID       string
	wsConn   *websocket.Conn
	terminal *Terminal
	recorder *Recorder
	ctx      context.Context
	cancel   context.CancelFunc
	dataChan chan rune
	tick     *time.Ticker
}

func NewTerminalSession(sessionId string, ws *websocket.Conn, cli *Cli, rows, cols int, recorder *Recorder) (*TerminalSession, error) {
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

	ctx, cancel := context.WithCancel(context.Background())
	tick := time.NewTicker(time.Millisecond * time.Duration(60))
	ts := &TerminalSession{
		ID:       sessionId,
		wsConn:   ws,
		terminal: terminal,
		recorder: recorder,
		ctx:      ctx,
		cancel:   cancel,
		dataChan: make(chan rune),
		tick:     tick,
	}
	return ts, nil
}

func (r TerminalSession) Start() {
	go r.readFormTerminal()
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
				logx.Errorf("关闭机器ssh终端失败: %s", err.Error())
			}
		}
	}
}

func (ts TerminalSession) readFormTerminal() {
	for {
		select {
		case <-ts.ctx.Done():
			return
		default:
			rn, size, err := ts.terminal.ReadRune()
			if err != nil {
				if err != io.EOF {
					logx.Error("机器ssh终端读取消息失败: ", err)
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
			if len(buf) > 0 {
				s := string(buf)
				if err := WriteMessage(ts.wsConn, s); err != nil {
					logx.Error("机器ssh终端发送消息至websocket失败: ", err)
					return
				}
				// 如果记录器存在，则记录操作回放信息
				if ts.recorder != nil {
					ts.recorder.Lock()
					ts.recorder.WriteData(OutPutType, s)
					ts.recorder.Unlock()
				}
				buf = []byte{}
			}
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

// 接收客户端ws发送过来的消息，并写入终端会话中。
func (ts *TerminalSession) receiveWsMsg() {
	wsConn := ts.wsConn
	for {
		select {
		case <-ts.ctx.Done():
			return
		default:
			// read websocket msg
			_, wsData, err := wsConn.ReadMessage()
			if err != nil {
				logx.Debugf("机器ssh终端读取websocket消息失败: %s", err.Error())
				return
			}
			// 解析消息
			msgObj := WsMsg{}
			if err := json.Unmarshal(wsData, &msgObj); err != nil {
				logx.Error("机器ssh终端消息解析失败: ", err)
			}
			switch msgObj.Type {
			case Resize:
				if msgObj.Cols > 0 && msgObj.Rows > 0 {
					if err := ts.terminal.WindowChange(msgObj.Rows, msgObj.Cols); err != nil {
						logx.Error("ssh pty change windows size failed")
					}
				}
			case Data:
				_, err := ts.terminal.Write([]byte(msgObj.Msg))
				if err != nil {
					logx.Debugf("机器ssh终端写入消息失败: %s", err)
				}
			case Ping:
				_, err := ts.terminal.SshSession.SendRequest("ping", true, nil)
				if err != nil {
					WriteMessage(wsConn, "\r\n\033[1;31m提示: 终端连接已断开...\033[0m")
					return
				}
			}
		}
	}
}

func WriteMessage(ws *websocket.Conn, msg string) error {
	return ws.WriteMessage(websocket.TextMessage, []byte(msg))
}
