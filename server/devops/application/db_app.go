package application

import (
	"database/sql"
	"errors"
	"fmt"
	"mayfly-go/base/biz"
	"mayfly-go/base/cache"
	"mayfly-go/base/global"
	"mayfly-go/base/model"
	"mayfly-go/server/devops/domain/entity"
	"mayfly-go/server/devops/domain/repository"
	"mayfly-go/server/devops/infrastructure/persistence"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Db interface {
	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.Db, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	Count(condition *entity.Db) int64

	// 根据条件获取
	GetDbBy(condition *entity.Db, cols ...string) error

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.Db

	Save(entity *entity.Db)

	// 删除数据库信息
	Delete(id uint64)

	// 获取数据库连接实例
	GetDbInstance(id uint64) *DbInstance
}

type dbAppImpl struct {
	dbRepo    repository.Db
	dbSqlRepo repository.DbSql
}

var DbApp Db = &dbAppImpl{
	dbRepo:    persistence.DbDao,
	dbSqlRepo: persistence.DbSqlDao,
}

// 分页获取数据库信息列表
func (d *dbAppImpl) GetPageList(condition *entity.Db, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return d.dbRepo.GetDbList(condition, pageParam, toEntity, orderBy...)
}

func (d *dbAppImpl) Count(condition *entity.Db) int64 {
	return d.dbRepo.Count(condition)
}

// 根据条件获取
func (d *dbAppImpl) GetDbBy(condition *entity.Db, cols ...string) error {
	return d.dbRepo.GetDb(condition, cols...)
}

// 根据id获取
func (d *dbAppImpl) GetById(id uint64, cols ...string) *entity.Db {
	return d.dbRepo.GetById(id, cols...)
}

func (d *dbAppImpl) Save(dbEntity *entity.Db) {
	// 默认tcp连接
	dbEntity.Network = "tcp"
	// 测试连接
	TestConnection(dbEntity)

	// 查找是否存在该库
	oldDb := &entity.Db{Host: dbEntity.Host, Port: dbEntity.Port, Database: dbEntity.Database}
	err := d.GetDbBy(oldDb)

	if dbEntity.Id == 0 {
		biz.IsTrue(err != nil, "该库已存在")
		d.dbRepo.Insert(dbEntity)
	} else {
		// 如果存在该库，则校验修改的库是否为该库
		if err == nil {
			biz.IsTrue(oldDb.Id == dbEntity.Id, "该库已存在")
		}
		// 先关闭数据库连接
		CloseDb(dbEntity.Id)
		d.dbRepo.Update(dbEntity)
	}
}

func (d *dbAppImpl) Delete(id uint64) {
	// 关闭连接
	CloseDb(id)
	d.dbRepo.Delete(id)
	// 删除该库下用户保存的所有sql信息
	d.dbSqlRepo.DeleteBy(&entity.DbSql{DbId: id})
}

var mutex sync.Mutex

func (da *dbAppImpl) GetDbInstance(id uint64) *DbInstance {
	mutex.Lock()
	defer mutex.Unlock()
	// Id不为0，则为需要缓存
	needCache := id != 0
	if needCache {
		load, ok := dbCache.Get(id)
		if ok {
			return load.(*DbInstance)
		}
	}

	d := da.GetById(id)
	biz.NotNil(d, "数据库信息不存在")
	global.Log.Infof("连接db: %s:%d/%s", d.Host, d.Port, d.Database)

	DB, err := sql.Open(d.Type, getDsn(d))
	biz.ErrIsNil(err, fmt.Sprintf("Open %s failed, err:%v\n", d.Type, err))
	perr := DB.Ping()
	if perr != nil {
		panic(biz.NewBizErr(fmt.Sprintf("数据库连接失败: %s", perr.Error())))
	}

	// 最大连接周期，超过时间的连接就close
	DB.SetConnMaxLifetime(100 * time.Second)
	// 设置最大连接数
	DB.SetMaxOpenConns(2)
	// 设置闲置连接数
	DB.SetMaxIdleConns(1)

	dbi := &DbInstance{Id: id, Type: d.Type, ProjectId: d.ProjectId, db: DB}
	if needCache {
		dbCache.Put(id, dbi)
	}
	return dbi
}

//------------------------------------------------------------------------------

// 客户端连接缓存，30分钟内没有访问则会被关闭
var dbCache = cache.NewTimedCache(30*time.Minute, 5*time.Second).
	WithUpdateAccessTime(true).
	OnEvicted(func(key interface{}, value interface{}) {
		global.Log.Info(fmt.Sprintf("删除db连接缓存 id: %d", key))
		value.(*DbInstance).Close()
	})

func GetDbInstanceByCache(id uint64) *DbInstance {
	if load, ok := dbCache.Get(fmt.Sprint(id)); ok {
		return load.(*DbInstance)
	}
	return nil
}

func TestConnection(d *entity.Db) {
	biz.NotNil(d, "数据库信息不存在")
	DB, err := sql.Open(d.Type, getDsn(d))
	biz.ErrIsNil(err, "Open %s failed, err:%v\n", d.Type, err)
	defer DB.Close()
	perr := DB.Ping()
	biz.ErrIsNilAppendErr(perr, "数据库连接失败: %s")
}

// db实例
type DbInstance struct {
	Id        uint64
	Type      string
	ProjectId uint64
	db        *sql.DB
}

// 执行查询语句
// 依次返回 列名数组，结果map，错误
func (d *DbInstance) SelectData(sql string) ([]string, []map[string]string, error) {
	sql = strings.Trim(sql, " ")
	isSelect := strings.HasPrefix(sql, "SELECT") || strings.HasPrefix(sql, "select")
	isShow := strings.HasPrefix(sql, "show")

	if !isSelect && !isShow {
		return nil, nil, errors.New("该sql非查询语句")
	}
	// 没加limit，则默认限制50条
	if isSelect && !strings.Contains(sql, "limit") && !strings.Contains(sql, "LIMIT") {
		sql = sql + " LIMIT 50"
	}

	rows, err := d.db.Query(sql)
	if err != nil {
		return nil, nil, err
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
	// 列名
	colNames := make([]string, 0)
	// 是否第一次遍历，列名数组只需第一次遍历时加入
	isFirst := true
	for rows.Next() {
		// 不Scan也会导致等待，该链接实际处于未工作的状态，然后也会导致连接数迅速达到最大
		err := rows.Scan(scans...)
		if err != nil {
			return nil, nil, err
		}
		// 每行数据
		rowData := make(map[string]string)
		// 把vals中的数据复制到row中
		for k, v := range vals {
			key := cols[k]
			// 如果是密码字段，则脱敏显示
			if key == "password" {
				v = []byte("******")
			}
			if isFirst {
				colNames = append(colNames, key)
			}
			// 这里把[]byte数据转成string
			rowData[key] = string(v)
		}
		//放入结果集
		result = append(result, rowData)
		isFirst = false
	}
	return colNames, result, nil
}

// 执行 update, insert, delete，建表等sql
// 返回影响条数和错误
func (d *DbInstance) Exec(sql string) (int64, error) {
	res, err := d.db.Exec(sql)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// 关闭连接
func (d *DbInstance) Close() {
	d.db.Close()
}

// 获取dataSourceName
func getDsn(d *entity.Db) string {
	if d.Type == "mysql" {
		return fmt.Sprintf("%s:%s@%s(%s:%d)/%s", d.Username, d.Password, d.Network, d.Host, d.Port, d.Database)
	}
	return ""
}

func CloseDb(id uint64) {
	if di := GetDbInstanceByCache(id); di != nil {
		di.Close()
		dbCache.Delete(id)
	}
}

//-----------------------------------元数据-------------------------------------------

const (
	// mysql 表信息元数据
	MYSQL_TABLE_MA = `SELECT table_name tableName, engine, table_comment tableComment, 
	create_time createTime from information_schema.tables
	WHERE table_schema = (SELECT database())`

	// mysql 表信息
	MYSQL_TABLE_INFO = `SELECT table_name tableName, table_comment tableComment, table_rows tableRows,
	data_length dataLength, index_length indexLength, create_time createTime 
	FROM information_schema.tables 
    WHERE table_schema = (SELECT database())`

	// mysql 索引信息
	MYSQL_INDEX_INFO = `SELECT index_name indexName, column_name columnName, index_type indexType,
	SEQ_IN_INDEX seqInIndex, INDEX_COMMENT indexComment
	FROM information_schema.STATISTICS 
    WHERE table_schema = (SELECT database()) AND table_name = '%s'`

	// 默认每次查询列元信息数量
	DEFAULT_COLUMN_SIZE = 2000

	// mysql 列信息元数据
	MYSQL_COLOUMN_MA = `SELECT table_name tableName, column_name columnName, column_type columnType,
	column_comment columnComment, column_key columnKey, extra, is_nullable nullable from information_schema.columns
	WHERE table_name in (%s) AND table_schema = (SELECT database()) ORDER BY tableName, ordinal_position limit %d, %d`

	// mysql 列信息元数据总数
	MYSQL_COLOUMN_MA_COUNT = `SELECT COUNT(*) maNum from information_schema.columns
	WHERE table_name in (%s) AND table_schema = (SELECT database())`
)

func (d *DbInstance) GetTableMetedatas() []map[string]string {
	var sql string
	if d.Type == "mysql" {
		sql = MYSQL_TABLE_MA
	}
	_, res, _ := d.SelectData(sql)
	return res
}

func (d *DbInstance) GetColumnMetadatas(tableNames ...string) []map[string]string {
	var sql, tableName string
	for i := 0; i < len(tableNames); i++ {
		if i != 0 {
			tableName = tableName + ", "
		}
		tableName = tableName + "'" + tableNames[i] + "'"
	}

	var countSqlTmp string
	var sqlTmp string
	if d.Type == "mysql" {
		countSqlTmp = MYSQL_COLOUMN_MA_COUNT
		sqlTmp = MYSQL_COLOUMN_MA
	}

	countSql := fmt.Sprintf(countSqlTmp, tableName)
	_, countRes, _ := d.SelectData(countSql)
	// 查询出所有列信息总数，手动分页获取所有数据
	maCount, _ := strconv.Atoi(countRes[0]["maNum"])
	// 计算需要查询的页数
	pageNum := maCount / DEFAULT_COLUMN_SIZE
	if maCount%DEFAULT_COLUMN_SIZE > 0 {
		pageNum++
	}

	res := make([]map[string]string, 0)
	for index := 0; index < pageNum; index++ {
		sql = fmt.Sprintf(sqlTmp, tableName, index*DEFAULT_COLUMN_SIZE, DEFAULT_COLUMN_SIZE)
		_, result, err := d.SelectData(sql)
		biz.ErrIsNilAppendErr(err, "获取数据库列信息失败: %s")
		res = append(res, result...)
	}
	return res
}

func (d *DbInstance) GetTableInfos() []map[string]string {
	var sql string
	if d.Type == "mysql" {
		sql = MYSQL_TABLE_INFO
	}
	_, res, _ := d.SelectData(sql)
	return res
}

func (d *DbInstance) GetTableIndex(tableName string) []map[string]string {
	var sql string
	if d.Type == "mysql" {
		sql = fmt.Sprintf(MYSQL_INDEX_INFO, tableName)
	}
	_, res, _ := d.SelectData(sql)
	return res
}

func (d *DbInstance) GetCreateTableDdl(tableName string) []map[string]string {
	var sql string
	if d.Type == "mysql" {
		sql = fmt.Sprintf("show create table %s ", tableName)
	}
	_, res, _ := d.SelectData(sql)
	return res
}
