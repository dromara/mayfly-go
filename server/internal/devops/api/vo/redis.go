package vo

import "time"

type Redis struct {
	Id *int64 `json:"id"`
	// Name       *string    `json:"name"`
	Host       *string    `json:"host"`
	Db         int        `json:"db"`
	ProjectId  *int64     `json:"projectId"`
	Project    *string    `json:"project"`
	Mode       *string    `json:"mode"`
	Remark     *string    `json:"remark"`
	Env        *string    `json:"env"`
	EnvId      *int64     `json:"envId"`
	CreateTime *time.Time `json:"createTime"`
	Creator    *string    `json:"creator"`
	CreatorId  *int64     `json:"creatorId"`
}

type Keys struct {
	Cursor map[string]uint64 `json:"cursor"`
	Keys   []*KeyInfo        `json:"keys"`
	DbSize int64             `json:"dbSize"`
}

type KeyInfo struct {
	Key  string `json:"key"`
	Ttl  int64  `json:"ttl"`
	Type string `json:"type"`
}
