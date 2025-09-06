package config

import "fmt"

const (
	AppName = "mayfly-go"
	Version = "v1.10.3"
)

func GetAppInfo() string {
	return fmt.Sprintf("[%s:%s]", AppName, Version)
}
