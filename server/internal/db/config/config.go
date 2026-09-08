package config

import (
	"cmp"
	sysapp "mayfly-go/internal/sys/application"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cast"
)

const (
	ConfigKeyDbms            string = "DbmsConfig"      // dbms相关配置信息
	ConfigKeyDbBackupRestore string = "DbBackupRestore" // 数据库备份
	ConfigKeyDbMysqlBin      string = "MysqlBin"        // mysql可执行文件配置
	ConfigKeyDbMariadbBin    string = "MariadbBin"      // mariadb可执行文件配置
)

type Dbms struct {
	QuerySqlSave bool // 是否记录查询类sql
	MaxResultSet int  // 允许sql查询的最大结果集数。注: 0=不限制
	SqlExecTl    int  // sql执行时间限制，超过该时间（单位：秒），执行将被取消

	MaskEnabled       bool     // 是否启用查询结果字段脱敏
	MaskExemptRoleIds []uint64 // 脱敏豁免角色id列表，命中角色的账号查询结果不脱敏
	MaskFailClosed    bool     // 脱敏计划构建失败时是否阻断查询：false降级为不脱敏（fail-open），true返回错误阻断查询（fail-close，安全敏感部署建议开启）
}

func GetDbms() *Dbms {
	c := sysapp.GetConfigApp().GetConfig(ConfigKeyDbms)
	jm := c.GetJsonM()

	dbmsConf := new(Dbms)
	dbmsConf.QuerySqlSave = c.ConvBool(jm.GetStr("querySqlSave"), false)
	dbmsConf.MaxResultSet = jm.GetInt("maxResultSet")
	dbmsConf.SqlExecTl = cmp.Or(jm.GetInt("sqlExecTl"), 60)
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

type DbBackupRestore struct {
	BackupPath   string // 备份文件路径呢
	TransferPath string // 数据库迁移文件存储路径
}

// 获取数据库备份配置
func GetDbBackupRestore() *DbBackupRestore {
	c := sysapp.GetConfigApp().GetConfig(ConfigKeyDbBackupRestore)
	jm := c.GetJsonM()

	dbrc := new(DbBackupRestore)

	dbrc.BackupPath = filepath.Join(cmp.Or(jm.GetStr("backupPath"), "./db/backup"))
	dbrc.TransferPath = filepath.Join(cmp.Or(jm.GetStr("transferPath"), "./db/transfer"))

	return dbrc
}

// mysql客户端可执行文件配置
type MysqlBin struct {
	Path            string // 可执行文件路径
	MysqlPath       string // mysql可执行文件路径
	MysqldumpPath   string // mysqldump可执行文件路径
	MysqlbinlogPath string // mysqlbinlog可执行文件路径
}

// 获取数据库备份配置
func GetMysqlBin(configKey string) *MysqlBin {
	c := sysapp.GetConfigApp().GetConfig(configKey)
	jm := c.GetJsonM()

	mbc := new(MysqlBin)

	path := jm.GetStr("path")
	if path == "" {
		path = "./db/mysql/bin"
	}
	mbc.Path = filepath.Join(path)

	var extName string
	if runtime.GOOS == "windows" {
		extName = ".exe"
	}
	mysqlPath := jm.GetStr("mysql")
	if mysqlPath == "" {
		mysqlPath = filepath.Join(path, "mysql"+extName)
	}
	mbc.MysqlPath = filepath.Join(mysqlPath)

	mysqldumpPath := jm.GetStr("mysqldump")
	if mysqldumpPath == "" {
		mysqldumpPath = filepath.Join(path, "mysqldump"+extName)
	}
	mbc.MysqldumpPath = filepath.Join(mysqldumpPath)

	mysqlbinlogPath := jm.GetStr("mysqlbinlog")
	if mysqlbinlogPath == "" {
		mysqlbinlogPath = filepath.Join(path, "mysqlbinlog"+extName)
	}
	mbc.MysqlbinlogPath = filepath.Join(mysqlbinlogPath)

	return mbc
}
