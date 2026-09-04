package resource

import (
	"context"
	"fmt"
	"strconv"

	dbapp "mayfly-go/internal/db/application"
	dbentity "mayfly-go/internal/db/domain/entity"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
)

// dbProvider 数据库资源提供者
//
// 权限口径与 ops 数据库列表一致：账号拥有「数据库实例+授权凭证+数据库」标签
// 即视为可操作，返回数据库资产（连接级）清单；具体物理库名由调用方按需展开
// （指定库名模式见 Detail.databases，自动获取模式经授权凭证实时列库）。
type dbProvider struct{}

func (p *dbProvider) Type() string { return TypeDb }

func (p *dbProvider) List(ctx context.Context, accountId uint64) (resources []*Resource, err error) {
	// 应用访问器底层为 ioc.Get，未注册时 panic；转 error 避免中断整个资源查询流程
	defer func() {
		if r := recover(); r != nil {
			resources = nil
			err = fmt.Errorf("db provider panic: %v", r)
		}
	}()

	tagTreeApp := tagapp.GetTagTreeApp()
	tags := tagTreeApp.GetAccountTags(accountId, &tagentity.TagTreeQuery{
		TypePaths: collx.AsArray(tagentity.NewTypePaths(tagentity.TagTypeDbInstance, tagentity.TagTypeAuthCert, tagentity.TagTypeDb)),
	})
	// 不存在可访问标签，即没有可操作数据
	if len(tags) == 0 {
		return []*Resource{}, nil
	}
	codes := tags.GetCodes()
	if len(codes) == 0 {
		return []*Resource{}, nil
	}

	dbs, err := dbapp.GetDbApp().GetPageList(&dbentity.DbQuery{
		PageParam: model.PageParam{PageNum: 1, PageSize: maxQuerySize},
		Codes:     codes,
	})
	if err != nil {
		return nil, err
	}

	resources = make([]*Resource, 0, len(dbs.List))
	for _, d := range dbs.List {
		if d == nil || d.Id == nil || d.Name == nil {
			continue
		}
		r := &Resource{
			Type:        TypeDb,
			Id:          strconv.FormatInt(*d.Id, 10),
			Code:        d.Code,
			Name:        *d.Name,
			Description: d.Code,
			Extra: collx.M{
				ExtraKeyAuthCertName:    d.AuthCertName,
				ExtraKeyGetDatabaseMode: d.GetDatabaseMode,
				ExtraKeyDatabase:        "",
			},
		}
		if d.Database != nil {
			r.Extra[ExtraKeyDatabase] = *d.Database
		}
		// 指定库名模式下透出可连接库名，供 LLM 直接填充 dbName 参数
		if d.GetDatabaseMode == dbentity.DbGetDatabaseModeAssign && d.Database != nil && *d.Database != "" {
			r.Detail = map[string]string{"databases": *d.Database}
		}
		resources = append(resources, r)
	}
	return resources, nil
}
