package api

import "mayfly-go/pkg/ioc"

func InitIoc() {
	ioc.Register(new(Ai))
	ioc.Register(new(AiPlugin))
}
