package keyvalue

import (
	"context"
	"strings"

	"mayfly-go/internal/redis/domain/entity"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
)

// ViewSet set 类型的成员视角
const ViewSet = "set"

// 集合运算类型，前端以 kind 提交，服务端白名单校验后才拼进命令
const (
	kindUnion = "union"
	kindDiff  = "diff"
	kindInter = "inter"
)

var setDesc = withDefault(&entity.ViewDescriptor{
	View:   ViewSet,
	Label:  "redis.viewSet",
	Types:  []entity.KeyType{entity.KeyTypeSet},
	Layout: "table",
	Caps: entity.Capabilities{
		Create: true, Update: true, Delete: true, BatchDelete: true,
		Keyword: true, CursorPaging: true, Ops: true,
	},
	// ConsoleHints 命令控制台的快捷命令模板，{key} 由前端换成当前 key 名
	ConsoleHints: []string{
		"SMEMBERS {key}",
		"SCARD {key}",
		"SISMEMBER {key} member",
		"SREM {key} member",
	},
	Columns: []entity.Column{column("value", "redis.colValue", "code", 0)},
	Form: form(
		textAreaField(argValues, "redis.colValues", 5, true),
	),
	UpdateForm: form(
		textAreaField(argValue, "redis.colValue", 4, false),
	),
	Ops: []entity.OpSpec{
		op("spop", "redis.opSPop", true, form(numberField(argCount, "redis.colCount", false))),
		op("combine", "redis.opCombine", true, form(
			selectField(argKind, "redis.colKind", option(kindUnion, "redis.kindUnion"), option(kindDiff, "redis.kindDiff"), option(kindInter, "redis.kindInter")),
			textFieldWith(argDestination, "redis.colDestination", "redis.colDestinationTips", true),
			textAreaField(argSourceKeys, "redis.colSourceKeys", 3, false),
		)),
	},
	// ReadCmd 面板读取本视角内容等价的命令名，触发策略判定与「申请查看」提单同口径
	ReadCmd: "SMEMBERS",
})

type setHandler struct{}

func init() { Register(&setHandler{}) }

func (s *setHandler) Descriptor() *entity.ViewDescriptor { return setDesc }

func (s *setHandler) Size(ctx context.Context, cmd redis.Cmdable, key string) (int64, error) {
	return cmd.SCard(ctx, key).Result()
}

func (s *setHandler) Load(ctx context.Context, cmd redis.Cmdable, q *entity.MemberQuery) (*entity.MemberPage, error) {
	total, err := s.Size(ctx, cmd, q.Key)
	if err != nil {
		return nil, err
	}

	values, cursor, err := cmd.SScan(ctx, q.Key, pageCursor(q.Cursor), scanMatch(q.Keyword), pageSize(q.Size)).Result()
	if err != nil {
		return nil, err
	}

	members := make([]*entity.Member, 0, len(values))
	for _, value := range values {
		members = append(members, &entity.Member{Value: value})
	}

	next := ""
	if cursor != 0 {
		next = cast.ToString(cursor)
	}
	return &entity.MemberPage{Total: total, Cursor: next, Members: members}, nil
}

func (s *setHandler) BuildWrite(_ context.Context, _ redis.Cmdable, w *entity.MemberWrite) ([][]any, error) {
	switch w.Op {
	case entity.MemberOpCreate:
		values := splitLines(w.Args[argValues])
		if len(values) == 0 {
			return nil, errInvalidArg("no set member to add")
		}
		return [][]any{appendAll([]any{"SADD", w.Key}, values)}, nil
	case entity.MemberOpUpdate:
		if w.Member == nil {
			return nil, errInvalidArg("set member is required")
		}
		value := trimArg(w.Args, argValue)
		if value == "" || w.Member.Value == value {
			return nil, errInvalidArg("set member value is unchanged")
		}
		// 集合成员没有「值」这一容器，改成员只能「加新 + 删旧」两条命令完成；
		// 先加后删：中途失败最坏是新旧成员并存，不会把成员删掉
		return [][]any{{"SADD", w.Key, value}, {"SREM", w.Key, w.Member.Value}}, nil
	case entity.MemberOpDelete:
		values := memberValues(w)
		if len(values) == 0 {
			return nil, errInvalidArg("no set member to delete")
		}
		return [][]any{appendAll([]any{"SREM", w.Key}, values)}, nil
	}
	return nil, errInvalidArg("set op: " + w.Op)
}

func (s *setHandler) PlanOp(ctx context.Context, cmd redis.Cmdable, req *entity.OpRequest) ([][]any, error) {
	switch req.Op {
	case "spop":
		args := []any{"SPOP", req.Key}
		if count := cast.ToInt64(req.Args[argCount]); count > 0 {
			return [][]any{append(args, count)}, nil
		}
		return [][]any{args}, nil
	case "combine":
		kind := strings.ToLower(trimArg(req.Args, argKind))
		store := map[string]string{kindUnion: "SUNIONSTORE", kindDiff: "SDIFFSTORE", kindInter: "SINTERSTORE"}[kind]
		if store == "" {
			return nil, errInvalidArg("set combine kind: " + kind)
		}
		destination := trimArg(req.Args, argDestination)
		if destination == "" {
			return nil, errInvalidArg("destination key is required")
		}

		// 参与运算的 key 包含当前 key，前端只需填其余 key
		keys := append([]string{req.Key}, splitKeys(req.Args[argSourceKeys])...)
		return [][]any{appendAll([]any{store, destination}, keys)}, nil
	}
	return nil, errInvalidArg("set op: " + req.Op)
}
