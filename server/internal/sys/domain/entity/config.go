package entity

import (
	"encoding/json"
	"mayfly-go/pkg/model"
	"strconv"
)

const (
	ConfigKeyUseLoginCaptcha string = "UseLoginCaptcha" // 是否使用登录验证码
	ConfigKeyDbQueryMaxCount string = "DbQueryMaxCount" // 数据库查询的最大数量
	ConfigKeyDbSaveQuerySQL  string = "DbSaveQuerySQL"  // 数据库是否记录查询相关sql
)

type Config struct {
	model.Model
	Name   string `json:"name"` // 配置名
	Key    string `json:"key"`  // 配置key
	Params string `json:"params"`
	Value  string `json:"value"`
	Remark string `json:"remark"`
}

func (a *Config) TableName() string {
	return "t_sys_config"
}

// 若配置信息不存在, 则返回传递的默认值.
// 否则只有value == "1"为true，其他为false
func (c *Config) BoolValue(defaultValue bool) bool {
	// 如果值不存在，则返回默认值
	if c.Id == 0 {
		return defaultValue
	}
	return c.Value == "1"
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
	if intV, err := strconv.Atoi(c.Value); err != nil {
		return defaultValue
	} else {
		return intV
	}
}
