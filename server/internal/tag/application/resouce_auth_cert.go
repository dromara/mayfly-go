package application

import (
	"context"
	"mayfly-go/internal/tag/application/dto"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
)

type ResourceAuthCert interface {
	base.App[*entity.ResourceAuthCert]

	// RelateAuthCert 关联资源授权凭证信息
	RelateAuthCert(ctx context.Context, param *dto.RelateAuthCert) error

	// SaveAuthCert 保存授权凭证信息
	SaveAuthCert(ctx context.Context, rac *entity.ResourceAuthCert) error

	DeleteAuthCert(ctx context.Context, id uint64) error

	// GetAuthCert 根据授权凭证名称获取授权凭证
	GetAuthCert(authCertName string) (*entity.ResourceAuthCert, error)

	//GetRealAuthCert 获取真实可连接鉴权的授权凭证，主要用于资源测试连接时
	GetRealAuthCert(authCert *entity.ResourceAuthCert) (*entity.ResourceAuthCert, error)

	// GetResourceAuthCert 获取资源授权凭证，优先获取默认账号，若不存在默认账号则返回特权账号，都不存在则返回第一个
	GetResourceAuthCert(resourceType entity.TagType, resourceCode string) (*entity.ResourceAuthCert, error)

	// FillAuthCertByAcs 根据授权凭证列表填充资源的授权凭证信息
	// @param authCerts 授权凭证列表
	// @param resources 实现了entity.IAuthCert接口的资源信息
	FillAuthCertByAcs(authCerts []*entity.ResourceAuthCert, resources ...entity.IAuthCert)

	// FillAuthCert 填充资源对应的授权凭证信息
	// @param resourceType 资源类型
	// @param resources 实现了entity.IAuthCert接口的资源信息
	FillAuthCert(resourceType int8, resources ...entity.IAuthCert)

	// FillAuthCertByAcNames 根据授权凭证名称填充资源对应的凭证信息
	FillAuthCertByAcNames(authCertNames []string, resources ...entity.IAuthCert)
}

type resourceAuthCertAppImpl struct {
	base.AppImpl[*entity.ResourceAuthCert, repository.ResourceAuthCert]

	tagTreeApp TagTree `inject:"TagTreeApp"`
}

// 注入Repo
func (r *resourceAuthCertAppImpl) InjectResourceAuthCertRepo(resourceAuthCertRepo repository.ResourceAuthCert) {
	r.Repo = resourceAuthCertRepo
}

