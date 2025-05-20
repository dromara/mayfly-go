package global

import (
	"mayfly-go/pkg/eventbus"

	"gorm.io/gorm"
)

var (
	Db *gorm.DB // gorm

	EventBus eventbus.Bus[any] = eventbus.New[any]()
)
