package application

import (
	"database/sql"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type Instance interface {
	// GetPageList 分页获取数据库实例
	GetPageList(condition *entity.InstanceQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any]

	Count(condition *entity.InstanceQuery) int64

	// GetInstanceBy 根据条件获取数据库实例
	GetInstanceBy(condition *entity.Instance, cols ...string) error

	// GetById 根据id获取数据库实例
	GetById(id uint64, cols ...string) *entity.Instance

	Save(instanceEntity *entity.Instance)

	// Delete 删除数据库信息
	Delete(id uint64)

	// GetDatabases 获取数据库实例的所有数据库列表
	GetDatabases(entity *entity.Instance) []string
}

func newInstanceApp(InstanceRepo repository.Instance) Instance {
	return &instanceAppImpl{
		instanceRepo: InstanceRepo,
	}
}

type instanceAppImpl struct {
	instanceRepo repository.Instance
}

// GetPageList 分页获取数据库实例
func (app *instanceAppImpl) GetPageList(condition *entity.InstanceQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any] {
	return app.instanceRepo.GetInstanceList(condition, pageParam, toEntity, orderBy...)
}

func (app *instanceAppImpl) Count(condition *entity.InstanceQuery) int64 {
	return app.instanceRepo.Count(condition)
}

// GetInstanceBy 根据条件获取数据库实例
func (app *instanceAppImpl) GetInstanceBy(condition *entity.Instance, cols ...string) error {
	return app.instanceRepo.GetInstance(condition, cols...)
}

// GetById 根据id获取数据库实例
func (app *instanceAppImpl) GetById(id uint64, cols ...string) *entity.Instance {
	return app.instanceRepo.GetById(id, cols...)
}

func (app *instanceAppImpl) Save(instanceEntity *entity.Instance) {
	// 默认tcp连接
	instanceEntity.Network = instanceEntity.GetNetwork()

	// 测试连接
	if instanceEntity.Password != "" {
		testConnection(instanceEntity)
	}

	// 查找是否存在该库
	oldInstance := &entity.Instance{Host: instanceEntity.Host, Port: instanceEntity.Port, Username: instanceEntity.Username}
	if instanceEntity.SshTunnelMachineId > 0 {
		oldInstance.SshTunnelMachineId = instanceEntity.SshTunnelMachineId
	}

	err := app.GetInstanceBy(oldInstance)
	if instanceEntity.Id == 0 {
		biz.NotEmpty(instanceEntity.Password, "密码不能为空")
		biz.IsTrue(err != nil, "该数据库实例已存在")
		instanceEntity.PwdEncrypt()
		app.instanceRepo.Insert(instanceEntity)
	} else {
		// 如果存在该库，则校验修改的库是否为该库
		if err == nil {
			biz.IsTrue(oldInstance.Id == instanceEntity.Id, "该数据库实例已存在")
		}
		instanceEntity.PwdEncrypt()
		app.instanceRepo.Update(instanceEntity)
	}
}

func (app *instanceAppImpl) Delete(id uint64) {
	app.instanceRepo.Delete(id)
}

// getInstanceConn 获取数据库连接数据库实例
func getInstanceConn(instance *entity.Instance, db string) (*sql.DB, error) {
	var conn *sql.DB
	var err error
	if instance.Type == entity.DbTypeMysql {
		conn, err = getMysqlDB(instance, db)
	} else if instance.Type == entity.DbTypePostgres {
		conn, err = getPgsqlDB(instance, db)
	}

	if err != nil {
		return nil, err
	}
	err = conn.Ping()
	if err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func testConnection(d *entity.Instance) {
	// 不指定数据库名称
	conn, err := getInstanceConn(d, "")
	biz.ErrIsNilAppendErr(err, "数据库连接失败: %s")
	defer conn.Close()
}

func (app *instanceAppImpl) GetDatabases(ed *entity.Instance) []string {
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

	dbConn, err := getInstanceConn(ed, metaDb)
	biz.ErrIsNilAppendErr(err, "数据库连接失败: %s")
	defer dbConn.Close()

	_, res, err := SelectDataByDb(dbConn, getDatabasesSql)
	biz.ErrIsNilAppendErr(err, "获取数据库列表失败")
	for _, re := range res {
		databases = append(databases, re["dbname"].(string))
	}
	return databases
}
