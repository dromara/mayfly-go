package config

import (
	"cmp"
	sysapp "mayfly-go/internal/sys/application"
	"strings"

	"github.com/spf13/cast"
)

const (
	// ConfigKeyDbms dbms 相关运行时配置（结果集上限/执行超时/脱敏开关）
	ConfigKeyDbms string = "DbmsConfig"
)

type Dbms struct {
	QuerySQLSave bool // 是否记录查询类sql
	MaxResultSet int  // 允许sql查询的最大结果集数。注: 0=不限制
	SQLExecTl    int  // sql执行时间限制，超过该时间（单位：秒），执行将被取消

	MaskEnabled       bool     // 是否启用查询结果字段脱敏
	MaskExemptRoleIds []uint64 // 脱敏豁免角色id列表，命中角色的账号查询结果不脱敏
	MaskFailClosed    bool     // 脱敏计划构建失败时是否阻断查询：false降级为不脱敏（fail-open），true返回错误阻断查询（fail-close，安全敏感部署建议开启）
}

func GetDbms() *Dbms {
	c := sysapp.GetConfigApp().GetConfig(ConfigKeyDbms)
	jm := c.GetJsonM()

	dbmsConf := new(Dbms)
	dbmsConf.QuerySQLSave = c.ConvBool(jm.GetStr("querySqlSave"), false)
	dbmsConf.MaxResultSet = jm.GetInt("maxResultSet")
	dbmsConf.SQLExecTl = cmp.Or(jm.GetInt("sqlExecTl"), 60)
	dbmsConf.MaskEnabled = c.ConvBool(jm.GetStr("maskEnabled"), false)
	dbmsConf.MaskFailClosed = c.ConvBool(jm.GetStr("maskFailClosed"), false)
	dbmsConf.MaskExemptRoleIds = parseMaskExemptRoleIds(jm["maskExemptRoleIds"])
	return dbmsConf
}

// parseMaskExemptRoleIds 解析脱敏豁免角色id：兼容逗号分隔字符串（系统配置动态表单存字符串）与数组两种格式
func parseMaskExemptRoleIds(val any) []uint64 {
	var strs []string
	switch v := val.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		strs = strings.Split(v, ",")
	case []any:
		for _, item := range v {
			strs = append(strs, cast.ToString(item))
		}
	default:
		return nil
	}

	ids := make([]uint64, 0, len(strs))
	for _, s := range strs {
		if s = strings.TrimSpace(s); s == "" {
			continue
		}
		ids = append(ids, uint64(cast.ToInt64(s)))
	}
	return ids
}
