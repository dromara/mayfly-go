package stringx

import (
	"fmt"
	"mayfly-go/pkg/utils/collx"
	"strings"
	"testing"
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

	res, _ := TemplateParse(tmpl, vars)
	res2 := strings.TrimSpace(res)
	fmt.Println(res2)
}
