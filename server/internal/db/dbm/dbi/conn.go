package dbi

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"reflect"
	"strconv"
	"strings"
)

// 游标遍历查询结果集处理函数
type WalkQueryRowsFunc func(row map[string]any, columns []*QueryColumn) error

// db实例连接信息
type DbConn struct {
	Id   string
	Info *DbInfo

	db *sql.DB
}

// 执行数据库查询返回的列信息
type QueryColumn struct {
	Name string `json:"name"` // 列名
	Type string `json:"type"` // 类型
}

// 执行查询语句
// 依次返回 列信息数组(顺序)，结果map，错误
func (d *DbConn) Query(querySql string, args ...any) ([]*QueryColumn, []map[string]any, error) {
	return d.QueryContext(context.Background(), querySql, args...)
}

// 执行查询语句
// 依次返回 列信息数组(顺序)，结果map，错误
func (d *DbConn) QueryContext(ctx context.Context, querySql string, args ...any) ([]*QueryColumn, []map[string]any, error) {
	result := make([]map[string]any, 0, 16)
	var queryColumns []*QueryColumn

	err := d.WalkQueryRows(ctx, querySql, func(row map[string]any, columns []*QueryColumn) error {
		if len(queryColumns) == 0 {
			queryColumns = columns
		}
		result = append(result, row)
		return nil
	}, args...)

	if err != nil {
		return nil, nil, wrapSqlError(err)
	}

	return queryColumns, result, nil
}

// 将查询结果映射至struct，可具体参考sqlx库
func (d *DbConn) Query2Struct(execSql string, dest any) error {
	rows, err := d.db.Query(execSql)
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
	return scanAll(rows, dest, false)
}

// 游标方式遍历查询结果集, walkFn返回error不为nil, 则跳出遍历
func (d *DbConn) WalkQueryRows(ctx context.Context, querySql string, walkFn WalkQueryRowsFunc, args ...any) error {
	return walkQueryRows(ctx, d.db, querySql, walkFn, args...)
}

// 游标方式遍历指定表的结果集, walkFn返回error不为nil, 则跳出遍历
func (d *DbConn) WalkTableRows(ctx context.Context, tableName string, walkFn WalkQueryRowsFunc) error {
	return d.WalkQueryRows(ctx, fmt.Sprintf("SELECT * FROM %s", tableName), walkFn)
}

// 执行 update, insert, delete，建表等sql
// 返回影响条数和错误
func (d *DbConn) Exec(sql string, args ...any) (int64, error) {
	return d.ExecContext(context.Background(), sql, args...)
}

// 事务执行 update, insert, delete，建表等sql，若tx == nil，则不使用事务
// 返回影响条数和错误
func (d *DbConn) TxExec(tx *sql.Tx, execSql string, args ...any) (int64, error) {
	return d.TxExecContext(context.Background(), tx, execSql, args...)
}

// 执行 update, insert, delete，建表等sql
// 返回影响条数和错误
func (d *DbConn) ExecContext(ctx context.Context, execSql string, args ...any) (int64, error) {
	return d.TxExecContext(ctx, nil, execSql, args...)
}

// 事务执行 update, insert, delete，建表等sql，若tx == nil，则不适用事务
// 返回影响条数和错误
func (d *DbConn) TxExecContext(ctx context.Context, tx *sql.Tx, execSql string, args ...any) (int64, error) {
	var res sql.Result
	var err error
	if tx != nil {
		res, err = tx.ExecContext(ctx, execSql, args...)
	} else {
		res, err = d.db.ExecContext(ctx, execSql, args...)
	}

	if err != nil {
		return 0, wrapSqlError(err)
	}
	return res.RowsAffected()
}

// 开启事务
func (d *DbConn) Begin() (*sql.Tx, error) {
	return d.db.Begin()
}

// 获取数据库dialect实现接口
func (d *DbConn) GetDialect() Dialect {
	return d.Info.Meta.GetDialect(d)
}

// 返回数据库连接状态
func (d *DbConn) Stats(ctx context.Context, execSql string, args ...any) sql.DBStats {
	return d.db.Stats()
}

