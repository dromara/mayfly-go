package config

import "fmt"

type App struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

func (a *App) GetAppInfo() string {
	return fmt.Sprintf("[%s:%s]", a.Name, a.Version)
}
