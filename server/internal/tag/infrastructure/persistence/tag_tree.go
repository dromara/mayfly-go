package persistence

import (
	"fmt"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"strings"
)

type tagTreeRepoImpl struct{}

func newTagTreeRepo() repository.TagTree {
	return new(tagTreeRepoImpl)
}

func (p *tagTreeRepoImpl) SelectByCondition(condition *entity.TagTreeQuery, toEntity interface{}, orderBy ...string) {
	sql := "SELECT DISTINCT(p.id), p.pid, p.code, p.code_path, p.name, p.remark, p.create_time, p.creator, p.update_time, p.modifier FROM t_tag_tree p WHERE 1 = 1 "
	if condition.Name != "" {
		sql = sql + " AND p.name LIKE '%" + condition.Name + "%'"
	}
	if condition.CodePath != "" {
		sql = fmt.Sprintf("%s AND p.code_path = '%s'", sql, condition.CodePath)
	}
	if len(condition.CodePaths) > 0 {
		strCodePaths := make([]string, 0)
		// 将字符串用''包裹
		for _, v := range condition.CodePaths {
			strCodePaths = append(strCodePaths, fmt.Sprintf("'%s'", v))
		}
		sql = fmt.Sprintf("%s AND p.code_path IN (%s)", sql, strings.Join(strCodePaths, ","))
	}
	if condition.CodePathLike != "" {
		sql = fmt.Sprintf("%s AND p.code_path LIKE '%s'", sql, condition.CodePathLike+"%")
	}
	if condition.Pid != 0 {
		sql = fmt.Sprintf("%s AND p.pid = %d ", sql, condition.Pid)
	}
	if len(condition.CodePathLikes) > 0 {
		sql = sql + " AND ("
		for i, v := range condition.CodePathLikes {
			if i == 0 {
				sql = sql + fmt.Sprintf("p.code_path LIKE '%s'", v+"%")
			} else {
				sql = sql + fmt.Sprintf(" OR p.code_path LIKE '%s'", v+"%")
			}
		}
		sql = sql + ")"
	}
	sql = sql + " ORDER BY p.code_path"
	model.GetListBySql2Model(sql, toEntity)
}

func (p *tagTreeRepoImpl) SelectById(id uint64) *entity.TagTree {
	pt := new(entity.TagTree)
	if err := model.GetById(pt, id); err != nil {
		return nil
	}
	return pt
}

func (a *tagTreeRepoImpl) GetBy(condition *entity.TagTree, cols ...string) error {
	return model.GetBy(condition, cols...)
}

func (p *tagTreeRepoImpl) Insert(tagTree *entity.TagTree) {
	biz.ErrIsNil(model.Insert(tagTree), "新增标签失败")
}

func (p *tagTreeRepoImpl) UpdateById(tagTree *entity.TagTree) {
	biz.ErrIsNil(model.UpdateById(tagTree), "更新标签失败")
}

func (p *tagTreeRepoImpl) Delete(id uint64) {
	model.DeleteById(new(entity.TagTree), id)
}
