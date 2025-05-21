package mcm

import (
	"mayfly-go/pkg/pool"
	"time"
)

var mcConnPool = make(map[string]pool.Pool)
var mcIdPool = make(map[uint64]pool.Pool)

func init() {
}
func getMcPool(authCertName string, getMachine func(string) (*MachineInfo, error)) (pool.Pool, error) {
	// 获取连接池，如果没有，则创建一个
	if p, ok := mcConnPool[authCertName]; !ok {
		var err error
		p, err = pool.NewChannelPool(&pool.Config{
			InitialCap:  1,                //资源池初始连接数
			MaxCap:      10,               //最大空闲连接数
			MaxIdle:     10,               //最大并发连接数
			IdleTimeout: 10 * time.Minute, // 连接最大空闲时间，过期则失效
			Factory: func() (interface{}, error) {
				mi, err := getMachine(authCertName)
				if err != nil {
					return nil, err
				}
				mi.Key = authCertName
				return mi.Conn()
			},
			Close: func(v interface{}) error {
				v.(*Cli).Close()
				return nil
			},
			Ping: func(v interface{}) error {
				return v.(*Cli).Ping()
			},
		})
		if err != nil {
			return nil, err
		}
		mcConnPool[authCertName] = p
		return p, nil
	} else {
		return p, nil
	}
}

func PutMachineCli(c *Cli) {
	if nil == c {
		return
	}
	if p, ok := mcConnPool[c.Info.AuthCertName]; ok {
		p.Put(c)
	}

}

// 从缓存中获取客户端信息，不存在则回调获取机器信息函数，并新建。
// @param 机器的授权凭证名
func GetMachineCli(authCertName string, getMachine func(string) (*MachineInfo, error)) (*Cli, error) {
	p, err := getMcPool(authCertName, getMachine)
	if err != nil {
		return nil, err
	}
	// 从连接池中获取一个可用的连接
	c, err := p.Get()
	if err != nil {
		return nil, err
	}
	return c.(*Cli), nil
}

// 删除指定机器缓存客户端，并关闭客户端连接
func DeleteCli(id uint64) {
	// 遍历所有机器连接实例，删除指定机器id关联的连接...
	delete(mcIdPool, id)
}
