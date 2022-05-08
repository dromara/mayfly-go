package form

type DbForm struct {
	Id        uint64
	Name      string `binding:"required" json:"name"`
	Type      string `binding:"required" json:"type"` // 类型，mysql oracle等
	Host      string `binding:"required" json:"host"`
	Port      int    `binding:"required" json:"port"`
	Username  string `binding:"required" json:"username"`
	Password  string `json:"password"`
	Database  string `binding:"required" json:"database"`
	ProjectId uint64 `binding:"required" json:"projectId"`
	Project   string `json:"project"`
	Env       string `json:"env"`
	EnvId     uint64 `binding:"required" json:"envId"`
}
