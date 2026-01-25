package req

// RouterApi
// 该接口的实现类注册到ioc中，则会自动将请求配置注册到路由中
type RouterApi interface {
	// ReqConfs 获取请求配置信息
	ReqConfs() *Confs
}

// RouterConfig 请求路由配置
type RouterConfig struct {
	ContextPath string // 请求路径上下文
}
