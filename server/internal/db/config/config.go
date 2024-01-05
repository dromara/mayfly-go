package config

import (
	sysapp "mayfly-go/internal/sys/application"
	"path/filepath"
	"runtime"
)

const (
	ConfigKeyDbSaveQuerySQL  string = "DbSaveQuerySQL"  // 数据库是否记录查询相关sql
	ConfigKeyDbQueryMaxCount string = "DbQueryMaxCount" // 数据库查询的最大数量
	ConfigKeyDbBackupRestore string = "DbBackupRestore" // 数据库备份
	ConfigKeyDbMysqlBin      string = "MysqlBin"        // mysql可执行文件配置
	ConfigKeyDbMariadbBin    string = "MariadbBin"      // mariadb可执行文件配置
)

// 获取数据库最大查询数量配置
func GetDbQueryMaxCount() int {
	return sysapp.GetConfigApp().GetConfig(ConfigKeyDbQueryMaxCount).IntValue(200)
}

// 获取数据库是否记录查询相关sql配置
func GetDbSaveQuerySql() bool {
	return sysapp.GetConfigApp().GetConfig(ConfigKeyDbSaveQuerySQL).BoolValue(false)
}

type DbBackupRestore struct {
	BackupPath string // 备份文件路径呢
}

// 获取数据库备份配置
func GetDbBackupRestore() *DbBackupRestore {
	c := sysapp.GetConfigApp().GetConfig(ConfigKeyDbBackupRestore)
	jm := c.GetJsonMap()

	dbrc := new(DbBackupRestore)

	backupPath := jm["backupPath"]
	if backupPath == "" {
		backupPath = "./db/backup"
	}
	dbrc.BackupPath = filepath.Join(backupPath)

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
