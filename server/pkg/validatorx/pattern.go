package validatorx

import (
	"mayfly-go/pkg/global"
	"regexp"

	"github.com/go-playground/validator/v10"
)

const CustomPatternTagName = "pattern"

var (
	regexpMap     map[string]*regexp.Regexp
	patternErrMsg map[string]string
)

// 注册自定义正则表达式校验规则
func RegisterCustomPatterns() {
	// 账号用户名校验
	RegisterPattern("account_username", "^[a-zA-Z0-9_]{5,20}$", "只允许输入5-20位大小写字母、数字、下划线")
}

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
		global.Log.Warnf("%s的正则校验规则不存在!", f.Param())
		return false
	}

	return reg.MatchString(f.Field().String())
}
