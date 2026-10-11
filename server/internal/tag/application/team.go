package application

import (
	"context"
	"mayfly-go/internal/tag/application/dto"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/internal/tag/imsg"
	"mayfly-go/internal/tag/infra/cache"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
)

type Team interface {
	base.App[*entity.Team]

	// 分页获取项目团队信息列表
	GetPageList(condition *entity.TeamQuery, orderBy ...string) (*model.PageResult[*entity.Team], error)

	// SaveTeam 保存团队信息
	SaveTeam(ctx context.Context, team *dto.SaveTeam) error

	Delete(ctx context.Context, id uint64) error

	//--------------- 团队成员相关接口 ---------------

	GetMemberPage(condition *entity.TeamMember, pageParam model.PageParam) (*model.PageResult[*entity.TeamMemberPO], error)

	SaveMember(ctx context.Context, tagTeamMember *entity.TeamMember) error

	DeleteMember(ctx context.Context, teamId, accountId uint64) error

	IsExistMember(ctx context.Context, teamId, accounId uint64) (bool, error)

	DeleteTag(tx context.Context, teamId, tagId uint64) error
}

type teamAppImpl struct {
	base.AppImpl[*entity.Team, repository.Team]

	teamMemberRepo   repository.TeamMember `inject:"T"`
	tagTreeRelateApp TagTreeRelate         `inject:"T"`
}

var _ Team = (*teamAppImpl)(nil)

var _ (Team) = (*teamAppImpl)(nil)

func (p *teamAppImpl) GetPageList(condition *entity.TeamQuery, orderBy ...string) (*model.PageResult[*entity.Team], error) {
	return p.GetRepo().GetPageList(condition, orderBy...)
}

func (p *teamAppImpl) SaveTeam(ctx context.Context, saveParam *dto.SaveTeam) error {
	team := &entity.Team{
		Name:              saveParam.Name,
		ValidityStartDate: &saveParam.ValidityStartDate.Time,
		ValidityEndDate:   &saveParam.ValidityEndDate.Time,
		Remark:            saveParam.Remark,
	}
	team.Id = saveParam.Id

	return p.Tx(ctx, func(ctx context.Context) error {
		if team.Id == 0 {
			// 存在性判断走带 ctx 的 Exists：它复用本事务连接，能看见本事务尚未提交的写入。
			// 原先用无 ctx 的 CountByCond，等于在事务外查重——两个并发同名创建都会通过校验
			exists, err := p.Exists(ctx, &entity.Team{Name: saveParam.Name})
			if err != nil {
				return err
			}
			if exists {
				return errorx.NewBizI(ctx, imsg.ErrNameExist)
			}

			if err := p.Insert(ctx, team); err != nil {
				return err
			}

			loginAccount := contextx.GetLoginAccount(ctx)
			logx.InfoContext(ctx, "Add [%s] to the [%s] team by default", loginAccount.Username, team.Name)

			teamMem := &entity.TeamMember{}
			teamMem.AccountId = loginAccount.Id
			teamMem.Username = loginAccount.Username
			teamMem.TeamId = team.Id
			if err := p.SaveMember(ctx, teamMem); err != nil {
				return err
			}
		} else {
			// 置空名称，防止变更
			team.Name = ""
			if err := p.UpdateById(ctx, team); err != nil {
				return err
			}
		}

		// 保存团队关联的标签信息
		if err := p.tagTreeRelateApp.RelateTag(ctx, entity.TagRelateTypeTeam, team.Id, saveParam.CodePaths...); err != nil {
			return err
		}

		// 团队关联账号的标签缓存必须等事务提交后再清，三个理由各自独立：
		//   - 基座读入口还不接 ctx，事务内的读走的是连接池而不是事务那条连接：既读不到本事务
		//     刚插入的成员，也会在事务占满连接池时等不到空闲连接而长时间不返回；
		//   - 本次事务里刚加进来的成员在提交前读不到，于是新成员的标签缓存每次都被漏清；
		//   - 缓存失效属于「提交后才该生效」的副作用，事务中途回滚却已清过缓存会留下脏读窗口。
		// 错误也不再吞掉：清理失败意味着账号继续看到旧标签，必须留下痕迹
		return base.AfterCommit(ctx, func(runCtx context.Context) error {
			teamMembers, err := p.teamMemberRepo.SelectByCond(&entity.TeamMember{TeamId: team.Id})
			if err != nil {
				// 只记日志不把错误回报给调用方：团队本身已提交成功，而缓存陈旧只会让成员短暂看到
				// 旧标签，下一次加载即可纠正。把它做成接口失败会让用户以为没保存而重复提交
				logx.ErrorfContext(runCtx, "failed to list team members for tag cache invalidation, team %d: %v", team.Id, err)
				return nil
			}
			for _, tm := range teamMembers {
				cache.DelAccountTagPaths(tm.AccountId)
			}
			return nil
		})
	})
}

func (p *teamAppImpl) Delete(ctx context.Context, id uint64) error {
	return p.Tx(ctx, func(ctx context.Context) error {
		return p.DeleteById(ctx, id)
	}, func(ctx context.Context) error {
		return p.teamMemberRepo.DeleteByCond(ctx, &entity.TeamMember{TeamId: id})
	}, func(ctx context.Context) error {
		return p.tagTreeRelateApp.DeleteByCond(ctx, &entity.TagTreeRelate{RelateType: entity.TagRelateTypeTeam, RelateId: id})
	})
}

// --------------- 团队成员相关接口 ---------------

func (p *teamAppImpl) GetMemberPage(condition *entity.TeamMember, pageParam model.PageParam) (*model.PageResult[*entity.TeamMemberPO], error) {
	return p.teamMemberRepo.GetPageList(condition, pageParam)
}

// 保存团队成员信息
func (p *teamAppImpl) SaveMember(ctx context.Context, teamMember *entity.TeamMember) error {
	teamMember.Id = 0
	exist, err := p.teamMemberRepo.IsExist(ctx, teamMember.TeamId, teamMember.AccountId)
	if err != nil {
		return err
	}
	if exist {
		return errorx.NewBizI(ctx, imsg.ErrMemeberExist)
	}
	return p.teamMemberRepo.Insert(ctx, teamMember)
}

// 删除团队成员信息
func (p *teamAppImpl) DeleteMember(ctx context.Context, teamId, accountId uint64) error {
	return p.teamMemberRepo.DeleteByCond(ctx, &entity.TeamMember{TeamId: teamId, AccountId: accountId})
}

func (p *teamAppImpl) IsExistMember(ctx context.Context, teamId, accounId uint64) (bool, error) {
	return p.teamMemberRepo.IsExist(ctx, teamId, accounId)
}

//--------------- 标签相关接口 ---------------

// 删除关联标签信息
func (p *teamAppImpl) DeleteTag(ctx context.Context, teamId, tagId uint64) error {
	return p.tagTreeRelateApp.DeleteByCond(ctx, &entity.TagTreeRelate{RelateType: entity.TagRelateTypeTeam, RelateId: teamId, TagId: tagId})
}
