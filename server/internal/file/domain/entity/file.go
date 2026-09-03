package entity

import "mayfly-go/pkg/model"

type File struct {
	model.Model
	model.ExtraData

	FileKey     string `json:"fileKey" gorm:"size:32;not null;"`   // 文件key
	Filename    string `json:"filename" gorm:"size:255;not null;"` // 文件名
	Path        string `json:"path" gorm:"size:500;"`              // 文件路径
	Size        int64  `json:"size"`
	StorageType string `json:"storageType" gorm:"size:20;not null;default:local;"` // 存储介质类型：local/s3，用于介质切换后仍按原介质读写存量文件
}

func (a *File) TableName() string {
	return "t_sys_file"
}
