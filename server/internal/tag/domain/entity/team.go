package entity

import "mayfly-go/pkg/model"

// 团队信息
type Team struct {
	model.Model

	Name   string `json:"name"`   // 名称
	Remark string `json:"remark"` // 备注说明
}
