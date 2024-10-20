package config

import (
	"cmp"
	sysapp "mayfly-go/internal/sys/application"
	"path/filepath"
	"runtime"

	"github.com/may-fly/cast"
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
}

func GetDbms() *Dbms {
	c := sysapp.GetConfigApp().GetConfig(ConfigKeyDbms)
	jm := c.GetJsonMap()

	dbmsConf := new(Dbms)
	dbmsConf.QuerySqlSave = c.ConvBool(jm["querySqlSave"], false)
	dbmsConf.MaxResultSet = cast.ToInt(jm["maxResultSet"])
	dbmsConf.SqlExecTl = cast.ToIntD(jm["sqlExecTl"], 60)
	return dbmsConf
}

type DbBackupRestore struct {
	BackupPath   string // 备份文件路径呢
	TransferPath string // 数据库迁移文件存储路径
}

// 获取数据库备份配置
func GetDbBackupRestore() *DbBackupRestore {
	c := sysapp.GetConfigApp().GetConfig(ConfigKeyDbBackupRestore)
	jm := c.GetJsonMap()

	dbrc := new(DbBackupRestore)

	dbrc.BackupPath = filepath.Join(cmp.Or(jm["backupPath"], "./db/backup"))
	dbrc.TransferPath = filepath.Join(cmp.Or(jm["transferPath"], "./db/transfer"))

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
	jm := c.GetJsonMap()

	mbc := new(MysqlBin)

	path := jm["path"]
	if path == "" {
		path = "./db/mysql/bin"
	}
	mbc.Path = filepath.Join(path)

	var extName string
	if runtime.GOOS == "windows" {
		extName = ".exe"
	}
	mysqlPath := jm["mysql"]
	if mysqlPath == "" {
		mysqlPath = filepath.Join(path, "mysql"+extName)
	}
	mbc.MysqlPath = filepath.Join(mysqlPath)

	mysqldumpPath := jm["mysqldump"]
	if mysqldumpPath == "" {
		mysqldumpPath = filepath.Join(path, "mysqldump"+extName)
	}
	mbc.MysqldumpPath = filepath.Join(mysqldumpPath)

	mysqlbinlogPath := jm["mysqlbinlog"]
	if mysqlbinlogPath == "" {
		mysqlbinlogPath = filepath.Join(path, "mysqlbinlog"+extName)
	}
	mbc.MysqlbinlogPath = filepath.Join(mysqlbinlogPath)

	return mbc
}
