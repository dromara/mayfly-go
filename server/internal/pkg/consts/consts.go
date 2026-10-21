package consts

import "sort"

const (
	AdminId = 1

	ResourceTypeMachine    int8 = 1
	ResourceTypeDbInstance int8 = 2
	ResourceTypeRedis      int8 = 3
	ResourceTypeMongo      int8 = 4
	ResourceTypeAuthCert   int8 = 5
	ResourceTypeEsInstance int8 = 6
	ResourceTypeContainer  int8 = 7
	ResourceTypeMqKafka    int8 = 8
	ResourceTypeMilvus     int8 = 9
	ResourceTypeDbName     int8 = 22 // 数据库名资源

	// imsg起始编号
	ImsgNumSys      = 10000
	ImsgNumAuth     = 20000
	ImsgNumTag      = 30000
	ImsgNumFlow     = 40000
	ImsgNumMachine  = 50000
	ImsgNumDb       = 60000
	ImsgNumRedis    = 70000
	ImsgNumMongo    = 80000
	ImsgNumMsg      = 90000
	ImsgNumEs       = 100000
	ImsgNumDocker   = 110000
	ImsgNumMqKafka  = 120000
	ImsgNumMilvus   = 130000
	ImsgNumAi       = 140000
	ImsgNumFile     = 150000
	ImsgNumAlert    = 160000
	ImsgNumLabel    = 170000
	ImsgNumAgentops = 180000
)

// KnownResourceTypes 全部已知资源类型，供注册声明自检：写错一个不存在的类型，
// 只会表现为「该场景在资源树里永远选不到、配了也永远不命中」，必须在能看见的时机就报出来。
// 新增资源类型时这里要同步，常量取值唯一性由 consts_test 守住
var KnownResourceTypes = []int8{
	ResourceTypeMachine, ResourceTypeDbInstance, ResourceTypeRedis, ResourceTypeMongo,
	ResourceTypeAuthCert, ResourceTypeEsInstance, ResourceTypeContainer, ResourceTypeMqKafka,
	ResourceTypeMilvus, ResourceTypeDbName,
}

// resourceTypeKeys 资源类型数值 → 稳定字符串键，是「资源类型叫什么」的唯一真源。
//
// 告警触发事件、AI 工具入参说明、资源可见面判定均读这张表：此前同一映射写在三处
// （alert_engine 的 switch、monitor 工具的两段说明文本）且已漂移——说明只列 6 类而
// 映射覆盖 8 类，模型按说明传 redis 会撞上「未注册评估器」硬失败。
// 凭证（AuthCert）与库名（DbName）不是可观测/可匹配的资源主体，有意不登记。
var resourceTypeKeys = map[int8]string{
	ResourceTypeMachine:    "machine",
	ResourceTypeDbInstance: "db",
	ResourceTypeRedis:      "redis",
	ResourceTypeMongo:      "mongo",
	ResourceTypeEsInstance: "es",
	ResourceTypeContainer:  "container",
	ResourceTypeMqKafka:    "kafka",
	ResourceTypeMilvus:     "milvus",
}

// ResourceTypeKey 返回资源类型的稳定字符串键；ok 为 false 表示该类型未登记键名
// （如凭证/库名，或新增但漏登记的类型），调用方需显式处理而非默默拼个数字串
func ResourceTypeKey(value int8) (key string, ok bool) {
	key, ok = resourceTypeKeys[value]
	return
}

// ResourceTypeWithKey 带键名的资源类型（按数值升序），供工具入参说明等需要稳定顺序的
// 场景由真源动态生成，不再手写一份类型清单
func ResourceTypeWithKey() []int8 {
	types := make([]int8, 0, len(resourceTypeKeys))
	for value := range resourceTypeKeys {
		types = append(types, value)
	}
	sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })
	return types
}

// IsResourceType 给定数值是否为已声明的资源类型
func IsResourceType(value int8) bool {
	for _, known := range KnownResourceTypes {
		if known == value {
			return true
		}
	}
	return false
}
