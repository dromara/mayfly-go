package guac

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mayfly-go/internal/machine/config"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"net"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// ReconnectableTunnel 支持自动重连的 Tunnel 包装器
type ReconnectableTunnel struct {
	tunnel      Tunnel
	query       url.Values
	parameters  map[string]string
	username    string
	maxRetries  int
	retryCount  atomic.Int32
	lastError   error
	closed      atomic.Bool
	reconnectMu sync.Mutex // 防止并发重连
}

// NewReconnectableTunnel 创建一个新的可重连 Tunnel
func NewReconnectableTunnel(tunnel Tunnel, query url.Values, parameters map[string]string, username string) *ReconnectableTunnel {
	return &ReconnectableTunnel{
		tunnel:     tunnel,
		query:      query,
		parameters: parameters,
		username:   username,
		maxRetries: 5,
	}
}

// AcquireReader 获取读取器
func (rt *ReconnectableTunnel) AcquireReader() InstructionReader {
	return rt.tunnel.AcquireReader()
}

// ReleaseReader 释放读取器
func (rt *ReconnectableTunnel) ReleaseReader() {
	rt.tunnel.ReleaseReader()
}

// HasQueuedReaderThreads 检查是否有排队的读取线程
func (rt *ReconnectableTunnel) HasQueuedReaderThreads() bool {
	return rt.tunnel.HasQueuedReaderThreads()
}

// AcquireWriter 获取写入器
func (rt *ReconnectableTunnel) AcquireWriter() io.Writer {
	return rt.tunnel.AcquireWriter()
}

// ReleaseWriter 释放写入器
func (rt *ReconnectableTunnel) ReleaseWriter() {
	rt.tunnel.ReleaseWriter()
}

// HasQueuedWriterThreads 检查是否有排队的写入线程
func (rt *ReconnectableTunnel) HasQueuedWriterThreads() bool {
	return rt.tunnel.HasQueuedWriterThreads()
}

// GetUUID 获取隧道 UUID
func (rt *ReconnectableTunnel) GetUUID() string {
	return rt.tunnel.GetUUID()
}

// ConnectionID 获取连接 ID
func (rt *ReconnectableTunnel) ConnectionID() string {
	return rt.tunnel.ConnectionID()
}

// Close 关闭隧道
func (rt *ReconnectableTunnel) Close() error {
	rt.closed.Store(true)
	return rt.tunnel.Close()
}

// IsClosed 检查隧道是否已关闭
func (rt *ReconnectableTunnel) IsClosed() bool {
	return rt.closed.Load()
}

// Reconnect 尝试重新连接（线程安全）
func (rt *ReconnectableTunnel) Reconnect() error {
	// 使用互斥锁防止并发重连
	rt.reconnectMu.Lock()
	defer rt.reconnectMu.Unlock()

	if rt.closed.Load() {
		return errors.New("tunnel is closed")
	}

	retryCount := rt.retryCount.Load()
	if retryCount >= int32(rt.maxRetries) {
		logx.Warnf("max retries (%d) reached, giving up", rt.maxRetries)
		return errors.New("max retries reached")
	}

	// 关闭当前隧道
	_ = rt.tunnel.Close()

	retryCount++
	rt.retryCount.Store(retryCount)
	logx.Warnf("reconnecting to guacd (attempt %d/%d)", retryCount, rt.maxRetries)

	// 等待一小段时间后重试
	time.Sleep(time.Second * time.Duration(retryCount))

	// 重新建立连接
	newTunnel, err := DoConnectWithoutRetry(rt.query, rt.parameters, rt.username)
	if err != nil {
		rt.lastError = err
		logx.Errorf("reconnect failed: %v", err)
		return err
	}

	rt.tunnel = newTunnel
	rt.retryCount.Store(0) // 重置计数
	logx.Info("reconnected to guacd successfully")
	return nil
}

// GetLastError 获取最后错误
func (rt *ReconnectableTunnel) GetLastError() error {
	return rt.lastError
}

// DoConnect 创建支持重连的 Tunnel
func DoConnect(query url.Values, parameters map[string]string, username string) (Tunnel, error) {
	tunnel, err := DoConnectWithoutRetry(query, parameters, username)
	if err != nil {
		return nil, err
	}
	return NewReconnectableTunnel(tunnel, query, parameters, username), nil
}

