package application

import (
	"database/sql"
	"errors"
	"fmt"
	"mayfly-go/internal/devops/domain/entity"
	"mayfly-go/internal/devops/domain/repository"
	"mayfly-go/internal/devops/infrastructure/persistence"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Db interface {
	// 分页获取
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
	// @param id 数据库实例id
	// @param db 数据库
	GetDbInstance(id uint64, db string) *DbInstance
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
	if dbEntity.Password != "" {
		TestConnection(*dbEntity)
	}

	// 查找是否存在该库
	oldDb := &entity.Db{Host: dbEntity.Host, Port: dbEntity.Port, EnvId: dbEntity.EnvId}
	err := d.GetDbBy(oldDb)

	if dbEntity.Id == 0 {
		biz.NotEmpty(dbEntity.Password, "密码不能为空")
		biz.IsTrue(err != nil, "该数据库实例已存在")
		d.dbRepo.Insert(dbEntity)
		return
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil {
		biz.IsTrue(oldDb.Id == dbEntity.Id, "该数据库实例已存在")
	}

	dbId := dbEntity.Id
	old := d.GetById(dbId)

	var oldDbs []interface{}
	for _, v := range strings.Split(old.Database, " ") {
		oldDbs = append(oldDbs, v)
	}

	var newDbs []interface{}
	for _, v := range strings.Split(dbEntity.Database, " ") {
		newDbs = append(newDbs, v)
	}
	// 比较新旧数据库列表，需要将移除的数据库相关联的信息删除
	_, delDb, _ := utils.ArrayCompare(newDbs, oldDbs, func(i1, i2 interface{}) bool {
		return i1.(string) == i2.(string)
	})
	for _, v := range delDb {
		// 先关闭数据库连接
		CloseDb(dbEntity.Id, v.(string))
		// 删除该库关联的所有sql记录
		d.dbSqlRepo.DeleteBy(&entity.DbSql{DbId: dbId, Db: v.(string)})
	}

	d.dbRepo.Update(dbEntity)

}

func (d *dbAppImpl) Delete(id uint64) {
	db := d.GetById(id)
	dbs := strings.Split(db.Database, " ")
	for _, v := range dbs {
		// 关闭连接
		CloseDb(id, v)
	}
	d.dbRepo.Delete(id)
	// 删除该库下用户保存的所有sql信息
	d.dbSqlRepo.DeleteBy(&entity.DbSql{DbId: id})
}

var mutex sync.Mutex

func (da *dbAppImpl) GetDbInstance(id uint64, db string) *DbInstance {
	mutex.Lock()
	defer mutex.Unlock()
	// Id不为0，则为需要缓存
	needCache := id != 0
	if needCache {
		load, ok := dbCache.Get(GetDbCacheKey(id, db))
		if ok {
			return load.(*DbInstance)
		}
	}

	d := da.GetById(id)
	biz.NotNil(d, "数据库信息不存在")
	biz.IsTrue(strings.Contains(d.Database, db), "未配置该库的操作权限")
	global.Log.Infof("连接db: %s:%d/%s", d.Host, d.Port, db)

	// 将数据库替换为要访问的数据库，原本数据库为空格拼接的所有库
	d.Database = db
	DB, err := sql.Open(d.Type, getDsn(d))
	biz.ErrIsNil(err, fmt.Sprintf("Open %s failed, err:%v\n", d.Type, err))
	perr := DB.Ping()
	if perr != nil {
		global.Log.Errorf("连接db失败: %s:%d/%s", d.Host, d.Port, db)
		panic(biz.NewBizErr(fmt.Sprintf("数据库连接失败: %s", perr.Error())))
	}

	// 最大连接周期，超过时间的连接就close
	// DB.SetConnMaxLifetime(100 * time.Second)
	// 设置最大连接数
	DB.SetMaxOpenConns(2)
	// 设置闲置连接数
	DB.SetMaxIdleConns(1)

	cacheKey := GetDbCacheKey(id, db)
	dbi := &DbInstance{Id: cacheKey, Type: d.Type, ProjectId: d.ProjectId, db: DB}
	if needCache {
		dbCache.Put(cacheKey, dbi)
	}
	return dbi
}

//------------------------------------------------------------------------------

// 客户端连接缓存，30分钟内没有访问则会被关闭, key为数据库实例id:数据库
var dbCache = cache.NewTimedCache(30*time.Minute, 5*time.Second).
	WithUpdateAccessTime(true).
	OnEvicted(func(key interface{}, value interface{}) {
		global.Log.Info(fmt.Sprintf("删除db连接缓存 id = %s", key))
		value.(*DbInstance).Close()
	})

func GetDbCacheKey(dbId uint64, db string) string {
	return fmt.Sprintf("%d:%s", dbId, db)
}

func GetDbInstanceByCache(id string) *DbInstance {
	if load, ok := dbCache.Get(id); ok {
		return load.(*DbInstance)
	}
	return nil
}

func TestConnection(d entity.Db) {
	// 验证第一个库是否可以连接即可
	d.Database = strings.Split(d.Database, " ")[0]
	DB, err := sql.Open(d.Type, getDsn(&d))
	biz.ErrIsNil(err, "Open %s failed, err:%v\n", d.Type, err)
	defer DB.Close()
	perr := DB.Ping()
	biz.ErrIsNilAppendErr(perr, "数据库连接失败: %s")
}

// db实例
type DbInstance struct {
	Id        string
	Type      string
	ProjectId uint64
	db        *sql.DB
}

// 执行查询语句
// 依次返回 列名数组，结果map，错误
func (d *DbInstance) SelectData(execSql string) ([]string, []map[string]interface{}, error) {
	execSql = strings.Trim(execSql, " ")
	isSelect := strings.HasPrefix(execSql, "SELECT") || strings.HasPrefix(execSql, "select")
	isShow := strings.HasPrefix(execSql, "show")

	if !isSelect && !isShow {
		return nil, nil, errors.New("该sql非查询语句")
	}
	// 没加limit，则默认限制50条
	if isSelect && !strings.Contains(execSql, "limit") && !strings.Contains(execSql, "LIMIT") {
		execSql = execSql + " LIMIT 50"
	}

	rows, err := d.db.Query(execSql)
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
	colTypes, _ := rows.ColumnTypes()
	// 这里表示一行填充数据
	scans := make([]interface{}, len(colTypes))
	// 这里表示一行所有列的值，用[]byte表示
	vals := make([][]byte, len(colTypes))
	// 这里scans引用vals，把数据填充到[]byte里
	for k := range vals {
		scans[k] = &vals[k]
	}

	result := make([]map[string]interface{}, 0)
	// 列名用于前端表头名称按照数据库与查询字段顺序显示
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
		rowData := make(map[string]interface{})
		// 把vals中的数据复制到row中
		for i, v := range vals {
			colType := colTypes[i]
			colName := colType.Name()
			// 字段类型名
			colScanType := colType.ScanType().Name()

			// 如果是密码字段，则脱敏显示
			if colName == "password" {
				v = []byte("******")
			}
			if isFirst {
				colNames = append(colNames, colName)
			}
			// 这里把[]byte数据转成string
			stringV := string(v)
			if stringV == "" {
				rowData[colName] = stringV
				continue
			}
			if strings.Contains(colScanType, "int") || strings.Contains(colScanType, "Int") {
				intV, _ := strconv.Atoi(stringV)
				switch colType.ScanType().Kind() {
				case reflect.Int8:
					rowData[colName] = int8(intV)
				case reflect.Uint8:
					rowData[colName] = uint8(intV)
				case reflect.Int64:
					rowData[colName] = int64(intV)
				case reflect.Uint64:
					rowData[colName] = uint64(intV)
				case reflect.Uint:
					rowData[colName] = uint(intV)
				default:
					rowData[colName] = intV
				}
				continue
			}
			if strings.Contains(colScanType, "float") || strings.Contains(colScanType, "Float") {
				floatV, _ := strconv.ParseFloat(stringV, 64)
				rowData[colName] = floatV
			} else {
				rowData[colName] = stringV
			}
		}
		// 放入结果集
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
		return fmt.Sprintf("%s:%s@%s(%s:%d)/%s?timeout=8s", d.Username, d.Password, d.Network, d.Host, d.Port, d.Database)
	}
	return ""
}

// 删除db缓存并关闭该数据库所有连接
func CloseDb(dbId uint64, db string) {
	dbCache.Delete(GetDbCacheKey(dbId, db))
}

//-----------------------------------元数据-------------------------------------------

const (
	// mysql 表信息元数据
	MYSQL_TABLE_MA = `SELECT table_name tableName, engine, table_comment tableComment, 
	create_time createTime from information_schema.tables
	WHERE table_schema = (SELECT database()) LIMIT 2000`

	// mysql 表信息
	MYSQL_TABLE_INFO = `SELECT table_name tableName, table_comment tableComment, table_rows tableRows,
	data_length dataLength, index_length indexLength, create_time createTime 
	FROM information_schema.tables 
    WHERE table_schema = (SELECT database()) LIMIT 2000`

	// mysql 索引信息
	MYSQL_INDEX_INFO = `SELECT index_name indexName, column_name columnName, index_type indexType,
	SEQ_IN_INDEX seqInIndex, INDEX_COMMENT indexComment
	FROM information_schema.STATISTICS 
    WHERE table_schema = (SELECT database()) AND table_name = '%s' LIMIT 500`

	// 默认每次查询列元信息数量
	DEFAULT_COLUMN_SIZE = 2000

	// mysql 列信息元数据
	MYSQL_COLOUMN_MA = `SELECT table_name tableName, column_name columnName, column_type columnType,
	column_comment columnComment, column_key columnKey, extra, is_nullable nullable from information_schema.columns
	WHERE table_name in (%s) AND table_schema = (SELECT database()) ORDER BY tableName, ordinal_position LIMIT %d, %d`

	// mysql 列信息元数据总数
	MYSQL_COLOUMN_MA_COUNT = `SELECT COUNT(*) maNum from information_schema.columns
	WHERE table_name in (%s) AND table_schema = (SELECT database())`
)

func (d *DbInstance) GetTableMetedatas() []map[string]interface{} {
	var sql string
	if d.Type == "mysql" {
		sql = MYSQL_TABLE_MA
	}
	_, res, _ := d.SelectData(sql)
	return res
}

func (d *DbInstance) GetColumnMetadatas(tableNames ...string) []map[string]interface{} {
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
	maCount := int(countRes[0]["maNum"].(int64))
	// 计算需要查询的页数
	pageNum := maCount / DEFAULT_COLUMN_SIZE
	if maCount%DEFAULT_COLUMN_SIZE > 0 {
		pageNum++
	}

	res := make([]map[string]interface{}, 0)
	for index := 0; index < pageNum; index++ {
		sql = fmt.Sprintf(sqlTmp, tableName, index*DEFAULT_COLUMN_SIZE, DEFAULT_COLUMN_SIZE)
		_, result, err := d.SelectData(sql)
		biz.ErrIsNilAppendErr(err, "获取数据库列信息失败: %s")
		res = append(res, result...)
	}
	return res
}

// 获取表的主键，目前默认第一列为主键
func (d *DbInstance) GetPrimaryKey(tablename string) string {
	return d.GetColumnMetadatas(tablename)[0]["columnName"].(string)
}

func (d *DbInstance) GetTableInfos() []map[string]interface{} {
	var sql string
	if d.Type == "mysql" {
		sql = MYSQL_TABLE_INFO
	}
	_, res, _ := d.SelectData(sql)
	return res
}

func (d *DbInstance) GetTableIndex(tableName string) []map[string]interface{} {
	var sql string
	if d.Type == "mysql" {
		sql = fmt.Sprintf(MYSQL_INDEX_INFO, tableName)
	}
	_, res, _ := d.SelectData(sql)
	return res
}

func (d *DbInstance) GetCreateTableDdl(tableName string) []map[string]interface{} {
	var sql string
	if d.Type == "mysql" {
		sql = fmt.Sprintf("show create table %s ", tableName)
	}
	_, res, _ := d.SelectData(sql)
	return res
}
