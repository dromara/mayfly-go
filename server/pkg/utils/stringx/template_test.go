package stringx

import (
	"fmt"
	"mayfly-go/pkg/utils/collx"
	"strings"
	"testing"

	"github.com/may-fly/cast"
)

func TestTemplateParse(t *testing.T) {
	tmpl := `
	{{if gt .cpu 10*5}}
		当前服务器[{{.asset.host}}]cpu使用率为{{.cpu}}
	{{end}}
	`
	vars := collx.M{
		"cpu": 60,
		"asset": collx.M{
			"host": "localhost:121",
		},
	}

	num1 := 12
	num2 := 12
	num3 := 2
	templ2 := "{{ eq .num1 121 }}"
	re, err := TemplateParse(templ2, collx.M{"num1": num1, "num2": num2, "num3": num3})

	fmt.Println(err, cast.ToBool(re))

	res, _ := TemplateParse(tmpl, vars)
	res2 := strings.TrimSpace(res)
	fmt.Println(res2)
}
