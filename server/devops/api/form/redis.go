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

type KeyInfo struct {
	Key   string `binding:"required" json:"key"`
	Timed int64
}

type StringValue struct {
	KeyInfo
	Value interface{} `binding:"required" json:"value"`
}

type HashValue struct {
	KeyInfo
	Value []map[string]interface{} `binding:"required" json:"value"`
}

type SetValue struct {
	KeyInfo
	Value []interface{} `binding:"required" json:"value"`
}
