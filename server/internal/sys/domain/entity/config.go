package entity

import (
	"cmp"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"

	"github.com/spf13/cast"
)

const (
	ConfigUseWartermark string = "UseWartermark" // 是否使用水印
)

type Config struct {
	model.Model
	Name       string `json:"name" gorm:"size:60;not null;"` // 配置名
	Key        string `json:"key" gorm:"size:60;not null;"`  // 配置key
	Params     string `json:"params" gorm:"size:4000"`
	Value      string `json:"value" gorm:"size:4000"`
	Remark     string `json:"remark" gorm:"size:255"`
	Permission string `json:"permission" gorm:"size:255;comment:操作权限"` // 可操作该配置的权限
}

func (a *Config) TableName() string {
	return "t_sys_config"
}

// 若配置信息不存在, 则返回传递的默认值.
func (c *Config) BoolValue(defaultValue bool) bool {
	// 如果值不存在，则返回默认值
	if c.Id == 0 {
		return defaultValue
	}
	return c.ConvBool(c.Value, defaultValue)
}

// GetJsonM 获取配置的json map值，如果配置不存在或值为空，则返回空map
func (c *Config) GetJsonM() collx.M {
	if c.Id == 0 || c.Value == "" {
		return collx.M{}
	}
	res, _ := jsonx.ToMapByStr(c.Value)
	return res
}

// 获取配置的int值，如果配置值非int或不存在，则返回默认值
func (c *Config) IntValue(defaultValue int) int {
	// 如果值不存在，则返回默认值
	if c.Id == 0 {
		return defaultValue
	}
	return cmp.Or(cast.ToInt(c.Value), defaultValue)
}

// 转换配置中的值为bool类型（默认"1"或"true"为true，其他为false）
func (c *Config) ConvBool(value string, defaultValue bool) bool {
	if value == "" {
		return defaultValue
	}
	return value == "1" || value == "true"
}
