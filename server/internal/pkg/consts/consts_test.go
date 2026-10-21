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

// TestResourceTypeKeys 资源类型键名表的注册自检：这张表是「资源类型叫什么」的唯一真源
// （告警触发事件、AI 工具入参说明、资源可见面判定都读它），此前同一映射散在三处且已漂移。
//
// 登记错一个类型值不会报错，只会表现为「触发器按类型永远匹不上」「工具把 redis 当成未知类型」，
// 所以必须在单测里钉住：键必须是已声明的资源类型、反查一致、顺序稳定
func TestResourceTypeKeys(t *testing.T) {
	registered := ResourceTypeWithKey()
	if len(registered) == 0 {
		t.Fatal("资源类型键名表不得为空，否则工具说明与触发器匹配都会失去类型口径")
	}

	// 升序：工具入参说明与下拉选项依赖稳定顺序
	for i := 1; i < len(registered); i++ {
		if registered[i-1] >= registered[i] {
			t.Fatalf("ResourceTypeWithKey 必须按数值升序，实际 %d 在 %d 之后", registered[i-1], registered[i])
		}
	}

	for _, value := range registered {
		if !IsResourceType(value) {
			t.Errorf("键名表登记了未声明的资源类型 %d（漏登记 KnownResourceTypes 会导致类型自检失效）", value)
		}
		key, ok := ResourceTypeKey(value)
		if !ok || key == "" {
			t.Errorf("资源类型 %d 在键名表里取不到键名", value)
		}
	}

	// 凭证与库名不是可观测/可匹配的资源主体，有意不登记：它们混进类型表会让
	// 「按资源类型匹配触发器」出现永远命中不到的错配
	for _, notAResource := range []int8{ResourceTypeAuthCert, ResourceTypeDbName} {
		if _, ok := ResourceTypeKey(notAResource); ok {
			t.Errorf("非资源主体类型 %d 不应登记键名", notAResource)
		}
	}

	// 未知值不得回退成某个真实键名（调用方靠 ok 分支做 fail-closed 判定）
	if key, ok := ResourceTypeKey(100); ok {
		t.Errorf("未知资源类型不应命中键名，实际返回 %q", key)
	}
}
