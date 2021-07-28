package form

type Redis struct {
	Id        uint64
	Host      string `binding:"required" json:"host"`
	Password  string `json:"password"`
	Db        int    `json:"db"`
	ProjectId uint64 `binding:"required" json:"projectId"`
	Project   string `json:"project"`
	Env       string `json:"env"`
	EnvId     uint64 `binding:"required" json:"envId"`
}

type KeyValue struct {
	Key   string      `binding:"required" json:"key"`
	Value interface{} `binding:"required" json:"value"`
	Timed uint64
}
