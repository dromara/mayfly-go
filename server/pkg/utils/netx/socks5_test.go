package netx

import (
	"bufio"
	"context"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// testSocks5Server 最小 SOCKS5 服务端（RFC1928 CONNECT + 可选 RFC1929 用户名密码认证），
// 只用于验证客户端拨号链路，支持 no-auth 与 user/pass 两种方式。
type testSocks5Server struct {
	ln          net.Listener
	requireAuth bool
	username    string
	password    string

	mu      sync.Mutex
	targets []string
}

func startTestSocks5Server(t *testing.T, requireAuth bool, username, password string) *testSocks5Server {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen socks5: %v", err)
	}
	s := &testSocks5Server{ln: ln, requireAuth: requireAuth, username: username, password: password}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go s.handle(conn)
		}
	}()
	t.Cleanup(func() { _ = ln.Close() })
	return s
}

func (s *testSocks5Server) addr() string {
	return s.ln.Addr().String()
}

func (s *testSocks5Server) dialedTargets() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.targets...)
}

func (s *testSocks5Server) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	reader := bufio.NewReader(conn)

	// 1. 方法协商
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil || header[0] != 0x05 {
		return
	}
	methods := make([]byte, int(header[1]))
	if _, err := io.ReadFull(reader, methods); err != nil {
		return
	}
	chosen := byte(0x00)
	if s.requireAuth {
		chosen = 0x02
	}
	if _, err := conn.Write([]byte{0x05, chosen}); err != nil {
		return
	}

	// 2. 用户名密码认证（可选）
	if s.requireAuth {
		verAndLen := make([]byte, 2)
		if _, err := io.ReadFull(reader, verAndLen); err != nil {
			return
		}
		user := make([]byte, int(verAndLen[1]))
		if _, err := io.ReadFull(reader, user); err != nil {
			return
		}
		pwdLen := make([]byte, 1)
		if _, err := io.ReadFull(reader, pwdLen); err != nil {
			return
		}
		pwd := make([]byte, int(pwdLen[0]))
		if _, err := io.ReadFull(reader, pwd); err != nil {
			return
		}
		if string(user) != s.username || string(pwd) != s.password {
			_, _ = conn.Write([]byte{0x01, 0x01})
			return
		}
		if _, err := conn.Write([]byte{0x01, 0x00}); err != nil {
			return
		}
	}

	// 3. CONNECT 请求
	req := make([]byte, 4)
	if _, err := io.ReadFull(reader, req); err != nil || req[0] != 0x05 || req[1] != 0x01 {
		return
	}
	var host string
	switch req[3] {
	case 0x01:
		ip := make([]byte, 4)
		if _, err := io.ReadFull(reader, ip); err != nil {
			return
		}
		host = net.IP(ip).String()
	case 0x03:
		length := make([]byte, 1)
		if _, err := io.ReadFull(reader, length); err != nil {
			return
		}
		domain := make([]byte, int(length[0]))
		if _, err := io.ReadFull(reader, domain); err != nil {
			return
		}
		host = string(domain)
	case 0x04:
		ip := make([]byte, 16)
		if _, err := io.ReadFull(reader, ip); err != nil {
			return
		}
		host = net.IP(ip).String()
	default:
		return
	}
	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(reader, portBytes); err != nil {
		return
	}
	port := int(portBytes[0])<<8 | int(portBytes[1])
	target := net.JoinHostPort(host, strconv.Itoa(port))

	s.mu.Lock()
	s.targets = append(s.targets, target)
	s.mu.Unlock()

	upstream, err := net.Dial("tcp", target)
	if err != nil {
		_, _ = conn.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer func() { _ = upstream.Close() }()
	if _, err := conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}); err != nil {
		return
	}

	done := make(chan struct{}, 2)
	go func() {
		// 用 reader 而非 conn：它可能已经缓冲了握手后的首段数据
		_, _ = io.Copy(upstream, reader)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(conn, upstream)
		done <- struct{}{}
	}()
	<-done
}

// startEchoServer 起一个把收到的数据原样回写的 TCP 服务
func startEchoServer(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen echo: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = conn.Close() }()
				_, _ = io.Copy(conn, conn)
			}()
		}
	}()
	return ln.Addr().String()
}

