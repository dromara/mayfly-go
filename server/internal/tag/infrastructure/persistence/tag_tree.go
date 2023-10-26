package persistence

import (
	"fmt"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"strings"
)

type tagTreeRepoImpl struct {
	base.RepoImpl[*entity.TagTree]
}

func newTagTreeRepo() repository.TagTree {
	return &tagTreeRepoImpl{base.RepoImpl[*entity.TagTree]{M: new(entity.TagTree)}}
}

func (p *tagTreeRepoImpl) SelectByCondition(condition *entity.TagTreeQuery, toEntity any, orderBy ...string) {
	sql := "SELECT DISTINCT(p.id), p.pid, p.code, p.code_path, p.name, p.remark, p.create_time, p.creator, p.update_time, p.modifier FROM t_tag_tree p WHERE p.is_deleted = 0 "
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
	gormx.GetListBySql2Model(sql, toEntity)
}
