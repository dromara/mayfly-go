package consts

import "testing"

// TestResourceTypesUnique 资源类型常量唯一性完备性契约测试：
// ResourceType 常量是前后端共享协议（前端 TagResourceTypeEnum 与之对应），
// 新增资源类型时必须在此登记，且值不得与已有类型冲突（重复会导致标签树/权限判断串类型）
func TestResourceTypesUnique(t *testing.T) {
	types := map[string]int8{
		"ResourceTypeMachine":    ResourceTypeMachine,
		"ResourceTypeDbInstance": ResourceTypeDbInstance,
		"ResourceTypeRedis":      ResourceTypeRedis,
		"ResourceTypeMongo":      ResourceTypeMongo,
		"ResourceTypeAuthCert":   ResourceTypeAuthCert,
		"ResourceTypeEsInstance": ResourceTypeEsInstance,
		"ResourceTypeContainer":  ResourceTypeContainer,
		"ResourceTypeMqKafka":    ResourceTypeMqKafka,
		"ResourceTypeMilvus":     ResourceTypeMilvus,
		"ResourceTypeDbName":     ResourceTypeDbName,
	}

	seen := make(map[int8]string, len(types))
	for name, v := range types {
		if other, ok := seen[v]; ok {
			t.Errorf("资源类型值冲突: %s 与 %s 均为 %d", name, other, v)
			continue
		}
		seen[v] = name
	}
}
