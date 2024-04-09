package application

import (
	"context"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/utils/collx"
)

type SaveAuthCertParam struct {
	ResourceCode string
	// 资源标签类型
	ResourceType entity.TagType

	// 授权凭证类型
	AuthCertTagType entity.TagType

	// 空数组则为删除该资源绑定的授权凭证
	AuthCerts []*entity.ResourceAuthCert
}

type ResourceAuthCert interface {
	base.App[*entity.ResourceAuthCert]

	// SaveAuthCert 保存资源授权凭证信息，不可放于事务中
	SaveAuthCert(ctx context.Context, param *SaveAuthCertParam) error

	// GetAuthCert 根据授权凭证名称获取授权凭证
	GetAuthCert(authCertName string) (*entity.ResourceAuthCert, error)

	// GetResourceAuthCert 获取资源授权凭证，默认获取特权账号，若没有则返回第一个
	GetResourceAuthCert(resourceType entity.TagType, resourceCode string) (*entity.ResourceAuthCert, error)

	// GetAccountAuthCert 获取账号有权限操作的授权凭证信息
	GetAccountAuthCert(accountId uint64, authCertTagType entity.TagType, tagPath ...string) []*entity.ResourceAuthCert

	// FillAuthCert 填充资源的授权凭证信息
	// @param resources 实现了entity.IAuthCert接口的资源信息
	FillAuthCert(authCerts []*entity.ResourceAuthCert, resources ...entity.IAuthCert)
}

type resourceAuthCertAppImpl struct {
	base.AppImpl[*entity.ResourceAuthCert, repository.ResourceAuthCert]

	tagTreeApp TagTree `inject:"TagTreeApp"`
}

// 注入Repo
func (r *resourceAuthCertAppImpl) InjectResourceAuthCertRepo(resourceAuthCertRepo repository.ResourceAuthCert) {
	r.Repo = resourceAuthCertRepo
}

func (r *resourceAuthCertAppImpl) SaveAuthCert(ctx context.Context, params *SaveAuthCertParam) error {
	resourceCode := params.ResourceCode
	resourceType := int8(params.ResourceType)
	resourceAuthCerts := params.AuthCerts
	authCertTagType := params.AuthCertTagType

	if authCertTagType == 0 {
		return errorx.NewBiz("资源授权凭证所属标签类型不能为空")
	}

	if resourceCode == "" {
		return errorx.NewBiz("资源授权凭证的资源编号不能为空")
	}

	// 删除授权信息
	if len(resourceAuthCerts) == 0 {
		if err := r.DeleteByCond(ctx, &entity.ResourceAuthCert{ResourceCode: resourceCode, ResourceType: resourceType}); err != nil {
			return err
		}

		// 删除该资源下的所有授权凭证资源标签
		if err := r.tagTreeApp.DeleteResource(ctx, &DelResourceTagParam{
			ResourceCode: resourceCode,
			ResourceType: params.ResourceType,
			ChildType:    authCertTagType,
		}); err != nil {
			return err
		}

		return nil
	}

	name2AuthCert := make(map[string]*entity.ResourceAuthCert, 0)
	for _, resourceAuthCert := range resourceAuthCerts {
		resourceAuthCert.ResourceCode = resourceCode
		resourceAuthCert.ResourceType = int8(resourceType)
		name2AuthCert[resourceAuthCert.Name] = resourceAuthCert

		existNameAc := &entity.ResourceAuthCert{Name: resourceAuthCert.Name}
		if r.GetBy(existNameAc) == nil && existNameAc.ResourceCode != resourceCode {
			return errorx.NewBiz("授权凭证的名称不能重复[%s]", resourceAuthCert.Name)
		}

		// 公共授权凭证，则无需进行密文加密，密文即为公共授权凭证名
		if resourceAuthCert.CiphertextType == entity.AuthCertCiphertextTypePublic {
			continue
		}

		// 密文加密
		if err := resourceAuthCert.CiphertextEncrypt(); err != nil {
			return errorx.NewBiz(err.Error())
		}
	}

	var oldAuthCert []*entity.ResourceAuthCert
	r.ListByCond(&entity.ResourceAuthCert{ResourceCode: resourceCode, ResourceType: resourceType}, &oldAuthCert)

	var adds, dels, unmodifys []string
	if len(oldAuthCert) == 0 {
		adds = collx.MapKeys(name2AuthCert)
	} else {
		oldNames := collx.ArrayMap(oldAuthCert, func(ac *entity.ResourceAuthCert) string {
			return ac.Name
		})
		adds, dels, unmodifys = collx.ArrayCompare[string](collx.MapKeys(name2AuthCert), oldNames)
	}

	addAuthCerts := make([]*entity.ResourceAuthCert, 0)
	for _, add := range adds {
		addAc := name2AuthCert[add]
		addAc.Id = 0
		addAuthCerts = append(addAuthCerts, addAc)
	}

	// 处理新增的授权凭证
	if len(addAuthCerts) > 0 {
		if err := r.BatchInsert(ctx, addAuthCerts); err != nil {
			return err
		}

		// 获取资源编号对应的资源标签信息
		var resourceTags []*entity.TagTree
		r.tagTreeApp.ListByCond(&entity.TagTree{Type: params.ResourceType, Code: resourceCode}, &resourceTags)
		// 资源标签id（相当于父tag id）
		resourceTagIds := collx.ArrayMap(resourceTags, func(tag *entity.TagTree) uint64 {
			return tag.Id
		})

		// 保存授权凭证类型的资源标签
		for _, authCert := range addAuthCerts {
			if err := r.tagTreeApp.SaveResource(ctx, &SaveResourceTagParam{
				ResourceCode: authCert.Name,
				ResourceType: authCertTagType,
				ResourceName: authCert.Username,
				TagIds:       resourceTagIds,
			}); err != nil {
				return err
			}
		}
	}

	for _, del := range dels {
		if err := r.DeleteByCond(ctx, &entity.ResourceAuthCert{ResourceCode: resourceCode, ResourceType: resourceType, Name: del}); err != nil {
			return err
		}
		// 删除对应授权凭证资源标签
		if err := r.tagTreeApp.DeleteResource(ctx, &DelResourceTagParam{
			ResourceCode: del,
			ResourceType: authCertTagType,
		}); err != nil {
			return err
		}
	}

	for _, unmodify := range unmodifys {
		unmodifyAc := name2AuthCert[unmodify]
		if unmodifyAc.Id == 0 {
			continue
		}
		if err := r.UpdateById(ctx, unmodifyAc); err != nil {
			return err
		}
	}

	return nil
}

