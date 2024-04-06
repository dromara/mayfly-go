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

	"github.com/gorilla/websocket"
)

// creates the tunnel to the remote machine (via guacd)
func DoConnect(query url.Values, parameters map[string]string, machineId uint64) (Tunnel, error) {
	conf := NewGuacamoleConfiguration()

	parameters["enable-wallpaper"] = "true" // 允许显示墙纸
	//parameters["resize-method"] = "reconnect"
	parameters["enable-font-smoothing"] = "true"
	parameters["enable-desktop-composition"] = "true"
	parameters["enable-menu-animations"] = "false"
	parameters["disable-bitmap-caching"] = "true"
	parameters["disable-offscreen-caching"] = "true"
	parameters["force-lossless"] = "true" // 无损压缩
	parameters["color-depth"] = "32"      //32 真彩（32位）；24 真彩（24位）；16 低色（16位）；8 256色

	// drive
	parameters["enable-drive"] = "true"
	parameters["drive-name"] = "Filesystem"
	parameters["create-drive-path"] = "true"
	parameters["drive-path"] = fmt.Sprintf("/rdp-file/%d", machineId)

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

	conf.AudioMimetypes = []string{"audio/L8", "audio/L16"}
	conf.ImageMimetypes = []string{"image/jpeg", "image/png", "image/webp"}

	logx.Debug("Connecting to guacd")

	machineConfig := config.GetMachine()
	if machineConfig.GuacdHost == "" {
		return nil, errorx.NewBiz("请前往'系统配置-机器配置'中配置guacd相关信息")
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

func WsToGuacd(ws *websocket.Conn, tunnel Tunnel, guacd io.Writer) {
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			logx.Warnf("Error reading message from ws: %v", err)
			_ = tunnel.Close()
			return
		}

		if bytes.HasPrefix(data, internalOpcodeIns) {
			// messages starting with the InternalDataOpcode are never sent to guacd
			continue
		}

		if _, err = guacd.Write(data); err != nil {
			logx.Warnf("Failed writing to guacd: %v", err)
			return
		}
	}
}

func GuacdToWs(ws *websocket.Conn, tunnel Tunnel, guacd InstructionReader) {
	buf := bytes.NewBuffer(make([]byte, 0, MaxGuacMessage*2))

	for {
		ins, err := guacd.ReadSome()
		if err != nil {
			logx.Warnf("Error reading message from guacd: %v", err)
			_ = tunnel.Close()
			return
		}

		if bytes.HasPrefix(ins, internalOpcodeIns) {
			// messages starting with the InternalDataOpcode are never sent to the websocket
			continue
		}
		logx.Debugf("guacd msg: %s", string(ins))
		if _, err = buf.Write(ins); err != nil {
			logx.Warnf("Failed to buffer guacd to ws: %v", err)
			return
		}

		// if the buffer has more data in it or we've reached the max buffer size, send the data and reset
		if !guacd.Available() || buf.Len() >= MaxGuacMessage {
			if err = ws.WriteMessage(1, buf.Bytes()); err != nil {
				if errors.Is(err, websocket.ErrCloseSent) {
					return
				}
				logx.Warnf("Failed sending message to ws: %v", err)
				return
			}
			buf.Reset()
		}
	}
}
