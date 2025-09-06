package imsg

import "mayfly-go/pkg/i18n"

var Zh_CN = map[i18n.MsgId]string{
	LogDockerContainerStop:    "容器-停止",
	LogDockerContainerRemove:  "容器-删除",
	LogDockerContainerRestart: "容器-重启",
	LogDockerContainerCreate:  "容器-创建",

	LogDockerImageRemove: "镜像-删除",
	LogDockerImageLoad:   "镜像-导入",

	ErrContainerConfExist: "容器配置已存在",
}
