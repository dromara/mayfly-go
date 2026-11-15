package sqlexec

import (
	"context"
	"strings"

	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type DbSQL interface {
	base.App[*entity.DbSQL]

	// Examples 同库已保存脚本（最多 limit 条，最新的在前）。
	//
	// 供 SQL 草稿作为「本库既有写法」示例：这是平台自有语料，不需要引入外部样本，
	// 也让生成风格与用户自己的脚本保持一致。只回语句文本，不回名称等元信息。
	Examples(ctx context.Context, dbId uint64, limit int) ([]string, error)
}

type DbSQLAppImpl struct {
	base.AppImpl[*entity.DbSQL, repository.DbSQL]
}

var _ DbSQL = (*DbSQLAppImpl)(nil)

// Examples 取同库脚本（按 id 倒序截断）。
//
// 不用 SQL 层 LIMIT 而取全量后截断：脚本表是用户手工维护的小表，且 pkg/model 的
// 查询条件没有 LIMIT 能力（引入它会牵动基座）。若将来脚本量级变大，应在仓储层加
// 分页入口，而不是在这里自己拼 SQL。
func (d *DbSQLAppImpl) Examples(ctx context.Context, dbId uint64, limit int) ([]string, error) {
	if dbId == 0 || limit <= 0 {
		return nil, nil
	}
	list, err := d.ListByCond(model.NewCond().Eq("db_id", dbId).OrderByDesc("id"), "id", "sql")
	if err != nil {
		return nil, err
	}
	examples := make([]string, 0, limit)
	for _, item := range list {
		if item == nil {
			continue
		}
		text := strings.TrimSpace(item.SQL)
		if text == "" {
			continue
		}
		examples = append(examples, text)
		if len(examples) >= limit {
			break
		}
	}
	return examples, nil
}
