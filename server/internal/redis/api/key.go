package api

import (
	"context"
	"mayfly-go/internal/redis/api/form"
	"mayfly-go/internal/redis/api/vo"
	"mayfly-go/internal/redis/domain/entity"
	"mayfly-go/internal/redis/rdm"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// 单次 scan 请求的时间预算：稀疏匹配下凑批循环可能扫完整库（百万 key 库为数秒），
// 预算耗尽即带已凑的 keys 与未归零游标返回，推进权交还前端（游标非零即还有更多）
const scanBudget = 250 * time.Millisecond

// scanShouldStop 凑批循环的退出判据：凑满目标数、游标归零（本节点扫完）、或时间预算耗尽。
// 纯函数便于单测覆盖三条件边界
func scanShouldStop(matched, target int64, cursor uint64, elapsed, budget time.Duration) bool {
	return matched >= target || cursor == 0 || elapsed >= budget
}

// scan获取redis的key列表信息
func (r *Redis) ScanKeys(rc *req.Ctx) {
	ri := r.getRedisConn(rc)

	form := rc.BindJson[form.RedisScanForm]()

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

		rc.ResData = &vo.Keys{Cursor: cursorRes, Keys: keys, DbSize: size, Summaries: r.summarizeBatch(rc, ri, keys)}
		return
	}

	start := time.Now()
	// 通配符或全匹配
	mode := ri.Info.Mode
	if mode == "" || mode == rdm.StandaloneMode || mode == rdm.SentinelMode {
		redisAddr := ri.Cli.Options().Addr
		cursorRes[redisAddr] = form.Cursor[redisAddr]
		for {
			ks, cursor, err := ri.Scan(cursorRes[redisAddr], form.Match, form.Count)
			biz.ErrIsNil(err)
			cursorRes[redisAddr] = cursor
			keys = append(keys, ks...)
			// 退出判据收在循环底部：每轮至少一次 SCAN，保证请求必有进展
			if scanShouldStop(int64(len(keys)), form.Count, cursor, time.Since(start), scanBudget) {
				break
			}
		}
	} else if mode == rdm.ClusterMode {
		mu := &sync.Mutex{}
		// 遍历所有master节点，并执行scan命令，合并keys；预算 deadline 全节点共享
		ri.ClusterCli.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
			redisAddr := client.Options().Addr
			nowCursor := form.Cursor[redisAddr]
			for {
				ks, cursor, _ := client.Scan(ctx, nowCursor, form.Match, form.Count).Result()
				var done bool
				// 遍历节点的内部回调函数使用异步调用：keys 的追加与退出判据都须在锁内，避免并发 append 的读竞争
				mu.Lock()
				cursorRes[redisAddr] = cursor
				nowCursor = cursor
				keys = append(keys, ks...)
				done = scanShouldStop(int64(len(keys)), form.Count, cursor, time.Since(start), scanBudget)
				mu.Unlock()
				if done {
					break
				}
			}
			return nil
		})
	}

	rc.ResData = &vo.Keys{Cursor: cursorRes, Keys: keys, DbSize: size, Summaries: r.summarizeBatch(rc, ri, keys)}
}

// summarizeBatch 本批 key 的摘要随扫描响应返回：树角标（类型/过期）无需前端再发独立摘要请求，
// 大库下「扫描批 + 摘要批」双链路会让请求数随 key 数线性翻倍。
// 摘要取不到不拖死 key 列表：降级为空集合，前端按无角标渲染
func (r *Redis) summarizeBatch(rc *req.Ctx, ri *rdm.RedisConn, keys []string) []*entity.KeySummary {
	if len(keys) == 0 {
		return nil
	}
	summaries, err := r.keyValueApp.SummarizeKeys(rc.MetaCtx, ri, keys)
	if err != nil {
		return nil
	}
	return summaries
}
