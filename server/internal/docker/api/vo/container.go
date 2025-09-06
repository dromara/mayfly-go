package vo

import (
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/model"
)

type ContainerConf struct {
	model.Model
	model.ExtraData
	tagentity.ResourceTags // 标签信息

	Addr   string `json:"addr"`
	Code   string `json:"code"`
	Name   string `json:"name"`
	Remark string `json:"remark"`
}

func (c *ContainerConf) GetCode() string {
	return c.Code
}
