package netx

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/proxy"
)

// OutboundDialer 出网拨号器：机器建链只依赖这一契约，新增代理协议（HTTP CONNECT 等）
// 只需实现该接口并在装配层注册，不改连接层分支。
type OutboundDialer interface {
	DialContext(ctx context.Context, network, address string, timeout time.Duration) (net.Conn, error)
}

// Socks5Config SOCKS5 出网代理配置。Addr 为空表示直连。
type Socks5Config struct {
	Addr     string // 代理地址 host:port
	Username string // 代理认证用户名（可选）
	Password string // 代理认证密码（可选）
}

var _ OutboundDialer = Socks5Config{}

// Enabled 是否配置了代理地址
func (c Socks5Config) Enabled() bool {
	return c.Addr != ""
}

// staticDialer 返回一个已经建立好的连接：x/net/proxy 的 SOCKS5 拨号器内部只通过
// forward dialer 拨代理地址，这里把「带超时拨好的代理连接」直接交给它，
// 从而能在连上代理后继续用 SetDeadline 约束 SOCKS5 协商阶段。
type staticDialer struct {
	conn net.Conn
}

func (d *staticDialer) Dial(_, _ string) (net.Conn, error) {
	if d.conn == nil {
		return nil, errors.New("socks5 forward conn is not initialized")
	}
	conn := d.conn
	d.conn = nil
	return conn, nil
}

// DialContext 经 SOCKS5 代理建连；未配置代理地址时等价直连。
//
// timeout 覆盖三段：拨代理 / SOCKS5 协商 / （由调用方继续约束的）目标握手。
// x/net/proxy 的 SOCKS5 实现在连上代理后不做 deadline 约束（内部没有 ctx 检查），
// 因此这里主动给底层连接设置读写截止时间，协商成功后清除，避免代理半死不活时无限挂起。
func (c Socks5Config) DialContext(ctx context.Context, network, address string, timeout time.Duration) (net.Conn, error) {
	var dialer net.Dialer
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
		dialer.Timeout = timeout
	}

	if !c.Enabled() {
		return dialer.DialContext(ctx, network, address)
	}
	rawConn, err := dialer.DialContext(ctx, "tcp", c.Addr)
	if err != nil {
		return nil, fmt.Errorf("dial socks5 proxy %s: %w", c.Addr, err)
	}

	var auth *proxy.Auth
	if c.Username != "" {
		auth = &proxy.Auth{User: c.Username, Password: c.Password}
	}
	socksDialer, err := proxy.SOCKS5("tcp", c.Addr, auth, &staticDialer{conn: rawConn})
	if err != nil {
		_ = rawConn.Close()
		return nil, fmt.Errorf("build socks5 dialer for %s: %w", c.Addr, err)
	}
	contextDialer, ok := socksDialer.(proxy.ContextDialer)
	if !ok {
		_ = rawConn.Close()
		return nil, fmt.Errorf("socks5 dialer for %s does not support context", c.Addr)
	}

	// SOCKS5 协商阶段（含 RFC1929 认证）的兜底时限
	if timeout > 0 {
		if err := rawConn.SetDeadline(time.Now().Add(timeout)); err != nil {
			_ = rawConn.Close()
			return nil, err
		}
	}
	conn, err := contextDialer.DialContext(ctx, network, address)
	if err != nil {
		_ = rawConn.Close()
		return nil, fmt.Errorf("socks5 connect %s via %s: %w", address, c.Addr, err)
	}
	// 协商完成后清除截止时间：后续 SSH 握手/长连接自行管理超时
	if err := rawConn.SetDeadline(time.Time{}); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}
