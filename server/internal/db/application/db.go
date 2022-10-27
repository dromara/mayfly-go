package application

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/internal/constant"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	machineapp "mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/infrastructure/machine"
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
	GetPageList(condition *entity.DbQuery, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	Count(condition *entity.DbQuery) int64

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

func newDbApp(dbRepo repository.Db, dbSqlRepo repository.DbSql) Db {
	return &dbAppImpl{
		dbRepo:    dbRepo,
		dbSqlRepo: dbSqlRepo,
	}
}

type dbAppImpl struct {
	dbRepo    repository.Db
	dbSqlRepo repository.DbSql
}

// 分页获取数据库信息列表
func (d *dbAppImpl) GetPageList(condition *entity.DbQuery, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return d.dbRepo.GetDbList(condition, pageParam, toEntity, orderBy...)
}

func (d *dbAppImpl) Count(condition *entity.DbQuery) int64 {
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
	oldDb := &entity.Db{Host: dbEntity.Host, Port: dbEntity.Port, TagId: dbEntity.TagId}
	err := d.GetDbBy(oldDb)

	if dbEntity.Id == 0 {
		biz.NotEmpty(dbEntity.Password, "密码不能为空")
		biz.IsTrue(err != nil, "该数据库实例已存在")
		dbEntity.PwdEncrypt()
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

	dbEntity.PwdEncrypt()
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
	ed.Network = ed.GetNetwork()
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

	_, res, err := SelectDataByDb(dbConn, getDatabasesSql, true)
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
	// 密码解密
	d.PwdDecrypt()
	biz.NotNil(d, "数据库信息不存在")
	biz.IsTrue(strings.Contains(d.Database, db), "未配置该库的操作权限")

	cacheKey := GetDbCacheKey(id, db)
	dbi := &DbInstance{Id: cacheKey, Type: d.Type, TagPath: d.TagPath, sshTunnelMachineId: d.SshTunnelMachineId}

	DB, err := GetDbConn(d, db)
	if err != nil {
		dbi.Close()
		global.Log.Errorf("连接db失败: %s:%d/%s", d.Host, d.Port, db)
		panic(biz.NewBizErr(fmt.Sprintf("数据库连接失败: %s", err.Error())))
	}

	// 最大连接周期，超过时间的连接就close
	// DB.SetConnMaxLifetime(100 * time.Second)
	// 设置最大连接数
	DB.SetMaxOpenConns(5)
	// 设置闲置连接数
	DB.SetMaxIdleConns(1)

	dbi.db = DB
	global.Log.Infof("连接db: %s:%d/%s", d.Host, d.Port, db)
	if needCache {
		dbCache.Put(cacheKey, dbi)
	}
	return dbi
}

//----------------------------------------  db instance  ------------------------------------

// db实例
type DbInstance struct {
	Id                 string
	Type               string
	TagPath            string
	db                 *sql.DB
	sshTunnelMachineId uint64
}

// 执行查询语句
// 依次返回 列名数组，结果map，错误
func (d *DbInstance) SelectData(execSql string) ([]string, []map[string]interface{}, error) {
	return SelectDataByDb(d.db, execSql, false)
}

// 将查询结果映射至struct，可具体参考sqlx库
func (d *DbInstance) SelectData2Struct(execSql string, dest interface{}) error {
	return Select2StructByDb(d.db, execSql, dest)
}

// 执行内部查询语句，不返回列名以及不限制行数
// 依次返回 结果map，错误
func (d *DbInstance) innerSelect(execSql string) ([]map[string]interface{}, error) {
	_, res, err := SelectDataByDb(d.db, execSql, true)
	return res, err
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

// 获取数据库元信息实现接口
func (di *DbInstance) GetMeta() DbMetadata {
	dbType := di.Type
	if dbType == entity.DbTypeMysql {
		return &MysqlMetadata{di: di}
	}
	if dbType == entity.DbTypePostgres {
		return &PgsqlMetadata{di: di}
	}
	return nil
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

//------------------------------------------------------------------------------

// 单次最大查询数据集
const Max_Rows = 2000

// 客户端连接缓存，指定时间内没有访问则会被关闭, key为数据库实例id:数据库
var dbCache = cache.NewTimedCache(constant.DbConnExpireTime, 5*time.Second).
	WithUpdateAccessTime(true).
	OnEvicted(func(key interface{}, value interface{}) {
		global.Log.Info(fmt.Sprintf("删除db连接缓存 id = %s", key))
		value.(*DbInstance).Close()
	})

func init() {
	machine.AddCheckSshTunnelMachineUseFunc(func(machineId uint64) bool {
		// 遍历所有db连接实例，若存在db实例使用该ssh隧道机器，则返回true，表示还在使用中...
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
		sshTunnelMachine := machineapp.GetMachineApp().GetSshTunnelMachine(d.SshTunnelMachineId)
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

func SelectDataByDb(db *sql.DB, selectSql string, isInner bool) ([]string, []map[string]interface{}, error) {
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
	rowNum := 0
	for rows.Next() {
		rowNum++
		// 非内部sql，则校验返回结果数量
		if !isInner {
			biz.IsTrue(rowNum <= Max_Rows, "结果集 > 2000, 请完善条件或分页信息")
		}

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
			// 如果是第一行，则将列名加入到列信息中，由于map是无序的，所有需要返回列名的有序数组
			if isFirst {
				colNames = append(colNames, colName)
			}
			rowData[colName] = valueConvert(v, colType)
		}
		// 放入结果集
		result = append(result, rowData)
		isFirst = false
	}
	return colNames, result, nil
}

// 将查询的值转为对应列类型的实际值，不全部转为字符串
func valueConvert(data []byte, colType *sql.ColumnType) interface{} {
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
	if strings.Contains(colScanType, "float") {
		floatV, _ := strconv.ParseFloat(stringV, 64)
		return floatV
	}

	return stringV
}

// 查询数据结果映射至struct。可参考sqlx库
func Select2StructByDb(db *sql.DB, selectSql string, dest interface{}) error {
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

// 删除db缓存并关闭该数据库所有连接
func CloseDb(dbId uint64, db string) {
	dbCache.Delete(GetDbCacheKey(dbId, db))
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
