package application

import (
	"context"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
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

	// SavePublicAuthCert 保存公共授权凭证信息
	SavePulbicAuthCert(ctx context.Context, rac *entity.ResourceAuthCert) error

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
		logx.DebugfContext(ctx, "SaveAuthCert[%d-%s]-删除所有关联的授权凭证信息", resourceType, resourceCode)
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
		logx.DebugfContext(ctx, "SaveAuthCert[%d-%s]-不存在已有的授权凭证信息, 为新增资源授权凭证", resourceType, resourceCode)
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
		logx.DebugfContext(ctx, "SaveAuthCert[%d-%s]-新增授权凭证-[%v]", resourceType, resourceCode, adds)
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

		if len(resourceTagIds) > 0 {
			// 保存授权凭证类型的资源标签
			for _, authCert := range addAuthCerts {
				logx.DebugfContext(ctx, "SaveAuthCert[%d-%s]-授权凭证标签[%d-%s]关联至所属资源标签下[%v]", resourceType, resourceCode, authCertTagType, authCert.Name, resourceTagIds)
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
	}

	for _, del := range dels {
		logx.DebugfContext(ctx, "SaveAuthCert[%d-%s]-删除授权凭证-[%v]", resourceType, resourceCode, del)
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
		logx.DebugfContext(ctx, "SaveAuthCert[%d-%s]-更新授权凭证-[%v]", resourceType, resourceCode, unmodify)
		// 置空用户名，不允许修改（TagTree的name关联了该值，修改了会造成数据不一致，懒得处理该同步。要修改用户名可通过重新删除旧凭证后添加一个授权凭证或通过关联公共凭证（公共凭证可修改用户名））
		if unmodifyAc.CiphertextType != entity.AuthCertCiphertextTypePublic {
			unmodifyAc.Username = ""
		}

		if err := r.UpdateById(ctx, unmodifyAc); err != nil {
			return err
		}
	}

	return nil
}

func (r *resourceAuthCertAppImpl) SavePulbicAuthCert(ctx context.Context, rac *entity.ResourceAuthCert) error {
	rac.Type = entity.AuthCertTypePublic
	rac.CiphertextEncrypt()
	rac.ResourceType = 0
	rac.ResourceCode = "-"
	if rac.Id == 0 {
		if r.CountByCond(&entity.ResourceAuthCert{Name: rac.Name}) > 0 {
			return errorx.NewBiz("授权凭证的名称不能重复[%s]", rac.Name)
		}
		return r.Insert(ctx, rac)
	}
	// 名称置空，防止被更新
	rac.Name = ""
	return r.UpdateById(ctx, rac)
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
	if len(resources) == 0 || len(authCerts) == 0 {
		return
	}

	// 资源编号 -> 资源
	resourceCode2Resource := collx.ArrayToMap(resources, func(ac entity.IAuthCert) string {
		return ac.GetCode()
	})

	for _, authCert := range authCerts {
		resource := resourceCode2Resource[authCert.ResourceCode]
		if resource == nil {
			logx.Debugf("FillAuthCert-授权凭证[%s]未匹配到对应的资源[%s]", authCert.Name, authCert.ResourceCode)
			continue
		}

		resource.SetAuthCert(entity.AuthCert{
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
		// 需要维持资源关联信息
		resourceCode := authCert.ResourceCode
		resourceType := authCert.ResourceType
		authCertType := authCert.Type

		// 如果是公共授权凭证，则密文为公共授权凭证名称，需要使用该名称再去获取对应的授权凭证
		authCert = &entity.ResourceAuthCert{Name: authCert.Ciphertext}
		if err := r.GetBy(authCert); err != nil {
			return nil, errorx.NewBiz("该公共授权凭证[%s]不存在", authCert.Ciphertext)
		}

		// 使用资源关联的凭证类型
		authCert.ResourceCode = resourceCode
		authCert.ResourceType = resourceType
		authCert.Type = authCertType
	}

	if err := authCert.CiphertextDecrypt(); err != nil {
		return nil, err
	}
	return authCert, nil
}
