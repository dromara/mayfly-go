package config

type Db struct {
	BackupPath  string    `yaml:"backup-path"`
	MysqlUtil   MysqlUtil `yaml:"mysqlutil-path"`
	MariadbUtil MysqlUtil `yaml:"mariadbutil-path"`
}

type MysqlUtil struct {
	Mysql       string `yaml:"mysql"`
	MysqlDump   string `yaml:"mysqldump"`
	MysqlBinlog string `yaml:"mysqlbinlog"`
}
