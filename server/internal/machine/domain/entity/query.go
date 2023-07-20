package entity

type MachineQuery struct {
	Ids     string `json:"ids" form:"ids"`
	Name    string `json:"name" form:"name"`
	Ip      string `json:"ip" form:"ip"` // IP地址
	TagPath string `json:"tagPath" form:"tagPath"`
	TagIds  []uint64
}

type AuthCertQuery struct {
	Id         uint64 `json:"id" form:"id"`
	Name       string `json:"name" form:"name"`
	AuthMethod string `json:"authMethod" form:"authMethod"` // IP地址
}
