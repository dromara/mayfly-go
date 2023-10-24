package starter

import (
	"log"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/logx"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func initDb() {
	global.Db = initGormDb()
}

func initGormDb() *gorm.DB {
	m := config.Conf.Mysql
	// 存在msyql数据库名，则优先使用mysql
	if m.Dbname != "" {
		return initMysql(m)
	}

	return initSqlite(config.Conf.Sqlite)
}

func initMysql(m config.Mysql) *gorm.DB {
	logx.Infof("连接mysql [%s]", m.Host)
	mysqlConfig := mysql.Config{
		DSN:                       m.Dsn(), // DSN data source name
		DefaultStringSize:         191,     // string 类型字段的默认长度
		DisableDatetimePrecision:  true,    // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true,    // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true,    // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: false,   // 根据版本自动配置
	}

	if db, err := gorm.Open(mysql.New(mysqlConfig), getGormConfig()); err != nil {
		logx.Panicf("连接mysql失败! [%s]", err.Error())
		return nil
	} else {
		sqlDB, _ := db.DB()
		sqlDB.SetMaxIdleConns(m.MaxIdleConns)
		sqlDB.SetMaxOpenConns(m.MaxOpenConns)
		return db
	}
}

func initSqlite(sc config.Sqlite) *gorm.DB {
	logx.Infof("连接sqlite [%s]", sc.Path)
	if db, err := gorm.Open(sqlite.Open(sc.Path), getGormConfig()); err != nil {
		logx.Panicf("连接sqlite失败! [%s]", err.Error())
		return nil
	} else {
		sqlDB, _ := db.DB()
		sqlDB.SetMaxIdleConns(sc.MaxIdleConns)
		sqlDB.SetMaxOpenConns(sc.MaxOpenConns)
		return db
	}
}

func getGormConfig() *gorm.Config {
	sqlLogLevel := logger.Error
	logConf := logx.GetConfig()
	// 如果为配置文件中配置的系统日志级别为debug，则打印gorm执行的sql信息
	if logConf.IsDebug() {
		sqlLogLevel = logger.Info
	}

	gormLogger := logger.New(
		log.New(logConf.GetLogOut(), "\r\n", log.LstdFlags), // io writer（日志输出的目标，前缀和日志包含的内容——译者注）
		logger.Config{
			SlowThreshold:             time.Second, // 慢 SQL 阈值
			LogLevel:                  sqlLogLevel, // 日志级别, 改为logger.Info即可显示sql语句
			IgnoreRecordNotFoundError: true,        // 忽略ErrRecordNotFound（记录未找到）错误
			Colorful:                  false,       // 禁用彩色打印
		},
	)

	return &gorm.Config{NamingStrategy: schema.NamingStrategy{
		TablePrefix:   "t_",
		SingularTable: true,
	}, Logger: gormLogger}
}
