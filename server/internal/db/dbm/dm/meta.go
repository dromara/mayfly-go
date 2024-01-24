package dm

import (
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"net/url"
	"strings"
)

func init() {
	dbi.Register(dbi.DbTypeDM, new(DmMeta))
}

type DmMeta struct {
}

func (md *DmMeta) GetSqlDb(d *dbi.DbInfo) (*sql.DB, error) {
	driverName := "dm"
	db := d.Database
	var dbParam string
	if db != "" {
		// dm database可以使用db/schema表示，方便连接指定schema, 若不存在schema则使用默认schema
		ss := strings.Split(db, "/")
		if len(ss) > 1 {
			dbParam = fmt.Sprintf("%s?schema=\"%s\"&escapeProcess=true", ss[0], ss[len(ss)-1])
		} else {
			dbParam = db + "?escapeProcess=true"
		}
	} else {
		dbParam = "?escapeProcess=true"
	}

	err := d.IfUseSshTunnelChangeIpPort()
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("dm://%s:%s@%s:%d/%s", d.Username, url.PathEscape(d.Password), d.Host, d.Port, dbParam)
	return sql.Open(driverName, dsn)
}

func (md *DmMeta) GetDialect(conn *dbi.DbConn) dbi.Dialect {
	return &DMDialect{conn}
}
