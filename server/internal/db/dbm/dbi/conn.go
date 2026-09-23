package dbi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"mayfly-go/internal/db/dbm/dbi/scan"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
)

// 游标遍历查询结果集处理函数
type WalkQueryRowsFunc func(row map[string]any, columns []*QueryColumn) error

// db实例连接信息
type DbConn struct {
	Id   string
	Info *DbInfo

	// tunnel 该连接建立时持有的中转通道句柄，连接关闭时释放；无通道时为 nil
	tunnel *Tunnel
}

/******************* pool.Conn impl *******************/

// 关闭连接
func (d *DbConn) Close() error {
	if db := d.Info.GetDb(); db != nil {
		// 释放随连接持有的中转通道（nil 接收者安全），与底层连接关闭解耦于任何业务模块
		defer d.tunnel.Close()
		if err := db.Close(); err != nil {
			logx.Errorf("关闭数据库实例[%s]连接失败: %v", d.Id, err)
			return err
		}
		logx.Debugf("dbm - conn close success, connId: %s", d.Id)
		d.Info.db = nil
	}

	return nil
}

func (d *DbConn) Ping() error {
	db := d.Info.GetDb()
	stats := db.Stats()
	logx.Debugf("[%s] db stats -> open: %d, idle: %d,  inUse: %d, maxOpen: %d", d.Info.Name, stats.OpenConnections, stats.Idle, stats.InUse, stats.MaxOpenConnections)
	if stats.OpenConnections == 0 {
		logx.Infof("[%s]-[%s] db stats: no open connections", d.Info.Name, d.Info.Database)
	}

	return db.Ping()
}

// 执行查询语句返回的列信息
type QueryColumn struct {
	Name string `json:"name"` // 列名
	Key  string `json:"key"`  // 列唯一标识
	Type string `json:"type"` // 数据类型

	Masked bool `json:"masked,omitempty"` // 该列是否已被脱敏

	DbDataType *DbDataType `json:"-"`
	valuer     Valuer      `json:"-"`
}

func NewQueryColumn(colName string, columnType *DbDataType) *QueryColumn {
	return &QueryColumn{
		Name:       colName,
		Key:        colName,
		Type:       columnType.Codec.Name,
		DbDataType: columnType,
		valuer:     columnType.Codec.Valuer(),
	}
}

func (qc *QueryColumn) getValuePtr() any {
	return qc.valuer.NewValuePtr()
}

// value 获取列值
func (qc *QueryColumn) value() any {
	return qc.valuer.Value()
}

func (qc *QueryColumn) SQLValue(val any) any {
	return qc.DbDataType.Codec.SQLValue(val)
}

func (d *DbConn) GetDb() *sql.DB {
	return d.Info.GetDb()
}

// 执行查询语句
// 依次返回 列信息数组(顺序)，结果map，错误
func (d *DbConn) Query(querySQL string, args ...any) ([]*QueryColumn, []map[string]any, error) {
	return d.QueryContext(context.Background(), querySQL, args...)
}

// 执行查询语句
// 依次返回 列信息数组(顺序)，结果map，错误
func (d *DbConn) QueryContext(ctx context.Context, querySQL string, args ...any) ([]*QueryColumn, []map[string]any, error) {
	result := make([]map[string]any, 0, 16)
	cols, err := d.WalkQueryRows(ctx, querySQL, func(row map[string]any, columns []*QueryColumn) error {
		result = append(result, row)
		return nil
	}, args...)

	return cols, result, err
}

// 将查询结果映射至struct，可具体参考sqlx库
func (d *DbConn) Query2Struct(execSQL string, dest any) error {
	db := d.Info.GetDb()
	rows, err := db.Query(execSQL)
	if err != nil {
		return err
	}
	// rows对象一定要close掉，如果出错，不关掉则会很迅速的达到设置最大连接数，
	// 后面的链接过来直接报错或拒绝，实际上也没有起效果
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	return scan.All(rows, dest, false)
}

