package application

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"mayfly-go/internal/constant"
	"mayfly-go/internal/devops/domain/entity"
	"mayfly-go/internal/devops/domain/repository"
	"mayfly-go/internal/devops/infrastructure/machine"
	"mayfly-go/internal/devops/infrastructure/persistence"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
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

	// 获取数据库实例的所有数据库列表
	GetDatabases(entity *entity.Db) []string
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
	dbEntity.Network = dbEntity.GetNetwork()

	// 测试连接
	if dbEntity.Password != "" {
		TestConnection(dbEntity)
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
		// 关闭数据库连接
		CloseDb(dbEntity.Id, v)
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

func (d *dbAppImpl) GetDatabases(ed *entity.Db) []string {
	databases := make([]string, 0)
	var dbConn *sql.DB
	var metaDb string
	var getDatabasesSql string
	if ed.Type == entity.DbTypeMysql {
		metaDb = "information_schema"
		getDatabasesSql = "SELECT SCHEMA_NAME AS dbname FROM SCHEMATA"
	} else {
		metaDb = "postgres"
		getDatabasesSql = "SELECT datname AS dbname FROM pg_database"
	}

	dbConn, err := GetDbConn(ed, metaDb)
	biz.ErrIsNilAppendErr(err, "数据库连接失败: %s")
	defer dbConn.Close()

	_, res, err := SelectDataByDb(dbConn, getDatabasesSql)
	biz.ErrIsNilAppendErr(err, "获取数据库列表失败")
	for _, re := range res {
		databases = append(databases, re["dbname"].(string))
	}
	return databases
}

var mutex sync.Mutex

func (da *dbAppImpl) GetDbInstance(id uint64, db string) *DbInstance {
	// Id不为0，则为需要缓存
	needCache := id != 0
	if needCache {
		load, ok := dbCache.Get(GetDbCacheKey(id, db))
		if ok {
			return load.(*DbInstance)
		}
	}
	mutex.Lock()
	defer mutex.Unlock()

	d := da.GetById(id)
	biz.NotNil(d, "数据库信息不存在")
	biz.IsTrue(strings.Contains(d.Database, db), "未配置该库的操作权限")

	cacheKey := GetDbCacheKey(id, db)
	dbi := &DbInstance{Id: cacheKey, Type: d.Type, ProjectId: d.ProjectId, sshTunnelMachineId: d.SshTunnelMachineId}

	DB, err := GetDbConn(d, db)
	if err != nil {
		dbi.Close()
		global.Log.Errorf("连接db失败: %s:%d/%s", d.Host, d.Port, db)
		panic(biz.NewBizErr(fmt.Sprintf("数据库连接失败: %s", err.Error())))
	}

	// 最大连接周期，超过时间的连接就close
	// DB.SetConnMaxLifetime(100 * time.Second)
	// 设置最大连接数
	DB.SetMaxOpenConns(2)
	// 设置闲置连接数
	DB.SetMaxIdleConns(1)

	dbi.db = DB
	global.Log.Infof("连接db: %s:%d/%s", d.Host, d.Port, db)
	if needCache {
		dbCache.Put(cacheKey, dbi)
	}
	return dbi
}

//------------------------------------------------------------------------------

// 客户端连接缓存，指定时间内没有访问则会被关闭, key为数据库实例id:数据库
var dbCache = cache.NewTimedCache(constant.DbConnExpireTime, 5*time.Second).
	WithUpdateAccessTime(true).
	OnEvicted(func(key interface{}, value interface{}) {
		global.Log.Info(fmt.Sprintf("删除db连接缓存 id = %s", key))
		value.(*DbInstance).Close()
	})

func init() {
	machine.AddCheckSshTunnelMachineUseFunc(func(machineId uint64) bool {
		// 遍历所有db连接实例，若存在redis实例使用该ssh隧道机器，则返回true，表示还在使用中...
		items := dbCache.Items()
		for _, v := range items {
			if v.Value.(*DbInstance).sshTunnelMachineId == machineId {
				return true
			}
		}
		return false
	})
}

func GetDbCacheKey(dbId uint64, db string) string {
	return fmt.Sprintf("%d:%s", dbId, db)
}

func GetDbInstanceByCache(id string) *DbInstance {
	if load, ok := dbCache.Get(id); ok {
		return load.(*DbInstance)
	}
	return nil
}

func TestConnection(d *entity.Db) {
	// 验证第一个库是否可以连接即可
	DB, err := GetDbConn(d, strings.Split(d.Database, " ")[0])
	biz.ErrIsNilAppendErr(err, "数据库连接失败: %s")
	defer DB.Close()
}

// 获取数据库连接
func GetDbConn(d *entity.Db, db string) (*sql.DB, error) {
	// SSH Conect
	if d.EnableSshTunnel == 1 && d.SshTunnelMachineId != 0 {
		sshTunnelMachine := MachineApp.GetSshTunnelMachine(d.SshTunnelMachineId)
		if d.Type == entity.DbTypeMysql {
			mysql.RegisterDialContext(d.Network, func(ctx context.Context, addr string) (net.Conn, error) {
				return sshTunnelMachine.GetDialConn("tcp", addr)
			})
		} else if d.Type == entity.DbTypePostgres {
			_, err := pq.DialOpen(&PqSqlDialer{sshTunnelMachine: sshTunnelMachine}, getDsn(d, db))
			if err != nil {
				panic(biz.NewBizErr(fmt.Sprintf("postgres隧道连接失败: %s", err.Error())))
			}
		}
	}

	DB, err := sql.Open(d.Type, getDsn(d, db))
	if err != nil {
		return nil, err
	}
	err = DB.Ping()
	if err != nil {
		DB.Close()
		return nil, err
	}

	return DB, nil
}

func SelectDataByDb(db *sql.DB, selectSql string) ([]string, []map[string]interface{}, error) {
	rows, err := db.Query(selectSql)
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
			if isFirst {
				colNames = append(colNames, colName)
			}
			if v == nil {
				rowData[colName] = nil
				continue
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

type PqSqlDialer struct {
	sshTunnelMachine *machine.SshTunnelMachine
}

func (pd *PqSqlDialer) Dial(network, address string) (net.Conn, error) {
	if sshConn, err := pd.sshTunnelMachine.GetDialConn("tcp", address); err == nil {
		// 将ssh conn包装，否则redis内部设置超时会报错,ssh conn不支持设置超时会返回错误: ssh: tcpChan: deadline not supported
		return &utils.WrapSshConn{Conn: sshConn}, nil
	} else {
		return nil, err
	}
}
func (pd *PqSqlDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return pd.Dial(network, address)
}

// db实例
type DbInstance struct {
	Id                 string
	Type               string
	ProjectId          uint64
	db                 *sql.DB
	sshTunnelMachineId uint64
}

// 执行查询语句
// 依次返回 列名数组，结果map，错误
func (d *DbInstance) SelectData(execSql string) ([]string, []map[string]interface{}, error) {
	execSql = strings.Trim(execSql, " ")
	isSelect := strings.HasPrefix(execSql, "SELECT") || strings.HasPrefix(execSql, "select")
	isShow := strings.HasPrefix(execSql, "show")
	isExplain := strings.HasPrefix(execSql, "explain")

	if !isSelect && !isShow && !isExplain {
		return nil, nil, errors.New("该sql非查询语句")
	}
	return SelectDataByDb(d.db, execSql)
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
	if d.db != nil {
		if err := d.db.Close(); err != nil {
			global.Log.Errorf("关闭数据库实例[%s]连接失败: %s", d.Id, err.Error())
		}
		d.db = nil
	}
}

// 获取dataSourceName
func getDsn(d *entity.Db, db string) string {
	var dsn string
	if d.Type == entity.DbTypeMysql {
		dsn = fmt.Sprintf("%s:%s@%s(%s:%d)/%s?timeout=8s", d.Username, d.Password, d.Network, d.Host, d.Port, db)
		if d.Params != "" {
			dsn = fmt.Sprintf("%s&%s", dsn, d.Params)
		}
		return dsn
	}

	if d.Type == entity.DbTypePostgres {
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", d.Host, d.Port, d.Username, d.Password, db)
		if d.Params != "" {
			dsn = fmt.Sprintf("%s %s", dsn, strings.Join(strings.Split(d.Params, "&"), " "))
		}
		return dsn
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
	MYSQL_COLUMN_MA = `SELECT table_name tableName, column_name columnName, column_type columnType,
	column_comment columnComment, column_key columnKey, extra, is_nullable nullable from information_schema.columns
	WHERE table_name in (%s) AND table_schema = (SELECT database()) ORDER BY tableName, ordinal_position LIMIT %d, %d`

	// mysql 列信息元数据总数
	MYSQL_COLOUMN_MA_COUNT = `SELECT COUNT(*) maNum from information_schema.columns
	WHERE table_name in (%s) AND table_schema = (SELECT database())`
)

const (
	// postgres 表信息元数据
	PGSQL_TABLE_MA = `SELECT obj_description(c.oid) AS "tableComment", c.relname AS "tableName" FROM pg_class c 
	JOIN pg_namespace n ON c.relnamespace = n.oid WHERE n.nspname = (select current_schema()) AND c.reltype > 0`

	PGSQL_TABLE_INFO = `SELECT obj_description(c.oid) AS "tableComment", c.relname AS "tableName" FROM pg_class c 
	JOIN pg_namespace n ON c.relnamespace = n.oid WHERE n.nspname = (select current_schema()) AND c.reltype > 0`

	PGSQL_INDEX_INFO = `SELECT indexname AS "indexName", indexdef AS "indexComment"
	FROM pg_indexes WHERE schemaname =  (select current_schema()) AND tablename = '%s'`

	PGSQL_COLUMN_MA = `SELECT
		C.relname AS "tableName",
		A.attname AS "columnName",
		concat_ws ( '', t.typname, SUBSTRING ( format_type ( a.atttypid, a.atttypmod ) FROM '\(.*\)' ) ) AS "columnType",
		d.description AS "columnComment" 
	FROM
		pg_attribute a LEFT JOIN pg_description d ON d.objoid = a.attrelid 
		AND d.objsubid = A.attnum
		LEFT JOIN pg_class c ON A.attrelid = c.oid
		LEFT JOIN pg_namespace pn ON c.relnamespace = pn.oid
		LEFT JOIN pg_type t ON a.atttypid = t.oid 
	WHERE
		A.attnum >= 0 
		AND pn.nspname = (select current_schema())
		AND C.relname in (%s)
	ORDER BY
		C.relname DESC,
		A.attnum ASC
	OFFSET %d LIMIT %d	
	`

	PGSQL_COLUMN_MA_COUNT = `SELECT COUNT(*) "maNum"
	FROM
		pg_attribute a LEFT JOIN pg_description d ON d.objoid = a.attrelid 
		AND d.objsubid = A.attnum
		LEFT JOIN pg_class c ON A.attrelid = c.oid
		LEFT JOIN pg_namespace pn ON c.relnamespace = pn.oid
		LEFT JOIN pg_type t ON a.atttypid = t.oid 
	WHERE
		A.attnum >= 0 
		AND pn.nspname = (select current_schema())
		AND C.relname in (%s)
	`
)

func (d *DbInstance) GetTableMetedatas() []map[string]interface{} {
	var sql string
	if d.Type == entity.DbTypeMysql {
		sql = MYSQL_TABLE_MA
	} else if d.Type == "postgres" {
		sql = PGSQL_TABLE_MA
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
	if d.Type == entity.DbTypeMysql {
		countSqlTmp = MYSQL_COLOUMN_MA_COUNT
		sqlTmp = MYSQL_COLUMN_MA
	} else if d.Type == entity.DbTypePostgres {
		countSqlTmp = PGSQL_COLUMN_MA_COUNT
		sqlTmp = PGSQL_COLUMN_MA
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
	if d.Type == entity.DbTypeMysql {
		sql = MYSQL_TABLE_INFO
	} else if d.Type == entity.DbTypePostgres {
		sql = PGSQL_TABLE_INFO
	}
	_, res, _ := d.SelectData(sql)
	return res
}

func (d *DbInstance) GetTableIndex(tableName string) []map[string]interface{} {
	var sql string
	if d.Type == entity.DbTypeMysql {
		sql = fmt.Sprintf(MYSQL_INDEX_INFO, tableName)
	} else if d.Type == entity.DbTypePostgres {
		sql = fmt.Sprintf(PGSQL_INDEX_INFO, tableName)
	}
	_, res, _ := d.SelectData(sql)
	return res
}

func (d *DbInstance) GetCreateTableDdl(tableName string) []map[string]interface{} {
	var sql string
	if d.Type == entity.DbTypeMysql {
		sql = fmt.Sprintf("show create table %s ", tableName)
	}
	_, res, _ := d.SelectData(sql)
	return res
}
