package mcm

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"testing"

	"mayfly-go/pkg/utils/netx"

	tagentity "mayfly-go/internal/tag/domain/entity"

	"golang.org/x/crypto/ssh"
)

// TestGetSshClientViaSocks5 端到端：经 SOCKS5 建链完成真实 SSH 握手并执行命令。
// 覆盖「拨代理 → SOCKS5 协商 → SSH 握手 → 会话执行」整条链路，而不是只测拨号函数。
func TestGetSshClientViaSocks5(t *testing.T) {
	sshAddr := startTestSSHServer(t)
	socksAddr, dialedTargets := startTestSocks5Server(t)

	_, sshPortStr, _ := net.SplitHostPort(sshAddr)
	sshPort, _ := strconv.Atoi(sshPortStr)
	mi := &MachineInfo{
		Name:            "socks5-e2e",
		Ip:              "127.0.0.1",
		Port:            sshPort,
		Username:        "audit",
		Password:        "secret",
		AuthMethod:      int8(tagentity.AuthCertCiphertextTypePassword),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Proxy:           &netx.Socks5Config{Addr: socksAddr},
	}

	client, err := mi.GetSshClient(nil)
	if err != nil {
		t.Fatalf("经 SOCKS5 建 SSH 连接失败: %v", err)
	}
	defer func() { _ = client.Close() }()

	session, err := client.NewSession()
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	defer func() { _ = session.Close() }()
	out, err := session.CombinedOutput("echo proxy-ok")
	if err != nil {
		t.Fatalf("run command: %v", err)
	}
	if string(out) != "proxy-ok: echo proxy-ok\n" {
		t.Fatalf("command output = %q", string(out))
	}

	targets := dialedTargets()
	if len(targets) != 1 || targets[0] != sshAddr {
		t.Fatalf("socks5 代理收到的目标 = %v, want [%s]", targets, sshAddr)
	}
}

// TestGetSshClientProxyUnreachable 代理不可达时必须快速失败，且错误指向代理
func TestGetSshClientProxyUnreachable(t *testing.T) {
	sshAddr := startTestSSHServer(t)
	_, sshPortStr, _ := net.SplitHostPort(sshAddr)
	sshPort, _ := strconv.Atoi(sshPortStr)

	mi := &MachineInfo{
		Name:            "socks5-unreachable",
		Ip:              "127.0.0.1",
		Port:            sshPort,
		Username:        "audit",
		Password:        "secret",
		AuthMethod:      int8(tagentity.AuthCertCiphertextTypePassword),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		// 拿一个刚关闭的端口当代理：连接会被拒绝
		Proxy: &netx.Socks5Config{Addr: closedTCPAddr(t)},
	}

	if _, err := mi.GetSshClient(nil); err == nil {
		t.Fatal("代理不可达时应返回错误")
	}
}

func closedTCPAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	return addr
}

// startTestSSHServer 起一个只接受 audit/secret 的本机 SSH 服务，exec 时回显命令
func startTestSSHServer(t *testing.T) string {
	t.Helper()
	hostKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(hostKey)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	serverConfig := &ssh.ServerConfig{
		PasswordCallback: func(meta ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if meta.User() == "audit" && string(password) == "secret" {
				return nil, nil
			}
			return nil, fmt.Errorf("auth denied for %s", meta.User())
		},
	}
	serverConfig.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen ssh: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveTestSSHConn(conn, serverConfig)
		}
	}()
	return ln.Addr().String()
}

func serveTestSSHConn(conn net.Conn, serverConfig *ssh.ServerConfig) {
	defer func() { _ = conn.Close() }()
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, serverConfig)
	if err != nil {
		return
	}
	defer func() { _ = sshConn.Close() }()
	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "only session is supported")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}
		go func() {
			defer func() { _ = channel.Close() }()
			for req := range requests {
				if req.Type != "exec" {
					_ = req.Reply(false, nil)
					continue
				}
				var payload struct{ Command string }
				if err := ssh.Unmarshal(req.Payload, &payload); err != nil {
					_ = req.Reply(false, nil)
					continue
				}
				if err := req.Reply(true, nil); err != nil {
					return
				}
				if _, err := channel.Write([]byte("proxy-ok: " + payload.Command + "\n")); err != nil {
					return
				}
				_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
				return
			}
		}()
	}
}

// startTestSocks5Server 最小 SOCKS5 CONNECT 服务（无认证），记录被拨的目标地址
func startTestSocks5Server(t *testing.T) (string, func() []string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen socks5: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	var mu sync.Mutex
	var targets []string
	record := func(target string) {
		mu.Lock()
		defer mu.Unlock()
		targets = append(targets, target)
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = conn.Close() }()
				reader := bufio.NewReader(conn)

				header := make([]byte, 2)
				if _, err := io.ReadFull(reader, header); err != nil {
					return
				}
				methods := make([]byte, int(header[1]))
				if _, err := io.ReadFull(reader, methods); err != nil {
					return
				}
				if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
					return
				}

				req := make([]byte, 4)
				if _, err := io.ReadFull(reader, req); err != nil || req[1] != 0x01 {
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
				default:
					return
				}
				portBytes := make([]byte, 2)
				if _, err := io.ReadFull(reader, portBytes); err != nil {
					return
				}
				target := net.JoinHostPort(host, strconv.Itoa(int(portBytes[0])<<8|int(portBytes[1])))
				record(target)

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
				go func() { _, _ = io.Copy(upstream, reader); done <- struct{}{} }()
				go func() { _, _ = io.Copy(conn, upstream); done <- struct{}{} }()
				<-done
			}()
		}
	}()

	return ln.Addr().String(), func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), targets...)
	}
}
