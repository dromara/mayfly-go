package config

import "fmt"

const (
	AppName = "mayfly-go"
	Version = "v1.5.2"
)

func GetAppInfo() string {
	return fmt.Sprintf("[%s:%s]", AppName, Version)
}
