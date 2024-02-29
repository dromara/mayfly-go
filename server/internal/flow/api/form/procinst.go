package form

type ProcinstTaskAudit struct {
	Id     uint64 `json:"id" binding:"required"`
	Remark string `json:"remark"`
}
