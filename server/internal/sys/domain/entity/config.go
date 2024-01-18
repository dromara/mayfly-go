package entity

import (
	"encoding/json"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/conv"
)

const (
	ConfigUseWartermark string = "UseWartermark" // 是否使用水印
)

type Config struct {
	model.Model
	Name       string `json:"name"` // 配置名
	Key        string `json:"key"`  // 配置key
	Params     string `json:"params" gorm:"column:params;type:varchar(1500)"`
	Value      string `json:"value" gorm:"column:value;type:varchar(1500)"`
	Remark     string `json:"remark"`
	Permission string `json:"permission"` // 可操作该配置的权限
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

// 值返回json map
func (c *Config) GetJsonMap() map[string]string {
	var res map[string]string
	if c.Id == 0 || c.Value == "" {
		return res
	}
	_ = json.Unmarshal([]byte(c.Value), &res)
	return res
}

// 获取配置的int值，如果配置值非int或不存在，则返回默认值
func (c *Config) IntValue(defaultValue int) int {
	// 如果值不存在，则返回默认值
	if c.Id == 0 {
		return defaultValue
	}
	return conv.Str2Int(c.Value, defaultValue)
}

// 转换配置中的值为bool类型（默认"1"或"true"为true，其他为false）
func (c *Config) ConvBool(value string, defaultValue bool) bool {
	if value == "" {
		return defaultValue
	}
	return value == "1" || value == "true"
}
