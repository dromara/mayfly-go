package application

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(new(labelAppImpl))
	ioc.Register(new(labelBindingAppImpl))
}

func GetLabelApp() Label {
	return ioc.Get[Label]()
}

func GetLabelBindingApp() LabelBinding {
	return ioc.Get[LabelBinding]()
}
