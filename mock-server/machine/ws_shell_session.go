package machine

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

type safeBuffer struct {
	buffer bytes.Buffer
	mu     sync.Mutex
}

func (w *safeBuffer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.Write(p)
}
func (w *safeBuffer) Bytes() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.Bytes()
}
func (w *safeBuffer) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buffer.Reset()
}

const (
	wsMsgCmd    = "cmd"
	wsMsgResize = "resize"
)

type WsMsg struct {
	Type string `json:"type"`
	Msg  string `json:"msg"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

type LogicSshWsSession struct {
	stdinPipe       io.WriteCloser
	comboOutput     *safeBuffer //ssh 终端混合输出
	logBuff         *safeBuffer //保存session的日志
	inputFilterBuff *safeBuffer //用来过滤输入的命令和ssh_filter配置对比的
	session         *ssh.Session
	wsConn          *websocket.Conn
}

func NewLogicSshWsSession(cols, rows int, cli *Cli, wsConn *websocket.Conn) (*LogicSshWsSession, error) {
	sshSession, err := cli.GetSession()
	if err != nil {
		return nil, err
	}

	stdinP, err := sshSession.StdinPipe()
	if err != nil {
		return nil, err
	}

	comboWriter := new(safeBuffer)
	logBuf := new(safeBuffer)
	inputBuf := new(safeBuffer)
	//ssh.stdout and stderr will write output into comboWriter
	sshSession.Stdout = comboWriter
	sshSession.Stderr = comboWriter

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // disable echo
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	// Request pseudo terminal
	if err := sshSession.RequestPty("xterm", rows, cols, modes); err != nil {
		return nil, err
	}
	// Start remote shell
	if err := sshSession.Shell(); err != nil {
		return nil, err
	}
	//sshSession.Run("top")
	return &LogicSshWsSession{
		stdinPipe:       stdinP,
		comboOutput:     comboWriter,
		logBuff:         logBuf,
		inputFilterBuff: inputBuf,
		session:         sshSession,
		wsConn:          wsConn,
	}, nil
}

//Close 关闭
func (sws *LogicSshWsSession) Close() {
	if sws.session != nil {
		sws.session.Close()
	}
	if sws.logBuff != nil {
		sws.logBuff = nil
	}
	if sws.comboOutput != nil {
		sws.comboOutput = nil
	}
}
func (sws *LogicSshWsSession) Start(quitChan chan bool) {
	go sws.receiveWsMsg(quitChan)
	go sws.sendComboOutput(quitChan)
}

//receiveWsMsg  receive websocket msg do some handling then write into ssh.session.stdin
func (sws *LogicSshWsSession) receiveWsMsg(exitCh chan bool) {
	wsConn := sws.wsConn
	//tells other go routine quit
	defer setQuit(exitCh)
	for {
		select {
		case <-exitCh:
			return
		default:
			//read websocket msg
			_, wsData, err := wsConn.ReadMessage()
			if err != nil {
				logs.Error("reading webSocket message failed: ", err)
				return
			}
			//unmashal bytes into struct
			msgObj := WsMsg{}
			if err := json.Unmarshal(wsData, &msgObj); err != nil {
				logs.Error("unmarshal websocket message failed：", err)
			}
			switch msgObj.Type {
			case wsMsgResize:
				//handle xterm.js size change
				if msgObj.Cols > 0 && msgObj.Rows > 0 {
					if err := sws.session.WindowChange(msgObj.Rows, msgObj.Cols); err != nil {
						logs.Error("ssh pty change windows size failed")
					}
				}
			case wsMsgCmd:
				//handle xterm.js stdin
				//decodeBytes, err := base64.StdEncoding.DecodeString(msgObj.Cmd)
				//if err != nil {
				//	logs.Error("websock cmd string base64 decoding failed")
				//	//panic(base.NewBizErr("websock cmd string base64 decoding failed"))
				//}
				sws.sendWebsocketInputCommandToSshSessionStdinPipe([]byte(msgObj.Msg))
			}
		}
	}
}

//sendWebsocketInputCommandToSshSessionStdinPipe
func (sws *LogicSshWsSession) sendWebsocketInputCommandToSshSessionStdinPipe(cmdBytes []byte) {
	if _, err := sws.stdinPipe.Write(cmdBytes); err != nil {
		logs.Error("ws cmd bytes write to ssh.stdin pipe failed")
	}
}

func (sws *LogicSshWsSession) sendComboOutput(exitCh chan bool) {
	wsConn := sws.wsConn
	//todo 优化成一个方法
	//tells other go routine quit
	defer setQuit(exitCh)

	//every 120ms write combine output bytes into websocket response
	tick := time.NewTicker(time.Millisecond * time.Duration(60))
	//for range time.Tick(120 * time.Millisecond){}
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if sws.comboOutput == nil {
				return
			}
			bs := sws.comboOutput.Bytes()
			if len(bs) > 0 {
				err := wsConn.WriteMessage(websocket.TextMessage, bs)
				if err != nil {
					logs.Error("ssh sending combo output to webSocket failed")
				}
				_, err = sws.logBuff.Write(bs)
				if err != nil {
					logs.Error("combo output to log buffer failed")
				}
				sws.comboOutput.buffer.Reset()
			}

		case <-exitCh:
			return
		}
	}
}

func (sws *LogicSshWsSession) Wait(quitChan chan bool) {
	if err := sws.session.Wait(); err != nil {
		setQuit(quitChan)
	}
}

func (sws *LogicSshWsSession) LogString() string {
	return sws.logBuff.buffer.String()
}

func setQuit(ch chan bool) {
	ch <- true
}
