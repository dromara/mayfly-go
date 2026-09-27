package keyvalue

import (
	"context"
	"fmt"
	"strings"

	"mayfly-go/internal/redis/domain/entity"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
)

// ViewHash hash 类型的 field/value 视角
const ViewHash = "hash"

// opHExpire 批量设置字段过期的操作标识
const opHExpire = "hexpire"

// ttlProbeKey 能力探测用的 key 名：探测走只读命令且命中不存在的 key，不会留下任何数据
const ttlProbeKey = "mayfly:redis:hash-field-ttl-probe"

var hashDesc = withDefault(&entity.ViewDescriptor{
	View:   ViewHash,
	Label:  "redis.viewHash",
	Types:  []entity.KeyType{entity.KeyTypeHash},
	Layout: "table",
	Caps: entity.Capabilities{
		Create: true, Update: true, Delete: true, BatchDelete: true,
		Keyword: true, CursorPaging: true, Ops: true,
	},
	// ConsoleHints 命令控制台的快捷命令模板，{key} 由前端换成当前 key 名
	ConsoleHints: []string{
		"HGETALL {key}",
		"HKEYS {key}",
		"HLEN {key}",
		"HDEL {key} field",
	},
	Columns: []entity.Column{
		sortableColumn("field", "redis.colField", "text", 220),
		column("value", "redis.colValue", "code", 0),
		column(extraTtl, "redis.colFieldTtl", "ttl", 120),
	},
	Form: form(
		textFieldWith(argField, "redis.colField", "redis.colFieldTips", true),
		textAreaField(argValue, "redis.colValue", 6, false),
		numberFieldWith(extraTtl, "redis.colFieldTtl", "redis.fieldTtlTips", false),
	),
	// 修改成员只改值：字段过期由 BuildWrite 按读取到的剩余秒续回，改值不会把过期设置弄丢，
	// 要单独调整过期走「更多操作 → 设置字段过期」
	UpdateForm: form(
		textFieldWith(argField, "redis.colField", "redis.colFieldTips", true),
		textAreaField(argValue, "redis.colValue", 6, false),
	),
	Ops: []entity.OpSpec{
		op("hincrby", "redis.opHIncrBy", true, form(
			textFieldWith(argField, "redis.colField", "", true),
			numberField(argIncrement, "redis.colIncrement", true),
		)),
		op(opHExpire, "redis.opHExpire", true, form(
			textAreaField(argFields, "redis.colFieldList", 3, true),
			numberFieldWith(extraTtl, "redis.colFieldTtl", "redis.fieldTtlOpTips", false),
		)),
		op("hrandfield", "redis.opHRandField", false, form(
			numberField(argCount, "redis.colCount", false),
		)),
	},
})

type hashHandler struct{}

func init() { Register(&hashHandler{}) }

func (h *hashHandler) Descriptor() *entity.ViewDescriptor { return hashDesc }

// supportsFieldTtl 探测实例是否支持 hash 字段级过期（Redis 7.4 起）。
// 用 HTTL 探一次不存在的 key：该命令只读无副作用，比拉全量 COMMAND 表便宜得多。
// 只有服务端明确报「unknown command」才算不支持；key 不存在的 nil 回复、类型不符等
// 都属于命令存在的表现
func supportsFieldTtl(ctx context.Context, cmd redis.Cmdable) bool {
	_, err := cmd.HTTL(ctx, ttlProbeKey, "field").Result()
	return err == nil || !strings.Contains(err.Error(), "unknown command")
}

// ProbeSupport 字段级过期是 Redis 7.4 才有的能力：不支持的实例上把过期列、新增表单里的过期入参
// 与批量设置过期操作一并摘掉，前端按收到的描述符渲染即可，无需知道版本这件事。
// 描述符是进程级共享的，因此返回副本，绝不原地修改
func (h *hashHandler) ProbeSupport(ctx context.Context, cmd redis.Cmdable, desc *entity.ViewDescriptor) *entity.ViewDescriptor {
	if supportsFieldTtl(ctx, cmd) {
		return desc
	}

	probed := *desc
	probed.Columns = withoutColumn(desc.Columns, extraTtl)
	probed.Form = withoutFormField(desc.Form, extraTtl)
	probed.Ops = withoutOp(desc.Ops, opHExpire)
	return &probed
}

func (h *hashHandler) Size(ctx context.Context, cmd redis.Cmdable, key string) (int64, error) {
	return cmd.HLen(ctx, key).Result()
}

func (h *hashHandler) Load(ctx context.Context, cmd redis.Cmdable, q *entity.MemberQuery) (*entity.MemberPage, error) {
	total, err := h.Size(ctx, cmd, q.Key)
	if err != nil {
		return nil, err
	}

	fields, cursor, err := cmd.HScan(ctx, q.Key, pageCursor(q.Cursor), scanMatch(q.Keyword), pageSize(q.Size)).Result()
	if err != nil {
		return nil, err
	}

	members := make([]*entity.Member, 0, len(fields)/2)
	names := make([]string, 0, len(fields)/2)
	for i := 0; i+1 < len(fields); i += 2 {
		members = append(members, &entity.Member{Field: fields[i], Value: fields[i+1]})
		names = append(names, fields[i])
	}
	h.attachFieldTtl(ctx, cmd, q.Key, members, names)

	next := ""
	if cursor != 0 {
		next = cast.ToString(cursor)
	}
	return &entity.MemberPage{Total: total, Cursor: next, Members: members}, nil
}

