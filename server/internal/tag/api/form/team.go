package form

type TeamMember struct {
	TeamId     uint64   `json:"teamId" binding:"required"`
	AccountIds []uint64 `json:"accountIds"`
}
