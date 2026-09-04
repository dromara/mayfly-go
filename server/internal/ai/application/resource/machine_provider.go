package resource

import (
	"context"
	"fmt"
	"strconv"

	machineapp "mayfly-go/internal/machine/application"
	machineentity "mayfly-go/internal/machine/domain/entity"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
)

// maxQuerySize 单类资源单次查询上限保护（资源量级有限，一次拉全后由 App 层统一过滤）
const maxQuerySize = 1000

// machineProvider 机器资源提供者
//
// 权限口径与 ops 机器列表一致：账号拥有「机器+授权凭证」标签即视为可操作，
// 仅返回启用状态的 ssh 机器（与 AI 机器命令执行工具的可用范围对齐）。
type machineProvider struct{}

func (p *machineProvider) Type() string { return TypeMachine }

func (p *machineProvider) List(ctx context.Context, accountId uint64) (resources []*Resource, err error) {
	// 应用访问器底层为 ioc.Get，未注册时 panic；转 error 避免中断整个资源查询流程
	defer func() {
		if r := recover(); r != nil {
			resources = nil
			err = fmt.Errorf("machine provider panic: %v", r)
		}
	}()

	tagTreeApp := tagapp.GetTagTreeApp()
	tags := tagTreeApp.GetAccountTags(accountId, &tagentity.TagTreeQuery{
		TypePaths: collx.AsArray(tagentity.NewTypePaths(tagentity.TagTypeMachine, tagentity.TagTypeAuthCert)),
	})
	// 不存在可操作的机器-授权凭证标签，即没有可操作数据
	if len(tags) == 0 {
		return []*Resource{}, nil
	}
	codes := collx.ArrayDeduplicate(tagentity.GetCodesByCodePaths(tagentity.TagTypeMachine, tags.GetCodePaths()...))
	if len(codes) == 0 {
		return []*Resource{}, nil
	}

	machines, err := machineapp.GetMachineApp().GetMachineList(&machineentity.MachineQuery{
		PageParam: model.PageParam{PageNum: 1, PageSize: maxQuerySize},
		Codes:     codes,
		Status:    machineentity.MachineStatusEnable,
		Protocol:  machineentity.MachineProtocolSsh,
	})
	if err != nil {
		return nil, err
	}

	resources = make([]*Resource, 0, len(machines.List))
	for _, m := range machines.List {
		if m == nil {
			continue
		}
		resources = append(resources, &Resource{
			Type:        TypeMachine,
			Id:          strconv.FormatUint(m.Id, 10),
			Code:        m.Code,
			Name:        m.Name,
			Description: fmt.Sprintf("%s:%d", m.Ip, m.Port),
			Detail: map[string]string{
				ExtraKeyIp:   m.Ip,
				ExtraKeyPort: strconv.Itoa(m.Port),
			},
			// 机器的 code 即授权凭证名（machinetool 经 GetCliByAc(code) 建连）
			Extra: collx.M{ExtraKeyAuthCertName: m.Code},
		})
	}
	return resources, nil
}
