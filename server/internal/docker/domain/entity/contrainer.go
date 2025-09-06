package entity

import "mayfly-go/pkg/model"

// 容器配置
type Container struct {
	model.Model
	model.ExtraData

	Code   string `json:"code" gorm:"size:32;comment:code"`        // code
	Name   string `json:"name" gorm:"size:32"`                     // 名称
	Addr   string `json:"addr" gorm:"size:64;not null;comment:地址"` // 地址
	Remark string `json:"remark" gorm:"comment:备注"`                // 备注
}
