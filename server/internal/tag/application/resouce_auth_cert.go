package application

import (
	"context"
	"mayfly-go/internal/tag/application/dto"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/internal/tag/imsg"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
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

	tagTreeApp TagTree `inject:"T"`
}

func (r *resourceAuthCertAppImpl) RelateAuthCert(ctx context.Context, params *dto.RelateAuthCert) error {
	resourceCode := params.ResourceCode
	resourceType := int8(params.ResourceType)
	resourceAuthCerts := params.AuthCerts

	if resourceCode == "" {
		return errorx.NewBiz("The resource code of the authorization credential cannot be empty")
	}
	if resourceType == 0 {
		return errorx.NewBiz("The resource type of the authorization credential cannot be empty")
	}

	// 删除授权信息
	if len(resourceAuthCerts) == 0 {
		logx.DebugfContext(ctx, "RelateAuthCert[%d-%s] - Remove all associated authorization credential information", resourceType, resourceCode)
		if err := r.DeleteByCond(ctx, &entity.ResourceAuthCert{ResourceCode: resourceCode, ResourceType: resourceType}); err != nil {
			return err
		}

		return nil
	}

	// 不存在授权凭证名，则随机生成
	for _, ac := range resourceAuthCerts {
		if ac.Name == "" {
			ac.Name = stringx.Rand(10)
		}
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
			return err
		}
	}

	oldAuthCert, _ := r.ListByCond(&entity.ResourceAuthCert{ResourceCode: resourceCode, ResourceType: resourceType})

	// 新增、删除以及不变的授权凭证名
	var addAcNames, delAcNames, unmodifyAcNames []string
	if len(oldAuthCert) == 0 {
		logx.DebugfContext(ctx, "RelateAuthCert[%d-%s] - There is no existing authorization credential information, the current operation is the authorization credential of the new resource", resourceType, resourceCode)
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
			return errorx.NewBizf("The name of the authorization credential cannot be repeated: [%s]", addAcName)
		}

		addAuthCerts = append(addAuthCerts, addAc)
	}

	// 处理新增的授权凭证
	if len(addAuthCerts) > 0 {
		logx.DebugfContext(ctx, "RelateAuthCert[%d-%s] - Add authorization credentials - [%v]", resourceType, resourceCode, addAcNames)
		if err := r.BatchInsert(ctx, addAuthCerts); err != nil {
			return err
		}
	}

	for _, delAcName := range delAcNames {
		logx.DebugfContext(ctx, "RelateAuthCert[%d-%s] - Delete authorization credentials - [%v]", resourceType, resourceCode, delAcName)
		if err := r.DeleteByCond(ctx, &entity.ResourceAuthCert{ResourceCode: resourceCode, ResourceType: resourceType, Name: delAcName}); err != nil {
			return err
		}
	}

	if len(unmodifyAcNames) > 0 {
		// 旧凭证名 -> 旧凭证
		oldName2AuthCert := collx.ArrayToMap(oldAuthCert, func(ac *entity.ResourceAuthCert) string {
			return ac.Name
		})
		for _, unmodifyAcName := range unmodifyAcNames {
			unmodifyAc := name2AuthCert[unmodifyAcName]

			oldAuthCert := oldName2AuthCert[unmodifyAcName]
			if !unmodifyAc.HasChanged(oldAuthCert) {
				logx.DebugfContext(ctx, "RelateAuthCert[%d-%s] - Authorization credential [%s] No field changes", resourceType, resourceCode, unmodifyAcName)
				continue
			}

			// 如果修改了用户名，且该凭证关联至标签，则需要更新对应的标签名（资源授权凭证类型的标签名为username）
			if oldAuthCert.Username != unmodifyAc.Username {
				r.tagTreeApp.UpdateTagName(ctx, entity.TagTypeAuthCert, unmodifyAcName, unmodifyAc.Username)
			}
			logx.DebugfContext(ctx, "RelateAuthCert[%d-%s] - Update Authorization credential - [%v]", resourceType, resourceCode, unmodifyAcName)
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
		return errorx.NewBiz("ac not found")
	}

	if rac.Type == entity.AuthCertTypePublic {
		if r.CountByCond(&entity.ResourceAuthCert{Ciphertext: rac.Name}) > 0 {
			return errorx.NewBizI(ctx, imsg.ErrPublicAcRelated)
		}
		// 公共授权凭证直接删除即可
		return r.DeleteById(ctx, id)
	}

	if r.CountByCond(&entity.ResourceAuthCert{ResourceCode: rac.ResourceCode, ResourceType: rac.ResourceType}) <= 1 {
		return errorx.NewBizI(ctx, imsg.ErrResourceNoBindAc)
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
		return nil, errorx.NewBiz("ac not found")
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
		return nil, errorx.NewBiz("An authorization credential account does not exist for this resource")
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
			logx.Debugf("FillAuthCert - Credential [%s] does not match resource [%s]", authCert.Name, authCert.ResourceCode)
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
	if rac.Name == "" {
		rac.Name = stringx.Rand(10)
	} else {
		if r.CountByCond(&entity.ResourceAuthCert{Name: rac.Name}) > 0 {
			return errorx.NewBizI(ctx, imsg.ErrAcNameExist, "acName", rac.Name)
		}
	}
	if rac.Type == 0 {
		rac.Type = entity.AuthCertTypePrivate
	}
	if rac.CiphertextType == 0 {
		rac.CiphertextType = entity.AuthCertCiphertextTypePassword
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

	var resourceTagCodePaths []string
	// 获取资源编号对应的资源标签信息
	resourceTags, _ := r.tagTreeApp.ListByCond(&entity.TagTree{Type: entity.TagType(resourceType), Code: resourceCode})
	// 资源标签tagPath（相当于父tag）
	resourceTagCodePaths = collx.ArrayMap(resourceTags, func(tag *entity.TagTree) string {
		return tag.CodePath
	})
	if len(resourceTagCodePaths) == 0 {
		return errorx.NewBizI(ctx, imsg.ErrResourceTagNotExist, "resourceCode", resourceCode)
	}

	// 如果密文类型不为公共凭证，则进行加密。公共凭证密文内容存的是明文的公共凭证名
	if rac.CiphertextType != entity.AuthCertCiphertextTypePublic {
		rac.CiphertextEncrypt()
	}

	return r.Tx(ctx, func(ctx context.Context) error {
		// 若存在需要关联到的资源标签，则关联到对应的资源标签下
		if len(resourceTagCodePaths) > 0 {
			logx.DebugfContext(ctx, "[%d-%s] - AC tag [%d-%s] associated to the resource tag [%v] ", resourceType, resourceCode, entity.TagTypeAuthCert, rac.Name, resourceTagCodePaths)
			return r.tagTreeApp.SaveResourceTag(ctx, &dto.SaveResourceTag{
				ResourceTag: &dto.ResourceTag{
					Code: rac.Name,
					Type: entity.TagTypeAuthCert,
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
		return errorx.NewBiz("ac not found")
	}

	if !oldRac.HasChanged(rac) {
		return nil
	}

	if oldRac.Type == entity.AuthCertTypePublic {
		// 如果旧凭证为公共凭证，则不允许修改凭证类型
		if rac.Type != entity.AuthCertTypePublic {
			return errorx.NewBizI(ctx, imsg.ErrPublicAcNotAllowModifyType)
		}

		if rac.CiphertextType == entity.AuthCertCiphertextTypePublic {
			// 公共授权凭证不允许绑定其他公共授权凭证
			return errorx.NewBiz("Public authorization credentials are not allowed to bind other public authorization credentials")
		}
	} else {
		if rac.Type == entity.AuthCertTypePublic {
			// 非公共授权凭证不允许修改为公共凭证
			return errorx.NewBiz("Non-public authorization credentials are not allowed to be modified to public credentials")
		}

		// 修改了用户名，则需要同步更新对应授权凭证标签里的名称
		if rac.Username != oldRac.Username && rac.ResourceType == int8(entity.TagTypeMachine) {
			if err := r.tagTreeApp.UpdateTagName(ctx, entity.TagTypeAuthCert, oldRac.Name, rac.Username); err != nil {
				return errorx.NewBiz("Synchronously updating the authorization credential tag name failed")
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
			return nil, errorx.NewBizI(context.Background(), imsg.ErrPublicAcNotExist, "acName", authCert.Ciphertext)
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