func (r *resourceAuthCertAppImpl) RelateAuthCert(ctx context.Context, params *dto.RelateAuthCert) error {
	resourceCode := params.ResourceCode
	resourceType := int8(params.ResourceType)
	resourceAuthCerts := params.AuthCerts

	if resourceCode == "" {
		return errorx.NewBiz("授权凭证的资源编号不能为空")
	}
	if resourceType == 0 {
		return errorx.NewBiz("授权凭证的资源类型不能为空")
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

		// 公共授权凭证，则无需进行密文加密，密文即为公共授权凭证名
		if resourceAuthCert.CiphertextType == entity.AuthCertCiphertextTypePublic {
			continue
		}

		// 密文加密
		if err := resourceAuthCert.CiphertextEncrypt(); err != nil {
			return errorx.NewBiz(err.Error())
		}
	}

	oldAuthCert, _ := r.ListByCond(&entity.ResourceAuthCert{ResourceCode: resourceCode, ResourceType: resourceType})

	// 新增、删除以及不变的授权凭证名
	var addAcNames, delAcNames, unmodifyAcNames []string
	if len(oldAuthCert) == 0 {
		logx.DebugfContext(ctx, "RelateAuthCert[%d-%s]-不存在已有的授权凭证信息, 为新增资源授权凭证", resourceType, resourceCode)
		addAcNames = collx.MapKeys(name2AuthCert)
	} else {
		oldNames := collx.ArrayMap(oldAuthCert, func(ac *entity.ResourceAuthCert) string {
			return ac.Name
		})
		addAcNames, delAcNames, unmodifyAcNames = collx.ArrayCompare[string](collx.MapKeys(name2AuthCert), oldNames)
	}

	addAuthCerts := make([]*entity.ResourceAuthCert, 0)
	for _, addAcName := range addAcNames {
		addAc := name2AuthCert[addAcName]
		addAc.Id = 0

		existNameAc := &entity.ResourceAuthCert{Name: addAcName}
		if r.GetByCond(existNameAc) == nil && existNameAc.ResourceCode != resourceCode {
			return errorx.NewBiz("授权凭证的名称不能重复[%s]", addAcName)
		}

		addAuthCerts = append(addAuthCerts, addAc)
	}

	// 处理新增的授权凭证
	if len(addAuthCerts) > 0 {
		logx.DebugfContext(ctx, "RelateAuthCert[%d-%s]-新增授权凭证-[%v]", resourceType, resourceCode, addAcNames)
		if err := r.BatchInsert(ctx, addAuthCerts); err != nil {
			return err
		}
	}

	for _, delAcName := range delAcNames {
		logx.DebugfContext(ctx, "RelateAuthCert[%d-%s]-删除授权凭证-[%v]", resourceType, resourceCode, delAcName)
		if err := r.DeleteByCond(ctx, &entity.ResourceAuthCert{ResourceCode: resourceCode, ResourceType: resourceType, Name: delAcName}); err != nil {
			return err
		}
	}

	if len(unmodifyAcNames) > 0 {
		// 旧凭证名 -> 旧凭证
		oldName2AuthCert := collx.ArrayToMap(oldAuthCert, func(ac *entity.ResourceAuthCert) string {
			return ac.Name
		})
		acTagType := GetResourceAuthCertTagType(params.ResourceType)
		for _, unmodifyAcName := range unmodifyAcNames {
			unmodifyAc := name2AuthCert[unmodifyAcName]

			oldAuthCert := oldName2AuthCert[unmodifyAcName]
			if !unmodifyAc.HasChanged(oldAuthCert) {
				logx.DebugfContext(ctx, "RelateAuthCert[%d-%s]-授权凭证[%s]未发生字段变更", resourceType, resourceCode, unmodifyAcName)
				continue
			}

			// 如果修改了用户名，且该凭证关联至标签，则需要更新对应的标签名（资源授权凭证类型的标签名为username）
			if oldAuthCert.Username != unmodifyAc.Username && acTagType != 0 {
				r.tagTreeApp.UpdateTagName(ctx, acTagType, unmodifyAcName, unmodifyAc.Username)
			}
			logx.DebugfContext(ctx, "RelateAuthCert[%d-%s]-更新授权凭证-[%v]", resourceType, resourceCode, unmodifyAcName)
			if err := r.UpdateByCond(ctx, unmodifyAc, &entity.ResourceAuthCert{Name: unmodifyAcName, ResourceCode: resourceCode, ResourceType: resourceType}); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *resourceAuthCertAppImpl) SaveAuthCert(ctx context.Context, rac *entity.ResourceAuthCert) error {
	if rac.Id == 0 {
		return r.addAuthCert(ctx, rac)
	}

	return r.updateAuthCert(ctx, rac)
}

func (r *resourceAuthCertAppImpl) DeleteAuthCert(ctx context.Context, id uint64) error {
	rac, err := r.GetById(id)
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
			return r.tagTreeApp.DeleteTagByParam(ctx, &dto.DelResourceTag{
				ResourceCode: rac.Name,
			})
		},
		func(ctx context.Context) error {
			return r.DeleteById(ctx, id)
		})
}

func (r *resourceAuthCertAppImpl) GetAuthCert(authCertName string) (*entity.ResourceAuthCert, error) {
	authCert := &entity.ResourceAuthCert{Name: authCertName}
	if err := r.GetByCond(authCert); err != nil {
		return nil, errorx.NewBiz("该授权凭证不存在")
	}

	return r.decryptAuthCert(authCert)
}

func (r *resourceAuthCertAppImpl) GetRealAuthCert(authCert *entity.ResourceAuthCert) (*entity.ResourceAuthCert, error) {
	// 如果使用的是公共授权凭证，则密文为凭证名称
	if authCert.CiphertextType == entity.AuthCertCiphertextTypePublic {
		return r.GetAuthCert(authCert.Ciphertext)
	}

	if authCert.Id != 0 && authCert.Ciphertext == "" {
		// 密文可能被清除，故需要重新获取
		ac, err := r.GetAuthCert(authCert.Name)
		if err != nil {
			return nil, err
		}
		authCert.Ciphertext = ac.Ciphertext
		return authCert, nil
	}

	return authCert, nil
}

