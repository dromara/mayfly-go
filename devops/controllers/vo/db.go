package vo

import "time"

type SelectDataDbVO struct {
	//models.BaseModel
	Id         *int64     `json:"id"`
	Name       *string    `json:"name"`
	Host       *string    `json:"host"`
	Port       *int       `json:"port"`
	Type       *string    `json:"type"`
	Database   *string    `json:"database"`
	CreateTime *time.Time `json:"createTime"`
	Creator    *string    `json:"creator"`
	CreatorId  *int64     `json:"creatorId"`
}
