package application

import (
	"database/sql"
	"fmt"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/machine/infrastructure/machine"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Db interface {
	base.App[*entity.Db]

	// 分页获取
	GetPageList(condition *entity.DbQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	Count(condition *entity.DbQuery) int64

	Save(entity *entity.Db) error

	// 删除数据库信息
	Delete(id uint64) error

	// 获取数据库连接实例
	// @param id 数据库实例id
	// @param db 数据库
	GetDbConnection(dbId uint64, dbName string) (*DbConnection, error)
}

func newDbApp(dbRepo repository.Db, dbSqlRepo repository.DbSql, dbInstanceApp Instance) Db {
	app := &dbAppImpl{
		dbSqlRepo:     dbSqlRepo,
		dbInstanceApp: dbInstanceApp,
	}
	app.Repo = dbRepo
	return app
}

type dbAppImpl struct {
	base.AppImpl[*entity.Db, repository.Db]

	dbSqlRepo     repository.DbSql
	dbInstanceApp Instance
}

// 分页获取数据库信息列表
func (d *dbAppImpl) GetPageList(condition *entity.DbQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return d.GetRepo().GetDbList(condition, pageParam, toEntity, orderBy...)
}

func (d *dbAppImpl) Count(condition *entity.DbQuery) int64 {
	return d.GetRepo().Count(condition)
}

func (d *dbAppImpl) Save(dbEntity *entity.Db) error {
	// 查找是否存在
	oldDb := &entity.Db{Name: dbEntity.Name, InstanceId: dbEntity.InstanceId}
	err := d.GetBy(oldDb)

	if dbEntity.Id == 0 {
		if err == nil {
			return errorx.NewBiz("该实例下数据库名已存在")
		}
		return d.Insert(dbEntity)
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil && oldDb.Id != dbEntity.Id {
		return errorx.NewBiz("该实例下数据库名已存在")
	}

	dbId := dbEntity.Id
	old, err := d.GetById(new(entity.Db), dbId)
	if err != nil {
		return errorx.NewBiz("该数据库不存在")
	}

	oldDbs := strings.Split(old.Database, " ")
	newDbs := strings.Split(dbEntity.Database, " ")
	// 比较新旧数据库列表，需要将移除的数据库相关联的信息删除
	_, delDb, _ := collx.ArrayCompare(newDbs, oldDbs, func(i1, i2 string) bool {
		return i1 == i2
	})

	for _, v := range delDb {
		// 关闭数据库连接
		CloseDb(dbEntity.Id, v)
		// 删除该库关联的所有sql记录
		d.dbSqlRepo.DeleteByCond(&entity.DbSql{DbId: dbId, Db: v})
	}

	return d.UpdateById(dbEntity)
}

func (d *dbAppImpl) Delete(id uint64) error {
	db, err := d.GetById(new(entity.Db), id)
	if err != nil {
		return errorx.NewBiz("该数据库不存在")
	}
	dbs := strings.Split(db.Database, " ")
	for _, v := range dbs {
		// 关闭连接
		CloseDb(id, v)
	}
	// 删除该库下用户保存的所有sql信息
	d.dbSqlRepo.DeleteByCond(&entity.DbSql{DbId: id})
	return d.DeleteById(id)
}

var mutex sync.Mutex

func (d *dbAppImpl) GetDbConnection(dbId uint64, dbName string) (*DbConnection, error) {
	cacheKey := GetDbCacheKey(dbId, dbName)

	// Id不为0，则为需要缓存
	needCache := dbId != 0
	if needCache {
		load, ok := dbCache.Get(cacheKey)
		if ok {
			return load.(*DbConnection), nil
		}
	}
	mutex.Lock()
	defer mutex.Unlock()

	db, err := d.GetById(new(entity.Db), dbId)
	if err != nil {
		return nil, errorx.NewBiz("数据库信息不存在")
	}
	if !strings.Contains(" "+db.Database+" ", " "+dbName+" ") {
		return nil, errorx.NewBiz("未配置数据库【%s】的操作权限", dbName)
	}

	instance, err := d.dbInstanceApp.GetById(new(entity.Instance), db.InstanceId)
	if err != nil {
		return nil, errorx.NewBiz("数据库实例不存在")
	}
	// 密码解密
	instance.PwdDecrypt()

	dbInfo := NewDbInfo(db, instance)
	dbInfo.Database = dbName
	dbi := &DbConnection{Id: cacheKey, Info: dbInfo}

	conn, err := getInstanceConn(instance, dbName)
	if err != nil {
		dbi.Close()
		logx.Errorf("连接db失败: %s:%d/%s", dbInfo.Host, dbInfo.Port, dbName)
		return nil, errorx.NewBiz(fmt.Sprintf("数据库连接失败: %s", err.Error()))
	}

	// 最大连接周期，超过时间的连接就close
	// conn.SetConnMaxLifetime(100 * time.Second)
	// 设置最大连接数
	conn.SetMaxOpenConns(5)
	// 设置闲置连接数
	conn.SetMaxIdleConns(1)

	dbi.db = conn
	logx.Infof("连接db: %s:%d/%s", dbInfo.Host, dbInfo.Port, dbName)
	if needCache {
		dbCache.Put(cacheKey, dbi)
	}
	return dbi, nil
}

//----------------------------------------  db instance  ------------------------------------

type DbInfo struct {
	Id                 uint64
	Name               string
	Type               entity.DbType // 类型，mysql oracle等
	Host               string
	Port               int
	Network            string
	Username           string
	TagPath            string
	Database           string
	SshTunnelMachineId int
}

func NewDbInfo(db *entity.Db, instance *entity.Instance) *DbInfo {
	return &DbInfo{
		Id:                 db.Id,
		Name:               db.Name,
		Type:               instance.Type,
		Host:               instance.Host,
		Port:               instance.Port,
		Username:           instance.Username,
		TagPath:            db.TagPath,
		SshTunnelMachineId: instance.SshTunnelMachineId,
	}
}

// 获取记录日志的描述
func (d *DbInfo) GetLogDesc() string {
	return fmt.Sprintf("DB[id=%d, tag=%s, name=%s, ip=%s:%d, database=%s]", d.Id, d.TagPath, d.Name, d.Host, d.Port, d.Database)
}

// db实例连接信息
type DbConnection struct {
	Id   string
	Info *DbInfo

	db *sql.DB
}

// 执行查询语句
// 依次返回 列名数组，结果map，错误
func (d *DbConnection) SelectData(execSql string) ([]string, []map[string]any, error) {
	return selectDataByDb(d.db, execSql)
}

// 将查询结果映射至struct，可具体参考sqlx库
func (d *DbConnection) SelectData2Struct(execSql string, dest any) error {
	return select2StructByDb(d.db, execSql, dest)
}

// WalkTableRecord 遍历表记录
func (d *DbConnection) WalkTableRecord(selectSql string, walk func(record map[string]any, columns []string)) error {
	return walkTableRecord(d.db, selectSql, walk)
}

// 执行 update, insert, delete，建表等sql
// 返回影响条数和错误
func (d *DbConnection) Exec(sql string) (int64, error) {
	res, err := d.db.Exec(sql)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// 获取数据库元信息实现接口
func (d *DbConnection) GetMeta() DbMetadata {
	switch d.Info.Type {
	case entity.DbTypeMysql:
		return &MysqlMetadata{di: d}
	case entity.DbTypePostgres:
		return &PgsqlMetadata{di: d}
	default:
		panic(fmt.Sprintf("invalid database type: %s", d.Info.Type))
	}
}

// 关闭连接
func (d *DbConnection) Close() {
	if d.db != nil {
		if err := d.db.Close(); err != nil {
			logx.Errorf("关闭数据库实例[%s]连接失败: %s", d.Id, err.Error())
		}
		d.db = nil
	}
}

//------------------------------------------------------------------------------

// 客户端连接缓存，指定时间内没有访问则会被关闭, key为数据库实例id:数据库
var dbCache = cache.NewTimedCache(consts.DbConnExpireTime, 5*time.Second).
	WithUpdateAccessTime(true).
	OnEvicted(func(key any, value any) {
		logx.Info(fmt.Sprintf("删除db连接缓存 id = %s", key))
		value.(*DbConnection).Close()
	})

func init() {
	machine.AddCheckSshTunnelMachineUseFunc(func(machineId int) bool {
		// 遍历所有db连接实例，若存在db实例使用该ssh隧道机器，则返回true，表示还在使用中...
		items := dbCache.Items()
		for _, v := range items {
			if v.Value.(*DbConnection).Info.SshTunnelMachineId == machineId {
				return true
			}
		}
		return false
	})
}

func GetDbCacheKey(dbId uint64, db string) string {
	return fmt.Sprintf("%d:%s", dbId, db)
}

func selectDataByDb(db *sql.DB, selectSql string) ([]string, []map[string]any, error) {
	// 列名用于前端表头名称按照数据库与查询字段顺序显示
	var colNames []string
	result := make([]map[string]any, 0, 16)
	err := walkTableRecord(db, selectSql, func(record map[string]any, columns []string) {
		result = append(result, record)
		if colNames == nil {
			colNames = make([]string, len(columns))
			copy(colNames, columns)
		}
	})
	if err != nil {
		return nil, nil, err
	}
	return colNames, result, nil
}

func walkTableRecord(db *sql.DB, selectSql string, walk func(record map[string]any, columns []string)) error {
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

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return err
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
			return err
		}
		// 每行数据
		rowData := make(map[string]any, lenCols)
		// 把values中的数据复制到row中
		for i, v := range values {
			rowData[colTypes[i].Name()] = valueConvert(v, colTypes[i])
		}
		walk(rowData, colNames)
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

// 删除db缓存并关闭该数据库所有连接
func CloseDb(dbId uint64, db string) {
	dbCache.Delete(GetDbCacheKey(dbId, db))
}
