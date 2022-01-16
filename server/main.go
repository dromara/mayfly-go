package main

import (
	"mayfly-go/base/starter"
)

func main() {
	starter.PrintBanner()
	starter.InitDb()
	starter.RunWebServer()
}
