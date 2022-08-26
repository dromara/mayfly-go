package form

type ConfigForm struct {
	Id     int
	Name   string `binding:"required"`
	Key    string `binding:"required"`
	Value  string
	Remark string `json:"remark"`
}
