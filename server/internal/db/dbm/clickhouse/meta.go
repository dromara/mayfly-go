package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

func init() {
	meta := new(Meta)
	dbi.Register(DbTypeClickHouse, meta)
}

const (
	DbTypeClickHouse dbi.DbType = "clickhouse"
)

var _ dbi.Meta = (*Meta)(nil)

type Meta struct {
}

func (cm *Meta) GetSqlDb(ctx context.Context, d *dbi.DbInfo) (*sql.DB, error) {
	// ClickHouse connection string
	// Format: clickhouse://username:password@host:port/database?param1=value1&param2=value2
	dbName := d.GetDatabase()
	if dbName == "" {
		dbName = "default"
	}

	// Build connection string
	dsn := fmt.Sprintf("clickhouse://%s:%s@%s:%d/%s", d.Username, d.Password, d.Host, d.Port, dbName)
	if d.Params != "" {
		dsn = fmt.Sprintf("%s?%s", dsn, d.Params)
	} else {
		// Add default parameters for better connection handling
		dsn = fmt.Sprintf("%s?dial_timeout=10s&read_timeout=30s", dsn)
	}

	const driverName = "clickhouse"
	return sql.Open(driverName, dsn)
}

func (cm *Meta) GetDialect(conn *dbi.DbConn) dbi.Dialect {
	return &ClickHouseDialect{dc: conn}
}

func (cm *Meta) GetMetadata(conn *dbi.DbConn) dbi.Metadata {
	return &ClickHouseMetadata{dc: conn}
}

func (cm *Meta) GetDbDataTypes() []*dbi.DbDataType {
	return collx.AsArray(
		UInt8, UInt16, UInt32, UInt64, Int8, Int16, Int32, Int64,
		Float32, Float64,
		String, FixedString,
		DateTime, DateTime64, Date, Date32,
		UUID,
		IPv4, IPv6,
		Bool,
		Decimal, Decimal32, Decimal64, Decimal128, Decimal256,
		Enum8, Enum16,
		Array, Tuple, Map, Nested, AggregateFunction, SimpleAggregateFunction,
		LowCardinality, Nullable,
	)
}

func (cm *Meta) GetCommonTypeConverter() dbi.CommonTypeConverter {
	return &commonTypeConverter{}
}

// Common type converter for ClickHouse
//
// 用于异构数据库迁移至clickhouse的类型转换。
// clickhouse的类型均不携带长度参数（如String、Int32），需清空源长度信息避免生成非法DDL
type commonTypeConverter struct{}

// clearLength 清空列的长度与精度。clickhouse类型均不携带长度/精度参数，
// 保留源长度会生成非法DDL；Decimal除外（必须携带精度，见Decimal方法）
func clearLength(col *dbi.Column) {
	col.CharMaxLength = 0
	col.NumPrecision = 0
	col.NumScale = 0
}

func (c *commonTypeConverter) Varchar(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return String
}

func (c *commonTypeConverter) Char(column *dbi.Column) *dbi.DbDataType {
	// clickhouse不使用FixedString承载：其必须显式指定长度且以\0定长填充，会改变数据语义
	clearLength(column)
	return String
}

func (c *commonTypeConverter) Text(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return String
}

func (c *commonTypeConverter) Mediumtext(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return String
}

func (c *commonTypeConverter) Longtext(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return String
}

func (c *commonTypeConverter) Bit(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return UInt8
}

func (c *commonTypeConverter) Bool(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return Bool
}

func (c *commonTypeConverter) Int1(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return Int8
}

func (c *commonTypeConverter) Int2(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return Int16
}

func (c *commonTypeConverter) Int4(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return Int32
}

func (c *commonTypeConverter) Int8(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return Int64
}

// Numeric 浮点不保留精度，统一Float64
func (c *commonTypeConverter) Numeric(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return Float64
}

// Decimal 必须保留精度：clickhouse的Decimal(P,S)依赖精度参数化，
// 由GenTableDDL输出时补齐S缺失的格式；不声明精度时CH等价Decimal(10,0)，
// 无界精确数值源列（pg的numeric、sqlite声明的numeric）必须补齐宽度，否则小数被静默截断，
// (38,19)为Decimal128可表示且整数部分仍能容纳int64的选择；精度超出CH上限(76)时收敛避免非法DDL
func (c *commonTypeConverter) Decimal(column *dbi.Column) *dbi.DbDataType {
	dbi.FillUnboundedDecimal(column, 38, 19)
	dbi.ClampDecimalPrecision(column, 76, 76)
	return Decimal
}

func (c *commonTypeConverter) UnsignedInt8(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	// 无符号bigint（最大2^64-1）必须映射UInt64，映射UInt8会导致大于255的值插入失败
	return UInt64
}

func (c *commonTypeConverter) UnsignedInt4(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return UInt32
}

func (c *commonTypeConverter) UnsignedInt2(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return UInt16
}

func (c *commonTypeConverter) UnsignedInt1(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return UInt8
}

func (c *commonTypeConverter) Date(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return Date
}

func (c *commonTypeConverter) Time(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return DateTime
}

func (c *commonTypeConverter) Datetime(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return DateTime
}

func (c *commonTypeConverter) Timestamp(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return DateTime
}

// 二进制以String承载（clickhouse String可存储任意字节序列）
func (c *commonTypeConverter) Binary(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return String
}

func (c *commonTypeConverter) Varbinary(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return String
}

func (c *commonTypeConverter) Mediumblob(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return String
}

func (c *commonTypeConverter) Blob(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return String
}

func (c *commonTypeConverter) Longblob(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return String
}

// Enum Enum8必须显式声明枚举值集合（如Enum8('a'=1)），源枚举定义迁移后不可知，退化为String
func (c *commonTypeConverter) Enum(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return String
}

func (c *commonTypeConverter) JSON(column *dbi.Column) *dbi.DbDataType {
	clearLength(column)
	return String
}
