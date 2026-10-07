package keyvalue

import (
	"context"

	"mayfly-go/internal/redis/domain/entity"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
)

// ViewZset zset 类型的分值/成员视角
const ViewZset = "zset"

var zsetDesc = withDefault(&entity.ViewDescriptor{
	View:   ViewZset,
	Label:  "redis.viewZset",
	Types:  []entity.KeyType{entity.KeyTypeZset},
	Layout: "table",
	Caps: entity.Capabilities{
		Create: true, Update: true, Delete: true, BatchDelete: true,
		Keyword: true, RankPaging: true, CursorPaging: true, Ops: true,
	},
	// ConsoleHints 命令控制台的快捷命令模板，{key} 由前端换成当前 key 名
	ConsoleHints: []string{
		"ZRANGE {key} 0 -1 WITHSCORES",
		"ZCARD {key}",
		"ZSCORE {key} member",
		"ZADD {key} score member",
	},
	Columns: []entity.Column{
		sortableColumn("score", "redis.colScore", "number", 140),
		column("value", "redis.colValue", "code", 0),
	},
	Form: form(
		numberField(argScore, "redis.colScore", true),
		textFieldWith(argValue, "redis.colValue", "", true),
	),
	Ops: []entity.OpSpec{
		op("zincrby", "redis.opZIncrBy", true, form(
			textFieldWith(argValue, "redis.colValue", "", true),
			numberField(argIncrement, "redis.colIncrement", true),
		)),
		op("zpopmax", "redis.opZPopMax", true, form(numberField(argCount, "redis.colCount", false))),
		op("zpopmin", "redis.opZPopMin", true, form(numberField(argCount, "redis.colCount", false))),
		op("zrank", "redis.opZRank", false, form(
			textFieldWith(argValue, "redis.colValue", "", true),
		)),
		op("zremrangebyscore", "redis.opZRemRangeByScore", true, form(
			textFieldWith(argMin, "redis.colMin", "redis.colMinTips", true),
			textFieldWith(argMax, "redis.colMax", "redis.colMaxTips", true),
		)),
	},
	// ReadCmd 面板读取本视角内容等价的命令名，触发策略判定与「申请查看」提单同口径
	ReadCmd: "ZRANGE",
})

// 分值区间参数的字段名，与 zremrangebyscore 的操作表单对应
const (
	argMin = "min"
	argMax = "max"
)

type zsetHandler struct{}

func init() { Register(&zsetHandler{}) }

func (z *zsetHandler) Descriptor() *entity.ViewDescriptor { return zsetDesc }

func (z *zsetHandler) Size(ctx context.Context, cmd redis.Cmdable, key string) (int64, error) {
	return cmd.ZCard(ctx, key).Result()
}

func (z *zsetHandler) Load(ctx context.Context, cmd redis.Cmdable, q *entity.MemberQuery) (*entity.MemberPage, error) {
	return loadSortedSet(ctx, cmd, q)
}

func (z *zsetHandler) BuildWrite(_ context.Context, _ redis.Cmdable, w *entity.MemberWrite) ([][]any, error) {
	switch w.Op {
	case entity.MemberOpCreate, entity.MemberOpUpdate:
		member := trimArg(w.Args, argValue)
		if member == "" {
			return nil, errInvalidArg("zset member is required")
		}
		score, err := parseScore(w.Args[argScore])
		if err != nil {
			return nil, err
		}
		// 改成员名等价于换成员：必须删旧成员，否则旧分值残留成脏数据；
		// 先 ZADD 后 ZREM：中途失败最坏是两个成员并存，不会把成员和分值一起丢掉
		if w.Op == entity.MemberOpUpdate && w.Member != nil && w.Member.Value != member {
			return [][]any{{"ZADD", w.Key, score, member}, {"ZREM", w.Key, w.Member.Value}}, nil
		}
		return [][]any{{"ZADD", w.Key, score, member}}, nil
	case entity.MemberOpDelete:
		members := memberValues(w)
		if len(members) == 0 {
			return nil, errInvalidArg("no zset member to delete")
		}
		return [][]any{appendAll([]any{"ZREM", w.Key}, members)}, nil
	}
	return nil, errInvalidArg("zset op: " + w.Op)
}

func (z *zsetHandler) PlanOp(ctx context.Context, cmd redis.Cmdable, req *entity.OpRequest) ([][]any, error) {
	member := trimArg(req.Args, argValue)
	switch req.Op {
	case "zincrby":
		if member == "" {
			return nil, errInvalidArg("zset member is required")
		}
		increment, err := parseScore(req.Args[argIncrement])
		if err != nil {
			return nil, err
		}
		return [][]any{{"ZINCRBY", req.Key, increment, member}}, nil
	case "zpopmax", "zpopmin":
		pop := "ZPOPMAX"
		if req.Op == "zpopmin" {
			pop = "ZPOPMIN"
		}
		args := []any{pop, req.Key}
		if count := cast.ToInt64(req.Args[argCount]); count > 0 {
			return [][]any{append(args, count)}, nil
		}
		return [][]any{args}, nil
	case "zrank":
		if member == "" {
			return nil, errInvalidArg("zset member is required")
		}
		return [][]any{{"ZREVRANK", req.Key, member}}, nil
	case "zremrangebyscore":
		min, max := trimArg(req.Args, argMin), trimArg(req.Args, argMax)
		if min == "" || max == "" {
			return nil, errInvalidArg("score range min/max is required")
		}
		return [][]any{{"ZREMRANGEBYSCORE", req.Key, min, max}}, nil
	}
	return nil, errInvalidArg("zset op: " + req.Op)
}

// loadSortedSet 有序集合的两种分页读取：
//   - 无关键字且无游标 → 按排名（分值倒序）区间读，翻页可精确定位；
//   - 有关键字或游标 → ZSCAN 游标读（scan 是唯一的按成员匹配方式）
//
// geo 视角在 Redis 内部就是 zset，因此与 zset 共用本函数，只在其上补齐经纬度
func loadSortedSet(ctx context.Context, cmd redis.Cmdable, q *entity.MemberQuery) (*entity.MemberPage, error) {
	total, err := cmd.ZCard(ctx, q.Key).Result()
	if err != nil {
		return nil, err
	}
	size := pageSize(q.Size)

	if q.Keyword != "" || q.Cursor != "" {
		vals, cursor, err := cmd.ZScan(ctx, q.Key, pageCursor(q.Cursor), scanMatch(q.Keyword), size).Result()
		if err != nil {
			return nil, err
		}
		members := make([]*entity.Member, 0, len(vals)/2)
		for i := 0; i+1 < len(vals); i += 2 {
			members = append(members, &entity.Member{Value: vals[i], Score: cast.ToFloat64(vals[i+1])})
		}
		return &entity.MemberPage{Total: total, Cursor: scanCursor(cursor), Members: members}, nil
	}

	if q.Offset >= total {
		return &entity.MemberPage{Total: total}, nil
	}
	sorted, err := cmd.ZRevRangeWithScores(ctx, q.Key, q.Offset, q.Offset+size-1).Result()
	if err != nil {
		return nil, err
	}
	members := make([]*entity.Member, 0, len(sorted))
	for _, item := range sorted {
		members = append(members, &entity.Member{Value: cast.ToString(item.Member), Score: item.Score})
	}
	return &entity.MemberPage{Total: total, Members: members}, nil
}

// scanCursor 游标归一：0 表示已遍历完成，前端据此禁用「加载更多」
func scanCursor(cursor uint64) string {
	if cursor == 0 {
		return ""
	}
	return cast.ToString(cursor)
}
