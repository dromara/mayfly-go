package application

import (
	"mayfly-go/internal/redis/domain/entity"
	"mayfly-go/internal/redis/domain/repository"
	"mayfly-go/internal/redis/rdm"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"strconv"
	"strings"
)

type Redis interface {
	base.App[*entity.Redis]

	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.RedisQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	Count(condition *entity.RedisQuery) int64

	Save(re *entity.Redis) error

	// 删除数据库信息
	Delete(id uint64) error

	// 获取数据库连接实例
	// id: 数据库实例id
	// db: 库号
	GetRedisConn(id uint64, db int) (*rdm.RedisConn, error)

	// 测试连接
	TestConn(re *entity.Redis) error
}

func newRedisApp(redisRepo repository.Redis) Redis {
	return &redisAppImpl{
		base.AppImpl[*entity.Redis, repository.Redis]{Repo: redisRepo},
	}
}

type redisAppImpl struct {
	base.AppImpl[*entity.Redis, repository.Redis]
}

// 分页获取redis列表
func (r *redisAppImpl) GetPageList(condition *entity.RedisQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return r.GetRepo().GetRedisList(condition, pageParam, toEntity, orderBy...)
}

func (r *redisAppImpl) Count(condition *entity.RedisQuery) int64 {
	return r.GetRepo().Count(condition)
}

func (r *redisAppImpl) Save(re *entity.Redis) error {
	// ’修改信息且密码不为空‘ or ‘新增’需要测试是否可连接
	if (re.Id != 0 && re.Password != "") || re.Id == 0 {
		if err := r.TestConn(re); err != nil {
			return errorx.NewBiz("Redis连接失败: %s", err.Error())
		}
	}

	// 查找是否存在该库
	oldRedis := &entity.Redis{Host: re.Host}
	if re.SshTunnelMachineId > 0 {
		oldRedis.SshTunnelMachineId = re.SshTunnelMachineId
	}
	err := r.GetBy(oldRedis)

	if re.Id == 0 {
		if err == nil {
			return errorx.NewBiz("该实例已存在")
		}
		re.PwdEncrypt()
		return r.Insert(re)
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil && oldRedis.Id != re.Id {
		return errorx.NewBiz("该实例已存在")
	}
	// 如果修改了redis实例的库信息，则关闭旧库的连接
	if oldRedis.Db != re.Db || oldRedis.SshTunnelMachineId != re.SshTunnelMachineId {
		for _, dbStr := range strings.Split(oldRedis.Db, ",") {
			db, _ := strconv.Atoi(dbStr)
			rdm.CloseConn(re.Id, db)
		}
	}
	re.PwdEncrypt()
	return r.UpdateById(re)
}

// 删除Redis信息
func (r *redisAppImpl) Delete(id uint64) error {
	re, err := r.GetById(new(entity.Redis), id)
	if err != nil {
		return errorx.NewBiz("该redis信息不存在")
	}
	// 如果存在连接，则关闭所有库连接信息
	for _, dbStr := range strings.Split(re.Db, ",") {
		db, _ := strconv.Atoi(dbStr)
		rdm.CloseConn(re.Id, db)
	}
	return r.DeleteById(id)
}

// 获取数据库连接实例
func (r *redisAppImpl) GetRedisConn(id uint64, db int) (*rdm.RedisConn, error) {
	return rdm.GetRedisConn(id, db, func() (*rdm.RedisInfo, error) {
		// 缓存不存在，则回调获取redis信息
		re, err := r.GetById(new(entity.Redis), id)
		if err != nil {
			return nil, errorx.NewBiz("redis信息不存在")
		}
		re.PwdDecrypt()

		return re.ToRedisInfo(db), nil
	})
}

func (r *redisAppImpl) TestConn(re *entity.Redis) error {
	db := 0
	if re.Db != "" {
		db, _ = strconv.Atoi(strings.Split(re.Db, ",")[0])
	}

	rc, err := re.ToRedisInfo(db).Conn()
	if err != nil {
		return err
	}
	rc.Close()
	return nil
}
