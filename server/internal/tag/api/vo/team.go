package vo

import "time"

// 团队成员信息
type TeamMember struct {
	Id         uint64     `json:"id"`
	TeamId     uint64     `json:"teamId"`
	AccountId  uint64     `json:"accountId"`
	Username   string     `json:"username"`
	Name       string     `json:"name"`
	Creator    string     `json:"creator"`
	CreateTime *time.Time `json:"createTime"`
}
