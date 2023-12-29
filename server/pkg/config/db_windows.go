package config

func (db *Db) Default() {
	if len(db.BackupPath) == 0 {
		db.BackupPath = "./backup"
	}

	if len(db.MysqlUtil.Mysql) == 0 {
		db.MysqlUtil.Mysql = "./mysqlutil/bin/mysql.exe"
	}
	if len(db.MysqlUtil.MysqlDump) == 0 {
		db.MysqlUtil.MysqlDump = "./mysqlutil/bin/mysqldump.exe"
	}
	if len(db.MysqlUtil.Mysql) == 0 {
		db.MysqlUtil.MysqlBinlog = "./mysqlutil/bin/mysqlbinlog.exe"
	}

	if len(db.MariadbUtil.Mysql) == 0 {
		db.MariadbUtil.Mysql = "./mariadbutil/bin/mariadb.exe"
	}
	if len(db.MariadbUtil.MysqlDump) == 0 {
		db.MariadbUtil.MysqlDump = "./mariadbutil/bin/mariadb-dump.exe"
	}
	if len(db.MariadbUtil.MysqlBinlog) == 0 {
		db.MariadbUtil.MysqlBinlog = "./mariadbutil/bin/mariadb-binlog.exe"
	}
}
