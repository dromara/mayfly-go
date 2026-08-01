package client

import (
	"fmt"
	"mayfly-go/cli/i18n"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

// 消息类型
const (
	MsgTypeResize = 1
	MsgTypeData   = 2
	MsgTypePing   = 3
)

// SSHSession SSH 终端会话
type SSHSession struct {
	conn      *websocket.Conn
	serverURL string
	token     string
	machineId uint64
	authCert  string
	cols      int
	rows      int
	done      chan struct{}
}

// NewSSHSession 创建新的 SSH 终端会话
func NewSSHSession(serverURL, token, authCert string, machineId uint64) *SSHSession {
	return &SSHSession{
		serverURL: serverURL,
		token:     token,
		machineId: machineId,
		authCert:  authCert,
		cols:      80,
		rows:      24,
		done:      make(chan struct{}),
	}
}

// Start 启动终端会话
func (s *SSHSession) Start() error {
	// 获取终端尺寸
	s.updateTerminalSize()

	// 连接 WebSocket
	if err := s.connect(); err != nil {
		return fmt.Errorf("%s: %w", i18n.T(i18n.MsgClientTerminalConnFailed), err)
	}
	defer s.conn.Close()

	// 设置终端为原始模式
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T(i18n.MsgClientTerminalModeFailed), err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// 清屏并显示光标
	fmt.Print("\033[?25h\033[2J\033[H")

	// 监听窗口大小变化
	s.listenResize()

	// 启动心跳
	go s.startHeartbeat()

	// 启动读取服务器数据的协程
	go s.readFromServer()

	// 主循环：读取stdin并发送到服务器
	s.writeToServer()

	// 恢复终端并显示光标
	fmt.Print("\033[?25h\r\n")
	return nil
}

// connect 连接 WebSocket
func (s *SSHSession) connect() error {
	wsURL := fmt.Sprintf("%s/api/machines/terminal/%s", s.serverURL, s.authCert)
	u, err := url.Parse(wsURL)
	if err != nil {
		return err
	}

	// 根据协议选择 ws 或 wss
	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}

	// 添加参数
	q := u.Query()
	q.Set("rows", fmt.Sprintf("%d", s.rows))
	q.Set("cols", fmt.Sprintf("%d", s.cols))
	u.RawQuery = q.Encode()

	// 设置请求头
	header := http.Header{}
	header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))

	// 连接
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		return err
	}

	s.conn = conn
	return nil
}

// readFromServer 从服务器读取数据并写入终端
func (s *SSHSession) readFromServer() {
	defer func() {
		// 确保 done 通道被关闭
		select {
		case <-s.done:
		default:
			close(s.done)
		}
	}()

	for {
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			// 忽略正常关闭和 EOF 错误
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				return
			}
			return
		}

		// 检测错误消息，如果是错误就关闭连接
		msgStr := string(message)
		if strings.Contains(msgStr, "failed to write data to the ssh terminal") {
			// 收到错误消息，说明 SSH 终端已关闭
			os.Stdout.Write([]byte("\r\n"))
			return
		}

		os.Stdout.Write(message)
	}
}

// writeToServer 从终端读取用户输入并发送到服务器
func (s *SSHSession) writeToServer() {
	// 创建一个 channel 来接收 stdin 输入
	inputCh := make(chan []byte, 1)
	// 创建一个 channel 来通知 stdin 读取 goroutine 停止
	stopCh := make(chan struct{})

	// 启动 goroutine 读取 stdin
	go func() {
		defer close(inputCh)
		buf := make([]byte, 1024)
		for {
			select {
			case <-stopCh:
				return
			default:
			}

			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}

			if n > 0 {
				// 复制数据
				data := make([]byte, n)
				copy(data, buf[:n])
				select {
				case inputCh <- data:
				case <-stopCh:
					return
				}
			}
		}
	}()

	defer close(stopCh)

	for {
		select {
		case data, ok := <-inputCh:
			if !ok {
				// stdin 关闭
				return
			}

			// 再次检查连接状态，防止竞态条件
			select {
			case <-s.done:
				return
			default:
			}

			message := fmt.Sprintf("%d|%s", MsgTypeData, string(data))
			if err := s.conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
				// 写入失败，静默退出，不输出错误
				return
			}
		case <-s.done:
			return
		}
	}
}

// updateTerminalSize 更新终端尺寸
func (s *SSHSession) updateTerminalSize() {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err == nil {
		s.cols = width / 8
		s.rows = height / 16

		if s.cols < 80 {
			s.cols = 80
		}
		if s.rows < 24 {
			s.rows = 24
		}
	}
}

// listenResize 监听窗口大小变化
func (s *SSHSession) listenResize() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)

	go func() {
		for range sigCh {
			s.updateTerminalSize()
			s.sendResize()
		}
	}()

	// 立即发送一次
	go func() {
		time.Sleep(100 * time.Millisecond)
		s.sendResize()
	}()
}

// sendResize 发送窗口大小调整
func (s *SSHSession) sendResize() {
	if s.conn == nil {
		return
	}
	message := fmt.Sprintf("%d|%d|%d", MsgTypeResize, s.rows, s.cols)
	// 静默发送，忽略错误
	s.conn.WriteMessage(websocket.TextMessage, []byte(message))
}

// startHeartbeat 启动心跳
func (s *SSHSession) startHeartbeat() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			message := fmt.Sprintf("%d|ping", MsgTypePing)
			// 静默发送，忽略错误
			s.conn.WriteMessage(websocket.TextMessage, []byte(message))
		case <-s.done:
			return
		}
	}
}

// GetAuthCertName 从机器信息中提取认证凭证名称
func GetAuthCertName(machine map[string]interface{}) string {
	// 从机器信息中获取 authCerts 数组
	if authCerts, ok := machine["authCerts"]; ok {
		if certs, ok := authCerts.([]interface{}); ok && len(certs) > 0 {
			// 取第一个凭证
			if cert, ok := certs[0].(map[string]interface{}); ok {
				if name, ok := cert["name"]; ok {
					return fmt.Sprintf("%v", name)
				}
			}
		}
	}

	// 如果没有 authCerts，尝试使用 id 作为备用
	if id, ok := machine["id"]; ok {
		return fmt.Sprintf("%v", id)
	}

	return ""
}
