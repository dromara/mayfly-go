package starter

import (
	"fmt"
	"log"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/logx"
	"strconv"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func initDB(dbConf DBConf) (*gorm.DB, error) {
	if dbConf.Dialect == DialectMySQL {
		return initMysql(dbConf)
	}

	return initSQLite(dbConf)
}

func initMysql(dbConf DBConf) (*gorm.DB, error) {
	logx.Infof("connecting to mysql [%s]", dbConf.Address)
	if dbConf.Config == "" {
		dbConf.Config = "charset=utf8&loc=Local&parseTime=true"
	}

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

// sqliteBusyTimeoutMs 锁冲突时的等待上限：SQLite 是单写者引擎，元数据库的写入大量来自请求与
// 定时任务，撞锁直接失败对用户就是「莫名报错」。5s 远小于请求超时，排不上也能自己退出
const sqliteBusyTimeoutMs = 5000

// sqliteDsn 给地址补上 busy 超时 pragma。已带查询串的地址用 & 追加，避免造出非法 DSN
func sqliteDsn(address string) string {
	pragma := "_pragma=busy_timeout(" + strconv.Itoa(sqliteBusyTimeoutMs) + ")"
	if strings.Contains(address, "?") {
		return address + "&" + pragma
	}
	return address + "?" + pragma
}

func initSQLite(dbConf DBConf) (*gorm.DB, error) {
	logx.Infof("connecting to sqlite [%s]", dbConf.Address)
	// 连接池沿用配置值，**刻意不收成一条连接**：单连接确实能让写入完全串行、从根上避开锁冲突，
	// 但代价是「事务内的读」永远等不到第二条连接而永久挂死 —— 基座那批不收 ctx 的历史读入口在
	// 业务事务链上大量存在（架构守卫只能看见直接写在 Tx 闭包里的那些，经仓储方法间接发生的读看不见），
	// 而永久挂死比一次可重试的锁错误严重得多。锁冲突改由 busy_timeout 排队解决
	db, err := gorm.Open(sqlite.Open(sqliteDsn(dbConf.Address)), getGormConfig())
	if err != nil {
		logx.Errorf("failed to connect to sqlite! [%s]", err.Error())
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sqlite handle: %w", err)
	}
	sqlDB.SetMaxIdleConns(dbConf.MaxIdleConns)
	sqlDB.SetMaxOpenConns(dbConf.MaxOpenConns)
	return db, nil
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

	// 表名策略取自 gormx 的单一真源：迁移端到端测试用的是同一份配置，
	// 两边解析出的表名才必然一致（详见 gormx.MetaNamingStrategy 的说明）
	return &gorm.Config{NamingStrategy: gormx.MetaNamingStrategy(), Logger: gormLogger}
}
