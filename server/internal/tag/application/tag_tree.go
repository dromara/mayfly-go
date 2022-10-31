package application

import (
	"fmt"
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
	"mayfly-go/pkg/biz"
	"strings"
)

type TagTree interface {
	ListByQuery(condition *entity.TagTreeQuery, toEntity interface{})

	GetById(id uint64) *entity.TagTree

	Save(project *entity.TagTree)

	Delete(id uint64)

	// 获取账号id拥有的可访问的标签id
	ListTagIdByAccountId(accountId uint64) []uint64

	// 获取以指定tagPath数组开头的所有标签id
	ListTagIdByPath(tagPath ...string) []uint64

	// 根据tagPath获取自身及其所有子标签信息
	ListTagByPath(tagPath ...string) []entity.TagTree

	// 根据账号id获取其可访问标签信息
	ListTagByAccountId(accountId uint64) []string

	// 账号是否有权限访问该标签关联的资源信息
	CanAccess(accountId uint64, tagPath string) error
}

func newTagTreeApp(tagTreeRepo repository.TagTree,
	tagTreeTeamRepo repository.TagTreeTeam,
	machineApp machineapp.Machine,
	redisApp redisapp.Redis,
	dbApp dbapp.Db,
	mongoApp mongoapp.Mongo) TagTree {
	return &tagTreeAppImpl{
		tagTreeRepo:     tagTreeRepo,
		tagTreeTeamRepo: tagTreeTeamRepo,
		machineApp:      machineApp,
		redisApp:        redisApp,
		dbApp:           dbApp,
		mongoApp:        mongoApp,
	}
}

type tagTreeAppImpl struct {
	tagTreeRepo     repository.TagTree
	tagTreeTeamRepo repository.TagTreeTeam
	machineApp      machineapp.Machine
	redisApp        redisapp.Redis
	mongoApp        mongoapp.Mongo
	dbApp           dbapp.Db
}

func (p *tagTreeAppImpl) Save(tag *entity.TagTree) {
	// 新建项目树节点信息
	if tag.Id == 0 {
		biz.IsTrue(!strings.Contains(tag.Code, entity.CodePathSeparator), "标识符不能包含'/'")
		if tag.Pid != 0 {
			parentTag := p.tagTreeRepo.SelectById(tag.Pid)
			biz.NotNil(parentTag, "父节点不存在")
			tag.CodePath = parentTag.CodePath + entity.CodePathSeparator + tag.Code
		} else {
			tag.CodePath = tag.Code
		}
		// 判断该路径是否存在
		var hasLikeTags []entity.TagTree
		p.tagTreeRepo.SelectByCondition(&entity.TagTreeQuery{CodePathLike: tag.CodePath}, &hasLikeTags)
		biz.IsTrue(len(hasLikeTags) == 0, "已存在该标签路径开头的标签, 请修改该标识code")

		// 校验同级标签，是否有以该code为开头的标识符
		p.tagTreeRepo.SelectByCondition(&entity.TagTreeQuery{Pid: tag.Pid}, &hasLikeTags)
		for _, v := range hasLikeTags {
			if strings.HasPrefix(tag.Code, v.Code) {
				panic(biz.NewBizErr(fmt.Sprintf("同级标签下的[%s]与[%s]存在相似开头字符, 请修改该标识code", v.Code, tag.Code)))
			}
		}
		p.tagTreeRepo.Insert(tag)
		return
	}

	// 防止误传导致被更新
	tag.Code = ""
	tag.CodePath = ""
	p.tagTreeRepo.UpdateById(tag)
}

func (p *tagTreeAppImpl) ListByQuery(condition *entity.TagTreeQuery, toEntity interface{}) {
	p.tagTreeRepo.SelectByCondition(condition, toEntity)
}

func (p *tagTreeAppImpl) GetById(tagId uint64) *entity.TagTree {
	return p.tagTreeRepo.SelectById(tagId)
}

func (p *tagTreeAppImpl) ListTagIdByAccountId(accountId uint64) []uint64 {
	// 获取该账号可操作的标签路径
	return p.ListTagIdByPath(p.ListTagByAccountId(accountId)...)
}

func (p *tagTreeAppImpl) ListTagByPath(tagPaths ...string) []entity.TagTree {
	var tags []entity.TagTree
	p.tagTreeRepo.SelectByCondition(&entity.TagTreeQuery{CodePathLikes: tagPaths}, &tags)
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

func (p *tagTreeAppImpl) CanAccess(accountId uint64, tagPath string) error {
	tagPaths := p.ListTagByAccountId(accountId)
	// 判断该资源标签是否为该账号拥有的标签或其子标签
	for _, v := range tagPaths {
		if strings.HasPrefix(tagPath, v) {
			return nil
		}
	}

	return biz.NewBizErr("您无权操作该资源")
}

func (p *tagTreeAppImpl) Delete(id uint64) {
	biz.IsTrue(p.machineApp.Count(&machineentity.MachineQuery{TagId: id}) == 0, "请先删除该项目关联的机器信息")
	biz.IsTrue(p.redisApp.Count(&redisentity.RedisQuery{TagId: id}) == 0, "请先删除该项目关联的redis信息")
	biz.IsTrue(p.dbApp.Count(&dbentity.DbQuery{TagId: id}) == 0, "请先删除该项目关联的数据库信息")
	biz.IsTrue(p.mongoApp.Count(&mongoentity.MongoQuery{TagId: id}) == 0, "请先删除该项目关联的Mongo信息")
	p.tagTreeRepo.Delete(id)
	// 删除该标签关联的团队信息
	p.tagTreeTeamRepo.DeleteBy(&entity.TagTreeTeam{TagId: id})
}
