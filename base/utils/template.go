package utils

import (
	"bytes"
	"text/template"
)

func parse(t *template.Template, vars interface{}) string {
	var tmplBytes bytes.Buffer

	err := t.Execute(&tmplBytes, vars)
	if err != nil {
		panic(err)
	}
	return tmplBytes.String()
}

// 模板字符串解析
// @param str 模板字符串
// @param vars 参数变量
func TemplateParse(str string, vars interface{}) string {
	tmpl, err := template.New("tmpl").Parse(str)

	if err != nil {
		panic(err)
	}
	return parse(tmpl, vars)
}
