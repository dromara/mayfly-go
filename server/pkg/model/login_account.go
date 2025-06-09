package model

type LoginAccount struct {
	Id       uint64
	Username string
	Token    string
}

func (la *LoginAccount) GetAesKey() string {
	return la.Token[:24]
}

// 系统账号
var SysAccount = &LoginAccount{
	Id:       0,
	Username: "-",
}
