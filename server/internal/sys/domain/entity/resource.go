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

// GenerateId 用户新增的资源（菜单/按钮）由应用侧发秒级时间戳主键，与官方迁移脚本为内置菜单
// 写死的 id（同为迁移动作时刻的秒级时间戳）落在同一条时间轴上：用户发号值就是创建时刻，
// 官方固定值是历史时刻，两者不靠自增区间的相邻关系避让。
//
// TimestampId 保证单进程同秒不重；与某条官方迁移的固定值同秒相撞是极低概率的显式主键冲突
// 报错，不会静默写坏数据
func (a *Resource) GenerateId() uint64 {
	return model.TimestampId()
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
