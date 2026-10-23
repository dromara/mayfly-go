package mcm

import (
	"testing"

	"mayfly-go/pkg/utils/netx"
)

// TestShouldUseProxy 代理只用于直连目标机：跳板链与被隧道改写地址的目标机都走隧道，不叠加代理
func TestShouldUseProxy(t *testing.T) {
	proxy := netx.Socks5Config{Addr: "127.0.0.1:1080"}

	direct := &MachineInfo{Ip: "10.0.0.1", Port: 22, Proxy: proxy}
	if !direct.shouldUseProxy() {
		t.Fatal("直连机器配了代理时应走代理")
	}

	// Conn() 会先把目标机 Ip/Port 改写为本地暴露地址；此时不能再交给代理，否则会去连 127.0.0.1:exposePort
	tunneled := &MachineInfo{
		Ip:               "127.0.0.1",
		Port:             2222,
		Proxy:            proxy,
		SshTunnelMachine: &MachineInfo{Id: 2, Ip: "10.0.0.2", Port: 22},
	}
	if tunneled.shouldUseProxy() {
		t.Fatal("跳板/隧道目标机不应叠加代理")
	}

	noProxy := &MachineInfo{Ip: "10.0.0.1", Port: 22}
	if noProxy.shouldUseProxy() {
		t.Fatal("未配置代理时应直连")
	}
}
