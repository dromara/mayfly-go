package entity

import "mayfly-go/pkg/model"

type Resource struct {
	model.Model
	Pid    int64  `json:"pid" gorm:"not null;comment:父节点id;"`
	UiPath string `json:"ui_path" gorm:"size:300;comment:唯一标识路径;"`        // 唯一标识路径
	Type   int8   `json:"type" gorm:"not null;comment:1：菜单路由；2：资源（按钮等）;"` // 1：菜单路由；2：资源（按钮等）
	Status int8   `json:"status" gorm:"not null;comment:状态；1:可用，-1:禁用;"`  // 1：可用；-1：不可用
	Code   string `json:"code" gorm:"size:300;comment:菜单路由为path，其他为唯一标识;"`
	Name   string `json:"name" gorm:"size:255;not null;"`
	Weight int    `json:"weight"`
	Meta   string `json:"meta" gorm:"size:500;"`
}

func (a *Resource) TableName() string {
	return "t_sys_resource"
}

func (m *Resource) FillBaseInfo(idGenType model.IdGenType, la *model.LoginAccount) {
	// id使用时间戳，减少id冲突概率
	m.Model.FillBaseInfo(model.IdGenTypeTimestamp, la)
}

const (
	ResourceStatusEnable  int8 = 1  // 启用状态
	ResourceStatusDisable int8 = -1 // 禁用状态

	// 资源状态
	ResourceTypeMenu       int8 = 1
	ResourceTypePermission int8 = 2

	// 唯一标识路径分隔符
	ResourceUiPathSp string = "/"
)
