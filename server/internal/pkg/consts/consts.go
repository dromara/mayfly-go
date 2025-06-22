package consts

const (
	AdminId = 1

	ResourceTypeMachine    int8 = 1
	ResourceTypeDbInstance int8 = 2
	ResourceTypeRedis      int8 = 3
	ResourceTypeMongo      int8 = 4
	ResourceTypeAuthCert   int8 = 5
	ResourceTypeEsInstance int8 = 6

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
)
