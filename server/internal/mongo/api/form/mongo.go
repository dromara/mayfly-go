package form

type Mongo struct {
	Id                 uint64   `json:"id"`
	Uri                string   `binding:"required" json:"uri"`
	SshTunnelMachineId int      `json:"sshTunnelMachineId"` // ssh隧道机器id
	Name               string   `binding:"required" json:"name"`
	TagId              []uint64 `binding:"required" json:"tagId"`
}

type MongoCommand struct {
	Database   string         `binding:"required" json:"database"`
	Collection string         `binding:"required" json:"collection"`
	Filter     map[string]any `json:"filter"`
}

type MongoRunCommand struct {
	Database string           `binding:"required" json:"database"`
	Command  []map[string]any `json:"command"`
}

type MongoFindCommand struct {
	MongoCommand
	Sort  map[string]any `json:"sort"`
	Skip  int64          `json:"skip"`
	Limit int64          `json:"limit"`
}

type MongoUpdateByIdCommand struct {
	MongoCommand
	DocId  any            `binding:"required" json:"docId"`
	Update map[string]any `json:"update"`
}

type MongoInsertCommand struct {
	MongoCommand
	Doc map[string]any `json:"doc"`
}
