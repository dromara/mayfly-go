package config

import "fmt"

const (
	AppName = "mayfly-go"
	Version = "v1.8.9"
)

func GetAppInfo() string {
	return fmt.Sprintf("[%s:%s]", AppName, Version)
}
