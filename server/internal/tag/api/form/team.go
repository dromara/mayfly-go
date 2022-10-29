package form

type TeamMember struct {
	TeamId     uint64   `json:"teamId"`
	AccountIds []uint64 `json:"accountIds"`
}
