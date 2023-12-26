package config

import "fmt"

const (
	AppName = "mayfly-go"
	Version = "v1.6.2"
)

func GetAppInfo() string {
	return fmt.Sprintf("[%s:%s]", AppName, Version)
}
