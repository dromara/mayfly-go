package keyvalue

import (
	"context"
	"strings"

	"mayfly-go/internal/redis/domain/entity"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
)

// ViewHyperLogLog HyperLogLog 视角：底层是带 "HYLL" 魔数的 string
const ViewHyperLogLog = "hyperloglog"

// hllMagic HyperLogLog 的序列化头部魔数，据此从普通 string 中自动识别
const hllMagic = "HYLL"

var hllDesc = &entity.ViewDescriptor{
	View:   ViewHyperLogLog,
	Label:  "redis.viewHyperLogLog",
	Types:  []entity.KeyType{entity.KeyTypeString},
	Layout: "value",
	Caps: entity.Capabilities{
		// 不提供 Update：整键只有一行「预估基数」，它是统计结果而非用户输入，改它等于把基数再 PFADD 进去
		Create: true,
	},
	// ConsoleHints 命令控制台的快捷命令模板，{key} 由前端换成当前 key 名
	ConsoleHints: []string{
		"PFCOUNT {key}",
		"PFADD {key} element",
	},
	Columns: []entity.Column{column("value", "redis.colPfCount", "number", 0)},
	Form: form(
		textAreaField(argValues, "redis.colPfValues", 5, true),
	),
	// 同其他单值视角：只有 PFADD 这一个写入入口，PFMERGE 这类跨键命令交给「命令控制台」
}

type hllHandler struct{}

func init() { Register(&hllHandler{}) }

func (h *hllHandler) Descriptor() *entity.ViewDescriptor { return hllDesc }

// Detect 读头部魔数判断是否为 HyperLogLog；string 视角是 string 类型的默认视角，因此本视角不设为默认
func (h *hllHandler) Detect(ctx context.Context, cmd redis.Cmdable, key string) bool {
	head, err := cmd.GetRange(ctx, key, 0, int64(len(hllMagic)-1)).Result()
	if err != nil {
		return false
	}
	return strings.HasPrefix(head, hllMagic)
}

func (h *hllHandler) Size(ctx context.Context, cmd redis.Cmdable, key string) (int64, error) {
	return cmd.PFCount(ctx, key).Result()
}

// Load HyperLogLog 只保存基数估计，无法枚举元素，因此整键就是一行
func (h *hllHandler) Load(ctx context.Context, cmd redis.Cmdable, q *entity.MemberQuery) (*entity.MemberPage, error) {
	count, err := h.Size(ctx, cmd, q.Key)
	if err != nil {
		return nil, err
	}
	return &entity.MemberPage{
		Total:   count,
		Members: []*entity.Member{{Value: cast.ToString(count)}},
	}, nil
}

func (h *hllHandler) BuildWrite(ctx context.Context, _ redis.Cmdable, w *entity.MemberWrite) ([][]any, error) {
	// HLL 是概率结构：既不能移除单个元素，也没有「改某一行」的语义（唯一一行是统计结果）
	if err := allowOps(w, ViewHyperLogLog, entity.MemberOpCreate); err != nil {
		return nil, err
	}
	values := splitLines(w.Args[argValues])
	if len(values) == 0 {
		return nil, errInvalidArg("no hyperloglog value to add")
	}
	return [][]any{appendAll([]any{"PFADD", w.Key}, values)}, nil
}
