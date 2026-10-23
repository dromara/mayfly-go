package mcm

import (
	"context"
	"fmt"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/netx"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// sshConnectTimeout SSH 建链总超时：TCP 连接与后续握手共用这一上限（20s 覆盖跨境/弱网链路）。
// ssh.ClientConfig.Timeout 只约束 TCP dial（x/crypto/ssh Dial 内部就是 DialTimeout），
// 对端接受连接却不回 banner 时握手会无限阻塞，因此必须由连接层自己兑底。
var sshConnectTimeout = 20 * time.Second

// 机器信息
type MachineInfo struct {
	model.ExtraData

	Id       uint64 `json:"id"`
	Name     string `json:"name"`
	Code     string `json:"code"`
	Protocol int    `json:"protocol"`

	Ip   string `json:"ip"` // IP地址
	Port int    `json:"-"`  // 端口号

	AuthCertName string `json:"authCertName"`
	AuthMethod   int8   `json:"-"` // 授权认证方式
	Username     string `json:"-"` // 用户名
	Password     string `json:"-"`
	Passphrase   string `json:"-"` // 私钥口令

	SshTunnelMachine *MachineInfo `json:"-"` // ssh隧道机器
	RemoteAddr       string       `json:"-"` // ssh隧道远程地址，格式 ip:port
	EnableRecorder   int8         `json:"-"` // 是否启用终端回放记录
	CodePath         []string     `json:"codePath"`

	// HostKeyCallback 主机公钥校验回调，由 application 层按信任策略（首次信任/指纹比对/失配拒绝）装配。
	// 未装配即连接属于编程错误，连接层按拒绝处理而非静默放行
	HostKeyCallback ssh.HostKeyCallback `json:"-"`

	// Proxy 出网代理拨号器，由 application 层按机器 extra 装配；nil 表示直连。
	// 仅作用于直连机器：跳板链内部通道、以及已被 SSH 隧道改写为本地暴露地址的目标机都走隧道，代理不叠加
	Proxy netx.OutboundDialer `json:"-"`
}

var _ (SshTunnelAble) = (*MachineInfo)(nil)

func (mi *MachineInfo) GetSshTunnelMachineId() int64 {
	if mi.SshTunnelMachine == nil {
		return 0
	}
	return int64(mi.SshTunnelMachine.Id)
}

func (mi *MachineInfo) GetRemoteAddr() string {
	if mi.RemoteAddr != "" {
		return mi.RemoteAddr
	}
	return fmt.Sprintf("%s:%d", mi.Ip, mi.Port)
}

func (mi *MachineInfo) UseSshTunnel() bool {
	return mi.SshTunnelMachine != nil
}

// HostKeyAddr 返回主机公钥信任键：连接改写前的原始目标地址（ip:port）。
// 不能用 ssh 回调传入的 hostname：跳板隧道会把连接地址改写为本地暴露地址；
// RemoteAddr 在改写前已被记录为原始地址，未走隧道时 Ip/Port 本身即原始值
func (mi *MachineInfo) HostKeyAddr() string {
	if mi.RemoteAddr != "" {
		return mi.RemoteAddr
	}
	return fmt.Sprintf("%s:%d", mi.Ip, mi.Port)
}

// GetSshClient 获取ssh客户端连接
func (mi *MachineInfo) GetSshClient(jumpClient *ssh.Client) (*ssh.Client, error) {
	if mi.HostKeyCallback == nil {
		// 信任策略未装配即发起连接属于装配缺陷而非可降级场景：
		// 跳过校验等于对中间人攻击不设防，宁可让这台机器连不上
		return nil, errorx.NewBizf("machine [%s] host key verify policy is not configured, connection rejected", mi.Name)
	}
	config := &ssh.ClientConfig{
		User:            mi.Username,
		HostKeyCallback: mi.HostKeyCallback,
		Timeout:         sshConnectTimeout,
	}
	if ciphers := mi.GetExtraString("ciphers"); ciphers != "" {
		config.Ciphers = strings.Split(ciphers, ",")
	}
	if keyExchanges := mi.GetExtraString("keyExchanges"); keyExchanges != "" {
		config.KeyExchanges = strings.Split(keyExchanges, ",")
	}

	if mi.AuthMethod == int8(tagentity.AuthCertCiphertextTypePassword) {
		config.Auth = []ssh.AuthMethod{ssh.Password(mi.Password)}
	} else if mi.AuthMethod == int8(tagentity.AuthCertCiphertextTypePrivateKey) {
		var key ssh.Signer
		var err error

		if len(mi.Passphrase) > 0 {
			key, err = ssh.ParsePrivateKeyWithPassphrase([]byte(mi.Password), []byte(mi.Passphrase))
		} else {
			key, err = ssh.ParsePrivateKey([]byte(mi.Password))
		}
		if err != nil {
			return nil, err
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(key)}
	}

	addr := fmt.Sprintf("%s:%d", mi.Ip, mi.Port)
	// 拨号地址用 JoinHostPort：IPv6 需要方括号；addr 仍作为 SSH hostname（与信任库 HostKeyAddr 同口径）
	dialAddr := net.JoinHostPort(mi.Ip, strconv.Itoa(mi.Port))

	// 一次建链只给一份总预算：拨通道 + SSH 握手串行共享，避免每段各超时导致实际等待翻倍
	deadline := time.Now().Add(sshConnectTimeout)
	var netConn net.Conn
	var err error
	switch {
	case jumpClient != nil:
		// 跳板通道本身的 Dial 同样没有 deadline，必须自己兜住（否则会挂住连接池创建锁）
		netConn, err = dialViaJump(jumpClient, dialAddr, time.Until(deadline))
	case mi.shouldUseProxy():
		netConn, err = mi.Proxy.DialContext(context.Background(), "tcp", dialAddr, time.Until(deadline))
	default:
		netConn, err = net.DialTimeout("tcp", dialAddr, time.Until(deadline))
	}
	if err != nil {
		return nil, err
	}
	return handshakeWithTimeout(netConn, addr, config, time.Until(deadline))
}

// shouldUseProxy 仅直连目标机时使用出网代理：跳板链内部通道、以及已被隧道改写为
// 本地暴露地址（127.0.0.1:exposePort）的目标机都走隧道，代理不叠加。
func (mi *MachineInfo) shouldUseProxy() bool {
	return mi.Proxy != nil && !mi.UseSshTunnel()
}

// dialViaJump 在给定预算内打开跳板通道。
// ssh.Client.Dial 走 mux.OpenChannel，没有 deadline：跳板进程假死时 channel open 会永久等待，
// 而本函数位于连接池创建锁内，必须自己兜底。超时后回收迟到的连接，避免连接与 goroutine 泄漏。
func dialViaJump(jumpClient *ssh.Client, addr string, timeout time.Duration) (net.Conn, error) {
	if timeout <= 0 {
		timeout = sshConnectTimeout
	}
	type dialResult struct {
		conn net.Conn
		err  error
	}
	done := make(chan dialResult, 1)
	go func() {
		conn, err := jumpClient.Dial("tcp", addr)
		done <- dialResult{conn: conn, err: err}
	}()

	select {
	case r := <-done:
		return r.conn, r.err
	case <-time.After(timeout):
		// channel open 无法主动取消；等它返回后立即关闭
		go func() {
			r := <-done
			if r.conn != nil {
				closeQuietly(r.conn)
			}
		}()
		return nil, errorx.NewBizf("jump host dial %s timed out after %s", addr, timeout)
	}
}

// handshakeWithTimeout 在限定时间内完成 SSH 握手。
//
// 为什么不能只靠 ClientConfig.Timeout：x/crypto/ssh 的 Dial 只用它做 TCP dial，
// 握手阶段不受约束；而连接池的 factory 是在创建锁内执行的，一次挂起的握手会把
// 终端、计划任务、状态采集全部堵在同一把锁上。超时后关闭底层连接即可让仍在阻塞的
// NewClientConn 立即以 IO 错误退出，等待方不会泄漏。
func handshakeWithTimeout(netConn net.Conn, addr string, config *ssh.ClientConfig, timeout time.Duration) (*ssh.Client, error) {
	if timeout <= 0 {
		timeout = config.Timeout
	}
	if timeout <= 0 {
		timeout = sshConnectTimeout
	}

	type handshakeResult struct {
		conn  ssh.Conn
		chans <-chan ssh.NewChannel
		reqs  <-chan *ssh.Request
		err   error
	}
	done := make(chan handshakeResult, 1)
	go func() {
		conn, chans, reqs, err := ssh.NewClientConn(netConn, addr, config)
		done <- handshakeResult{conn: conn, chans: chans, reqs: reqs, err: err}
	}()

	select {
	case r := <-done:
		if r.err != nil {
			closeQuietly(netConn)
			return nil, r.err
		}
		return ssh.NewClient(r.conn, r.chans, r.reqs), nil
	case <-time.After(timeout):
		// 关闭底层连接打断仍在进行的握手，避免 goroutine 与连接泄漏
		closeQuietly(netConn)
		<-done
		return nil, errorx.NewBizf("ssh handshake with %s timed out after %s", addr, timeout)
	}
}

// closeQuietly 关闭底层连接：失败只记日志，不覆盖调用方正在返回的握手错误
func closeQuietly(conn net.Conn) {
	if err := conn.Close(); err != nil {
		logx.Debugf("close ssh connection failed: %v", err)
	}
}

// 连接
func (mi *MachineInfo) Conn(ctx context.Context) (*Cli, error) {
	logx.Infof("the machine[%s] is connecting: %s:%d", mi.Name, mi.Ip, mi.Port)

	// 如果使用了ssh隧道，则修改机器ip port为暴露的ip port
	err := mi.IfUseSshTunnelChangeIpPort(ctx, false)
	if err != nil {
		return nil, errorx.NewBizf("ssh tunnel connection failed: %s", err.Error())
	}

	cli := &Cli{Info: mi}
	sshClient, err := mi.GetSshClient(nil)
	if err != nil {
		CloseSshTunnel(mi)
		return nil, err
	}
	cli.sshClient = sshClient
	return cli, nil
}

// 如果使用了ssh隧道，则修改机器ip port为暴露的ip port
func (mi *MachineInfo) IfUseSshTunnelChangeIpPort(ctx context.Context, out bool) error {
	if !mi.UseSshTunnel() {
		return nil
	}

	mi.RemoteAddr = mi.GetRemoteAddr()
	originId := mi.Id
	if originId == 0 {
		// 随机设置一个id，如果使用了隧道则用于临时保存隧道
		mi.Id = uint64(time.Now().Nanosecond())
	}

	stm := mi.SshTunnelMachine
	sshTunnelMachine, err := GetSshTunnelMachine(ctx, int(stm.Id), func(u uint64) (*MachineInfo, error) {
		return stm, nil
	})
	if err != nil {
		return err
	}
	exposeIp, exposePort, err := sshTunnelMachine.OpenSshTunnel(mi)
	if err != nil {
		return err
	}

	// 是否获取局域网的本地IP
	if out {
		exposeIp = netx.GetOutBoundIP()
	}

	// 修改机器ip地址
	mi.Ip = exposeIp
	mi.Port = exposePort
	return nil
}
