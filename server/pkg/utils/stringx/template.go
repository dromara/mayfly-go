package stringx

import (
	"bytes"
	"text/template"
)

func parse(t *template.Template, vars any) (string, error) {
	var tmplBytes bytes.Buffer

	err := t.Execute(&tmplBytes, vars)
	if err != nil {
		return "", err
	}
	return tmplBytes.String(), nil
}

// 模板字符串解析
// @param str 模板字符串
// @param vars 参数变量
func TemplateParse(str string, vars any) (string, error) {
	tmpl, err := template.New("tmpl").Parse(str)

	if err != nil {
		return "", err
	}
	return parse(tmpl, vars)
}
