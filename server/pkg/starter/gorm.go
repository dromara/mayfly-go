package starter

import (
	"log"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/global"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func initDb() {
	global.Db = gormMysql()
}

func gormMysql() *gorm.DB {
	m := config.Conf.Mysql
	if m == nil || m.Dbname == "" {
		global.Log.Panic("未找到数据库配置信息")
		return nil
	}
	global.Log.Infof("连接mysql [%s]", m.Host)
	mysqlConfig := mysql.Config{
		DSN:                       m.Dsn(), // DSN data source name
		DefaultStringSize:         191,     // string 类型字段的默认长度
		DisableDatetimePrecision:  true,    // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true,    // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true,    // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: false,   // 根据版本自动配置
	}

	sqlLogLevel := logger.Error
	logConf := config.Conf.Log
	// 如果为配置文件中配置的系统日志级别为debug，则打印gorm执行的sql信息
	if logConf.Level == "debug" {
		sqlLogLevel = logger.Info
	}
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer（日志输出的目标，前缀和日志包含的内容——译者注）
		logger.Config{
			SlowThreshold:             time.Second, // 慢 SQL 阈值
			LogLevel:                  sqlLogLevel, // 日志级别, 改为logger.Info即可显示sql语句
			IgnoreRecordNotFoundError: true,        // 忽略ErrRecordNotFound（记录未找到）错误
			Colorful:                  true,        // 禁用彩色打印
		},
	)

	ormConfig := &gorm.Config{NamingStrategy: schema.NamingStrategy{
		TablePrefix:   "t_",
		SingularTable: true,
	}, Logger: gormLogger}

	if db, err := gorm.Open(mysql.New(mysqlConfig), ormConfig); err != nil {
		global.Log.Panicf("连接mysql失败! [%s]", err.Error())
		return nil
	} else {
		sqlDB, _ := db.DB()
		sqlDB.SetMaxIdleConns(m.MaxIdleConns)
		sqlDB.SetMaxOpenConns(m.MaxOpenConns)
		return db
	}
}
