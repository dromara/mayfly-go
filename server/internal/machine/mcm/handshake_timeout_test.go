package mcm

import (
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	tagentity "mayfly-go/internal/tag/domain/entity"

	"golang.org/x/crypto/ssh"
)

// TestGetSshClientHandshakeTimeout 对端接受 TCP 却不回 SSH banner 时，握手必须按超时失败。
// 旧实现只给 TCP dial 设时限（ssh.Dial 内部即 DialTimeout），握手本身无限阻塞；
// 而连接池 factory 在创建锁内执行，一次挂起会堵住终端/计划任务/状态采集。
func TestGetSshClientHandshakeTimeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	var mu sync.Mutex
	var accepted []net.Conn
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			// 保持连接但既不读也不写：模拟挂起的 SSH 握手
			mu.Lock()
			accepted = append(accepted, conn)
			mu.Unlock()
		}
	}()
	defer func() {
		mu.Lock()
		defer mu.Unlock()
		for _, conn := range accepted {
			conn.Close()
		}
	}()

	oldTimeout := sshConnectTimeout
	sshConnectTimeout = 500 * time.Millisecond
	defer func() { sshConnectTimeout = oldTimeout }()

	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("split addr: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}

	mi := &MachineInfo{
		Name:            "handshake-timeout",
		Ip:              "127.0.0.1",
		Port:            port,
		Username:        "audit",
		AuthMethod:      int8(tagentity.AuthCertCiphertextTypePassword),
		Password:        "audit",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	start := time.Now()
	_, err = mi.GetSshClient(nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("挂起的握手应超时失败，实际返回成功")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("错误应说明握手超时，实际: %v", err)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("握手超时未生效，耗时 %s", elapsed)
	}
}
