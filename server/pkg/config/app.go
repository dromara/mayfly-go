package config

import "fmt"

const (
	AppName = "mayfly-go"
	Version = "v1.2.10"
)

func GetAppInfo() string {
	return fmt.Sprintf("[%s:%s]", AppName, Version)
}
