package persistence

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type procdefPolicyHisRepoImpl struct {
	base.RepoImpl[*entity.ProcdefPolicyHis]
}

func newProcdefPolicyHisRepo() repository.ProcdefPolicyHis {
	return &procdefPolicyHisRepoImpl{}
}

// ListByProcdef 按 id 倒序取最近 limit 条变更记录。
//
// 用 id 而不是 create_time 排序：同一秒内的多次保存按时间排会并列，
// 而时间线必须能确定「哪一条在前」，自增主键就是写入顺序
func (r *procdefPolicyHisRepoImpl) ListByProcdef(procdefId uint64, limit int) ([]*entity.ProcdefPolicyHis, error) {
	if limit <= 0 {
		limit = defaultPolicyHistoryLimit
	}
	page, err := r.PageByCond(model.NewCond().Eq("procdef_id", procdefId).OrderBy("id DESC"), model.PageParam{PageNum: 1, PageSize: limit})
	if err != nil {
		return nil, err
	}
	return page.List, nil
}

// defaultPolicyHistoryLimit 未指定条数上限时返回的变更记录数量
const defaultPolicyHistoryLimit = 50