// 关闭连接
func (d *DbConn) Close() {
	if d.db != nil {
		if err := d.db.Close(); err != nil {
			logx.Errorf("关闭数据库实例[%s]连接失败: %s", d.Id, err.Error())
		}
		// 如果是达梦并且使用了ssh隧道，则需要手动将其关闭
		if d.Info.Type == DbTypeDM && d.Info.SshTunnelMachineId > 0 {
			mcm.CloseSshTunnelMachine(d.Info.SshTunnelMachineId, fmt.Sprintf("db:%d", d.Info.Id))
		}
		d.db = nil
	}
}

// 游标方式遍历查询rows, walkFn error不为nil, 则跳出遍历
func walkQueryRows(ctx context.Context, db *sql.DB, selectSql string, walkFn WalkQueryRowsFunc, args ...any) error {
	rows, err := db.QueryContext(ctx, selectSql, args...)

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

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return err
	}
	lenCols := len(colTypes)
	// 列名用于前端表头名称按照数据库与查询字段顺序显示
	cols := make([]*QueryColumn, lenCols)
	// 这里表示一行填充数据
	scans := make([]any, lenCols)
	// 这里表示一行所有列的值，用[]byte表示
	values := make([][]byte, lenCols)
	for k, colType := range colTypes {
		// 处理字段名，如果为空，则命名为匿名列
		colName := colType.Name()
		if colName == "" {
			colName = fmt.Sprintf("<anonymous%d>", k+1)
		}
		cols[k] = &QueryColumn{Name: colName, Type: colType.DatabaseTypeName()}
		// 这里scans引用values，把数据填充到[]byte里
		scans[k] = &values[k]
	}

	for rows.Next() {
		// 不Scan也会导致等待，该链接实际处于未工作的状态，然后也会导致连接数迅速达到最大
		if err := rows.Scan(scans...); err != nil {
			return err
		}
		// 每行数据
		rowData := make(map[string]any, lenCols)
		// 把values中的数据复制到row中
		for i, v := range values {
			rowData[cols[i].Name] = valueConvert(v, colTypes[i])
		}
		if err = walkFn(rowData, cols); err != nil {
			logx.Error("游标遍历查询结果集出错,退出遍历: %s", err.Error())
			return err
		}
	}

	return nil
}

// 将查询的值转为对应列类型的实际值，不全部转为字符串
func valueConvert(data []byte, colType *sql.ColumnType) any {
	if data == nil {
		return nil
	}
	// 列的数据库类型名
	colDatabaseTypeName := strings.ToLower(colType.DatabaseTypeName())

	// 如果类型是bit，则直接返回第一个字节即可
	if strings.Contains(colDatabaseTypeName, "bit") {
		return data[0]
	}

	// 这里把[]byte数据转成string
	stringV := string(data)
	if stringV == "" {
		return ""
	}
	if colType == nil || colType.ScanType() == nil {
		return stringV
	}
	colScanType := strings.ToLower(colType.ScanType().Name())

	if strings.Contains(colScanType, "int") {
		// 如果长度超过16位，则返回字符串，因为前端js长度大于16会丢失精度
		if len(stringV) > 16 {
			return stringV
		}
		intV, _ := strconv.Atoi(stringV)
		switch colType.ScanType().Kind() {
		case reflect.Int8:
			return int8(intV)
		case reflect.Uint8:
			return uint8(intV)
		case reflect.Int64:
			return int64(intV)
		case reflect.Uint64:
			return uint64(intV)
		case reflect.Uint:
			return uint(intV)
		default:
			return intV
		}
	}
	if strings.Contains(colScanType, "float") || strings.Contains(colDatabaseTypeName, "decimal") {
		floatV, _ := strconv.ParseFloat(stringV, 64)
		return floatV
	}

	return stringV
}

// 包装sql执行相关错误
func wrapSqlError(err error) error {
	if err == context.Canceled {
		return errorx.NewBiz("取消执行")
	}
	if err == context.DeadlineExceeded {
		return errorx.NewBiz("执行超时")
	}
	return err
}
