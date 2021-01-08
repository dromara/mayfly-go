package form

// 登录表单
type LoginForm struct {
	Username string `valid:"Required"`
	Password string `valid:"Required"`
}

type MachineRunForm struct {
	MachineId int64  `valid:"Required"`
	Cmd       string `valid:"Required"`
}

type DbSqlSaveForm struct {
	Sql  string `valid:"Required"`
	Type int    `valid:"Required"`
}
