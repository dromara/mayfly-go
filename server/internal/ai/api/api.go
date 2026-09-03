package api

import "mayfly-go/pkg/ioc"

func InitIoc() {
	ioc.Register(new(AiDB))
	ioc.Register(new(Ai))
	ioc.Register(new(AiPlugin))
}
