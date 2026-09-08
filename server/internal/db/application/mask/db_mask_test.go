package mask

// 脱敏豁免角色与开关门控的单元测试：
//   - isExemptAccount：配置的豁免角色命中判定 + 零值planCache.exempts(nil map)写入安全性（IOC以new()创建实例的场景）
//   - buildRowMasker：MaskEnabled=false时直接不脱敏
//
// 通过ioc注册仅覆写GetConfig/GetAccountRoles的fake实现，避免依赖真实数据库与系统配置

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	sysapp "mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/ioc"
)

// fakeMaskConfigApp 仅覆写GetConfig，其余接口方法在本测试中不会被调用
type fakeMaskConfigApp struct {
	sysapp.Config
	value *entity.Config
}

func (f *fakeMaskConfigApp) GetConfig(_ string) *entity.Config {
	if f.value == nil {
		return &entity.Config{}
	}
	return f.value
}

// fakeMaskRoleApp 仅覆写GetAccountRoles
type fakeMaskRoleApp struct {
	sysapp.Role
	roleIds []uint64
}

func (f *fakeMaskRoleApp) GetAccountRoles(_ uint64) ([]*entity.AccountRole, error) {
	roles := make([]*entity.AccountRole, 0, len(f.roleIds))
	for _, id := range f.roleIds {
		roles = append(roles, &entity.AccountRole{RoleId: id})
	}
	return roles, nil
}

// maskTestDbmsJson 构造Dbms配置实体的json值（Id需非零，GetJsonM对Id==0返回空map；maskExemptRoleIds以字符串数组存储，与GetStrSlice解析一致）
func maskTestDbmsJson(maskEnabled bool, exemptRoleIds ...uint64) *entity.Config {
	ids := make([]string, 0, len(exemptRoleIds))
	for _, id := range exemptRoleIds {
		ids = append(ids, fmt.Sprintf("\"%d\"", id))
	}
	return &entity.Config{Id: 1, Value: fmt.Sprintf(`{"maskEnabled":%t,"maskExemptRoleIds":[%s]}`, maskEnabled, strings.Join(ids, ","))}
}

func TestMaskExemptAccountAndEnabledGate(t *testing.T) {
	fakeConf := &fakeMaskConfigApp{}
	ioc.Register(fakeConf)

	// 场景1：未配置豁免角色 → 不豁免
	fakeConf.value = maskTestDbmsJson(true)
	m := &MaskAppImpl{roleApp: &fakeMaskRoleApp{roleIds: []uint64{2}}}
	assert.False(t, m.isExemptAccount(context.Background(), 9))

	// 场景2：命中豁免角色 → 豁免；同时回归验证零值planCache.exempts(nil map)写入不panic
	fakeConf.value = maskTestDbmsJson(true, 2)
	assert.True(t, m.isExemptAccount(context.Background(), 9))
	// 命中豁免缓存
	assert.True(t, m.isExemptAccount(context.Background(), 9))

	// 场景3：账号角色未命中豁免角色列表
	m2 := &MaskAppImpl{roleApp: &fakeMaskRoleApp{roleIds: []uint64{3}}}
	assert.False(t, m2.isExemptAccount(context.Background(), 9))

	// 场景4：MaskEnabled=false时buildRowMasker直接返回(nil,nil)（开关门控先于其他逻辑）
	fakeConf.value = maskTestDbmsJson(false)
	masker, err := m.buildRowMasker(context.Background(), nil, nil, nil)
	assert.Nil(t, masker)
	assert.NoError(t, err)
}
