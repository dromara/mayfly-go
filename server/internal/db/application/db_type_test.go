package application

// GetDbTypesByDbIds 回归测试：数据同步列表的方言图标依赖「库id -> 所属实例类型」的批量映射，
// 需锁定同实例多库去重、空输入不查库、以及库/实例已被删除时不产出脏类型这三条语义。
//
// 运行：cd server && go test -count=1 -run TestGetDbTypesByDbIds ./internal/db/application/

import (
	"testing"

	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// stub 仓储：只借用 RepoImpl 的通用查询，仓储自有的分页方法在本测试中无用例
type stubDbRepo struct {
	*base.RepoImpl[*entity.Db]
}

func (s *stubDbRepo) GetPageList(*entity.DbQuery, ...string) (*model.PageResult[*entity.DbListPO], error) {
	return nil, nil
}

type stubDbInstanceRepo struct {
	*base.RepoImpl[*entity.DbInstance]
}

func (s *stubDbInstanceRepo) GetPageList(*entity.DbInstanceQuery, ...string) (*model.PageResult[*entity.DbInstance], error) {
	return nil, nil
}

// setupDbTypeTest 内存 sqlite 建表并接管全局 db（gormx 默认使用 global.Db），返回仅接好仓储的库应用
func setupDbTypeTest(t *testing.T) (*dbAppImpl, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&entity.Db{}, &entity.DbInstance{}))

	prev := global.Db
	global.Db = db
	t.Cleanup(func() { global.Db = prev })

	instanceApp := &instanceAppImpl{}
	instanceApp.Repo = &stubDbInstanceRepo{RepoImpl: &base.RepoImpl[*entity.DbInstance]{}}

	app := &dbAppImpl{}
	app.Repo = &stubDbRepo{RepoImpl: &base.RepoImpl[*entity.Db]{}}
	app.dbInstanceApp = instanceApp
	return app, db
}

func newTestInstance(t *testing.T, db *gorm.DB, name, dbType string) uint64 {
	t.Helper()
	inst := &entity.DbInstance{Name: name, Type: dbType, Host: "127.0.0.1", Port: 3306}
	inst.FillBaseInfo(model.IdGenTypeNone, nil)
	require.NoError(t, db.Create(inst).Error)
	return inst.Id
}

func newTestDb(t *testing.T, db *gorm.DB, name string, instanceId uint64) uint64 {
	t.Helper()
	d := &entity.Db{Name: name, Code: name, InstanceId: instanceId}
	d.FillBaseInfo(model.IdGenTypeNone, nil)
	require.NoError(t, db.Create(d).Error)
	return d.Id
}

func TestGetDbTypesByDbIds(t *testing.T) {
	app, db := setupDbTypeTest(t)

	mysqlInstId := newTestInstance(t, db, "local-mysql", "mysql")
	newTestInstance(t, db, "unused", "redis")
	pgDbId := newTestDb(t, db, "pg_db", newTestInstance(t, db, "local-pg", "pgsql"))
	// 同一实例下的两个库，用于验证按实例去重后仍能各自命中类型
	mysqlDb1 := newTestDb(t, db, "mysql_db1", mysqlInstId)
	mysqlDb2 := newTestDb(t, db, "mysql_db2", mysqlInstId)
	// 悬空引用：库还在但实例已被删除，不能凭空补一个类型
	danglingDbId := newTestDb(t, db, "dangling", 99999)

	got, err := app.GetDbTypesByDbIds([]uint64{pgDbId, mysqlDb1, mysqlDb2, mysqlDb1, danglingDbId, 404})
	require.NoError(t, err)

	assert.Equal(t, "pgsql", got[pgDbId])
	assert.Equal(t, "mysql", got[mysqlDb1])
	assert.Equal(t, "mysql", got[mysqlDb2])
	assert.NotContains(t, got, danglingDbId, "实例已删除时不应返回类型")
	assert.Len(t, got, 3, "不存在的库不应出现在结果里")

	empty, err := app.GetDbTypesByDbIds(nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}
