package application

import (
	"context"

	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/ioc"
)

// accountPermissionResource 只承接权限码判定所需的列；code 允许 NULL，故用指针。
type accountPermissionResource struct {
	Code *string `gorm:"column:code"`
	Type int8    `gorm:"column:type"`
}

// LoadPermissionCodes 从持久层加载账号权限码，供 pkg/req 在权限缓存未命中时回填。
// 与 GET /sys/accounts/permissions 共用 Resource.GetAccountResources，避免两套判定口径。
// 返回 ok=false 表示查询失败：调用方必须按无权限处理（fail-closed），不能放行。
func LoadPermissionCodes(accountId uint64) ([]string, bool) {
	resourceApp := ioc.Get[Resource]()
	var resources []accountPermissionResource
	if err := resourceApp.GetAccountResources(context.Background(), accountId, &resources); err != nil {
		return nil, false
	}
	codes := make([]string, 0, len(resources))
	for _, r := range resources {
		if r.Type == entity.ResourceTypeMenu || r.Code == nil || *r.Code == "" {
			continue
		}
		codes = append(codes, *r.Code)
	}
	return codes, true
}
