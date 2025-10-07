package netx

import (
	"mayfly-go/pkg/logx"
	"net"
	"strings"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

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

var (
	// ip2region数据所在路径，可在（https://gitee.com/lionsoul/ip2region/tree/master/data）处下载
	ip2RegionDbPath = "./ip2region.xdb"
	useIp2Region    = true
	vectorIndex     []byte
)

// 获取ip归属地信息
func Ip2Region(ip string) string {
	if !useIp2Region {
		return ""
	}

	if vectorIndex == nil {
		// 1、从 dbPath 加载 VectorIndex 缓存，把下述 vIndex 变量全局到内存里面。
		vIndex, err := xdb.LoadVectorIndexFromFile(ip2RegionDbPath)
		// 第一次加载失败，则默认调整为不使用ip2Region
		if err != nil {
			logx.Errorf("failed to load ip2region vector index from `%s`: %s\n", ip2RegionDbPath, err)
			useIp2Region = false
			return ""
		}

		vectorIndex = vIndex
	}

	// 2、用全局的 vIndex 创建带 VectorIndex 缓存的查询对象。
	searcher, err := xdb.NewWithVectorIndex(xdb.IPv4, ip2RegionDbPath, vectorIndex)
	if err != nil {
		logx.Errorf("failed to create searcher with vector index: %s\n", err)
		return ""
	}

	defer searcher.Close()

	// do the search
	region, err := searcher.SearchByStr(ip)
	if err != nil {
		logx.Warnf("failed to SearchIP(%s): %s\n", ip, err)
		return ""
	}
	return region
}

func GetOutBoundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return "0.0.0.0"
	}
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ip := strings.Split(localAddr.String(), ":")[0]
	return ip
}
