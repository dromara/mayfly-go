package form

type ResourceForm struct {
	Pid    int
	Id     int
	Code   string `binding:"required"`
	Name   string `binding:"required"`
	Type   int    `binding:"required,oneof=1 2"`
	Weight int
	Meta   map[string]interface{}
}

type MenuResourceMeta struct {
	RouteName   string `binding:"required"`
	Component   string `binding:"required"`
	Redirect    string
	Path        string `binding:"required"`
	IsKeepAlive bool   //
	IsHide      bool   // 是否在菜单栏显示，默认显示
	IsAffix     bool   // tag标签是否不可删除
	IsIframe    bool
	Link        string
}
