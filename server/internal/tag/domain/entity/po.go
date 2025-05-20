package entity

import "time"

type TeamMemberPO struct {
	Id         uint64     `json:"id"`
	TeamId     uint64     `json:"teamId"`
	AccountId  uint64     `json:"accountId"`
	Username   string     `json:"username"`
	Name       string     `json:"name"`
	Creator    string     `json:"creator"`
	CreateTime *time.Time `json:"createTime"`
}