func (r *resourceAuthCertAppImpl) GetResourceAuthCert(resourceType entity.TagType, resourceCode string) (*entity.ResourceAuthCert, error) {
	resourceAuthCerts, err := r.ListByCond(&entity.ResourceAuthCert{
		ResourceType: int8(resourceType),
		ResourceCode: resourceCode,
	})
	if err != nil {
		return nil, err
	}

	if len(resourceAuthCerts) == 0 {
		return nil, errorx.NewBiz("该资源不存在授权凭证账号")
	}

	for _, resourceAuthCert := range resourceAuthCerts {
		if resourceAuthCert.Type == entity.AuthCertTypePrivateDefault {
			return r.decryptAuthCert(resourceAuthCert)
		}
	}

	for _, resourceAuthCert := range resourceAuthCerts {
		if resourceAuthCert.Type == entity.AuthCertTypePrivileged {
			return r.decryptAuthCert(resourceAuthCert)
		}
	}

	return r.decryptAuthCert(resourceAuthCerts[0])
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

func (r *resourceAuthCertAppImpl) FillAuthCertByAcNames(authCertNames []string, resources ...entity.IAuthCert) {
	acs, _ := r.ListByCond(model.NewCond().In("name", authCertNames))
	r.FillAuthCertByAcs(acs, resources...)
}

func (r *resourceAuthCertAppImpl) FillAuthCert(resourceType int8, resources ...entity.IAuthCert) {
	if len(resources) == 0 {
		return
	}

	resourceCodes := collx.ArrayMap(resources, func(ac entity.IAuthCert) string {
		return ac.GetCode()
	})
	acs, _ := r.ListByCond(model.NewCond().In("resource_code", resourceCodes).Eq("resource_type", resourceType))
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
		rac.ResourceType = -2
		return r.Insert(ctx, rac)
	}

	resourceCode := rac.ResourceCode
	resourceType := rac.ResourceType
	// 资源对应的授权凭证标签类型，若为0则说明该资源不需要关联至资源tagTree
	authCertTagType := GetResourceAuthCertTagType(entity.TagType(resourceType))

	var resourceTagCodePaths []string
	// 如果该资源存在对应的授权凭证标签类型，则说明需要关联至tagTree，否则直接从授权凭证库中验证资源编号是否正确即可（一个资源最少有一个授权凭证）
	if authCertTagType != 0 {
		// 获取资源编号对应的资源标签信息
		resourceTags, _ := r.tagTreeApp.ListByCond(&entity.TagTree{Type: entity.TagType(resourceType), Code: resourceCode})
		// 资源标签tagPath（相当于父tag）
		resourceTagCodePaths = collx.ArrayMap(resourceTags, func(tag *entity.TagTree) string {
			return tag.CodePath
		})
		if len(resourceTagCodePaths) == 0 {
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
		if len(resourceTagCodePaths) > 0 {
			logx.DebugfContext(ctx, "[%d-%s]-授权凭证标签[%d-%s]关联至所属资源标签下[%v]", resourceType, resourceCode, authCertTagType, rac.Name, resourceTagCodePaths)
			return r.tagTreeApp.SaveResourceTag(ctx, &dto.SaveResourceTag{
				ResourceTag: &dto.ResourceTag{
					Code: rac.Name,
					Type: GetResourceAuthCertTagType(entity.TagType(resourceType)),
					Name: rac.Username,
				},
				ParentTagCodePaths: resourceTagCodePaths,
			})
		}
		return nil
	}, func(ctx context.Context) error {
		return r.Insert(ctx, rac)
	})
}

// updateAuthCert 更新授权凭证
func (r *resourceAuthCertAppImpl) updateAuthCert(ctx context.Context, rac *entity.ResourceAuthCert) error {
	oldRac, err := r.GetById(rac.Id)
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
				if err := r.tagTreeApp.UpdateTagName(ctx, authCertTagType, oldRac.Name, rac.Username); err != nil {
					return errorx.NewBiz("同步更新授权凭证标签名称失败")
				}
			}
		}
	}

	// 密文存的不是公共授权凭证名，则进行密文加密处理
	if rac.CiphertextType != entity.AuthCertCiphertextTypePublic {
		rac.CiphertextEncrypt()
	}

	// 防止误更新
	rac.Name = ""
	rac.ResourceCode = ""
	rac.ResourceType = 0
	return r.UpdateById(ctx, rac)
}

// 解密授权凭证信息
func (r *resourceAuthCertAppImpl) decryptAuthCert(authCert *entity.ResourceAuthCert) (*entity.ResourceAuthCert, error) {
	if authCert.CiphertextType == entity.AuthCertCiphertextTypePublic {
		// 如果是公共授权凭证，则密文为公共授权凭证名称，需要使用该名称再去获取对应的授权凭证
		realAuthCert := &entity.ResourceAuthCert{Name: authCert.Ciphertext}
		if err := r.GetByCond(realAuthCert); err != nil {
			return nil, errorx.NewBiz("该公共授权凭证[%s]不存在", authCert.Ciphertext)
		}

		// 使用该凭证关联的公共凭证进行密文等内容覆盖
		authCert.Username = realAuthCert.Username
		authCert.Ciphertext = realAuthCert.Ciphertext
		authCert.CiphertextType = realAuthCert.CiphertextType
		authCert.Extra = realAuthCert.Extra
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

	if resourceType == entity.TagTypeDb {
		return entity.TagTypeDbAuthCert
	}

	// 该资源不存在对应的授权凭证标签，即tag_tree不关联该资源的授权凭证
	return 0
}
