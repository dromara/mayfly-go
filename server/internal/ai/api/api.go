package api

import "mayfly-go/pkg/ioc"

func InitIoc() {
	// 注册AI SQL API组件
	ioc.Register(new(AiDB))
}
