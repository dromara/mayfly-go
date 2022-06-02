package main

import (
	"mayfly-go/pkg/starter"
)

func main() {
	starter.PrintBanner()
	starter.InitDb()
	starter.RunWebServer()
}
