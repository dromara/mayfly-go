package utils

import (
	"errors"
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

// 从指定路径加载yaml文件
func LoadYml(path string, out interface{}) error {
	yamlFileBytes, readErr := ioutil.ReadFile(path)
	if readErr != nil {
		return readErr
	}
	// yaml解析
	err := yaml.Unmarshal(yamlFileBytes, out)
	if err != nil {
		return errors.New("无法解析 [" + path + "] -- " + err.Error())
	}
	return nil
}

func LoadYmlByString(yamlStr string, out interface{}) error {
	// yaml解析
	return yaml.Unmarshal([]byte(yamlStr), out)
}
