package keyvalue

import (
	"context"

	"mayfly-go/internal/redis/domain/entity"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
)

// ViewList list 类型的下标/值视角
const ViewList = "list"

// 入队方向：前端表单以 position 提交，head 对应 LPUSH，其余一律按 tail 处理
const (
	positionHead = "head"
	positionTail = "tail"
)

var listDesc = withDefault(&entity.ViewDescriptor{
	View:   ViewList,
	Label:  "redis.viewList",
	Types:  []entity.KeyType{entity.KeyTypeList},
	Layout: "table",
	Caps: entity.Capabilities{
		Create: true, Update: true, Delete: true, BatchDelete: true,
		RankPaging: true, Ops: true,
	},
	// ConsoleHints 命令控制台的快捷命令模板，{key} 由前端换成当前 key 名
	ConsoleHints: []string{
		"LRANGE {key} 0 -1",
		"LLEN {key}",
		"RPUSH {key} value",
		"LPOP {key}",
	},
	Columns: []entity.Column{
		column("index", "redis.colIndex", "number", 110),
		column("value", "redis.colValue", "code", 0),
	},
	Form: form(
		selectField(argPosition, "redis.colPosition", option(positionTail, "redis.positionTail"), option(positionHead, "redis.positionHead")),
		textAreaField(argValues, "redis.colValues", 5, true),
	),
	UpdateForm: form(
		numberField(argIndex, "redis.colIndex", true),
		textFieldWith(argValue, "redis.colValue", "", false),
	),
	Ops: []entity.OpSpec{
		op("ltrim", "redis.opLTrim", true, form(
			numberField(argStart, "redis.colStart", true),
			numberField(argEnd, "redis.colEnd", true),
		)),
		op("lpop", "redis.opLPop", true, form(numberField(argCount, "redis.colCount", false))),
		op("rpop", "redis.opRPop", true, form(numberField(argCount, "redis.colCount", false))),
	},
	// ReadCmd 面板读取本视角内容等价的命令名，触发策略判定与「申请查看」提单同口径
	ReadCmd: "LRANGE",
})

type listHandler struct{}

func init() { Register(&listHandler{}) }

func (l *listHandler) Descriptor() *entity.ViewDescriptor { return listDesc }

func (l *listHandler) Size(ctx context.Context, cmd redis.Cmdable, key string) (int64, error) {
	return cmd.LLen(ctx, key).Result()
}

func (l *listHandler) Load(ctx context.Context, cmd redis.Cmdable, q *entity.MemberQuery) (*entity.MemberPage, error) {
	total, err := l.Size(ctx, cmd, q.Key)
	if err != nil {
		return nil, err
	}
	if q.Offset >= total {
		return &entity.MemberPage{Total: total}, nil
	}

	size := pageSize(q.Size)
	values, err := cmd.LRange(ctx, q.Key, q.Offset, q.Offset+size-1).Result()
	if err != nil {
		return nil, err
	}

	members := make([]*entity.Member, 0, len(values))
	for i, value := range values {
		members = append(members, &entity.Member{Index: q.Offset + int64(i), Value: value})
	}
	return &entity.MemberPage{Total: total, Members: members}, nil
}

func (l *listHandler) BuildWrite(ctx context.Context, cmd redis.Cmdable, w *entity.MemberWrite) ([][]any, error) {
	switch w.Op {
	case entity.MemberOpCreate:
		values := splitLines(w.Args[argValues])
		if len(values) == 0 {
			return nil, errInvalidArg("no list value to push")
		}
		push := "RPUSH"
		if trimArg(w.Args, argPosition) == positionHead {
			push = "LPUSH"
		}
		return [][]any{appendAll([]any{push, w.Key}, values)}, nil
	case entity.MemberOpUpdate:
		if w.Member == nil {
			return nil, errInvalidArg("list index is required")
		}
		index, err := parseInt64(w.Args[argIndex])
		if err != nil {
			return nil, err
		}
		return [][]any{{"LSET", w.Key, index, w.Args[argValue]}}, nil
	case entity.MemberOpDelete:
		if len(w.Members) == 0 && w.Member == nil {
			return nil, errInvalidArg("no list index to delete")
		}
		// LREM 按值删除，因此先读回该行当前值；带 count=1 只删一个，避免误删内容相同的其它行
		cmds := make([][]any, 0, 4)
		for _, member := range pickMembers(w) {
			value, err := cmd.LIndex(ctx, w.Key, member.Index).Result()
			if err == redis.Nil {
				continue
			}
			if err != nil {
				return nil, err
			}
			cmds = append(cmds, []any{"LREM", w.Key, 1, value})
		}
		if len(cmds) == 0 {
			return nil, ErrKeyNotFound(ctx)
		}
		return cmds, nil
	}
	return nil, errInvalidArg("list op: " + w.Op)
}

func (l *listHandler) PlanOp(ctx context.Context, cmd redis.Cmdable, req *entity.OpRequest) ([][]any, error) {
	switch req.Op {
	case "ltrim":
		start, err := parseInt64(req.Args[argStart])
		if err != nil {
			return nil, err
		}
		end, err := parseInt64(req.Args[argEnd])
		if err != nil {
			return nil, err
		}
		return [][]any{{"LTRIM", req.Key, start, end}}, nil
	case "lpop", "rpop":
		push := "LPOP"
		if req.Op == "rpop" {
			push = "RPOP"
		}
		args := []any{push, req.Key}
		if count := cast.ToInt64(req.Args[argCount]); count > 0 {
			return [][]any{append(args, count)}, nil
		}
		return [][]any{args}, nil
	}
	return nil, errInvalidArg("list op: " + req.Op)
}