// WalkQueryRows 游标方式遍历查询结果集, walkFn返回error不为nil, 则跳出遍历并取消查询
func (d *DbConn) WalkQueryRows(ctx context.Context, querySQL string, walkFn WalkQueryRowsFunc, args ...any) ([]*QueryColumn, error) {
	if qcs, err := d.walkQueryRows(ctx, querySQL, walkFn, args...); err != nil {
		// 如果是手动停止 则默认返回当前已遍历查询的数据即可
		// walkFn返回的StopWalkQueryError可能被包装，需用errors.As而非类型断言
		var stopErr *StopWalkQueryError
		if errors.As(err, &stopErr) {
			return qcs, nil
		}
		return qcs, wrapSQLError(err)
	} else {
		return qcs, nil
	}
}

// WalkTableRows 游标方式遍历指定表的结果集, walkFn返回error不为nil, 则跳出遍历并取消查询
func (d *DbConn) WalkTableRows(ctx context.Context, tableName string, walkFn WalkQueryRowsFunc) ([]*QueryColumn, error) {
	return d.WalkQueryRows(ctx, fmt.Sprintf("SELECT * FROM %s", tableName), walkFn)
}

// 执行 update, insert, delete，建表等sql
// 返回影响条数和错误
func (d *DbConn) Exec(sql string, args ...any) (int64, error) {
	return d.ExecContext(context.Background(), sql, args...)
}

// 事务执行 update, insert, delete，建表等sql，若tx == nil，则不使用事务
// 返回影响条数和错误
func (d *DbConn) TxExec(tx *sql.Tx, execSQL string, args ...any) (int64, error) {
	return d.TxExecContext(context.Background(), tx, execSQL, args...)
}

// 执行 update, insert, delete，建表等sql
// 返回影响条数和错误
func (d *DbConn) ExecContext(ctx context.Context, execSQL string, args ...any) (int64, error) {
	return d.TxExecContext(ctx, nil, execSQL, args...)
}

// 事务执行 update, insert, delete，建表等sql，若tx == nil，则不适用事务
// 返回影响条数和错误
func (d *DbConn) TxExecContext(ctx context.Context, tx *sql.Tx, execSQL string, args ...any) (int64, error) {
	var res sql.Result
	var err error
	db := d.Info.GetDb()
	if tx != nil {
		res, err = tx.ExecContext(ctx, execSQL, args...)
	} else {
		res, err = db.ExecContext(ctx, execSQL, args...)
	}

	if err != nil {
		return 0, wrapSQLError(err)
	}
	// 执行成功的 DDL 改变了库结构：立即失效服务端元数据缓存（判据见 DbInfo.invalidateIfDDL）。
	// 事务内语句执行成功但事务后来回滚时会多失效一次，下次内省重新查库即可，不影响正确性
	d.Info.invalidateIfDDL(execSQL)
	return res.RowsAffected()
}

// Begin 开启事务
func (d *DbConn) Begin() (*sql.Tx, error) {
	db := d.Info.GetDb()
	return db.Begin()
}

// GetDialect 获取数据库dialect实现接口
func (d *DbConn) GetDialect() Dialect {
	return d.Info.Backend.GetDialect(d.Info)
}

// Metadata 创建新的 Schema 元数据访问入口，并接入该连接的跨请求元数据缓存。
// reader 实例仍按请求新建，但其读穿透共享 DbInfo 上的 schemaCache（跨请求）；方言 provider 不感知缓存。
func (d *DbConn) Metadata() *MetadataReader {
	return NewMetadataReader(d.Info.Backend, d.Info.Backend.GetMetadataProvider(d.Info), d.Info.Backend.GetServerInfo(d.Info), d.Info.SchemaCache())
}

// GetDbDataType 获取定义的数据库数据类型
func (d *DbConn) GetDbDataType(dataType string) *DbDataType {
	return GetDbDataType(d.Info.Type, dataType)
}

