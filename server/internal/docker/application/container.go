package application

import (
	"context"
	"mayfly-go/internal/docker/application/dto"
	"mayfly-go/internal/docker/dkm"
	"mayfly-go/internal/docker/domain/entity"
	"mayfly-go/internal/docker/domain/repository"
	"mayfly-go/internal/docker/imsg"
	tagapp "mayfly-go/internal/tag/application"
	tagdto "mayfly-go/internal/tag/application/dto"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/stringx"
)

type Container interface {
	base.App[*entity.Container]

	GetContainerPage(condition *entity.ContainerQuery, orderBy ...string) (*model.PageResult[*entity.Container], error)

	// SaveContainer 保存容器配置信息
	SaveContainer(context.Context, *dto.SaveContainer) error

	// DeleteContainer 删除容器配置信息
	DeleteContainer(context.Context, uint64) error

	// GetContaienrCli 获取容器客户端
	GetContainerCli(context.Context, uint64) (*dkm.Client, error)
}

type containerAppImpl struct {
	base.AppImpl[*entity.Container, repository.Container]

	tagApp tagapp.TagTree `inject:"T"`
}

var _ (Container) = (*containerAppImpl)(nil)

func (c *containerAppImpl) GetContainerPage(condition *entity.ContainerQuery, orderBy ...string) (*model.PageResult[*entity.Container], error) {
	return c.Repo.GetContainerPage(condition, orderBy...)
}

func (c *containerAppImpl) SaveContainer(ctx context.Context, saveContainer *dto.SaveContainer) error {
	container := saveContainer.Container
	tagCodePaths := saveContainer.TagCodePaths
	resourceType := tagentity.TagTypeContainer

	oldContainer := &entity.Container{
		Addr: container.Addr,
	}

	err := c.GetByCond(oldContainer)

	if container.Id == 0 {
		if err == nil {
			return errorx.NewBizI(ctx, imsg.ErrContainerConfExist)
		}

		// 生成随机编号
		container.Code = stringx.Rand(10)

		return c.Tx(ctx, func(ctx context.Context) error {
			if err := c.Insert(ctx, container); err != nil {
				return err
			}

			return c.tagApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
				ResourceTag: &tagdto.ResourceTag{
					Code: container.Code,
					Name: container.Name,
					Type: resourceType,
				},
				ParentTagCodePaths: tagCodePaths,
			})
		})
	}

	if err == nil && container.Id != oldContainer.Id {
		return errorx.NewBizI(ctx, imsg.ErrContainerConfExist)
	}
	if oldContainer.Code == "" {
		oldContainer, _ = c.GetById(container.Id)
	}

	dkm.CloseCli(oldContainer.Id)
	return c.Tx(ctx, func(ctx context.Context) error {
		if err := c.UpdateById(ctx, container); err != nil {
			return err
		}

		if oldContainer.Name != container.Name {
			if err := c.tagApp.UpdateTagName(ctx, tagentity.TagTypeMachine, oldContainer.Code, container.Name); err != nil {
				return err
			}
		}

		return c.tagApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
			ResourceTag: &tagdto.ResourceTag{
				Code: oldContainer.Code,
				Name: container.Name,
				Type: resourceType,
			},
			ParentTagCodePaths: tagCodePaths,
		})
	})
}

func (c *containerAppImpl) DeleteContainer(ctx context.Context, id uint64) error {
	container, err := c.GetById(id)
	if err != nil {
		return err
	}

	dkm.CloseCli(id)
	return c.Tx(ctx, func(ctx context.Context) error {
		if err := c.DeleteById(ctx, id); err != nil {
			return err
		}

		return c.tagApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
			ResourceTag: &tagdto.ResourceTag{
				Code: container.Code,
				Type: tagentity.TagTypeContainer,
			},
		})
	})

}

func (c *containerAppImpl) GetContainerCli(ctx context.Context, id uint64) (*dkm.Client, error) {
	return dkm.GetCli(id, func(u uint64) (*dkm.ContainerServer, error) {
		containerConf, err := c.GetById(u)
		if err != nil {
			return nil, err
		}
		return &dkm.ContainerServer{
			Id:   id,
			Addr: containerConf.Addr,
		}, nil
	})
}
