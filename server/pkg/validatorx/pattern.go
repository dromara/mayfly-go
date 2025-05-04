package validatorx

import (
	"mayfly-go/pkg/logx"
	"regexp"

	"github.com/go-playground/validator/v10"
)

const CustomPatternTagName = "pattern"

var (
	regexpMap     map[string]*regexp.Regexp
	patternErrMsg map[string]string
)

// 注册自定义正则表达式
func RegisterPattern(patternName string, regexpStr string, errMsg string) {
	if regexpMap == nil {
		regexpMap = make(map[string]*regexp.Regexp, 0)
		patternErrMsg = make(map[string]string)
	}
	regexpMap[patternName] = regexp.MustCompile(regexpStr)
	patternErrMsg[patternName] = errMsg
}

func patternValidFunc(f validator.FieldLevel) bool {
	reg := regexpMap[f.Param()]
	if reg == nil {
		logx.Warnf("%s的正则校验规则不存在!", f.Param())
		return false
	}

	return reg.MatchString(f.Field().String())
}
