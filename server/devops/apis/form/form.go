package form

type MachineForm struct {
	Id   uint64 `json:"id"`
	Name string `json:"name" valid:"Required"`
	// IP地址
	Ip string `json:"ip" valid:"Required"`
	// 用户名
	Username string `json:"username" valid:"Required"`
	Password string `json:"password" valid:"Required"`
	// 端口号
	Port int `json:"port" valid:"Required"`
}

type MachineRunForm struct {
	MachineId int64  `valid:"Required"`
	Cmd       string `valid:"Required"`
}

type MachineFileForm struct {
	Id        uint64
	Name      string `valid:"Required"`
	MachineId uint64 `valid:"Required"`
	Type      int    `valid:"Required"`
	Path      string `valid:"Required"`
}

type MachineScriptForm struct {
	Id          uint64
	Name        string `valid:"Required"`
	MachineId   uint64 `valid:"Required"`
	Type        int    `valid:"Required"`
	Description string `valid:"Required"`
	Script      string `valid:"Required"`
}

type DbSqlSaveForm struct {
	Sql  string `valid:"Required"`
	Type int    `valid:"Required"`
}

type MachineFileUpdateForm struct {
	Content string `valid:"Required"`
	Id      uint64 `valid:"Required"`
	Path    string `valid:"Required"`
}
