package main

import (
	_ "mayfly-go/internal/auth/init"
	_ "mayfly-go/internal/common/init"
	_ "mayfly-go/internal/db/init"
	_ "mayfly-go/internal/machine/init"
	_ "mayfly-go/internal/mongo/init"
	_ "mayfly-go/internal/msg/init"
	_ "mayfly-go/internal/redis/init"
	_ "mayfly-go/internal/sys/init"
	_ "mayfly-go/internal/tag/init"
	"mayfly-go/pkg/starter"
)

func main() {
	starter.RunWebServer()
}
