package entity

import (
	"mayfly-go/base/model"
)

type Machine struct {
	model.Model
	ProjectId   uint64 `json:"projectId"`
	ProjectName string `json:"projectName"`
	Name        string `json:"name"`
	// IP地址
	Ip string `json:"ip"`
	// 用户名
	Username string `json:"username"`
	Password string `json:"-"`
	// 端口号
	Port int `json:"port"`
}