// DoConnectWithoutRetry 原始的连接函数（不支持重连）
func DoConnectWithoutRetry(query url.Values, parameters map[string]string, username string) (Tunnel, error) {
	conf := NewGuacamoleConfiguration()

	// 创建 parameters 的副本，避免并发修改导致的 "concurrent map writes" 错误
	paramsCopy := make(map[string]string, len(parameters)+20)
	for k, v := range parameters {
		paramsCopy[k] = v
	}
	parameters = paramsCopy

	parameters["client-name"] = "mayfly"
	parameters["enable-wallpaper"] = "true"
	parameters["resize-method"] = "display-update"
	parameters["enable-font-smoothing"] = "true"
	parameters["enable-desktop-composition"] = "false"
	parameters["enable-menu-animations"] = "false"
	parameters["disable-bitmap-caching"] = "true"
	parameters["disable-offscreen-caching"] = "true"
	parameters["force-lossless"] = "true" // 无损压缩
	parameters["color-depth"] = "32"      //32 真彩（32 位）；24 真彩（24 位）；16 低色（16 位）；8 256 色

	// drive
	parameters["enable-drive"] = "true"
	parameters["drive-name"] = "Filesystem"
	parameters["create-drive-path"] = "true"
	parameters["drive-path"] = fmt.Sprintf("/rdp-file/%s", username)

	conf.Protocol = parameters["scheme"]
	conf.Parameters = parameters
	conf.OptimalScreenWidth = 800
	conf.OptimalScreenHeight = 600

	var err error

	if query.Get("width") != "" {
		conf.OptimalScreenWidth, err = strconv.Atoi(query.Get("width"))
		if err != nil || conf.OptimalScreenWidth == 0 {
			logx.Error("Invalid width")
			conf.OptimalScreenWidth = 800
		}
	}

	if query.Get("height") != "" {
		conf.OptimalScreenHeight, err = strconv.Atoi(query.Get("height"))
		if err != nil || conf.OptimalScreenHeight == 0 {
			logx.Error("Invalid height")
			conf.OptimalScreenHeight = 600
		}
	}

	//conf.ConnectionID = uuid.New().String()

	conf.AudioMimetypes = []string{"audio/L16", "rate=44100", "channels=2"}
	conf.ImageMimetypes = []string{"image/jpeg", "image/png", "image/webp"}

	logx.Debug("Connecting to guacd")

	machineConfig := config.GetMachine()
	if machineConfig.GuacdHost == "" {
		return nil, errorx.NewBiz("please configure guacd connection info in 'System Config - Machine Config'")
	}
	guacdAddr := fmt.Sprintf("%v:%v", machineConfig.GuacdHost, machineConfig.GuacdPort)
	addr, err := net.ResolveTCPAddr("tcp", guacdAddr)
	if err != nil {
		logx.Error("error resolving guacd address", err)
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		logx.Error("error while connecting to guacd", err)
		return nil, err
	}

	stream := NewStream(conn, SocketTimeout)

	logx.Debug("Connected to guacd")
	//conf.ConnectionID = uuid.New().String()

	logx.Debugf("Starting handshake with %#v", conf)
	err = stream.Handshake(conf)
	if err != nil {
		return nil, err
	}
	logx.Debug("Socket configured")
	return NewSimpleTunnel(stream), nil
}

// WsToGuacd 处理从 WebSocket 到 Guacd 的消息转发，支持自动重连
func WsToGuacd(ws *websocket.Conn, tunnel Tunnel, guacd io.Writer) {
	var reconnectableTunnel *ReconnectableTunnel
	if rt, ok := tunnel.(*ReconnectableTunnel); ok {
		reconnectableTunnel = rt
	}

	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			logx.Warnf("error reading from websocket: %v", err)
			// WebSocket 连接失败，直接退出，避免重复读取
			return
		}

		if bytes.HasPrefix(data, internalOpcodeIns) {
			// messages starting with the InternalDataOpcode are never sent to guacd
			continue
		}

		if _, err = guacd.Write(data); err != nil {
			logx.Warnf("error writing to guacd: %v", err)
			// 判断是否是 WebSocket 关闭导致的错误
			if errors.Is(err, websocket.ErrCloseSent) ||
				err.Error() == "EOF" ||
				err.Error() == "use of closed network connection" {
				logx.Warnf("websocket connection closed, stopping WsToGuacd")
				return
			}
			// 尝试重连
			if err := reconnectableTunnel.Reconnect(); err != nil {
				logx.Errorf("failed to reconnect: %v", err)
				return
			}
			// 重连成功后重新获取 writer
			guacd = reconnectableTunnel.AcquireWriter()
			// 继续循环
			continue
		}
	}
}

// GuacdToWs 处理从 Guacd 到 WebSocket 的消息转发，支持自动重连
func GuacdToWs(ws *websocket.Conn, tunnel Tunnel, guacd InstructionReader) {
	var reconnectableTunnel *ReconnectableTunnel
	if rt, ok := tunnel.(*ReconnectableTunnel); ok {
		reconnectableTunnel = rt
	}

	buf := bytes.NewBuffer(make([]byte, 0, MaxGuacMessage*2))

	for {
		ins, err := guacd.ReadSome()
		if err != nil {
			logx.Warnf("error reading message from guacd: %v", err)
			// 尝试重连
			if err := reconnectableTunnel.Reconnect(); err != nil {
				logx.Errorf("failed to reconnect: %v", err)
				return
			}
			// 重连成功后重新获取 reader
			guacd = reconnectableTunnel.AcquireReader()
			// 继续循环
			continue
		}

		if bytes.HasPrefix(ins, internalOpcodeIns) {
			// messages starting with the InternalDataOpcode are never sent to the websocket
			continue
		}
		logx.Debugf("guacd msg: %s", string(ins))
		if _, err = buf.Write(ins); err != nil {
			logx.Warnf("failed to buffer guacd to ws: %v", err)
			return
		}

		// if the buffer has more data in it or we've reached the max buffer size, send the data and reset
		if !guacd.Available() || buf.Len() >= MaxGuacMessage {
			if err = ws.WriteMessage(1, buf.Bytes()); err != nil {
				if errors.Is(err, websocket.ErrCloseSent) {
					return
				}
				logx.Warnf("failed sending message to ws: %v", err)
				// 判断是否是 WebSocket 关闭导致的错误，如果是则直接退出
				if err.Error() == "EOF" ||
					err.Error() == "use of closed network connection" ||
					err.Error() == "broken pipe" {
					logx.Warnf("websocket connection closed, stopping GuacdToWs")
					return
				}
				// 尝试重连
				if err := reconnectableTunnel.Reconnect(); err != nil {
					logx.Errorf("failed to reconnect: %v", err)
					return
				}
				// 重连成功后重置缓冲区并继续
				buf.Reset()
				continue
			}
			buf.Reset()
		}
	}
}
