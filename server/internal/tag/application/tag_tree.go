package application

import (
	"context"
	dbapp "mayfly-go/internal/db/application"
	dbentity "mayfly-go/internal/db/domain/entity"
	machineapp "mayfly-go/internal/machine/application"
	machineentity "mayfly-go/internal/machine/domain/entity"
	mongoapp "mayfly-go/internal/mongo/application"
	mongoentity "mayfly-go/internal/mongo/domain/entity"
	redisapp "mayfly-go/internal/redis/application"
	redisentity "mayfly-go/internal/redis/domain/entity"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/gormx"
	"strings"
)

type TagTree interface {
	base.App[*entity.TagTree]

	ListByQuery(condition *entity.TagTreeQuery, toEntity any)

	Save(ctx context.Context, tt *entity.TagTree) error

	Delete(ctx context.Context, id uint64) error

	// 获取账号id拥有的可访问的标签id
	ListTagIdByAccountId(accountId uint64) []uint64

	// 获取以指定tagPath数组开头的所有标签id
	ListTagIdByPath(tagPath ...string) []uint64

	// 根据tagPath获取自身及其所有子标签信息
	ListTagByPath(tagPath ...string) []entity.TagTree

	// 根据账号id获取其可访问标签信息
	ListTagByAccountId(accountId uint64) []string

	// 查询账号id可访问的资源相关联的标签信息
	// @param model对应资源的实体信息，如Machinie、Db等等
	ListTagByAccountIdAndResource(accountId uint64, model any) []string

	// 账号是否有权限访问该标签关联的资源信息
	CanAccess(accountId uint64, tagPath string) error
}

func newTagTreeApp(tagTreeRepo repository.TagTree,
	tagTreeTeamRepo repository.TagTreeTeam,
	machineApp machineapp.Machine,
	redisApp redisapp.Redis,
	dbApp dbapp.Db,
	mongoApp mongoapp.Mongo) TagTree {
	tagTreeApp := &tagTreeAppImpl{
		tagTreeTeamRepo: tagTreeTeamRepo,
		machineApp:      machineApp,
		redisApp:        redisApp,
		dbApp:           dbApp,
		mongoApp:        mongoApp,
	}
	tagTreeApp.Repo = tagTreeRepo
	return tagTreeApp
}

type tagTreeAppImpl struct {
	base.AppImpl[*entity.TagTree, repository.TagTree]

	tagTreeTeamRepo repository.TagTreeTeam
	machineApp      machineapp.Machine
	redisApp        redisapp.Redis
	mongoApp        mongoapp.Mongo
	dbApp           dbapp.Db
}

func (p *tagTreeAppImpl) Save(ctx context.Context, tag *entity.TagTree) error {
	// 新建项目树节点信息
	if tag.Id == 0 {
		if strings.Contains(tag.Code, entity.CodePathSeparator) {
			return errorx.NewBiz("标识符不能包含'/'")
		}
		if tag.Pid != 0 {
			parentTag, err := p.GetById(new(entity.TagTree), tag.Pid)
			if err != nil {
				return errorx.NewBiz("父节点不存在")
			}
			tag.CodePath = parentTag.CodePath + tag.Code + entity.CodePathSeparator
		} else {
			tag.CodePath = tag.Code + entity.CodePathSeparator
		}
		// 判断该路径是否存在
		var hasLikeTags []entity.TagTree
		p.GetRepo().SelectByCondition(&entity.TagTreeQuery{CodePathLike: tag.CodePath}, &hasLikeTags)
		if len(hasLikeTags) > 0 {
			return errorx.NewBiz("已存在该标签路径开头的标签, 请修改该标识code")
		}

		return p.Insert(ctx, tag)
	}

	// 防止误传导致被更新
	tag.Code = ""
	tag.CodePath = ""
	return p.UpdateById(ctx, tag)
}

func (p *tagTreeAppImpl) ListByQuery(condition *entity.TagTreeQuery, toEntity any) {
	p.GetRepo().SelectByCondition(condition, toEntity)
}

func (p *tagTreeAppImpl) ListTagIdByAccountId(accountId uint64) []uint64 {
	// 获取该账号可操作的标签路径
	return p.ListTagIdByPath(p.ListTagByAccountId(accountId)...)
}

func (p *tagTreeAppImpl) ListTagByPath(tagPaths ...string) []entity.TagTree {
	var tags []entity.TagTree
	p.GetRepo().SelectByCondition(&entity.TagTreeQuery{CodePathLikes: tagPaths}, &tags)
	return tags
}

func (p *tagTreeAppImpl) ListTagIdByPath(tagPaths ...string) []uint64 {
	tagIds := make([]uint64, 0)
	if len(tagPaths) == 0 {
		return tagIds
	}

	tags := p.ListTagByPath(tagPaths...)
	for _, v := range tags {
		tagIds = append(tagIds, v.Id)
	}
	return tagIds
}

func (p *tagTreeAppImpl) ListTagByAccountId(accountId uint64) []string {
	return p.tagTreeTeamRepo.SelectTagPathsByAccountId(accountId)
}

func (p *tagTreeAppImpl) ListTagByAccountIdAndResource(accountId uint64, entity any) []string {
	var res []string

	tagIds := p.ListTagIdByAccountId(accountId)
	if len(tagIds) == 0 {
		return res
	}

	global.Db.Model(entity).Distinct("tag_path").Where("tag_id in ?", tagIds).Scopes(gormx.UndeleteScope).Order("tag_path asc").Find(&res)
	return res
}

func (p *tagTreeAppImpl) CanAccess(accountId uint64, tagPath string) error {
	tagPaths := p.ListTagByAccountId(accountId)
	// 判断该资源标签是否为该账号拥有的标签或其子标签
	for _, v := range tagPaths {
		if strings.HasPrefix(tagPath, v) {
			return nil
		}
	}

	return errorx.NewBiz("您无权操作该资源")
}

func (p *tagTreeAppImpl) Delete(ctx context.Context, id uint64) error {
	tagIds := [1]uint64{id}
	if p.machineApp.Count(&machineentity.MachineQuery{TagIds: tagIds[:]}) > 0 {
		return errorx.NewBiz("请先删除该标签关联的机器信息")
	}
	if p.redisApp.Count(&redisentity.RedisQuery{TagIds: tagIds[:]}) > 0 {
		return errorx.NewBiz("请先删除该标签关联的redis信息")
	}
	if p.dbApp.Count(&dbentity.DbQuery{TagIds: tagIds[:]}) > 0 {
		return errorx.NewBiz("请先删除该标签关联的数据库信息")
	}
	if p.mongoApp.Count(&mongoentity.MongoQuery{TagIds: tagIds[:]}) > 0 {
		return errorx.NewBiz("请先删除该标签关联的Mongo信息")
	}

	p.DeleteById(ctx, id)
	// 删除该标签关联的团队信息
	return p.tagTreeTeamRepo.DeleteByCond(ctx, &entity.TagTreeTeam{TagId: id})
}
