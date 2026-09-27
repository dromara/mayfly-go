package keyvalue

import (
	"context"

	"mayfly-go/internal/redis/domain/entity"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
)

// ViewBitmap 位图视角：Redis 的 bitmap 就是一个 string，因此本视角服务 string 类型。
// 无法从内容上判定一个 string 是否为位图，故不作为 string 的默认视角，由用户手动切换
const ViewBitmap = "bitmap"

// 位图按字节窗口读取，因此一次读取的位数必须是 8 的倍数
const bitmapMinBits = 64

// Extra 中的字节值与可打印字符列，同时是表格列名
const (
	extraByte = "byte"
	extraChar = "char"
)

var bitmapDesc = &entity.ViewDescriptor{
	View:   ViewBitmap,
	Label:  "redis.viewBitmap",
	Types:  []entity.KeyType{entity.KeyTypeString},
	Layout: "table",
	Caps: entity.Capabilities{
		Create: true, Update: true, Delete: true,
		RankPaging: true, Ops: true,
	},
	// ConsoleHints 命令控制台的快捷命令模板，{key} 由前端换成当前 key 名
	ConsoleHints: []string{
		"GETBIT {key} 0",
		"SETBIT {key} 0 1",
		"BITCOUNT {key}",
		"BITPOS {key} 1",
	},
	Columns: []entity.Column{
		column("index", "redis.colBitOffset", "number", 130),
		column("value", "redis.colBit", "tag", 90),
		column(extraByte, "redis.colByte", "number", 110),
		column(extraChar, "redis.colChar", "text", 90),
	},
	// 表单 prop 与 Member 字段同名：index 为位偏移、value 为位值，前端用行数据即可通用回填
	Form: form(
		numberField(argIndex, "redis.colBitOffset", true),
		selectField(argValue, "redis.colBit", option("1", "redis.bitOne"), option("0", "redis.bitZero")),
	),
	Ops: []entity.OpSpec{
		op("bitcount", "redis.opBitCount", false, form(
			numberField(argStart, "redis.colStart", false),
			numberField(argEnd, "redis.colEnd", false),
		)),
		op("bitpos", "redis.opBitPos", false, form(
			selectField(argBit, "redis.colBit", option("1", "redis.bitOne"), option("0", "redis.bitZero")),
			numberField(argStart, "redis.colStart", false),
			numberField(argEnd, "redis.colEnd", false),
		)),
	},
}

type bitmapHandler struct{}

func init() { Register(&bitmapHandler{}) }

func (b *bitmapHandler) Descriptor() *entity.ViewDescriptor { return bitmapDesc }

// Size 位图的元素是「位」，因此总数与成员分页的 Total 同源（字节数 * 8）。
// 置位数属于统计口径，走 bitcount 操作；拿它当成员总数会让两处读数互相矛盾
func (b *bitmapHandler) Size(ctx context.Context, cmd redis.Cmdable, key string) (int64, error) {
	bits, err := cmd.StrLen(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return bits * 8, nil
}

// Load 按位偏移读取一段位图：以字节为单位取回原始内容后逐位展开（Redis 位序为大端，字节内高位在前）
func (b *bitmapHandler) Load(ctx context.Context, cmd redis.Cmdable, q *entity.MemberQuery) (*entity.MemberPage, error) {
	size := pageSize(q.Size)
	if bits := size % 8; bits != 0 {
		size += 8 - bits
	}
	if size < bitmapMinBits {
		size = bitmapMinBits
	}

	startByte := q.Offset / 8
	raw, err := cmd.GetRange(ctx, q.Key, startByte, startByte+size/8-1).Result()
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return &entity.MemberPage{Total: b.totalBits(ctx, cmd, q.Key)}, nil
	}

	members := make([]*entity.Member, 0, len(raw)*8)
	for i := 0; i < len(raw); i++ {
		char := raw[i]
		for bit := 7; bit >= 0; bit-- {
			set := (char >> uint(bit)) & 1
			members = append(members, &entity.Member{
				Index: startByte*8 + int64(i)*8 + int64(7-bit),
				Value: cast.ToString(set),
				Extra: map[string]string{extraByte: cast.ToString(char), extraChar: printableChar(char)},
			})
		}
	}
	return &entity.MemberPage{Total: b.totalBits(ctx, cmd, q.Key), Members: members}, nil
}

func (b *bitmapHandler) BuildWrite(_ context.Context, _ redis.Cmdable, w *entity.MemberWrite) ([][]any, error) {
	if w.Op != entity.MemberOpCreate && w.Op != entity.MemberOpUpdate && w.Op != entity.MemberOpDelete {
		return nil, errInvalidArg("bitmap op: " + w.Op)
	}
	args := w.Args
	if w.Op == entity.MemberOpDelete {
		// 位图没有「删除某一位」，置 0 即为删除语义
		args = map[string]string{argIndex: args[argIndex], argValue: "0"}
	} else if args[argValue] == "" {
		return nil, errInvalidArg("bitmap bit value must be 0 or 1")
	}

	offset, err := parseInt64(args[argIndex])
	if err != nil {
		return nil, err
	}
	bit, err := parseInt64(args[argValue])
	if err != nil || (bit != 0 && bit != 1) {
		return nil, errInvalidArg("bitmap bit value must be 0 or 1")
	}
	if offset < 0 {
		return nil, errInvalidArg("bitmap offset must not be negative")
	}
	return [][]any{{"SETBIT", w.Key, offset, bit}}, nil
}

func (b *bitmapHandler) PlanOp(ctx context.Context, cmd redis.Cmdable, req *entity.OpRequest) ([][]any, error) {
	switch req.Op {
	case "bitcount":
		start, end := req.Args[argStart], req.Args[argEnd]
		if start == "" && end == "" {
			return [][]any{{"BITCOUNT", req.Key}}, nil
		}
		return [][]any{{"BITCOUNT", req.Key, cast.ToInt64(start), cast.ToInt64(end)}}, nil
	case "bitpos":
		bit, err := parseInt64(req.Args[argBit])
		if err != nil || (bit != 0 && bit != 1) {
			return nil, errInvalidArg("bitmap bit value must be 0 or 1")
		}
		args := []any{"BITPOS", req.Key, bit}
		if req.Args[argStart] != "" {
			args = append(args, cast.ToInt64(req.Args[argStart]))
			if req.Args[argEnd] != "" {
				args = append(args, cast.ToInt64(req.Args[argEnd]))
			}
		}
		return [][]any{args}, nil
	}
	return nil, errInvalidArg("bitmap op: " + req.Op)
}

// totalBits 位图总位数 = 字节数 * 8，Redis 的位图长度总是按整字节增长
func (b *bitmapHandler) totalBits(ctx context.Context, cmd redis.Cmdable, key string) int64 {
	return cmd.StrLen(ctx, key).Val() * 8
}

func printableChar(char byte) string {
	if char < 0x20 || char > 0x7e {
		return "."
	}
	return string(rune(char))
}
