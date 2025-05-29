package esi

import (
	"context"
	"encoding/base64"
	"fmt"
	machineapp "mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/httpx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/structx"
	"net/http"
	"strings"
)

type EsVersion string

type EsInfo struct {
	model.ExtraData // 连接需要的其他额外参数（json字符串）

	InstanceId uint64 // 实例id
	Name       string

	Host     string
	Port     int
	Network  string
	Username string
	Password string

	Version        EsVersion // 数据库版本信息，用于语法兼容
	DefaultVersion bool      // 经过查询数据库版本信息后，是否仍然使用默认版本

	CodePath           []string
	SshTunnelMachineId int
	useSshTunnel       bool // 是否使用系统自己实现的ssh隧道连接,而非库自带的

	OriginUrl     string // 原始url
	baseUrl       string // 发起http请求的基本url
	authorization string // 发起http请求携带的认证信息
}

// 获取记录日志的描述
func (di *EsInfo) GetLogDesc() string {
	return fmt.Sprintf("ES[id=%d, tag=%s, name=%s, ip=%s:%d]", di.InstanceId, di.CodePath, di.Name, di.Host, di.Port)
}

// 连接数据库
func (di *EsInfo) Conn(ctx context.Context) (*EsConn, map[string]any, error) {
	// 使用basic加密用户名和密码
	if di.Username != "" && di.Password != "" {
		encodeString := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", di.Username, di.Password)))
		di.authorization = fmt.Sprintf("Basic %s", encodeString)
	}

	// 使用ssh隧道
	err := di.IfUseSshTunnelChangeIpPort(ctx)
	if err != nil {
		logx.Errorf("es ssh failed: %s, err:%s", di.baseUrl, err.Error())
		return nil, nil, errorx.NewBiz("es ssh failed: %s", err.Error())
	}

	// 尝试获取es版本信息，调用接口：get /
	res, err := di.Ping()
	if err != nil {
		logx.Errorf("es ping failed: %s, err:%s", di.baseUrl, err.Error())
		return nil, nil, errorx.NewBiz("es ping failed: %s", err.Error())
	}

	esc := &EsConn{Id: di.InstanceId, Info: di}
	err = esc.StartProxy()
	if err != nil {
		logx.Errorf("es porxy failed: %s, err:%s", di.baseUrl, err.Error())
		return nil, nil, err
	}

	if di.OriginUrl != di.baseUrl {
		logx.Infof("es porxy success: %s => %s", di.baseUrl, di.OriginUrl)
	} else {
		logx.Infof("es porxy success: %s", di.baseUrl)
	}

	return esc, res, nil
}

func (di *EsInfo) Ping() (map[string]any, error) {
	return di.ExecApi("get", "", nil)
}

// ExecApi 执行api
func (di *EsInfo) ExecApi(method, path string, data any, timeoutSecond ...int) (map[string]any, error) {
	request := httpx.NewReq(di.baseUrl + path)
	if di.authorization != "" {
		request.Header("Authorization", di.authorization)
	}
	if len(timeoutSecond) > 0 { // 设置超时时间
		request.Timeout(timeoutSecond[0])
	}

	switch strings.ToUpper(method) {
	case http.MethodGet:
		if data != nil {
			return request.GetByQuery(structx.ToMap(data)).BodyToMap()
		}
		return request.Get().BodyToMap()

	case http.MethodPost:
		return request.PostObj(data).BodyToMap()
	case http.MethodPut:
		return request.PutObj(data).BodyToMap()
	}

	return nil, errorx.NewBiz("不支持的请求方法: %s", method)

}

// 如果使用了ssh隧道，将其host port改变其本地映射host port
func (di *EsInfo) IfUseSshTunnelChangeIpPort(ctx context.Context) error {
	// 开启ssh隧道
	if di.SshTunnelMachineId > 0 {
		stm, err := GetSshTunnel(ctx, di.SshTunnelMachineId)
		if err != nil {
			return err
		}
		exposedIp, exposedPort, err := stm.OpenSshTunnel(fmt.Sprintf("es:%d", di.InstanceId), di.Host, di.Port)
		if err != nil {
			return err
		}
		di.Host = exposedIp
		di.Port = exposedPort
		di.useSshTunnel = true
		di.baseUrl = fmt.Sprintf("http://%s:%d", exposedIp, exposedPort)
	} else {
		di.baseUrl = fmt.Sprintf("http://%s:%d", di.Host, di.Port)
	}
	return nil
}

// 根据ssh tunnel机器id返回ssh tunnel
func GetSshTunnel(ctx context.Context, sshTunnelMachineId int) (*mcm.SshTunnelMachine, error) {
	return machineapp.GetMachineApp().GetSshTunnelMachine(ctx, sshTunnelMachineId)
}
