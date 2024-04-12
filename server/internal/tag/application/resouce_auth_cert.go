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

type RelateAuthCertParam struct {
	ResourceCode string

	// 资源标签类型
	ResourceType entity.TagType

	// 空数组则为删除该资源绑定的授权凭证
	AuthCerts []*entity.ResourceAuthCert
}

type ResourceAuthCert interface {
	base.App[*entity.ResourceAuthCert]

	// RelateAuthCert 关联资源授权凭证信息
	RelateAuthCert(ctx context.Context, param *RelateAuthCertParam) error

	// RelateAuthCert2ResourceTag 关联授权凭证标签至指定资源标签下
	RelateAuthCert2ResourceTag(ctx context.Context, param *RelateAuthCertParam) error

	// SaveAuthCert 保存授权凭证信息
	SaveAuthCert(ctx context.Context, rac *entity.ResourceAuthCert) error

	DeleteAuthCert(ctx context.Context, id uint64) error

	// GetAuthCert 根据授权凭证名称获取授权凭证
	GetAuthCert(authCertName string) (*entity.ResourceAuthCert, error)

	// GetResourceAuthCert 获取资源授权凭证，默认获取特权账号，若没有则返回第一个
	GetResourceAuthCert(resourceType entity.TagType, resourceCode string) (*entity.ResourceAuthCert, error)

	// GetAccountAuthCert 获取账号有权限操作的授权凭证信息
	GetAccountAuthCert(accountId uint64, authCertTagType entity.TagType, tagPath ...string) []*entity.ResourceAuthCert

	// FillAuthCertByAcs 根据授权凭证列表填充资源的授权凭证信息
	// @param authCerts 授权凭证列表
	// @param resources 实现了entity.IAuthCert接口的资源信息
	FillAuthCertByAcs(authCerts []*entity.ResourceAuthCert, resources ...entity.IAuthCert)

	// FillAuthCert 填充资源对应的授权凭证信息
	// @param resourceType 资源类型
	// @param resources 实现了entity.IAuthCert接口的资源信息
	FillAuthCert(resourceType int8, resources ...entity.IAuthCert)
}

type resourceAuthCertAppImpl struct {
	base.AppImpl[*entity.ResourceAuthCert, repository.ResourceAuthCert]

	tagTreeApp TagTree `inject:"TagTreeApp"`
}

// 注入Repo
func (r *resourceAuthCertAppImpl) InjectResourceAuthCertRepo(resourceAuthCertRepo repository.ResourceAuthCert) {
	r.Repo = resourceAuthCertRepo
}

