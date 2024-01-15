package dbi

import (
	"fmt"
	machineapp "mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/mcm"
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
	Sid      string // oracle数据库需要指定sid
	Network  string
	Username string
	Password string
	Params   string
	Database string

	TagPath            []string
	SshTunnelMachineId int

	Meta Meta
}

// 获取记录日志的描述
func (d *DbInfo) GetLogDesc() string {
	return fmt.Sprintf("DB[id=%d, tag=%s, name=%s, ip=%s:%d, database=%s]", d.Id, d.TagPath, d.Name, d.Host, d.Port, d.Database)
}

// 连接数据库
func (dbInfo *DbInfo) Conn(meta Meta) (*DbConn, error) {
	if meta == nil {
		return nil, errorx.NewBiz("数据库元信息接口不能为空")
	}

	// 赋值Meta，方便后续获取dialect等
	dbInfo.Meta = meta
	database := dbInfo.Database

	conn, err := meta.GetSqlDb(dbInfo)
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

// 如果使用了ssh隧道，将其host port改变其本地映射host port
func (di *DbInfo) IfUseSshTunnelChangeIpPort() error {
	// 开启ssh隧道
	if di.SshTunnelMachineId > 0 {
		sshTunnelMachine, err := GetSshTunnel(di.SshTunnelMachineId)
		if err != nil {
			return err
		}
		exposedIp, exposedPort, err := sshTunnelMachine.OpenSshTunnel(fmt.Sprintf("db:%d", di.Id), di.Host, di.Port)
		if err != nil {
			return err
		}
		di.Host = exposedIp
		di.Port = exposedPort
	}
	return nil
}

// 根据ssh tunnel机器id返回ssh tunnel
func GetSshTunnel(sshTunnelMachineId int) (*mcm.SshTunnelMachine, error) {
	return machineapp.GetMachineApp().GetSshTunnelMachine(sshTunnelMachineId)
}

// 获取连接id
func GetDbConnId(dbId uint64, db string) string {
	if dbId == 0 {
		return ""
	}

	return fmt.Sprintf("%d:%s", dbId, db)
}
