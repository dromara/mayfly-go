package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/devops/domain/entity"
	"mayfly-go/internal/devops/domain/repository"
	"mayfly-go/internal/devops/infrastructure/persistence"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

type Redis interface {
	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.Redis, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	Count(condition *entity.Redis) int64

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

func (r *redisAppImpl) Count(condition *entity.Redis) int64 {
	return r.redisRepo.Count(condition)
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
	// ’修改信息且密码不为空‘ or ‘新增’需要测试是否可连接
	if (re.Id != 0 && re.Password != "") || re.Id == 0 {
		TestRedisConnection(re)
	}

	// 查找是否存在该库
	oldRedis := &entity.Redis{Host: re.Host, Db: re.Db}
	err := r.GetRedisBy(oldRedis)

	if re.Id == 0 {
		biz.IsTrue(err != nil, "该库已存在")
		r.redisRepo.Insert(re)
	} else {
		// 如果存在该库，则校验修改的库是否为该库
		if err == nil {
			biz.IsTrue(oldRedis.Id == re.Id, "该库已存在")
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
		load, ok := redisCache.Get(id)
		if ok {
			return load.(*RedisInstance)
		}
	}
	// 缓存不存在，则回调获取redis信息
	re := r.GetById(id)
	biz.NotNil(re, "redis信息不存在")

	redisMode := re.Mode
	ri := &RedisInstance{Id: id, ProjectId: re.ProjectId, Mode: redisMode}
	if redisMode == "" || redisMode == entity.RedisModeStandalone {
		rcli := getRedisCient(re)
		// 测试连接
		_, e := rcli.Ping(context.Background()).Result()
		if e != nil {
			rcli.Close()
			panic(biz.NewBizErr(fmt.Sprintf("redis连接失败: %s", e.Error())))
		}
		ri.Cli = rcli
	} else if redisMode == entity.RedisModeCluster {
		ccli := getRedisClusterClient(re)
		// 测试连接
		_, e := ccli.Ping(context.Background()).Result()
		if e != nil {
			ccli.Close()
			panic(biz.NewBizErr(fmt.Sprintf("redis集群连接失败: %s", e.Error())))
		}
		ri.ClusterCli = ccli
	}

	global.Log.Infof("连接redis: %s", re.Host)
	if needCache {
		redisCache.Put(re.Id, ri)
	}
	return ri
}

func getRedisCient(re *entity.Redis) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:        re.Host,
		Password:    re.Password, // no password set
		DB:          re.Db,       // use default DB
		DialTimeout: 8 * time.Second,
	})
}

func getRedisClusterClient(re *entity.Redis) *redis.ClusterClient {
	return redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:       strings.Split(re.Host, ","),
		Password:    re.Password,
		DialTimeout: 8 * time.Second,
	})
}

//------------------------------------------------------------------------------

// redis客户端连接缓存，30分钟内没有访问则会被关闭
var redisCache = cache.NewTimedCache(30*time.Minute, 5*time.Second).
	WithUpdateAccessTime(true).
	OnEvicted(func(key interface{}, value interface{}) {
		global.Log.Info(fmt.Sprintf("删除redis连接缓存 id = %d", key))
		value.(*RedisInstance).Close()
	})

// 移除redis连接缓存并关闭redis连接
func CloseRedis(id uint64) {
	redisCache.Delete(id)
}

func TestRedisConnection(re *entity.Redis) {
	var cmd redis.Cmdable
	if re.Mode == "" || re.Mode == entity.RedisModeStandalone {
		rcli := getRedisCient(re)
		defer rcli.Close()
		cmd = rcli
	} else if re.Mode == entity.RedisModeCluster {
		ccli := getRedisClusterClient(re)
		defer ccli.Close()
		cmd = ccli
	}

	// 测试连接
	_, e := cmd.Ping(context.Background()).Result()
	biz.ErrIsNilAppendErr(e, "Redis连接失败: %s")
}

// redis实例
type RedisInstance struct {
	Id         uint64
	ProjectId  uint64
	Mode       string
	Cli        *redis.Client
	ClusterCli *redis.ClusterClient
}

// 获取命令执行接口的具体实现
func (r *RedisInstance) GetCmdable() redis.Cmdable {
	redisMode := r.Mode
	if redisMode == "" || redisMode == entity.RedisModeStandalone {
		return r.Cli
	}
	if r.Mode == entity.RedisModeCluster {
		return r.ClusterCli
	}
	return nil
}

func (r *RedisInstance) Scan(cursor uint64, match string, count int64) ([]string, uint64) {
	keys, newcursor, err := r.GetCmdable().Scan(context.Background(), cursor, match, count).Result()
	biz.ErrIsNilAppendErr(err, "scan失败: %s")
	return keys, newcursor
}

func (r *RedisInstance) Close() {
	if r.Mode == entity.RedisModeStandalone {
		r.Cli.Close()
		return
	}
	if r.Mode == entity.RedisModeCluster {
		r.ClusterCli.Close()
	}
}
