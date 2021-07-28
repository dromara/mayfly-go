package utils

import (
	"errors"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

func ValidJsonString(schemal, json string) error {
	scheme, jsonLoader := gojsonschema.NewStringLoader(schemal), gojsonschema.NewStringLoader(json)

	result, err := gojsonschema.Validate(scheme, jsonLoader)
	if err != nil {
		return err
	}
	if result.Valid() {
		return nil
	}
	errs := make([]string, 0)
	for _, desc := range result.Errors() {
		errs = append(errs, desc.String())
	}

	return errors.New(strings.Join(errs, "|"))
}
