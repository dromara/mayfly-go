package entity

import "mayfly-go/base/model"

type Mongo struct {
	model.Model

	Name      string `orm:"column(name)" json:"name"`
	Uri       string `orm:"column(uri)" json:"uri"`
	ProjectId uint64 `json:"projectId"`
	Project   string `json:"project"`
	EnvId     uint64 `json:"envId"`
	Env       string `json:"env"`
}
