package entity

type SysLogQuery struct {
	CreatorId   uint64 `json:"creatorId" form:"creatorId"`
	Type        int8   `json:"type" form:"type"`
	Description string `json:"description" form:"description"`
}
