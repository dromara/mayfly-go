package keyvalue

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"mayfly-go/internal/redis/domain/entity"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
)

// ViewStream stream 类型的条目视角
const ViewStream = "stream"

// Entry 的推送时间（毫秒）来自 entry id 的 ms 段，作为派生列放进 Extra
const extraTime = "time"

var streamDesc = withDefault(&entity.ViewDescriptor{
	View:   ViewStream,
	Label:  "redis.viewStream",
	Types:  []entity.KeyType{entity.KeyTypeStream},
	Layout: "table",
	Caps: entity.Capabilities{
		Create: true, Delete: true, BatchDelete: true,
		RankPaging: true, CursorPaging: true, Ops: true,
	},
	// ConsoleHints 命令控制台的快捷命令模板，{key} 由前端换成当前 key 名
	ConsoleHints: []string{
		"XRANGE {key} - +",
		"XLEN {key}",
		"XINFO STREAM {key}",
		"XADD {key} * field value",
	},
	Columns: []entity.Column{
		column("id", "redis.colEntryId", "text", 190),
		column(extraTime, "redis.colTime", "time", 180),
		column("value", "redis.colField", "code", 0),
	},
	// 表单以 value 提交多行 key=value，与列表的摘要列同名，改行时可直接回填；列头只需「字段」，写入提示跟着表单走
	Form: form(
		textFieldWith(argId, "redis.colEntryId", "redis.colEntryIdTips", false),
		textAreaField(argValue, "redis.colFields", 5, true),
	),
	Ops: []entity.OpSpec{
		op("xtrim", "redis.opXTrim", true, form(
			numberField(argMaxLen, "redis.colMaxLen", true),
		)),
		op("xgroupcreate", "redis.opXGroupCreate", true, form(
			textFieldWith(argGroup, "redis.colGroup", "redis.colGroupTips", true),
		)),
		op("xack", "redis.opXAck", true, form(
			textFieldWith(argGroup, "redis.colGroup", "", true),
			textAreaField(argIds, "redis.colEntryIds", 3, true),
		)),
		op("xinfo", "redis.opXInfo", false, nil),
	},
	// ReadCmd 面板读取本视角内容等价的命令名，触发策略判定与「申请查看」提单同口径
	ReadCmd: "XRANGE",
})

// 批量条目 id 的入参名，与 xack 的操作表单对应
const argIds = "ids"

type streamHandler struct{}

func init() { Register(&streamHandler{}) }

func (s *streamHandler) Descriptor() *entity.ViewDescriptor { return streamDesc }

func (s *streamHandler) Size(ctx context.Context, cmd redis.Cmdable, key string) (int64, error) {
	return cmd.XLen(ctx, key).Result()
}

func (s *streamHandler) Load(ctx context.Context, cmd redis.Cmdable, q *entity.MemberQuery) (*entity.MemberPage, error) {
	total, err := s.Size(ctx, cmd, q.Key)
	if err != nil {
		return nil, err
	}

	start := q.Cursor
	if start == "" {
		start = "-"
	}
	messages, err := cmd.XRangeN(ctx, q.Key, start, "+", pageSize(q.Size)).Result()
	if err != nil {
		return nil, err
	}

	members := make([]*entity.Member, 0, len(messages))
	for _, message := range messages {
		members = append(members, toStreamMember(message))
	}

	next := ""
	if len(members) > 0 {
		next = streamNextCursor(members[len(members)-1].Id)
	}
	return &entity.MemberPage{Total: total, Cursor: next, Members: members}, nil
}

