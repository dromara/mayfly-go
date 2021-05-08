package entity

import (
	"mayfly-go/base/model"
)

type Account struct {
	model.Model

	Username string `json:"username"`
	Password string `json:"-"`
	Status   int8   `json:"status"`
}
