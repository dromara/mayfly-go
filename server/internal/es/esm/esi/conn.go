package esi

import (
	"crypto/tls"
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
	mcm.CloseSshTunnel(d.Info)
	return nil
}

func (d *EsConn) Ping() error {
	// 首先检查d是否为nil
	if d == nil {
		return fmt.Errorf("es connection is nil")
	}

	// 然后检查d.Info是否为nil，这是避免空指针异常的关键
	if d.Info == nil {
		return fmt.Errorf("es Info is nil")
	}
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

	// Configure TLS to skip certificate verification for non-compliant certificates
	if targetURL.Scheme == "https" {
		d.proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

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
