package esi

import (
	"fmt"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/logx"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type EsConn struct {
	Id   uint64
	Info *EsInfo

	proxy *httputil.ReverseProxy
}

/******************* pool.Conn impl *******************/

func (d *EsConn) Close() error {
	// 如果是使用了ssh隧道转发，则需要手动将其关闭
	if d.Info.useSshTunnel {
		mcm.CloseSshTunnelMachine(uint64(d.Info.SshTunnelMachineId), fmt.Sprintf("es:%d", d.Id))
	}
	return nil
}

func (d *EsConn) Ping() error {
	_, err := d.Info.Ping()
	return err
}

// StartProxy 开始代理
func (d *EsConn) StartProxy() error {
	// 目标 URL
	targetURL, err := url.Parse(d.Info.baseUrl)
	if err != nil {
		logx.Errorf("Error parsing URL: %v", err)
		return err
	}
	// 创建反向代理
	d.proxy = httputil.NewSingleHostReverseProxy(targetURL)
	// 设置 proxy buffer pool
	d.proxy.BufferPool = NewBufferPool()
	return nil
}

func (d *EsConn) Proxy(w http.ResponseWriter, r *http.Request, path string) {
	r.URL.Path = path
	if d.Info.authorization != "" {
		r.Header.Set("Authorization", d.Info.authorization)
	}
	r.Header.Set("connection", "keep-alive")
	r.Header.Set("Accept", "application/json")
	d.proxy.ServeHTTP(w, r)
}
