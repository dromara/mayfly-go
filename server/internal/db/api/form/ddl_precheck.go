package form

// DdlPrecheck 预检入参：传当前编辑器里的整段脚本，切分与分类都由后端完成
type DdlPrecheck struct {
	Db  string `json:"db" validate:"required"`
	Sql string `json:"sql" validate:"required"`
}