// buildQueryColumns 依据结果集列类型构建列信息数组与对应的扫描占位符。
// dbType 用于解析各列的数据库数据类型（决定值的编解码器）；空列名以 <anonymousN> 命名。
// 由 DbConn.walkQueryRows 与 DbInfo.QueryContext 共用，确保两条查询路径的列建模完全一致。
func buildQueryColumns(colTypes []*sql.ColumnType, dbType DbType) ([]*QueryColumn, []any) {
	cols := make([]*QueryColumn, len(colTypes))
	scans := make([]any, len(colTypes))
	for k, colType := range colTypes {
		colName := colType.Name()
		if colName == "" {
			colName = fmt.Sprintf("<anonymous%d>", k+1)
		}
		qc := NewQueryColumn(colName, GetDbDataType(dbType, colType.DatabaseTypeName()))
		cols[k] = qc
		scans[k] = qc.getValuePtr()
	}
	return cols, scans
}

// scanRowMap 将已 Scan 的一行值装配为 map。重复列名以列序号消歧并回写 QueryColumn.Key，
// 避免同名列（如 SELECT a.id, b.id）相互覆盖导致静默丢列。
func scanRowMap(cols []*QueryColumn) map[string]any {
	rowData := make(map[string]any, len(cols))
	for i := range cols {
		colname := cols[i].Name
		if _, e := rowData[colname]; e {
			colname = colname + strconv.Itoa(i)
			cols[i].Key = colname
		}
		rowData[colname] = cols[i].value()
	}
	return rowData
}

// 游标方式遍历查询rows, walkFn error不为nil, 则退出遍历
func (d *DbConn) walkQueryRows(ctx context.Context, selectSQL string, walkFn WalkQueryRowsFunc, args ...any) ([]*QueryColumn, error) {
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	db := d.Info.GetDb()
	rows, err := db.QueryContext(cancelCtx, selectSQL, args...)
	if err != nil {
		return nil, err
	}
	// rows对象一定要close掉，如果出错，不关掉则会很迅速的达到设置最大连接数，
	// 后面的链接过来直接报错或拒绝，实际上也没有起效果
	defer rows.Close()

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	// 列名用于前端表头名称按照数据库与查询字段顺序显示
	cols, scans := buildQueryColumns(colTypes, d.Info.Type)

	for rows.Next() {
		// 不Scan也会导致等待，该链接实际处于未工作的状态，然后也会导致连接数迅速达到最大
		if err := rows.Scan(scans...); err != nil {
			return cols, err
		}
		if err = walkFn(scanRowMap(cols), cols); err != nil {
			// StopWalkQueryError 是调用方主动结束遍历的控制流信号（取到目标行即停、超出行数上限等），
			// 并非结果集读取失败；统一按 ERROR 打印会让每次正常提前结束都刷一条假错误，
			// 淹没真正的遍历失败，故仅降级为 Debug
			var stopErr *StopWalkQueryError
			if errors.As(err, &stopErr) {
				logx.DebugfContext(ctx, "[%s] stop walking query result set early: %s", selectSQL, err.Error())
			} else {
				logx.ErrorfContext(ctx, "[%s] cursor traversal query result set error, exit traversal: %s", selectSQL, err.Error())
			}
			cancelFunc()
			return cols, err
		}
	}

	// rows.Next()返回false除正常遍历完毕外，还可能是游标中途IO/网络错误（断连、超时等），
	// 若不检查rows.Err()会静默按正常结束处理——dump导出场景将产出截断的备份文件且无任何报错，
	// 属不可逆的数据丢失，必须显式失败
	if err := rows.Err(); err != nil {
		return cols, wrapSQLError(err)
	}

	return cols, nil
}

// 包装sql执行相关错误
func wrapSQLError(err error) error {
	if errors.Is(err, context.Canceled) {
		return errorx.NewBiz("execution cancel")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errorx.NewBiz("execution timeout")
	}
	return err
}

// StopWalkQueryError 自定义的停止遍历查询错误类型
type StopWalkQueryError struct {
	Reason string
}

// Error 实现 error 接口
func (e *StopWalkQueryError) Error() string {
	return fmt.Sprintf("stop walk query: %s", e.Reason)
}

// NewStopWalkQueryError 创建一个带有reason的StopWalkQueryError
func NewStopWalkQueryError(reason string) *StopWalkQueryError {
	return &StopWalkQueryError{Reason: reason}
}
