package form

type ResourceForm struct {
	Pid    int                    `valid:"Required"`
	Id     int                    `valid:"Required"`
	Code   string                 `valid:"Required"`
	Name   string                 `valid:"Required"`
	Type   int                    `valid:"Required"`
	Weight int                    `valid:"Required"`
	Meta   map[string]interface{} `valid:"Required"`
}

type MenuResourceMeta struct {
	RouteName   string `valid:"Required"`
	Component   string `valid:"Required"`
	Redirect    string
	Path        string `valid:"Required"`
	IsKeepAlive bool   //
	IsHide      bool   // 是否在菜单栏显示，默认显示
	IsAffix     bool   // tag标签是否不可删除
	IsIframe    bool
	Link        string
}
