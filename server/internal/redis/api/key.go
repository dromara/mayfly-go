package api

import (
	"context"
	"fmt"
	"mayfly-go/internal/redis/api/form"
	"mayfly-go/internal/redis/api/vo"
	"mayfly-go/internal/redis/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// scan获取redis的key列表信息
func (r *Redis) Scan(rc *req.Ctx) {
	ri := r.getRedisIns(rc)

	form := &form.RedisScanForm{}
	ginx.BindJsonAndValid(rc.GinCtx, form)

	cmd := ri.GetCmdable()
	ctx := context.Background()

	kis := make([]*vo.KeyInfo, 0)
	var cursorRes map[string]uint64 = make(map[string]uint64)

	mode := ri.Info.Mode
	if mode == "" || mode == entity.RedisModeStandalone || mode == entity.RedisModeSentinel {
		redisAddr := ri.Cli.Options().Addr
		// 汇总所有的查询出来的键值
		var keys []string
		// 有通配符或空时使用scan，非模糊匹配直接匹配key
		if form.Match == "" || strings.ContainsAny(form.Match, "*") {
			cursorRes[redisAddr] = form.Cursor[redisAddr]
			for {
				ks, cursor := ri.Scan(cursorRes[redisAddr], form.Match, form.Count)
				cursorRes[redisAddr] = cursor
				if len(ks) > 0 {
					// 返回了数据则追加总集合中
					keys = append(keys, ks...)
				}
				// 匹配的数量满足用户需求退出
				if int32(len(keys)) >= int32(form.Count) {
					break
				}
				// 匹配到最后退出
				if cursor == 0 {
					break
				}
			}
		} else {
			// 精确匹配
			keys = append(keys, form.Match)
		}

		var keyInfoSplit []string
		if len(keys) > 0 {
			keyInfosLua := `local result = {}
							-- KEYS[1]为第1个参数,lua数组下标从1开始
							for i = 1, #KEYS do
								local ttl = redis.call('ttl', KEYS[i]);
								local keyType = redis.call('type', KEYS[i]);
								table.insert(result, string.format("%d,%s", ttl, keyType['ok']));
							end;
							return table.concat(result, ".");`
			// 通过lua获取 ttl,type.ttl2,type2格式，以便下面切割获取ttl和type。避免多次调用ttl和type函数
			keyInfos, err := cmd.Eval(ctx, keyInfosLua, keys).Result()
			biz.ErrIsNilAppendErr(err, "执行lua脚本获取key信息失败: %s")
			keyInfoSplit = strings.Split(keyInfos.(string), ".")
		}

		for i, k := range keys {
			ttlType := strings.Split(keyInfoSplit[i], ",")
			ttl, _ := strconv.Atoi(ttlType[0])
			// 没有存在该key,则跳过
			if ttl == -2 {
				continue
			}
			ki := &vo.KeyInfo{Key: k, Type: ttlType[1], Ttl: int64(ttl)}
			kis = append(kis, ki)
		}
	} else if mode == entity.RedisModeCluster {
		var keys []string
		// 有通配符或空时使用scan，非模糊匹配直接匹配key
		if form.Match == "" || strings.ContainsAny(form.Match, "*") {
			mu := &sync.Mutex{}
			// 遍历所有master节点，并执行scan命令，合并keys
			ri.ClusterCli.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
				redisAddr := client.Options().Addr
				nowCursor := form.Cursor[redisAddr]
				for {
					ks, cursor, _ := client.Scan(ctx, nowCursor, form.Match, form.Count).Result()
					// 遍历节点的内部回调函数使用异步调用，如不加锁会导致集合并发错误
					mu.Lock()
					cursorRes[redisAddr] = cursor
					nowCursor = cursor
					if len(ks) > 0 {
						// 返回了数据则追加总集合中
						keys = append(keys, ks...)
					}
					mu.Unlock()
					// 匹配的数量满足用户需求退出
					if int32(len(keys)) >= int32(form.Count) {
						break
					}
					// 匹配到最后退出
					if cursor == 0 {
						break
					}
				}
				return nil
			})

		} else {
			// 精确匹配
			keys = append(keys, form.Match)
		}

		// 因为redis集群模式执行lua脚本key必须位于同一slot中，故单机获取的方式不适合
		// 使用lua获取key的ttl以及类型，减少网络调用
		keyInfoLua := `local ttl = redis.call('ttl', KEYS[1]);
					   local keyType = redis.call('type', KEYS[1]);
					   return string.format("%d,%s", ttl, keyType['ok'])`
		for _, k := range keys {
			keyInfo, err := cmd.Eval(ctx, keyInfoLua, []string{k}).Result()
			biz.ErrIsNilAppendErr(err, "执行lua脚本获取key信息失败: %s")
			ttlType := strings.Split(keyInfo.(string), ",")
			ttl, _ := strconv.Atoi(ttlType[0])
			// 没有存在该key,则跳过
			if ttl == -2 {
				continue
			}
			ki := &vo.KeyInfo{Key: k, Type: ttlType[1], Ttl: int64(ttl)}
			kis = append(kis, ki)
		}
	}

	size, _ := cmd.DBSize(context.TODO()).Result()
	rc.ResData = &vo.Keys{Cursor: cursorRes, Keys: kis, DbSize: size}
}

func (r *Redis) TtlKey(rc *req.Ctx) {
	ri, key := r.checkKeyAndGetRedisIns(rc)
	ttl, err := ri.GetCmdable().TTL(context.Background(), key).Result()
	biz.ErrIsNilAppendErr(err, "ttl失败: %s")

	if ttl == -1 {
		rc.ResData = -1
	} else {
		rc.ResData = ttl.Seconds()
	}
}

func (r *Redis) DeleteKey(rc *req.Ctx) {
	ri, key := r.checkKeyAndGetRedisIns(rc)
	rc.ReqParam = fmt.Sprintf("%s -> 删除key: %s", ri.Info.GetLogDesc(), key)
	ri.GetCmdable().Del(context.Background(), key)
}

func (r *Redis) RenameKey(rc *req.Ctx) {
	form := &form.Rename{}
	ginx.BindJsonAndValid(rc.GinCtx, form)

	ri := r.getRedisIns(rc)
	rc.ReqParam = fmt.Sprintf("%s -> 重命名key[%s] -> [%s]", ri.Info.GetLogDesc(), form.Key, form.NewKey)
	ri.GetCmdable().Rename(context.Background(), form.Key, form.NewKey)
}

func (r *Redis) ExpireKey(rc *req.Ctx) {
	form := &form.Expire{}
	ginx.BindJsonAndValid(rc.GinCtx, form)

	ri := r.getRedisIns(rc)
	rc.ReqParam = fmt.Sprintf("%s -> 重置key[%s]过期时间为%d", ri.Info.GetLogDesc(), form.Key, form.Seconds)
	ri.GetCmdable().Expire(context.Background(), form.Key, time.Duration(form.Seconds)*time.Second)
}

// 移除过期时间
func (r *Redis) PersistKey(rc *req.Ctx) {
	ri, key := r.checkKeyAndGetRedisIns(rc)
	rc.ReqParam = fmt.Sprintf("%s -> 移除key[%s]的过期时间", ri.Info.GetLogDesc(), key)
	ri.GetCmdable().Persist(context.Background(), key)
}

// 清空库
func (r *Redis) FlushDb(rc *req.Ctx) {
	ri := r.getRedisIns(rc)
	rc.ReqParam = fmt.Sprintf("%s -> flushdb", ri.Info.GetLogDesc())
	ri.GetCmdable().FlushDB(context.Background())
}
