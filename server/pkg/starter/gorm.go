package starter

import (
	"log"
	"mayfly-go/pkg/logx"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func initDB(dbConf DBConf) (*gorm.DB, error) {
	if dbConf.Dialect == DialectMySQL {
		return initMysql(dbConf)
	}

	return initSqlite(dbConf)
}

func initMysql(dbConf DBConf) (*gorm.DB, error) {
	logx.Infof("connecting to mysql [%s]", dbConf.Address)
	mysqlConfig := mysql.Config{
		DSN:                       dbConf.Username + ":" + dbConf.Password + "@tcp(" + dbConf.Address + ")/" + dbConf.Name + "?" + dbConf.Config, // DSN data source name
		DefaultStringSize:         191,                                                                                                           // string 类型字段的默认长度
		DisableDatetimePrecision:  true,                                                                                                          // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true,                                                                                                          // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true,                                                                                                          // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: false,                                                                                                         // 根据版本自动配置
	}

	if db, err := gorm.Open(mysql.New(mysqlConfig), getGormConfig()); err != nil {
		logx.Errorf("failed to connect to mysql! [%s]", err.Error())
		return nil, err
	} else {
		sqlDB, _ := db.DB()
		sqlDB.SetMaxIdleConns(dbConf.MaxIdleConns)
		sqlDB.SetMaxOpenConns(dbConf.MaxOpenConns)

		// 如果是开发环境时，打印sql语句
		if logx.GetConfig().IsDebug() {
			db = db.Debug()
		}
		return db, nil
	}
}

func initSqlite(dbConf DBConf) (*gorm.DB, error) {
	logx.Infof("connecting to sqlite [%s]", dbConf.Address)
	if db, err := gorm.Open(sqlite.Open(dbConf.Address), getGormConfig()); err != nil {
		logx.Errorf("failed to connect to sqlite! [%s]", err.Error())
		return nil, err
	} else {
		sqlDB, _ := db.DB()
		sqlDB.SetMaxIdleConns(dbConf.MaxIdleConns)
		sqlDB.SetMaxOpenConns(dbConf.MaxOpenConns)
		return db, nil
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
