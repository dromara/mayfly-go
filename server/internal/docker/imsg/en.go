package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	LogDockerContainerStop:    "Container - Stop",
	LogDockerContainerRestart: "Container - Restart",
	LogDockerContainerRemove:  "Container - Remove",
	LogDockerContainerCreate:  "Container - Create",

	LogDockerImageRemove: "Image - Remove",
	LogDockerImageLoad:   "Image - Load",

	ErrContainerConfExist: "Container conf already exists",
}
