package jsonx

import (
	"fmt"
	"testing"
)

const jsonStr = `{
	"username": "test",
	"age": 12,
	"person": {
	  "name": {
		"first": "Leonid",
		"last": "Bugaev",
		"fullName": "Leonid Bugaev"
	  },
	  "github": {
		"handle": "buger",
		"followers": 109
	  },
	  "avatars": [
		{ "url": "https://avatars1.githubusercontent.com/u/14009?v=3&s=460", "type": "thumbnail" }
	  ]
	},
	"company": {
	  "name": "Acme"
	}
  }`

func TestGetString(t *testing.T) {
	// val, err := GetString(jsonStr, "username")

	// 错误路径
	// val, err := GetString(jsonStr, "username1")

	// 含有数组的
	val, err := GetString(jsonStr, "person.avatars.0.url")

	if err != nil {
		fmt.Println("error: ", err.Error())
	} else {
		fmt.Println(val)
	}
}

func TestGetInt(t *testing.T) {
	val, _ := GetInt(jsonStr, "age")
	val2, _ := GetInt(jsonStr, "person.github.followers")
	fmt.Println(val, ",", val2)
}
