package imsg

import (
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/i18n"
)

func init() {
	i18n.AppendLangMsg(i18n.Zh_CN, Zh_CN)
	i18n.AppendLangMsg(i18n.En, En)
}

const (
	LogDockerContainerStop = iota + consts.ImsgNumDocker
	LogDockerContainerRemove
	LogDockerContainerRestart
	LogDockerContainerCreate

	LogDockerImageRemove
	LogDockerImageLoad

	ErrContainerConfExist
)
