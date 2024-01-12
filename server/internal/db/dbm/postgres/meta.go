package postgres

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	machineapp "mayfly-go/internal/machine/application"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/netx"
	"net"
	"strings"
	"sync"
	"time"

	pq "gitee.com/liuzongyang/libpq"
)

var (
	meta dbi.Meta
	once sync.Once
)

func GetMeta() dbi.Meta {
	once.Do(func() {
		meta = new(PostgresMeta)
	})
	return meta
}

type PostgresMeta struct {
}

func (md *PostgresMeta) GetSqlDb(d *dbi.DbInfo) (*sql.DB, error) {
	driverName := string(d.Type)
	// SSH Conect
	if d.SshTunnelMachineId > 0 {
		// 如果使用了隧道，则使用`postgres:ssh:隧道机器id`注册名
		driverName = fmt.Sprintf("postgres:ssh:%d", d.SshTunnelMachineId)
		if !collx.ArrayContains(sql.Drivers(), driverName) {
			sql.Register(driverName, &PqSqlDialer{sshTunnelMachineId: d.SshTunnelMachineId})
		}
		sql.Drivers()
	}

	db := d.Database
	var dbParam string
	exsitSchema := false
	if db != "" {
		// postgres database可以使用db/schema表示，方便连接指定schema, 若不存在schema则使用默认schema
		ss := strings.Split(db, "/")
		if len(ss) > 1 {
			exsitSchema = true
			dbParam = fmt.Sprintf("dbname=%s search_path=%s", ss[0], ss[len(ss)-1])
		} else {
			dbParam = "dbname=" + db
		}
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s %s sslmode=disable connect_timeout=8", d.Host, d.Port, d.Username, d.Password, dbParam)
	// 存在额外指定参数，则拼接该连接参数
	if d.Params != "" {
		// 存在指定的db，则需要将dbInstance配置中的parmas排除掉dbname和search_path
		if db != "" {
			paramArr := strings.Split(d.Params, "&")
			paramArr = collx.ArrayRemoveFunc(paramArr, func(param string) bool {
				if strings.HasPrefix(param, "dbname=") {
					return true
				}
				if exsitSchema && strings.HasPrefix(param, "search_path") {
					return true
				}
				return false
			})
			d.Params = strings.Join(paramArr, " ")
		}
		dsn = fmt.Sprintf("%s %s", dsn, strings.Join(strings.Split(d.Params, "&"), " "))
	}

	return sql.Open(driverName, dsn)
}

func (md *PostgresMeta) GetDialect(conn *dbi.DbConn) dbi.Dialect {
	return &PgsqlDialect{conn}
}

// pgsql dialer
type PqSqlDialer struct {
	sshTunnelMachineId int
}

func (d *PqSqlDialer) Open(name string) (driver.Conn, error) {
	return pq.DialOpen(d, name)
}

func (pd *PqSqlDialer) Dial(network, address string) (net.Conn, error) {
	sshTunnel, err := machineapp.GetMachineApp().GetSshTunnelMachine(pd.sshTunnelMachineId)
	if err != nil {
		return nil, err
	}
	if sshConn, err := sshTunnel.GetDialConn("tcp", address); err == nil {
		// 将ssh conn包装，否则会返回错误: ssh: tcpChan: deadline not supported
		return &netx.WrapSshConn{Conn: sshConn}, nil
	} else {
		return nil, err
	}
}

func (pd *PqSqlDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return pd.Dial(network, address)
}
