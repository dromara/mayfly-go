package application

import (
	"context"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"

	"gorm.io/gorm"
)

type SaveTeamParam struct {
	Id     uint64 `json:"id"`
	Name   string `json:"name" binding:"required"` // 名称
	Remark string `json:"remark"`                  // 备注说明

	Tags []uint64 `json:"tags"` // 关联标签信息
}

type Team interface {
	// 分页获取项目团队信息列表
	GetPageList(condition *entity.TeamQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	// Save 保存团队信息
	Save(ctx context.Context, team *SaveTeamParam) error

	Delete(ctx context.Context, id uint64) error

	//--------------- 团队成员相关接口 ---------------

	GetMemberPage(condition *entity.TeamMember, pageParam *model.PageParam, toEntity any) (*model.PageResult[any], error)

	SaveMember(ctx context.Context, tagTeamMember *entity.TeamMember)

	DeleteMember(tx context.Context, teamId, accountId uint64)

	IsExistMember(teamId, accounId uint64) bool

	//--------------- 关联项目相关接口 ---------------

	ListTagIds(teamId uint64) []uint64

	DeleteTag(tx context.Context, teamId, tagId uint64) error
}

type teamAppImpl struct {
	teamRepo        repository.Team        `inject:"TeamRepo"`
	teamMemberRepo  repository.TeamMember  `inject:"TeamMemberRepo"`
	tagTreeTeamRepo repository.TagTreeTeam `inject:"TagTreeTeamRepo"`
}

func (p *teamAppImpl) GetPageList(condition *entity.TeamQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return p.teamRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

func (p *teamAppImpl) Save(ctx context.Context, saveParam *SaveTeamParam) error {
	team := &entity.Team{Name: saveParam.Name, Remark: saveParam.Remark}
	team.Id = saveParam.Id

	if team.Id == 0 {
		if p.teamRepo.CountByCond(&entity.Team{Name: saveParam.Name}) > 0 {
			return errorx.NewBiz("团队名[%s]已存在", saveParam.Name)
		}

		if err := p.teamRepo.Insert(ctx, team); err != nil {
			return err
		}

		loginAccount := contextx.GetLoginAccount(ctx)
		logx.DebugfContext(ctx, "将[%s]默认加入至[%s]团队", loginAccount.Username, team.Name)

		teamMem := &entity.TeamMember{}
		teamMem.AccountId = loginAccount.Id
		teamMem.Username = loginAccount.Username
		teamMem.TeamId = team.Id
		p.SaveMember(ctx, teamMem)
	} else {
		// 置空名称，防止变更
		team.Name = ""
		if err := p.teamRepo.UpdateById(ctx, team); err != nil {
			return err
		}
	}

	// 保存团队关联的标签信息
	teamId := team.Id
	var addIds, delIds []uint64
	if saveParam.Id == 0 {
		addIds = saveParam.Tags
	} else {
		// 将[]uint64转为[]any
		oIds := p.ListTagIds(team.Id)
		// 比较新旧两合集
		addIds, delIds, _ = collx.ArrayCompare(saveParam.Tags, oIds)
	}

	addTeamTags := make([]*entity.TagTreeTeam, 0)
	for _, v := range addIds {
		ptt := &entity.TagTreeTeam{TeamId: teamId, TagId: v}
		addTeamTags = append(addTeamTags, ptt)
	}
	if len(addTeamTags) > 0 {
		logx.DebugfContext(ctx, "团队[%s]新增关联的标签信息: [%v]", team.Name, addTeamTags)
		p.tagTreeTeamRepo.BatchInsert(ctx, addTeamTags)
	}

	for _, v := range delIds {
		p.DeleteTag(ctx, teamId, v)
	}
	if len(delIds) > 0 {
		logx.DebugfContext(ctx, "团队[%s]删除关联的标签信息: [%v]", team.Name, delIds)
	}

	return nil
}

func (p *teamAppImpl) Delete(ctx context.Context, id uint64) error {
	return gormx.Tx(
		func(db *gorm.DB) error {
			return p.teamRepo.DeleteByIdWithDb(ctx, db, id)
		},
		func(db *gorm.DB) error {
			return p.teamMemberRepo.DeleteByCondWithDb(ctx, db, &entity.TeamMember{TeamId: id})
		},
		func(db *gorm.DB) error {
			return p.tagTreeTeamRepo.DeleteByCondWithDb(ctx, db, &entity.TagTreeTeam{TeamId: id})
		},
	)
}

// --------------- 团队成员相关接口 ---------------

func (p *teamAppImpl) GetMemberPage(condition *entity.TeamMember, pageParam *model.PageParam, toEntity any) (*model.PageResult[any], error) {
	return p.teamMemberRepo.GetPageList(condition, pageParam, toEntity)
}

// 保存团队成员信息
func (p *teamAppImpl) SaveMember(ctx context.Context, teamMember *entity.TeamMember) {
	teamMember.Id = 0
	biz.IsTrue(!p.teamMemberRepo.IsExist(teamMember.TeamId, teamMember.AccountId), "该成员已存在")
	p.teamMemberRepo.Insert(ctx, teamMember)
}

// 删除团队成员信息
func (p *teamAppImpl) DeleteMember(ctx context.Context, teamId, accountId uint64) {
	p.teamMemberRepo.DeleteByCond(ctx, &entity.TeamMember{TeamId: teamId, AccountId: accountId})
}

func (p *teamAppImpl) IsExistMember(teamId, accounId uint64) bool {
	return p.teamMemberRepo.IsExist(teamId, accounId)
}

//--------------- 标签相关接口 ---------------

func (p *teamAppImpl) ListTagIds(teamId uint64) []uint64 {
	tags := &[]entity.TagTreeTeam{}
	p.tagTreeTeamRepo.ListByCondOrder(&entity.TagTreeTeam{TeamId: teamId}, tags)
	ids := make([]uint64, 0)
	for _, v := range *tags {
		ids = append(ids, v.TagId)
	}
	return ids
}

// 删除关联项目信息
func (p *teamAppImpl) DeleteTag(ctx context.Context, teamId, tagId uint64) error {
	return p.tagTreeTeamRepo.DeleteByCond(ctx, &entity.TagTreeTeam{TeamId: teamId, TagId: tagId})
}
