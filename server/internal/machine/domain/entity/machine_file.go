package entity

import "mayfly-go/pkg/model"

type MachineFile struct {
	model.Model

	Name      string `json:"name" gorm:"not null;size:50;comment:机器文件配置（linux一切皆文件，故也可以表示目录）"` // 机器文件配置（linux一切皆文件，故也可以表示目录）
	MachineId uint64 `json:"machineId" gorm:"not null;comment:机器id"`                           // 机器id
	Type      string `json:"type" gorm:"not null;size:32;comment:1：目录；2：文件"`                   // 1：目录；2：文件
	Path      string `json:"path" gorm:"not null;size:150;comment:路径"`                         // 路径
}
