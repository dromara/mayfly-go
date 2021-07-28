package utils

import (
	"fmt"
	"testing"

	"github.com/xeipuuv/gojsonschema"
)

func TestJsonSchemal(t *testing.T) {
	schema := `{
		"$schema": "http://json-schema.org/draft-04/schema#",
		"title": "Product",
		"description": "A product from Acme's catalog",
		"type": "object",
		"properties": {
			"id": {
				"description": "The unique identifier for a product",
				"type": "integer"
			},
			"name": {
				"description": "Name of the product",
				"type": "string"
			},
			"price": {
				"type": "number",
				"minimum": 0,
				"exclusiveMinimum": true
			}
		},
		"required": ["id", "name", "price"]
	}
	`

	json := `{"id": 1, "name": "test", "price": -21}`

	err := ValidJsonString(schema, json)
	fmt.Print(err)
}

func TestJs(t *testing.T) {
	schemaLoader := gojsonschema.NewStringLoader(`{"type": "object","properties":{"a":{"type":"object"}},"required":["a"]}`) // json格式
	documentLoader := gojsonschema.NewStringLoader(`{"a":"b"}`)                                                              // 待校验的json数据

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		panic(err.Error())
	}

	if result.Valid() {
		fmt.Printf("The document is valid\n")
	} else {
		fmt.Printf("The document is not valid. see errors :\n")
		for _, desc := range result.Errors() {
			fmt.Printf("- %s\n", desc)
		}
	}
}
