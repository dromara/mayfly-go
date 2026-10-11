package entity

import "mayfly-go/pkg/model"

// AccessMode 文件访问模式：决定能否经通用文件路由（GET /sys/files/:key）匿名读取。
//
// 该路由不校验 token，是因为头像、AI 附件等展示类资产由 <img src>/<a href> 直接引用，
// 浏览器无法为其附加 Authorization 头。但若所有文件都吃这条匿名路由，任何业务产物（数据库
// 备份/导出文件）只要 key 泄露就能无凭证拉走明文数据，因此按用途区分访问模式。
type AccessMode int8

const (
	// FileAccessModePublic 展示类资产（头像、AI 附件等），允许匿名读取
	FileAccessModePublic AccessMode = 1

	// FileAccessModePrivate 数据类产物，禁止经通用文件路由读取，必须由所属业务模块声明
	// 权限码的下载入口访问（如迁移导出文件走 GET /dbTransfer/files/:id/content）
	FileAccessModePrivate AccessMode = 2
)

type File struct {
	model.Model
	model.ExtraData

	FileKey     string     `json:"fileKey" gorm:"size:32;not null;"`   // 文件key
	Filename    string     `json:"filename" gorm:"size:255;not null;"` // 文件名
	Path        string     `json:"path" gorm:"size:500;"`              // 文件路径
	Size        int64      `json:"size"`
	StorageType string     `json:"storageType" gorm:"size:20;not null;default:local;"` // 存储介质类型：local/s3，用于介质切换后仍按原介质读写存量文件
	AccessMode  AccessMode `json:"accessMode" gorm:"type:tinyint;not null;default:1;comment:访问模式 1公开可匿名读 2私有须走业务模块下载入口"`
}

func (a *File) TableName() string {
	return "t_sys_file"
}

// IsPublic 报告该文件是否允许匿名读取。
//
// 判定刻意写成「只有显式等于 public 才公开」而不是「不等于 private 就公开」：
// 历史遗留或写入方漏赋值时字段是零值 0（既非 1 也非 2），若按「非 private 即公开」解释，
// 这些行会变成匿名可下的数据文件。未知取值一律按私有处理（fail-closed），
// 宁可让一次下载失败可查，也不能把数据静默暴露给匿名请求。
func (a *File) IsPublic() bool {
	return a.AccessMode == FileAccessModePublic
}
