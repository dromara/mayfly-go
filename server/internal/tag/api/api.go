package api

import "mayfly-go/pkg/ioc"

func InitIoc() {
	ioc.Register(new(ResourceAuthCert))
	ioc.Register(new(ResourceOpLog))
	ioc.Register(new(TagTree))
	ioc.Register(new(Team))
}
