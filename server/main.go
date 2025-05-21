package main

import (
	_ "mayfly-go/internal/auth/init"
	_ "mayfly-go/internal/common/init"
	_ "mayfly-go/internal/db/init"
	_ "mayfly-go/internal/es/init"
	_ "mayfly-go/internal/file/init"
	_ "mayfly-go/internal/flow/init"
	_ "mayfly-go/internal/machine/init"
	_ "mayfly-go/internal/mongo/init"
	_ "mayfly-go/internal/msg/init"
	"mayfly-go/internal/pkg/starter"
	_ "mayfly-go/internal/redis/init"
	_ "mayfly-go/internal/sys/init"
	_ "mayfly-go/internal/tag/init"
)

func main() {
	starter.RunWebServer()
}