func (r *resourceAuthCertAppImpl) GetAuthCert(authCertName string) (*entity.ResourceAuthCert, error) {
	authCert := &entity.ResourceAuthCert{Name: authCertName}
	if err := r.GetBy(authCert); err != nil {
		return nil, errorx.NewBiz("该授权凭证不存在")
	}

	return r.decryptAuthCert(authCert)
}

func (r *resourceAuthCertAppImpl) GetResourceAuthCert(resourceType entity.TagType, resourceCode string) (*entity.ResourceAuthCert, error) {
	var resourceAuthCerts []*entity.ResourceAuthCert
	if err := r.ListByCond(&entity.ResourceAuthCert{
		ResourceType: int8(resourceType),
		ResourceCode: resourceCode,
	}, &resourceAuthCerts); err != nil {
		return nil, err
	}

	if len(resourceAuthCerts) == 0 {
		return nil, errorx.NewBiz("该资源不存在授权凭证账号")
	}

	for _, resourceAuthCert := range resourceAuthCerts {
		if resourceAuthCert.Type == entity.AuthCertTypePrivileged {
			return r.decryptAuthCert(resourceAuthCert)
		}
	}

	return r.decryptAuthCert(resourceAuthCerts[0])
}

func (r *resourceAuthCertAppImpl) GetAccountAuthCert(accountId uint64, authCertTagType entity.TagType, tagPath ...string) []*entity.ResourceAuthCert {
	// 获取用户有权限操作的授权凭证资源标签
	tagQuery := &entity.TagTreeQuery{
		Type:          authCertTagType,
		CodePathLikes: tagPath,
	}
	authCertTags := r.tagTreeApp.GetAccountTagResources(accountId, tagQuery)

	// 获取所有授权凭证名称
	authCertNames := collx.ArrayMap(authCertTags, func(tag *entity.TagTree) string {
		return tag.Code
	})

	var authCerts []*entity.ResourceAuthCert
	r.GetRepo().ListByWheres(collx.M{
		"name in ?": collx.ArrayDeduplicate(authCertNames),
	}, &authCerts)

	return authCerts
}

func (r *resourceAuthCertAppImpl) FillAuthCert(authCerts []*entity.ResourceAuthCert, resources ...entity.IAuthCert) {
	if len(resources) == 0 {
		return
	}

	// 资源编号 -> 资源
	resourceCode2Resouce := collx.ArrayToMap(resources, func(ac entity.IAuthCert) string {
		return ac.GetCode()
	})

	for _, authCert := range authCerts {
		resourceCode2Resouce[authCert.ResourceCode].SetAuthCert(entity.AuthCert{
			Name:           authCert.Name,
			Username:       authCert.Username,
			Type:           authCert.Type,
			CiphertextType: authCert.CiphertextType,
		})
	}
}

// 解密授权凭证信息
func (r *resourceAuthCertAppImpl) decryptAuthCert(authCert *entity.ResourceAuthCert) (*entity.ResourceAuthCert, error) {
	if authCert.CiphertextType == entity.AuthCertCiphertextTypePublic {
		// 如果是公共授权凭证，则密文为公共授权凭证名称，需要使用该名称再去获取对应的授权凭证
		authCert = &entity.ResourceAuthCert{Name: authCert.Ciphertext}
		if err := r.GetBy(authCert); err != nil {
			return nil, errorx.NewBiz("该公共授权凭证[%s]不存在", authCert.Ciphertext)
		}
	}

	if err := authCert.CiphertextDecrypt(); err != nil {
		return nil, err
	}
	return authCert, nil
}
