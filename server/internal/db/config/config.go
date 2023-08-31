package config

import sysapp "mayfly-go/internal/sys/application"

const (
	ConfigKeyDbSaveQuerySQL  string = "DbSaveQuerySQL"  // 数据库是否记录查询相关sql
	ConfigKeyDbQueryMaxCount string = "DbQueryMaxCount" // 数据库查询的最大数量
)

// 获取数据库最大查询数量配置
func GetDbQueryMaxCount() int {
	return sysapp.GetConfigApp().GetConfig(ConfigKeyDbQueryMaxCount).IntValue(200)
}

// 获取数据库是否记录查询相关sql配置
func GetDbSaveQuerySql() bool {
	return sysapp.GetConfigApp().GetConfig(ConfigKeyDbSaveQuerySQL).BoolValue(false)
}
