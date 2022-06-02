package form

type Mongo struct {
	Id        uint64
	Uri       string `binding:"required" json:"uri"`
	Name      string `binding:"required" json:"name"`
	ProjectId uint64 `binding:"required" json:"projectId"`
	Project   string `json:"project"`
	Env       string `json:"env"`
	EnvId     uint64 `binding:"required" json:"envId"`
}

type MongoCommand struct {
	Database   string                 `binding:"required" json:"database"`
	Collection string                 `binding:"required" json:"collection"`
	Filter     map[string]interface{} `json:"filter"`
}

type MongoRunCommand struct {
	Database string                 `binding:"required" json:"database"`
	Command  map[string]interface{} `json:"command"`
}

type MongoFindCommand struct {
	MongoCommand
	Sort  map[string]interface{} `json:"sort"`
	Skip  int64
	Limit int64
}

type MongoUpdateByIdCommand struct {
	MongoCommand
	DocId  interface{}            `binding:"required" json:"docId"`
	Update map[string]interface{} `json:"update"`
}

type MongoInsertCommand struct {
	MongoCommand
	Doc map[string]interface{} `json:"doc"`
}
