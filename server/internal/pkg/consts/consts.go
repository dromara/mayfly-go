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
)
