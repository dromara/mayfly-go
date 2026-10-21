package resource

import (
	"context"
	"fmt"
	"slices"
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

	// 机器 → 可操作凭证名列表（取自标签路径的 AuthCert 段）。
	//
	// 为什么不能拿机器 code 当凭证名：凭证表没有 code 列，它的唯一标识就是 name（如
	// tVUcRFKwOC），而机器 code 是另一个值（如 Umaq3B2KHkKg）。旧写法把两者当成同一个，
	// 导致模型从资源清单读到的是一个不存在的凭证，MachineCommandExec 必然 `ac not found`
	// 且它无论怎么猜都猜不中（实测 10/10 失败）；参数补全卡片也把错值封在选项里，用户
	// 补全后照样失败。一台机器可挂多张凭证，故按 (机器, 凭证) 展开
	certsByMachine := make(map[string][]string, len(codes))
	for _, codePath := range tags.GetCodePaths() {
		machineCode := tagentity.CodePath(codePath).GetCode(tagentity.TagTypeMachine)
		authCertName := tagentity.CodePath(codePath).GetCode(tagentity.TagTypeAuthCert)
		if machineCode == "" || authCertName == "" {
			continue
		}
		if !slices.Contains(certsByMachine[machineCode], authCertName) {
			certsByMachine[machineCode] = append(certsByMachine[machineCode], authCertName)
		}
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
		// 没有可操作凭证的机器不产出资源：旧行为会产出一条凭证名错误的资源，模型照着调用
		// 只会得到 `ac not found`，不如一开始就不给它一个走不通的选项
		for _, authCertName := range certsByMachine[m.Code] {
			resources = append(resources, &Resource{
				Type:        TypeMachine,
				Id:          strconv.FormatUint(m.Id, 10),
				Code:        m.Code,
				Name:        m.Name,
				Description: fmt.Sprintf("%s:%d", m.Ip, m.Port),
				// 凭证名放进 Detail 对模型可见：它调用 MachineCommandExec 时必须用这个值，
				// 藏在 Extra 里它看不到，只能猜（实测猜什么都有）
				Detail: map[string]string{
					ExtraKeyIp:           m.Ip,
					ExtraKeyPort:         strconv.Itoa(m.Port),
					ExtraKeyAuthCertName: authCertName,
				},
				Extra: collx.M{ExtraKeyAuthCertName: authCertName},
			})
		}
	}
	return resources, nil
}
