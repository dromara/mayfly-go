package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"net"
	"sync"

	"github.com/go-sql-driver/mysql"
)

var (
	meta dbi.Meta
	once sync.Once
)

func GetMeta() dbi.Meta {
	once.Do(func() {
		meta = new(MysqlMeta)
	})
	return meta
}

type MysqlMeta struct {
}

func (md *MysqlMeta) GetSqlDb(d *dbi.DbInfo) (*sql.DB, error) {
	// SSH Conect
	if d.SshTunnelMachineId > 0 {
		sshTunnelMachine, err := dbi.GetSshTunnel(d.SshTunnelMachineId)
		if err != nil {
			return nil, err
		}
		mysql.RegisterDialContext(d.Network, func(ctx context.Context, addr string) (net.Conn, error) {
			return sshTunnelMachine.GetDialConn("tcp", addr)
		})
	}
	// 设置dataSourceName  -> 更多参数参考：https://github.com/go-sql-driver/mysql#dsn-data-source-name
	dsn := fmt.Sprintf("%s:%s@%s(%s:%d)/%s?timeout=8s", d.Username, d.Password, d.Network, d.Host, d.Port, d.Database)
	if d.Params != "" {
		dsn = fmt.Sprintf("%s&%s", dsn, d.Params)
	}
	const driverName = "mysql"
	return sql.Open(driverName, dsn)
}

func (md *MysqlMeta) GetDialect(conn *dbi.DbConn) dbi.Dialect {
	return &MysqlDialect{conn}
}
