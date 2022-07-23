package utils

import "net"

// GetAvailablePort 获取可用端口
func GetAvailablePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}

	defer func(l *net.TCPListener) {
		_ = l.Close()
	}(l)
	return l.Addr().(*net.TCPAddr).Port, nil
}
