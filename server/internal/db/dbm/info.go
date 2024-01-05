package dbm

import (
	"database/sql"
	"fmt"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
)

type DbInfo struct {
	InstanceId uint64 // 实例id
	Id         uint64 // dbId
	Name       string

	Type     DbType // 类型，mysql postgres等
	Host     string
	Port     int
	Network  string
	Username string
	Password string
	Params   string
	Database string

	TagPath            []string
	SshTunnelMachineId int
}

// 获取记录日志的描述
func (d *DbInfo) GetLogDesc() string {
	return fmt.Sprintf("DB[id=%d, tag=%s, name=%s, ip=%s:%d, database=%s]", d.Id, d.TagPath, d.Name, d.Host, d.Port, d.Database)
}

// 连接数据库
func (dbInfo *DbInfo) Conn() (*DbConn, error) {
	var conn *sql.DB
	var err error
	database := dbInfo.Database

	switch dbInfo.Type {
	case DbTypeMysql, DbTypeMariadb:
		conn, err = getMysqlDB(dbInfo)
	case DbTypePostgres:
		conn, err = getPgsqlDB(dbInfo)
	case DbTypeDM:
		conn, err = getDmDB(dbInfo)
	default:
		return nil, errorx.NewBiz("invalid database type: %s", dbInfo.Type)
	}

	if err != nil {
		logx.Errorf("连接db失败: %s:%d/%s, err:%s", dbInfo.Host, dbInfo.Port, database, err.Error())
		return nil, errorx.NewBiz(fmt.Sprintf("数据库连接失败: %s", err.Error()))
	}

	err = conn.Ping()
	if err != nil {
		logx.Errorf("db ping失败: %s:%d/%s, err:%s", dbInfo.Host, dbInfo.Port, database, err.Error())
		return nil, errorx.NewBiz(fmt.Sprintf("数据库连接失败: %s", err.Error()))
	}

	dbc := &DbConn{Id: GetDbConnId(dbInfo.Id, database), Info: dbInfo}

	// 最大连接周期，超过时间的连接就close
	// conn.SetConnMaxLifetime(100 * time.Second)
	// 设置最大连接数
	conn.SetMaxOpenConns(5)
	// 设置闲置连接数
	conn.SetMaxIdleConns(1)
	dbc.db = conn
	logx.Infof("连接db: %s:%d/%s", dbInfo.Host, dbInfo.Port, database)

	return dbc, nil
}

// 获取连接id
func GetDbConnId(dbId uint64, db string) string {
	if dbId == 0 {
		return ""
	}

	return fmt.Sprintf("%d:%s", dbId, db)
}