func (s *streamHandler) BuildWrite(_ context.Context, _ redis.Cmdable, w *entity.MemberWrite) ([][]any, error) {
	switch w.Op {
	case entity.MemberOpCreate:
		fields, err := streamFields(w.Args[argValue])
		if err != nil {
			return nil, err
		}
		id := trimArg(w.Args, argId)
		if id == "" {
			// 不指定 id 时由服务端按「毫秒-序号」自增，保证严格大于当前尾部 id
			id = "*"
		}
		args := appendAll([]any{"XADD", w.Key, id}, fields)
		return [][]any{args}, nil
	case entity.MemberOpUpdate:
		// 条目内容随 id 固定，Redis 不允许用同一 id 覆盖写入，因此改内容只能新增条目并删除旧条目
		return nil, errInvalidArg("stream entry is immutable, add a new entry instead")
	case entity.MemberOpDelete:
		ids := pickFields(w, func(m *entity.Member) string { return m.Id })
		if len(ids) == 0 {
			return nil, errInvalidArg("no stream entry to delete")
		}
		return [][]any{appendAll([]any{"XDEL", w.Key}, ids)}, nil
	}
	return nil, errInvalidArg("stream op: " + w.Op)
}

func (s *streamHandler) PlanOp(ctx context.Context, cmd redis.Cmdable, req *entity.OpRequest) ([][]any, error) {
	switch req.Op {
	case "xtrim":
		maxLen, err := parseInt64(req.Args[argMaxLen])
		if err != nil {
			return nil, err
		}
		// 近似裁剪（~）只按宏观节点删除，成本远低于精确裁剪，因此作为默认且唯一策略
		return [][]any{{"XTRIM", req.Key, "MAXLEN", "~", maxLen}}, nil
	case "xgroupcreate":
		group := trimArg(req.Args, argGroup)
		if group == "" {
			return nil, errInvalidArg("consumer group name is required")
		}
		// MKSTREAM：消费组是排他创建语义，组已存在时服务端会返回 BUSYGROUP，不额外先读再写
		return [][]any{{"XGROUP", "CREATE", req.Key, group, "$", "MKSTREAM"}}, nil
	case "xack":
		group := trimArg(req.Args, argGroup)
		ids := splitKeys(req.Args[argIds])
		if group == "" || len(ids) == 0 {
			return nil, errInvalidArg("consumer group and entry ids are required")
		}
		return [][]any{appendAll([]any{"XACK", req.Key, group}, ids)}, nil
	case "xinfo":
		return [][]any{{"XINFO", "STREAM", req.Key}}, nil
	}
	return nil, errInvalidArg("stream op: " + req.Op)
}

// toStreamMember 条目 → 行数据：字段集合进 Extra 供展开查看，Value 为按 key 排序的摘要（顺序稳定才可比较）
func toStreamMember(message redis.XMessage) *entity.Member {
	keys := make([]string, 0, len(message.Values))
	for key := range message.Values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	extras := make(map[string]string, len(keys))
	summary := make([]string, 0, len(keys))
	for _, key := range keys {
		value := toString(message.Values[key])
		extras[key] = value
		summary = append(summary, key+"="+value)
	}
	extras[extraTime] = streamEntryMillis(message.ID)

	return &entity.Member{Id: message.ID, Value: strings.Join(summary, "\n"), Extra: extras}
}

// streamFields 解析 "k=v" 多行输入为 XADD 的字段参数（按 key、value 交替展开）
func streamFields(text string) ([]string, error) {
	lines := splitLines(text)
	args := make([]string, 0, len(lines)*2)
	for _, line := range lines {
		key, value, found := strings.Cut(line, "=")
		if !found || strings.TrimSpace(key) == "" {
			return nil, errInvalidArg("stream field must be written as key=value, got: " + line)
		}
		args = append(args, strings.TrimSpace(key), value)
	}
	if len(args) == 0 {
		return nil, errInvalidArg("no stream field to add")
	}
	return args, nil
}

// streamEntryMillis entry id 的 ms 段，即条目的推送时间（毫秒时间戳）
func streamEntryMillis(id string) string {
	ms, _, _ := strings.Cut(id, "-")
	return ms
}

// streamNextCursor 下一页起始 id：seq + 1，避免同一条目在翻页边界重复出现
func streamNextCursor(id string) string {
	ms, seq, found := strings.Cut(id, "-")
	if !found {
		return ""
	}
	num, err := strconv.ParseUint(seq, 10, 64)
	if err != nil {
		return ""
	}
	return ms + "-" + strconv.FormatUint(num+1, 10)
}

// toString 兼容 int/嵌套值，统一转成可展示文本
func toString(val any) string {
	return cast.ToString(val)
}
