package form

type ConfigForm struct {
	Id     int    `json:"id"`
	Name   string `json:"name"`
	Key    string `json:"key"`
	Params string `json:"params"`
	Value  string `json:"value"`
	Remark string `json:"remark"`
}
