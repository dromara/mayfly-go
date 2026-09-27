package keyvalue

import (
	"context"
	"fmt"
	"strings"

	"mayfly-go/internal/redis/domain/entity"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
)

const (
	// DefaultPageSize 视角成员分页的默认页大小
	DefaultPageSize = 50
	// MaxPageSize 成员分页上限，避免单次拉取把服务端内存与前端表格打满
	MaxPageSize = 1000
)

// keepTTL 为「整体覆盖值」的写命令补上 EX 以保住原有过期时间：
// 不补则 SET 类命令会静默清除 TTL，用户改一次值就丢了过期策略；
// KEEPTTL 需 Redis 6+，用 EX + 当前剩余秒数在旧版本上等效且兼容
func keepTTL(ctx context.Context, cmd redis.Cmdable, key string, args []any) []any {
	ttl := cmd.TTL(ctx, key).Val()
	if ttl > 0 {
		return append(args, "EX", int(ttl.Seconds()))
	}
	return args
}

// trimArg 取表单参数值并去掉首尾空白
func trimArg(args map[string]string, key string) string {
	return strings.TrimSpace(args[key])
}

// splitLines 多行输入 → 值列表，用于一次添加多个元素/字段
func splitLines(text string) []string {
	vals := make([]string, 0, 4)
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimRight(line, "\r"); line != "" {
			vals = append(vals, line)
		}
	}
	return vals
}

// splitKeys 逗号/空白/换行分隔的 key 或成员列表 → 切片
func splitKeys(text string) []string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == ' ' || r == '\t' || r == ';'
	})
	res := make([]string, 0, len(fields))
	for _, field := range fields {
		if field != "" {
			res = append(res, field)
		}
	}
	return res
}

// parseScore 解析 score 等数值参数，非法时报业务错误而非静默归零
func parseScore(val any) (float64, error) {
	score, err := cast.ToFloat64E(val)
	if err != nil {
		return 0, errInvalidArg(fmt.Sprintf("score must be a number, got %v", val))
	}
	return score, nil
}

// parseInt64 解析整数参数（下标、偏移、数量），非法时报业务错误
func parseInt64(val any) (int64, error) {
	num, err := cast.ToInt64E(val)
	if err != nil {
		return 0, errInvalidArg(fmt.Sprintf("argument must be an integer, got %v", val))
	}
	return num, nil
}

// pageSize 单页成员数归一：未传或非法用默认值，超上限则截断
func pageSize(size int64) int64 {
	if size <= 0 {
		return DefaultPageSize
	}
	if size > MaxPageSize {
		return MaxPageSize
	}
	return size
}

// setExtra 写入成员的派生列：Extra 按需惰性创建，避免给没有派生数据的行也分配 map
func setExtra(extra map[string]string, key, val string) map[string]string {
	if extra == nil {
		extra = make(map[string]string, 2)
	}
	extra[key] = val
	return extra
}

// appendAll 把字符串切片展开为命令参数（SET key v1 v2 ...）
func appendAll(args []any, vals []string) []any {
	for _, val := range vals {
		args = append(args, val)
	}
	return args
}

// 成员表单与扩展操作入参的字段名，前后端契约的一部分：
// 前端把表单值原样放进 args，处理器按名取值，因此新增视角只需新增常量与读取代码
const (
	argValue       = "value"
	argBinary      = "binary" // string 视角的二进制（base64）开关
	extraBinary    = "binary" // Member.Extra 标记：内容为 base64 编码
	argField       = "field"
	argScore       = "score"
	argIndex       = "index"
	argId          = "id"
	argMember      = "member"
	argMemberTo    = "memberTo"
	argValues      = "values"
	argPosition    = "position"
	argBit         = "bit"
	argStart       = "start"
	argEnd         = "end"
	argCount       = "count"
	argIncrement   = "increment"
	argLongitude   = "longitude"
	argLatitude    = "latitude"
	argRadius      = "radius"
	argUnit        = "unit"
	argKind        = "kind"
	argDestination = "destination"
	argSourceKeys  = "sourceKeys"
	argGroup       = "group"
	argMaxLen      = "maxLen"
	argFields      = "fields" // 多字段入参（批量设置字段过期等）
	// hash 字段级过期的派生列与表单 prop 同名，编辑回填因此不需要额外映射代码
	extraTtl = "ttl"
)

// allowOps 校验成员写操作是否被该视角支持：视角外的 op 一律报错，绝不退化成新增
func allowOps(w *entity.MemberWrite, view string, ops ...string) error {
	for _, allow := range ops {
		if w.Op == allow {
			return nil
		}
	}
	return errInvalidArg(view + " does not support the member op: " + w.Op)
}

// pickFields 收集写请求里的成员标识：优先批量 members，单行时回退 member
func pickFields(w *entity.MemberWrite, pick func(*entity.Member) string) []string {
	members := w.Members
	if len(members) == 0 && w.Member != nil {
		members = []*entity.Member{w.Member}
	}
	vals := make([]string, 0, len(members))
	for _, m := range members {
		if val := pick(m); val != "" {
			vals = append(vals, val)
		}
	}
	return vals
}

// pickMembers 批量操作的完整行数据（需要下标、id 等复合定位信息时使用）
func pickMembers(w *entity.MemberWrite) []*entity.Member {
	if len(w.Members) > 0 {
		return w.Members
	}
	if w.Member == nil {
		return nil
	}
	return []*entity.Member{w.Member}
}

// memberValues 批量删除时取成员值（set/list/stream 以值或 id 定位）
func memberValues(w *entity.MemberWrite) []string {
	return pickFields(w, func(m *entity.Member) string { return m.Value })
}
