package db

import (
	"database/sql"
	"errors"
	"fmt"
	"mayfly-go/base/biz"
	"mayfly-go/devops/models"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var dbCache sync.Map

// db实例
type DbInstance struct {
	Id   uint64
	Type string
	db   *sql.DB
}

// 执行查询语句
func (d *DbInstance) SelectData(sql string) ([]map[string]string, error) {
	sql = strings.Trim(sql, " ")
	if !strings.HasPrefix(sql, "SELECT") && !strings.HasPrefix(sql, "select") {
		return nil, errors.New("该sql非查询语句")
	}
	rows, err := d.db.Query(sql)
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
	cols, _ := rows.Columns()
	// 这里表示一行填充数据
	scans := make([]interface{}, len(cols))
	// 这里表示一行所有列的值，用[]byte表示
	vals := make([][]byte, len(cols))
	// 这里scans引用vals，把数据填充到[]byte里
	for k := range vals {
		scans[k] = &vals[k]
	}
	result := make([]map[string]string, 0)
	for rows.Next() {
		// 不Scan也会导致等待，该链接实际处于未工作的状态，然后也会导致连接数迅速达到最大
		err := rows.Scan(scans...)
		if err != nil {
			return nil, err
		}
		// 每行数据
		rowData := make(map[string]string)
		// 把vals中的数据复制到row中
		for k, v := range vals {
			key := cols[k]
			// 这里把[]byte数据转成string
			rowData[key] = string(v)
		}
		//放入结果集
		result = append(result, rowData)
	}
	return result, nil
}

// 执行 update, insert, delete，建表等sql
//
// 返回影响条数和错误
func (d *DbInstance) Exec(sql string) (int64, error) {
	res, err := d.db.Exec(sql)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// 关闭连接，并从缓存中移除
func (d *DbInstance) Close() {
	d.db.Close()
	dbCache.Delete(d.Id)
}

// 获取dataSourceName
func getDsn(d *models.Db) string {
	if d.Type == "mysql" {
		return fmt.Sprintf("%s:%s@%s(%s:%d)/%s", d.Username, d.Password, d.Network, d.Host, d.Port, d.Database)
	}
	return ""
}

func GetDbInstance(id uint64) *DbInstance {
	// Id不为0，则为需要缓存
	needCache := id != 0
	if needCache {
		load, ok := dbCache.Load(id)
		if ok {
			return load.(*DbInstance)
		}
	}
	d := models.GetDbById(uint64(id))
	biz.NotNil(d, "数据库信息不存在")
	DB, err := sql.Open(d.Type, getDsn(d))
	biz.ErrIsNil(err, fmt.Sprintf("Open %s failed, err:%v\n", d.Type, err))
	perr := DB.Ping()
	if perr != nil {
		panic(biz.NewBizErr(fmt.Sprintf("数据库连接失败: %s", perr.Error())))
	}

	// 最大连接周期，超过时间的连接就close
	DB.SetConnMaxLifetime(100 * time.Second)
	// 设置最大连接数
	DB.SetMaxOpenConns(5)
	// 设置闲置连接数
	DB.SetMaxIdleConns(1)

	dbi := &DbInstance{Id: id, Type: d.Type, db: DB}
	if needCache {
		dbCache.LoadOrStore(d.Id, dbi)
	}
	return dbi
}
