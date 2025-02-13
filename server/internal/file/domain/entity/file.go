package entity

import "mayfly-go/pkg/model"

type File struct {
	model.Model

	FileKey  string `json:"fikeKey" gorm:"size:32;not null;"`   // 文件key
	Filename string `json:"filename" gorm:"size:255;not null;"` // 文件名
	Path     string `json:"path" gorm:"size:500;"`              // 文件路径
	Size     int64  `json:"size"`
}

func (a *File) TableName() string {
	return "t_sys_file"
}
