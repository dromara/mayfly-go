package consts

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
	ImsgNumSys     = 10000
	ImsgNumAuth    = 20000
	ImsgNumTag     = 30000
	ImsgNumFlow    = 40000
	ImsgNumMachine = 50000
	ImsgNumDb      = 60000
	ImsgNumRedis   = 70000
	ImsgNumMongo   = 80000
	ImsgNumMsg     = 90000
	ImsgNumEs      = 100000
	ImsgNumDocker  = 110000
	ImsgNumMqKafka = 120000
	ImsgNumMilvus  = 130000
	ImsgNumAi      = 140000
	ImsgNumFile    = 150000
	ImsgNumAlert   = 160000
	ImsgNumLabel   = 170000
)

// KnownResourceTypes 全部已知资源类型，供注册声明自检：写错一个不存在的类型，
// 只会表现为「该场景在资源树里永远选不到、配了也永远不命中」，必须在能看见的时机就报出来。
// 新增资源类型时这里要同步，常量取值唯一性由 consts_test 守住
var KnownResourceTypes = []int8{
	ResourceTypeMachine, ResourceTypeDbInstance, ResourceTypeRedis, ResourceTypeMongo,
	ResourceTypeAuthCert, ResourceTypeEsInstance, ResourceTypeContainer, ResourceTypeMqKafka,
	ResourceTypeMilvus, ResourceTypeDbName,
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
