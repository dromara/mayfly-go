package dbm

import (
	"database/sql"
	"fmt"
	"mayfly-go/pkg/logx"
	"reflect"
	"strconv"
	"strings"
)

// db实例连接信息
type DbConn struct {
	Id   string
	Info *DbInfo

	db *sql.DB
}

// 执行查询语句
// 依次返回 列名数组，结果map，错误
func (d *DbConn) SelectData(execSql string) ([]string, []map[string]any, error) {
	return selectDataByDb(d.db, execSql)
}

// 将查询结果映射至struct，可具体参考sqlx库
func (d *DbConn) SelectData2Struct(execSql string, dest any) error {
	return select2StructByDb(d.db, execSql, dest)
}

// WalkTableRecord 遍历表记录
func (d *DbConn) WalkTableRecord(selectSql string, walk func(record map[string]any, columns []string)) error {
	_, err := walkTableRecord(d.db, selectSql, walk)
	return err
}

// 执行 update, insert, delete，建表等sql
// 返回影响条数和错误
func (d *DbConn) Exec(sql string) (int64, error) {
	res, err := d.db.Exec(sql)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// 获取数据库元信息实现接口
func (d *DbConn) GetMeta() DbMetadata {
	switch d.Info.Type {
	case DbTypeMysql:
		return &MysqlMetadata{dc: d}
	case DbTypePostgres:
		return &PgsqlMetadata{dc: d}
	default:
		panic(fmt.Sprintf("invalid database type: %s", d.Info.Type))
	}
}

// 关闭连接
func (d *DbConn) Close() {
	if d.db != nil {
		if err := d.db.Close(); err != nil {
			logx.Errorf("关闭数据库实例[%s]连接失败: %s", d.Id, err.Error())
		}
		d.db = nil
	}
}

func selectDataByDb(db *sql.DB, selectSql string) ([]string, []map[string]any, error) {
	result := make([]map[string]any, 0, 16)
	columns, err := walkTableRecord(db, selectSql, func(record map[string]any, columns []string) {
		result = append(result, record)
	})
	if err != nil {
		return nil, nil, err
	}
	return columns, result, nil
}

func walkTableRecord(db *sql.DB, selectSql string, walk func(record map[string]any, columns []string)) ([]string, error) {
	rows, err := db.Query(selectSql)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	lenCols := len(colTypes)
	// 列名用于前端表头名称按照数据库与查询字段顺序显示
	colNames := make([]string, lenCols)
	// 这里表示一行填充数据
	scans := make([]any, lenCols)
	// 这里表示一行所有列的值，用[]byte表示
	values := make([][]byte, lenCols)
	for k, colType := range colTypes {
		colNames[k] = colType.Name()
		// 这里scans引用values，把数据填充到[]byte里
		scans[k] = &values[k]
	}

	for rows.Next() {
		// 不Scan也会导致等待，该链接实际处于未工作的状态，然后也会导致连接数迅速达到最大
		if err := rows.Scan(scans...); err != nil {
			return nil, err
		}
		// 每行数据
		rowData := make(map[string]any, lenCols)
		// 把values中的数据复制到row中
		for i, v := range values {
			rowData[colTypes[i].Name()] = valueConvert(v, colTypes[i])
		}
		walk(rowData, colNames)
	}

	return colNames, nil
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

// 查询数据结果映射至struct。可参考sqlx库
func select2StructByDb(db *sql.DB, selectSql string, dest any) error {
	rows, err := db.Query(selectSql)
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