// attachFieldTtl 补充字段剩余过期秒数（派生列，随成员行一起返回，前端按 ttl 语义渲染）。
// 低版本实例、或整个 hash 从未设过字段过期时 HTTL 可能报错/返回空，一律留空该列，
// 不能让一屏数据的读取被可选信息拖失败
func (h *hashHandler) attachFieldTtl(ctx context.Context, cmd redis.Cmdable, key string, members []*entity.Member, fields []string) {
	if len(fields) == 0 {
		return
	}
	ttls, err := cmd.HTTL(ctx, key, fields...).Result()
	if err != nil || len(ttls) != len(members) {
		return
	}
	for i, member := range members {
		member.Extra = setExtra(member.Extra, extraTtl, cast.ToString(ttls[i]))
	}
}

func (h *hashHandler) BuildWrite(_ context.Context, _ redis.Cmdable, w *entity.MemberWrite) ([][]any, error) {
	field := trimArg(w.Args, argField)

	switch w.Op {
	case entity.MemberOpCreate:
		if field == "" {
			return nil, errInvalidArg("hash field is required")
		}
		ttl, err := parseFieldTtl(w.Args)
		if err != nil {
			return nil, err
		}
		return fieldTtlCmds([][]any{{"HSET", w.Key, field, w.Args[argValue]}}, w.Key, field, ttl), nil
	case entity.MemberOpUpdate:
		if field == "" || w.Member == nil {
			return nil, errInvalidArg("hash field is required")
		}
		cmds := [][]any{{"HSET", w.Key, field, w.Args[argValue]}}
		// 改 field 等于换键，必须删掉旧 field，否则旧值永久残留成脏数据；
		// 顺序是先写新再删旧：中途失败最坏是新旧并存（看得见、可再删），反过来会直接丢值
		if old := w.Member.Field; old != "" && old != field {
			cmds = append(cmds, []any{"HDEL", w.Key, old})
		}
		ttl, err := parseFieldTtl(w.Args)
		if err != nil {
			return nil, err
		}
		if ttl < 0 {
			// HSET 会清掉字段原有的过期（Redis 语义），因此没填过期时按读取到的剩余秒续回。
			// 页面停留过久时这个数最多让字段比原计划多活一次编辑间隔，不影响值
			ttl = remainingFieldTtl(w.Member)
		}
		return fieldTtlCmds(cmds, w.Key, field, ttl), nil
	case entity.MemberOpDelete:
		fields := pickFields(w, func(m *entity.Member) string { return m.Field })
		if len(fields) == 0 {
			return nil, errInvalidArg("no hash field to delete")
		}
		return [][]any{appendAll([]any{"HDEL", w.Key}, fields)}, nil
	}
	return nil, errInvalidArg("hash op: " + w.Op)
}

func (h *hashHandler) PlanOp(ctx context.Context, cmd redis.Cmdable, req *entity.OpRequest) ([][]any, error) {
	switch req.Op {
	case "hincrby":
		field := trimArg(req.Args, argField)
		if field == "" {
			return nil, errInvalidArg("hash field is required")
		}
		return [][]any{{"HINCRBY", req.Key, field, cast.ToInt64(req.Args[argIncrement])}}, nil
	case opHExpire:
		fields := splitLines(req.Args[argFields])
		if len(fields) == 0 {
			return nil, errInvalidArg("no hash field to expire")
		}
		ttl, err := parseFieldTtl(req.Args)
		if err != nil {
			return nil, err
		}
		if ttl <= 0 {
			return [][]any{appendAll([]any{"HPERSIST", req.Key, "FIELDS", len(fields)}, fields)}, nil
		}
		return [][]any{appendAll([]any{"HEXPIRE", req.Key, ttl, "FIELDS", len(fields)}, fields)}, nil
	case "hrandfield":
		args := []any{"HRANDFIELD", req.Key}
		if count := cast.ToInt64(req.Args[argCount]); count > 0 {
			return [][]any{append(args, count)}, nil
		}
		return [][]any{args}, nil
	}
	return nil, errInvalidArg("hash op: " + req.Op)
}

// parseFieldTtl 解析字段过期入参：返回 -1 表示未填（不改动现有过期），
// 0 表示清除过期，正数为剩余秒数；非整数值报业务错误而不是静默当成 0
func parseFieldTtl(args map[string]string) (int64, error) {
	raw := strings.TrimSpace(args[extraTtl])
	if raw == "" {
		return -1, nil
	}
	seconds, err := cast.ToInt64E(raw)
	if err != nil {
		return 0, errInvalidArg(fmt.Sprintf("hash field ttl must be an integer, got %q", raw))
	}
	return max(seconds, 0), nil
}

// remainingFieldTtl 取成员行上记录的剩余过期秒数；无过期或该实例未提供此信息时返回 -1（不改动）
func remainingFieldTtl(member *entity.Member) int64 {
	if member == nil || member.Extra == nil {
		return -1
	}
	seconds := cast.ToInt64(member.Extra[extraTtl])
	if seconds <= 0 {
		return -1
	}
	return seconds
}

// fieldTtlCmds 按 ttl 语义追加字段过期命令：>0 续期，0 清除，<0 不追加。
//
// 清除必须走 HPERSIST：HEXPIRE 传 0 或负数的语义是「立即过期」，会把字段直接删掉
func fieldTtlCmds(cmds [][]any, key, field string, ttl int64) [][]any {
	switch {
	case ttl > 0:
		return append(cmds, []any{"HEXPIRE", key, ttl, "FIELDS", 1, field})
	case ttl == 0:
		return append(cmds, []any{"HPERSIST", key, "FIELDS", 1, field})
	}
	return cmds
}
