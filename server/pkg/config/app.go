package config

import "fmt"

const (
	AppName = "mayfly-go"
	Version = "v1.8.3"
)

func GetAppInfo() string {
	return fmt.Sprintf("[%s:%s]", AppName, Version)
}
