package api

import (
	"context"
	"mayfly-go/internal/redis/api/form"
	"mayfly-go/internal/redis/api/vo"
	"mayfly-go/internal/redis/rdm"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
)

// scan获取redis的key列表信息
func (r *Redis) ScanKeys(rc *req.Ctx) {
	ri := r.getRedisConn(rc)

	form := req.BindJsonAndValid[*form.RedisScanForm](rc)

	cmd := ri.GetCmdable()
	ctx := context.Background()

	keys := make([]string, 0)
	var cursorRes map[string]uint64 = make(map[string]uint64)

	size, _ := cmd.DBSize(ctx).Result()

	if form.Match != "" && !strings.ContainsAny(form.Match, "*") {
		// 精确匹配, 判断是否存在
		res, err := cmd.Exists(ctx, form.Match).Result()
		if err == nil && res != 0 {
			keys = append(keys, form.Match)
		}

		rc.ResData = &vo.Keys{Cursor: cursorRes, Keys: keys, DbSize: size}
		return
	}

	// 通配符或全匹配
	mode := ri.Info.Mode
	if mode == "" || mode == rdm.StandaloneMode || mode == rdm.SentinelMode {
		redisAddr := ri.Cli.Options().Addr
		cursorRes[redisAddr] = form.Cursor[redisAddr]
		for {
			ks, cursor, err := ri.Scan(cursorRes[redisAddr], form.Match, form.Count)
			biz.ErrIsNil(err)
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
	} else if mode == rdm.ClusterMode {
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
	}

	rc.ResData = &vo.Keys{Cursor: cursorRes, Keys: keys, DbSize: size}
}

func (r *Redis) KeyInfo(rc *req.Ctx) {
	ri, key := r.checkKeyAndGetRedisConn(rc)
	cmd := ri.GetCmdable()
	ctx := context.Background()
	ttl, err := cmd.TTL(ctx, key).Result()
	biz.ErrIsNilAppendErr(err, "ttl error: %s")

	ttlInt := -1
	if ttl != -1 {
		ttlInt = int(ttl.Seconds())
	}

	typeRes, err := cmd.Type(ctx, key).Result()
	biz.ErrIsNilAppendErr(err, "get key type error: %s")

	rc.ResData = &vo.KeyInfo{
		Key:  key,
		Ttl:  ttlInt,
		Type: typeRes,
	}
}

func (r *Redis) TtlKey(rc *req.Ctx) {
	ri, key := r.checkKeyAndGetRedisConn(rc)
	ttl, err := ri.GetCmdable().TTL(context.Background(), key).Result()
	biz.ErrIsNilAppendErr(err, "ttl error: %s")

	if ttl == -1 {
		rc.ResData = -1
	} else {
		rc.ResData = ttl.Seconds()
	}
}

func (r *Redis) MemoryUsage(rc *req.Ctx) {
	ri, key := r.checkKeyAndGetRedisConn(rc)
	rc.ResData = ri.GetCmdable().MemoryUsage(context.Background(), key).Val()
}
