package config

import "fmt"

const (
	AppName = "mayfly-go"
	Version = "v1.9.0"
)

func GetAppInfo() string {
	return fmt.Sprintf("[%s:%s]", AppName, Version)
}
