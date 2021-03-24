package form

type MockData struct {
	Method        string   `valid:"Required" json:"method"`
	Enable        uint     `json:"enable"`
	Description   string   `valid:"Required" json:"description"`
	Data          string   `valid:"Required" json:"data"`
	EffectiveUser []string `json:"effectiveUser"`
}

type Machine struct {
	Name     string `json:"name"`
	Ip       string `json:"ip"`       // IP地址
	Username string `json:"username"` // 用户名
	Password string `json:"-"`
	Port     int    `json:"port"` // 端口号
}

type MachineService struct {
	Name    string `json:"name"`
	Ip      string `json:"ip"`      // IP地址
	Service string `json:"service"` // 服务命令
}