func (r *resourceAuthCertAppImpl) RelateAuthCert(ctx context.Context, params *RelateAuthCertParam) error {
	resourceCode := params.ResourceCode
	resourceType := int8(params.ResourceType)
	resourceAuthCerts := params.AuthCerts

	if resourceCode == "" {
		return errorx.NewBiz("资源授权凭证的资源编号不能为空")
	}
	if resourceType == 0 {
		return errorx.NewBiz("资源类型不能为空")
	}

	// 删除授权信息
	if len(resourceAuthCerts) == 0 {
		logx.DebugfContext(ctx, "RelateAuthCert[%d-%s]-删除所有关联的授权凭证信息", resourceType, resourceCode)
		if err := r.DeleteByCond(ctx, &entity.ResourceAuthCert{ResourceCode: resourceCode, ResourceType: resourceType}); err != nil {
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
		if resourceAuthCert.Id == 0 && r.GetBy(existNameAc) == nil && existNameAc.ResourceCode != resourceCode {
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

	// 新增、删除以及不变的授权凭证名
	var adds, dels, unmodifys []string
	if len(oldAuthCert) == 0 {
		logx.DebugfContext(ctx, "RelateAuthCert[%d-%s]-不存在已有的授权凭证信息, 为新增资源授权凭证", resourceType, resourceCode)
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
		logx.DebugfContext(ctx, "RelateAuthCert[%d-%s]-新增授权凭证-[%v]", resourceType, resourceCode, adds)
		if err := r.BatchInsert(ctx, addAuthCerts); err != nil {
			return err
		}
	}

	for _, del := range dels {
		logx.DebugfContext(ctx, "RelateAuthCert[%d-%s]-删除授权凭证-[%v]", resourceType, resourceCode, del)
		if err := r.DeleteByCond(ctx, &entity.ResourceAuthCert{ResourceCode: resourceCode, ResourceType: resourceType, Name: del}); err != nil {
			return err
		}
	}

	if len(unmodifys) > 0 {
		// 旧凭证名 -> 旧凭证
		oldName2AuthCert := collx.ArrayToMap(oldAuthCert, func(ac *entity.ResourceAuthCert) string {
			return ac.Name
		})
		acTagType := GetResourceAuthCertTagType(params.ResourceType)
		for _, unmodify := range unmodifys {
			unmodifyAc := name2AuthCert[unmodify]
			if unmodifyAc.Id == 0 {
				continue
			}

			oldAuthCert := oldName2AuthCert[unmodify]
			if !unmodifyAc.HasChanged(oldAuthCert) {
				logx.DebugfContext(ctx, "RelateAuthCert[%d-%s]-授权凭证[%s]未发生字段变更", resourceType, resourceCode, unmodify)
				continue
			}

			// 如果修改了用户名，且该凭证关联至标签，则需要更新对应的标签名（资源授权凭证类型的标签名为username）
			if oldAuthCert.Username != unmodifyAc.Username && acTagType != 0 {
				r.updateAuthCertTagName(ctx, unmodify, acTagType, unmodifyAc.Username)
			}
			logx.DebugfContext(ctx, "RelateAuthCert[%d-%s]-更新授权凭证-[%v]", resourceType, resourceCode, unmodify)
			if err := r.UpdateById(ctx, unmodifyAc); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *resourceAuthCertAppImpl) RelateAuthCert2ResourceTag(ctx context.Context, param *RelateAuthCertParam) error {
	return r.tagTreeApp.RelateTagsByCodeAndType(ctx, &RelateTagsByCodeAndTypeParam{
		ParentTagCode: param.ResourceCode,
		ParentTagType: param.ResourceType,
		TagType:       GetResourceAuthCertTagType(param.ResourceType),
		Tags: collx.ArrayMap(param.AuthCerts, func(rac *entity.ResourceAuthCert) ITag {
			return rac
		}),
	})
}

func (r *resourceAuthCertAppImpl) SaveAuthCert(ctx context.Context, rac *entity.ResourceAuthCert) error {
	if rac.Id == 0 {
		return r.addAuthCert(ctx, rac)
	}

	return r.updateAuthCert(ctx, rac)
}

func (r *resourceAuthCertAppImpl) DeleteAuthCert(ctx context.Context, id uint64) error {
	rac, err := r.GetById(new(entity.ResourceAuthCert), id)
	if err != nil {
		return errorx.NewBiz("授权凭证不存在")
	}

	if rac.Type == entity.AuthCertTypePublic {
		if r.CountByCond(&entity.ResourceAuthCert{Ciphertext: rac.Name}) > 0 {
			return errorx.NewBiz("该公共授权凭证[%s]已被关联", rac.Name)
		}
		// 公共授权凭证直接删除即可
		return r.DeleteById(ctx, id)
	}

	if r.CountByCond(&entity.ResourceAuthCert{ResourceCode: rac.ResourceCode, ResourceType: rac.ResourceType}) <= 1 {
		return errorx.NewBiz("资源至少需要绑定一个授权凭证，无法删除该凭证[%s]", rac.Name)
	}

	return r.Tx(ctx,
		func(ctx context.Context) error {
			// 删除对应授权凭证标签
			return r.tagTreeApp.DeleteTagByParam(ctx, &DelResourceTagParam{
				ResourceCode: rac.Name,
			})
		},
		func(ctx context.Context) error {
			return r.DeleteById(ctx, id)
		})
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
	authCertTags := r.tagTreeApp.GetAccountTags(accountId, tagQuery)

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

func (r *resourceAuthCertAppImpl) FillAuthCertByAcs(authCerts []*entity.ResourceAuthCert, resources ...entity.IAuthCert) {
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

func (r *resourceAuthCertAppImpl) FillAuthCert(resourceType int8, resources ...entity.IAuthCert) {
	if len(resources) == 0 {
		return
	}

	resourceCodes := collx.ArrayMap(resources, func(ac entity.IAuthCert) string {
		return ac.GetCode()
	})
	var acs []*entity.ResourceAuthCert
	r.Repo.ListByWheres(collx.M{"resource_code in ?": resourceCodes, "resource_type = ?": resourceType}, &acs)
	r.FillAuthCertByAcs(acs, resources...)
}

// addAuthCert 添加授权凭证
func (r *resourceAuthCertAppImpl) addAuthCert(ctx context.Context, rac *entity.ResourceAuthCert) error {
	if r.CountByCond(&entity.ResourceAuthCert{Name: rac.Name}) > 0 {
		return errorx.NewBiz("授权凭证的名称不能重复[%s]", rac.Name)
	}
	// 公共凭证
	if rac.Type == entity.AuthCertTypePublic {
		rac.ResourceCode = "-"
		rac.CiphertextEncrypt()
		rac.Type = -2
		return r.Insert(ctx, rac)
	}

	resourceCode := rac.ResourceCode
	resourceType := rac.ResourceType
	// 资源对应的授权凭证标签类型，若为0则说明该资源不需要关联至资源tagTree
	authCertTagType := GetResourceAuthCertTagType(entity.TagType(resourceType))

	var resourceTagIds []uint64
	// 如果该资源存在对应的授权凭证标签类型，则说明需要关联至tagTree，否则直接从授权凭证库中验证资源编号是否正确即可（一个资源最少有一个授权凭证）
	if authCertTagType != 0 {
		// 获取资源编号对应的资源标签信息
		var resourceTags []*entity.TagTree
		r.tagTreeApp.ListByCond(&entity.TagTree{Type: entity.TagType(resourceType), Code: resourceCode}, &resourceTags)
		// 资源标签id（相当于父tag id）
		resourceTagIds = collx.ArrayMap(resourceTags, func(tag *entity.TagTree) uint64 {
			return tag.Id
		})
		if len(resourceTagIds) == 0 {
			return errorx.NewBiz("资源标签不存在[%s], 请检查资源编号是否正确", resourceCode)
		}
	} else {
		if r.CountByCond(&entity.ResourceAuthCert{ResourceCode: resourceCode, ResourceType: resourceType}) == 0 {
			return errorx.NewBiz("该授权凭证关联的资源信息不存在, 请检查资源编号")
		}
	}

	// 如果密文类型不为公共凭证，则进行加密。公共凭证密文内容存的是明文的公共凭证名
	if rac.CiphertextType != entity.AuthCertCiphertextTypePublic {
		rac.CiphertextEncrypt()
	}

	return r.Tx(ctx, func(ctx context.Context) error {
		// 若存在需要关联到的资源标签，则关联到对应的资源标签下
		if len(resourceTagIds) > 0 {
			logx.DebugfContext(ctx, "[%d-%s]-授权凭证标签[%d-%s]关联至所属资源标签下[%v]", resourceType, resourceCode, authCertTagType, rac.Name, resourceTagIds)
			return r.tagTreeApp.SaveResourceTag(ctx, &SaveResourceTagParam{
				Code:         rac.Name,
				Type:         GetResourceAuthCertTagType(entity.TagType(resourceType)),
				Name:         rac.Username,
				ParentTagIds: resourceTagIds,
			})
		}
		return nil
	}, func(ctx context.Context) error {
		return r.Insert(ctx, rac)
	})
}

// updateAuthCert 更新授权凭证
func (r *resourceAuthCertAppImpl) updateAuthCert(ctx context.Context, rac *entity.ResourceAuthCert) error {
	oldRac, err := r.GetById(new(entity.ResourceAuthCert), rac.Id)
	if err != nil {
		return errorx.NewBiz("该授权凭证不存在")
	}

	if !oldRac.HasChanged(rac) {
		return nil
	}

	if oldRac.Type == entity.AuthCertTypePublic {
		// 如果旧凭证为公共凭证，则不允许修改凭证类型
		if rac.Type != entity.AuthCertTypePublic {
			return errorx.NewBiz("公共授权凭证不允许修改凭证类型")
		}

		if rac.CiphertextType == entity.AuthCertCiphertextTypePublic {
			return errorx.NewBiz("公共授权凭证不允许绑定其他公共授权凭证")
		}
	} else {
		if rac.Type == entity.AuthCertTypePublic {
			return errorx.NewBiz("非公共授权凭证不允许修改为公共凭证")
		}

		// 修改了用户名，则需要同步更新对应授权凭证标签里的名称
		if rac.Username != oldRac.Username && rac.ResourceType == int8(entity.TagTypeMachine) {
			authCertTagType := GetResourceAuthCertTagType(entity.TagType(oldRac.ResourceType))
			if authCertTagType != 0 {
				if err := r.updateAuthCertTagName(ctx, oldRac.Name, authCertTagType, rac.Username); err != nil {
					return errorx.NewBiz("同步更新授权凭证标签名称失败")
				}
			}
		}
	}

	// 防止误更新
	rac.Name = ""
	rac.ResourceCode = ""
	rac.ResourceType = 0
	return r.UpdateById(ctx, rac)
}

// updateAuthCertTagName 同步更新授权凭证的标签名，防止标签展示不一致
func (r *resourceAuthCertAppImpl) updateAuthCertTagName(ctx context.Context, authCertName string, autyCertTagType entity.TagType, newTagName string) error {
	return r.tagTreeApp.UpdateByWheres(ctx, &entity.TagTree{Name: newTagName}, collx.M{"code = ?": authCertName, "type = ?": autyCertTagType})
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

// GetResourceAuthCertTagType 根据资源类型，获取对应的授权凭证标签类型，return 0 说明该资源授权凭证不关联至tagTree
func GetResourceAuthCertTagType(resourceType entity.TagType) entity.TagType {
	if resourceType == entity.TagTypeMachine {
		return entity.TagTypeMachineAuthCert
	}

	// 该资源不存在对应的授权凭证标签，即tag_tree不关联该资源的授权凭证
	return 0
}
