package form

type ConfigForm struct {
	Id     int
	Name   string
	Key    string
	Params string
	Value  string
	Remark string `json:"remark"`
}
