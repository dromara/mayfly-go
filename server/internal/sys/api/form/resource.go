package form

type ResourceForm struct {
	Pid    int            `json:"pid"`
	Id     int            `json:"id"`
	Code   string         `json:"code" binding:"required"`
	Name   string         `json:"name" binding:"required"`
	Type   int            `json:"type" binding:"required,oneof=1 2"`
	Weight int            `json:"weight"`
	Meta   map[string]any `json:"meta"`
}

type MenuResourceMeta struct {
	RouteName   string `json:"routeName" binding:"required"`
	Component   string `json:"component" binding:"required"`
	Redirect    string `json:"redirect"`
	Path        string `json:"path" binding:"required"`
	IsKeepAlive bool   `json:"isKeepAlive"` //
	IsHide      bool   `json:"isHide"`      // 是否在菜单栏显示，默认显示
	IsAffix     bool   `json:"isAffix"`     // tag标签是否不可删除
	IsIframe    bool   `json:"isIframe"`
	Link        string `json:"link"`
}
