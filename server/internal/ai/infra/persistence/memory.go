package persistence

import (
	"context"
	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/domain/repository"
	"mayfly-go/pkg/base"
	"strings"
)

type memoryRepoImpl struct {
	base.RepoImpl[*entity.Memory]
}

var _ repository.Memory = (*memoryRepoImpl)(nil)

func newMemoryRepo() repository.Memory {
	return &memoryRepoImpl{}
}

func (m *memoryRepoImpl) SelectByUser(ctx context.Context, userId string, keywords []string, tags []string, limit int) ([]*entity.Memory, error) {
	if limit <= 0 {
		limit = 100
	}

	sql := strings.Builder{}
	sql.WriteString("SELECT * FROM t_ai_memory WHERE user_id = ? AND is_deleted = 0")
	args := []any{userId}

	// 关键词模糊匹配（content / tags 任一命中），LIKE 通配符转义防注入误匹配
	for _, kw := range keywords {
		kw = escapeLike(kw)
		if kw == "" {
			continue
		}
		sql.WriteString(" AND (content LIKE ? OR tags LIKE ?)")
		args = append(args, "%"+kw+"%", "%"+kw+"%")
	}

	// 标签过滤：tags 列存 JSON 数组字符串，含任一目标标签即命中
	for _, tag := range tags {
		tag = escapeLike(tag)
		if tag == "" {
			continue
		}
		sql.WriteString(" AND tags LIKE ?")
		args = append(args, "%\""+tag+"\"%")
	}

	sql.WriteString(" ORDER BY update_time DESC LIMIT ?")
	args = append(args, limit)

	var memories []*entity.Memory
	if err := m.SelectBySql(sql.String(), &memories, args...); err != nil {
		return nil, err
	}
	return memories, nil
}

// escapeLike 转义 LIKE 通配符（% 与 _），避免用户输入干扰匹配语义
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	return strings.ReplaceAll(s, "_", "\\_")
}
