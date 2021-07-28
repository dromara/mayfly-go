package application

import (
	"mayfly-go/base/biz"
	"mayfly-go/base/model"
	"mayfly-go/server/devops/domain/entity"
	"mayfly-go/server/devops/domain/repository"
	"mayfly-go/server/devops/infrastructure/persistence"
	"sync"

	"github.com/go-redis/redis"
)

type Redis interface {
	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.Redis, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.Redis

	// 根据条件获取
	GetRedisBy(condition *entity.Redis, cols ...string) error

	Save(entity *entity.Redis)

	// 删除数据库信息
	Delete(id uint64)

	// 获取数据库连接实例
	GetRedisInstance(id uint64) *RedisInstance
}

type redisAppImpl struct {
	redisRepo repository.Redis
}

var RedisApp Redis = &redisAppImpl{
	redisRepo: persistence.RedisDao,
}

// 分页获取机器脚本信息列表
func (r *redisAppImpl) GetPageList(condition *entity.Redis, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return r.redisRepo.GetRedisList(condition, pageParam, toEntity, orderBy...)
}

// 根据id获取
func (r *redisAppImpl) GetById(id uint64, cols ...string) *entity.Redis {
	return r.redisRepo.GetById(id, cols...)
}

// 根据条件获取
func (r *redisAppImpl) GetRedisBy(condition *entity.Redis, cols ...string) error {
	return r.redisRepo.GetRedis(condition, cols...)
}

func (r *redisAppImpl) Save(re *entity.Redis) {
	TestRedisConnection(re)

	// 查找是否存在该库
	oldRedis := &entity.Redis{Host: re.Host, Db: re.Db}
	err := r.GetRedisBy(oldRedis)

	if re.Id == 0 {
		biz.IsTrue(err != nil, "该库已存在")
		r.redisRepo.Insert(re)
	} else {
		// 如果存在该库，则校验修改的库是否为该库
		if err == nil {
			biz.IsTrue(re.Id == re.Id, "该库已存在")
		}
		// 先关闭数据库连接
		CloseRedis(re.Id)
		r.redisRepo.Update(re)
	}
}

// 删除Redis信息
func (r *redisAppImpl) Delete(id uint64) {
	CloseRedis(id)
	r.redisRepo.Delete(id)
}

// 获取数据库连接实例
func (r *redisAppImpl) GetRedisInstance(id uint64) *RedisInstance {
	// Id不为0，则为需要缓存
	needCache := id != 0
	if needCache {
		load, ok := redisCache.Load(id)
		if ok {
			return load.(*RedisInstance)
		}
	}
	// 缓存不存在，则回调获取redis信息
	re := r.GetById(id)
	biz.NotNil(re, "redis信息不存在")
	rcli := redis.NewClient(&redis.Options{
		Addr:     re.Host,
		Password: re.Password, // no password set
		DB:       re.Db,       // use default DB
	})
	// 测试连接
	_, e := rcli.Ping().Result()
	biz.ErrIsNilAppendErr(e, "redis连接失败: %s")

	ri := &RedisInstance{Id: id, Cli: rcli}
	if needCache {
		redisCache.LoadOrStore(re.Id, ri)
	}
	return ri
}

//------------------------------------------------------------------------------

var redisCache sync.Map

// redis实例
type RedisInstance struct {
	Id  uint64
	Cli *redis.Client
}

// 关闭redis连接
func CloseRedis(id uint64) {
	if load, ok := redisCache.Load(id); ok {
		load.(*RedisInstance).Cli.Close()
		redisCache.Delete(id)
	}
}

func TestRedisConnection(re *entity.Redis) {
	rcli := redis.NewClient(&redis.Options{
		Addr:     re.Host,
		Password: re.Password, // no password set
		DB:       re.Db,       // use default DB
	})
	defer rcli.Close()
	// 测试连接
	_, e := rcli.Ping().Result()
	biz.ErrIsNilAppendErr(e, "Redis连接失败: %s")
}

func (r *RedisInstance) Scan(cursor uint64, match string, count int64) ([]string, uint64) {
	keys, newcursor, err := r.Cli.Scan(cursor, match, count).Result()
	biz.ErrIsNilAppendErr(err, "scan失败: %s")
	return keys, newcursor
}
