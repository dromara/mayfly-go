package dm

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"net/url"
	"strings"
)

func init() {
	dbi.Register(DbTypeDM, new(Meta))
}

const (
	DbTypeDM dbi.DbType = "dm"
)

type Meta struct {
}

func (dm *Meta) GetSqlDb(ctx context.Context, d *dbi.DbInfo) (*sql.DB, error) {
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

	err := d.IfUseSshTunnelChangeIpPort(ctx)
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("dm://%s:%s@%s:%d/%s", d.Username, url.PathEscape(d.Password), d.Host, d.Port, dbParam)
	return sql.Open(driverName, dsn)
}

func (dm *Meta) GetDialect(conn *dbi.DbConn) dbi.Dialect {
	return &DMDialect{dc: conn}
}

func (dm *Meta) GetMetadata(conn *dbi.DbConn) dbi.Metadata {
	return &DMMetadata{
		dc: conn,
	}
}

func (sm *Meta) GetDbDataTypes() []*dbi.DbDataType {
	return collx.AsArray[*dbi.DbDataType](CHAR, VARCHAR, TEXT, LONG, LONGVARCHAR, IMAGE, LONGVARBINARY, CLOB,
		BLOB,
		NUMERIC, DECIMAL, NUMBER, INTEGER, INT, BIGINT, TINYINT, BYTE, SMALLINT, BIT, DOUBLE, FLOAT,
		TIME, DATE, TIMESTAMP, DATETIME,
		ST_CURVE, ST_LINESTRING, ST_GEOMCOLLECTION, ST_GEOMETRY, ST_MULTICURVE, ST_MULTILINESTRING,
		ST_MULTIPOINT, ST_MULTIPOLYGON, ST_MULTISURFACE, ST_POINT, ST_POLYGON, ST_SURFACE,
		TABLES,
	)
}

func (mm *Meta) GetCommonTypeConverter() dbi.CommonTypeConverter {
	return &commonTypeConverter{}
}
