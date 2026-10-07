package keyvalue

import (
	"context"
	"encoding/base64"
	"strings"
	"unicode/utf8"

	"mayfly-go/internal/redis/domain/entity"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
)

// ViewString string 类型的文本/二进制视角
const ViewString = "string"

var stringDesc = withDefault(&entity.ViewDescriptor{
	View:   ViewString,
	Label:  "redis.viewString",
	Types:  []entity.KeyType{entity.KeyTypeString},
	Layout: "value",
	Caps: entity.Capabilities{
		Create: true,
		Update: true,
	},
	// ConsoleHints 命令控制台的快捷命令模板，{key} 由前端换成当前 key 名
	ConsoleHints: []string{
		"GET {key}",
		"TTL {key}",
		"STRLEN {key}",
		"TYPE {key}",
	},
	// ReadCmd 面板读取本视角内容等价的命令名，触发策略判定与「申请查看」提单同口径
	ReadCmd: "GET",
	Columns: []entity.Column{column("value", "redis.colValue", "code", 0)},
	Form: form(
		textAreaField(argValue, "redis.colValue", 8, false),
		switchField(argBinary, "redis.binaryBase64"),
	),
	// 单值视角不挂额外操作：整个 key 就是一个值，改完保存等价于 SET；
	// APPEND / INCRBY / STRLEN 要么是同一件事的另一种写法，要么是纯查询，都由「命令控制台」页签承担
})

type stringHandler struct{}

func init() { Register(&stringHandler{}) }

func (s *stringHandler) Descriptor() *entity.ViewDescriptor {
	return stringDesc
}

func (s *stringHandler) Size(ctx context.Context, cmd redis.Cmdable, key string) (int64, error) {
	return cmd.StrLen(ctx, key).Result()
}

func (s *stringHandler) Load(ctx context.Context, cmd redis.Cmdable, q *entity.MemberQuery) (*entity.MemberPage, error) {
	value, err := cmd.Get(ctx, q.Key).Result()
	if err == redis.Nil {
		return &entity.MemberPage{}, nil
	}
	if err != nil {
		return nil, err
	}

	member := &entity.Member{Value: value}
	if !utf8.ValidString(value) {
		member.Value = base64.StdEncoding.EncodeToString([]byte(value))
		member.Extra = map[string]string{extraBinary: "true"}
	}
	return &entity.MemberPage{Total: int64(len(value)), Members: []*entity.Member{member}}, nil
}

func (s *stringHandler) BuildWrite(ctx context.Context, cmd redis.Cmdable, w *entity.MemberWrite) ([][]any, error) {
	// 清空 string 值等价于删除 key，交由 key 级删除操作处理，避免语义分叉
	if err := allowOps(w, ViewString, entity.MemberOpCreate, entity.MemberOpUpdate); err != nil {
		return nil, err
	}

	value, err := decodeValue(w.Args)
	if err != nil {
		return nil, err
	}
	return [][]any{keepTTL(ctx, cmd, w.Key, []any{"SET", w.Key, value})}, nil
}

// decodeValue 解析成员表单里的值：二进制开关打开时按 base64 解码（非法 UTF-8 内容无法直接穿过 JSON）
func decodeValue(args map[string]string) (string, error) {
	value := args[argValue]
	if !isBinaryArg(args) {
		return value, nil
	}
	return decodeBase64(value)
}

// decodeBase64 解码 base64 内容，空内容视为空值
func decodeBase64(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	decoded, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return "", errInvalidArg("value is not a valid base64 text")
	}
	return string(decoded), nil
}

// isBinaryArg 成员表单的二进制开关是否为真（表单值可能是 bool 字符串或空）
func isBinaryArg(args map[string]string) bool {
	return strings.EqualFold(cast.ToString(args[argBinary]), "true")
}
