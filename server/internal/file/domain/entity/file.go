package entity

import "mayfly-go/pkg/model"

type File struct {
	model.Model

	FileKey  string `json:"fikeKey"`  // 文件key
	Filename string `json:"filename"` // 文件名
	Path     string `json:"path" `    // 文件路径
	Size     int64  `json:"size"`
}

func (a *File) TableName() string {
	return "t_sys_file"
}
