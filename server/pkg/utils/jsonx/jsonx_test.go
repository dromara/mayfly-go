package jsonx

import (
	"fmt"
	"testing"

	"github.com/buger/jsonparser"
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
	val, err := GetString(jsonStr, "person.avatars.[0].url")

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

// 官方demo
func TestJsonParser(t *testing.T) {
	data := []byte(jsonStr)
	// You can specify key path by providing arguments to Get function
	jsonparser.Get(data, "person", "name", "fullName")

	// There is `GetInt` and `GetBoolean` helpers if you exactly know key data type
	jsonparser.GetInt(data, "person", "github", "followers")

	// When you try to get object, it will return you []byte slice pointer to data containing it
	// In `company` it will be `{"name": "Acme"}`
	jsonparser.Get(data, "company")

	// If the key doesn't exist it will throw an error
	var size int64
	if value, err := jsonparser.GetInt(data, "company", "size"); err == nil {
		size = value
		fmt.Println(size)
	}

	// You can use `ArrayEach` helper to iterate items [item1, item2 .... itemN]
	jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		fmt.Println(jsonparser.Get(value, "url"))
	}, "person", "avatars")

	// Or use can access fields by index!
	jsonparser.GetString(data, "person", "avatars", "[0]", "url")

	// You can use `ObjectEach` helper to iterate objects { "key1":object1, "key2":object2, .... "keyN":objectN }
	jsonparser.ObjectEach(data, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		fmt.Printf("Key: '%s'\n Value: '%s'\n Type: %s\n", string(key), string(value), dataType)
		return nil
	}, "person", "name")

	// The most efficient way to extract multiple keys is `EachKey`

	paths := [][]string{
		[]string{"person", "name", "fullName"},
		[]string{"person", "avatars", "[0]", "url"},
		[]string{"company", "url"},
	}

	jsonparser.EachKey(data, func(idx int, value []byte, vt jsonparser.ValueType, err error) {
		switch idx {
		case 0: // []string{"person", "name", "fullName"}
			{
			}
		case 1: // []string{"person", "avatars", "[0]", "url"}
			{
			}
		case 2: // []string{"company", "url"},
			{
			}
		}
	}, paths...)
}