func assertEcho(t *testing.T, cfg Socks5Config, target string) {
	t.Helper()
	conn, err := cfg.DialContext(context.Background(), "tcp", target, 2*time.Second)
	if err != nil {
		t.Fatalf("dial target %s: %v", target, err)
	}
	defer func() { _ = conn.Close() }()

	if _, err := conn.Write([]byte("ping-through-proxy")); err != nil {
		t.Fatalf("write: %v", err)
	}
	buf := make([]byte, len("ping-through-proxy"))
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if string(buf) != "ping-through-proxy" {
		t.Fatalf("echo = %q", string(buf))
	}
}

// TestSocks5DialContext 经 SOCKS5 代理建连并回显，代理应记录目标地址
func TestSocks5DialContext(t *testing.T) {
	echoAddr := startEchoServer(t)
	proxyServer := startTestSocks5Server(t, false, "", "")

	assertEcho(t, Socks5Config{Addr: proxyServer.addr()}, echoAddr)

	targets := proxyServer.dialedTargets()
	if len(targets) != 1 || targets[0] != echoAddr {
		t.Fatalf("socks5 server dialed %v, want [%s]", targets, echoAddr)
	}
}

// TestSocks5DialContextAuth 代理要求用户名密码时正确携带认证信息
func TestSocks5DialContextAuth(t *testing.T) {
	echoAddr := startEchoServer(t)
	proxyServer := startTestSocks5Server(t, true, "audit", "secret")

	assertEcho(t, Socks5Config{Addr: proxyServer.addr(), Username: "audit", Password: "secret"}, echoAddr)
}

// TestSocks5DialContextAuthRejected 认证失败必须返回错误而不是静默拨到目标
func TestSocks5DialContextAuthRejected(t *testing.T) {
	echoAddr := startEchoServer(t)
	proxyServer := startTestSocks5Server(t, true, "audit", "secret")

	cfg := Socks5Config{Addr: proxyServer.addr(), Username: "audit", Password: "wrong"}
	conn, err := cfg.DialContext(context.Background(), "tcp", echoAddr, 2*time.Second)
	if err == nil {
		_ = conn.Close()
		t.Fatal("认证失败应返回错误")
	}
	if len(proxyServer.dialedTargets()) != 0 {
		t.Fatal("认证失败时不应向目标发起连接")
	}
}

// TestSocks5DialContextDirect 未配置代理地址时等价直连，不应经过任何代理
func TestSocks5DialContextDirect(t *testing.T) {
	echoAddr := startEchoServer(t)
	assertEcho(t, Socks5Config{}, echoAddr)
}

// TestSocks5DialContextHandshakeTimeout 代理接受 TCP 但不回协商包时，必须按超时返回
func TestSocks5DialContextHandshakeTimeout(t *testing.T) {
	echoAddr := startEchoServer(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen hang proxy: %v", err)
	}
	defer func() { _ = ln.Close() }()

	var mu sync.Mutex
	var held []net.Conn
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			held = append(held, conn)
			mu.Unlock()
		}
	}()
	defer func() {
		mu.Lock()
		defer mu.Unlock()
		for _, conn := range held {
			_ = conn.Close()
		}
	}()

	cfg := Socks5Config{Addr: ln.Addr().String()}
	start := time.Now()
	conn, err := cfg.DialContext(context.Background(), "tcp", echoAddr, 400*time.Millisecond)
	elapsed := time.Since(start)
	if err == nil {
		_ = conn.Close()
		t.Fatal("代理不回协商包时应超时失败")
	}
	if !strings.Contains(err.Error(), "socks5 connect") {
		t.Fatalf("错误应指向 socks5 协商阶段，实际: %v", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("协商超时未生效，耗时 %s", elapsed)
	}
}

// TestSocks5DialContextDirectTimeout 未配置代理时也必须遵守 timeout（导出 API 不能留无限等待分支）
func TestSocks5DialContextDirectTimeout(t *testing.T) {
	// TEST-NET-1 段常见于黑洞路由；即便环境直接返回不可达，也必须表现为「快速失败」
	start := time.Now()
	conn, err := (Socks5Config{}).DialContext(context.Background(), "tcp", "192.0.2.1:81", 300*time.Millisecond)
	elapsed := time.Since(start)
	if err == nil {
		_ = conn.Close()
		t.Fatal("不可达目标应返回错误")
	}
	if elapsed > 3*time.Second {
		t.Fatalf("直连超时未生效，耗时 %s", elapsed)
	}
}
