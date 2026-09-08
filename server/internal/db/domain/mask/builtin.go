package mask

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"mayfly-go/pkg/errorx"
)

// 算法名常量
const (
	AlgoFull         = "full"         // 全掩码
	AlgoPartial      = "partial"      // 部分保留
	AlgoHash         = "hash"         // 摘要
	AlgoRegexReplace = "regexReplace" // 正则替换
	AlgoPhone        = "phone"        // 手机号预设
	AlgoEmail        = "email"        // 邮箱预设
	AlgoIdcard       = "idcard"       // 身份证预设
	AlgoBankCard     = "bankCard"     // 银行卡预设
)

func init() {
	Register(fullAlgo{})
	Register(partialAlgo{})
	Register(hashAlgo{})
	Register(regexReplaceAlgo{})
	Register(phoneAlgo{})
	Register(emailAlgo{})
	Register(idcardAlgo{})
	Register(bankCardAlgo{})
}

// ErrAlgorithmNotFound 算法未注册
func ErrAlgorithmNotFound(name string) error {
	return errorx.NewBizf("mask algorithm not found: %s", name)
}

// ErrInvalidPattern 规则模式非法
func ErrInvalidPattern(pattern string, err error) error {
	return errorx.NewBizf("invalid mask rule pattern %q: %s", pattern, err.Error())
}

const defaultMaskChar = "*"

// fullAlgo 全掩码：所有字符替换为掩码字符，如 abcdef -> ******
type fullAlgo struct{}

func (fullAlgo) Name() string { return AlgoFull }

func (fullAlgo) Mask(value string, params Params) string {
	maskChar := params.Str("maskChar", defaultMaskChar)
	n := utf8.RuneCountInString(value)
	return strings.Repeat(maskChar, n)
}

// partialAlgo 部分保留：保留前 keepFirst 位与后 keepLast 位，中间以掩码字符填充，如 13800001234 -> 138****1234
type partialAlgo struct{}

func (partialAlgo) Name() string { return AlgoPartial }

func (partialAlgo) Mask(value string, params Params) string {
	// 参数钳位：负值会导致切片越界panic，直接归零
	keepFirst := max(params.Int("keepFirst", 0), 0)
	keepLast := max(params.Int("keepLast", 0), 0)
	maskChar := params.Str("maskChar", defaultMaskChar)

	runes := []rune(value)
	n := len(runes)
	if n == 0 {
		return value
	}
	// 保留位数不小于总长度时不做脱敏，避免出现全掩码后仍可推断长度为0的异常表现
	if keepFirst+keepLast >= n {
		return value
	}
	var sb strings.Builder
	sb.WriteString(string(runes[:keepFirst]))
	sb.WriteString(strings.Repeat(maskChar, n-keepFirst-keepLast))
	sb.WriteString(string(runes[n-keepLast:]))
	return sb.String()
}

// hashAlgo 摘要算法：对值取哈希并截取前 keepLen 位，如 abc123 -> e99a18c4
type hashAlgo struct{}

func (hashAlgo) Name() string { return AlgoHash }

func (hashAlgo) Mask(value string, params Params) string {
	algo := params.Str("algo", "md5")
	keepLen := params.Int("keepLen", 8)

	var sum []byte
	switch strings.ToLower(algo) {
	case "sha1":
		h := sha1.Sum([]byte(value))
		sum = h[:]
	case "sha256":
		h := sha256.Sum256([]byte(value))
		sum = h[:]
	default:
		h := md5.Sum([]byte(value))
		sum = h[:]
	}
	res := hex.EncodeToString(sum)
	if keepLen > 0 && keepLen < len(res) {
		res = res[:keepLen]
	}
	return res
}

// regexCache 正则编译缓存：regexReplace算法的替换正则来自规则参数，
// 查询结果集逐行调用Mask，若每单元格重新编译在大结果集下有严重性能开销
var regexCache sync.Map

// getCompiledRegex 获取已编译正则，编译失败返回nil（调用方降级处理）
func getCompiledRegex(expr string) *regexp.Regexp {
	if v, ok := regexCache.Load(expr); ok {
		if re, ok := v.(*regexp.Regexp); ok {
			return re
		}
		return nil
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		// 缓存编译失败结果，避免每行重复编译失败
		regexCache.Store(expr, (*regexp.Regexp)(nil))
		return nil
	}
	regexCache.Store(expr, re)
	return re
}

// regexReplaceAlgo 正则替换：按正则表达式替换，支持 $1 分组引用保留部分内容
type regexReplaceAlgo struct{}

func (regexReplaceAlgo) Name() string { return AlgoRegexReplace }

func (regexReplaceAlgo) Mask(value string, params Params) string {
	expr := params.Str("regex", "")
	if expr == "" {
		return value
	}
	re := getCompiledRegex(expr)
	if re == nil {
		// 正则非法时降级为全掩码，避免原文泄露
		return fullAlgo{}.Mask(value, Params{})
	}
	replacement := params.Str("replacement", strings.Repeat(defaultMaskChar, 4))
	return re.ReplaceAllString(value, replacement)
}

// phoneAlgo 手机号预设：保留前3后4，如 13800001234 -> 138****1234
type phoneAlgo struct{}

func (phoneAlgo) Name() string { return AlgoPhone }

func (a phoneAlgo) Mask(value string, params Params) string {
	return partialAlgo{}.Mask(value, Params{"keepFirst": 3, "keepLast": 4, "maskChar": params.Str("maskChar", defaultMaskChar)})
}

// idcardAlgo 身份证预设：保留前3后4，如 110101199001011234 -> 110***********1234
type idcardAlgo struct{}

func (idcardAlgo) Name() string { return AlgoIdcard }

func (a idcardAlgo) Mask(value string, params Params) string {
	return partialAlgo{}.Mask(value, Params{"keepFirst": 3, "keepLast": 4, "maskChar": params.Str("maskChar", defaultMaskChar)})
}

// bankCardAlgo 银行卡预设：保留前4后4，如 6222020000112345 -> 6222********2345
type bankCardAlgo struct{}

func (bankCardAlgo) Name() string { return AlgoBankCard }

func (a bankCardAlgo) Mask(value string, params Params) string {
	return partialAlgo{}.Mask(value, Params{"keepFirst": 4, "keepLast": 4, "maskChar": params.Str("maskChar", defaultMaskChar)})
}

// emailAlgo 邮箱预设：本地部分保留首字符，域名完整保留，如 test@qq.com -> t***@qq.com
type emailAlgo struct{}

func (emailAlgo) Name() string { return AlgoEmail }

func (emailAlgo) Mask(value string, params Params) string {
	maskChar := params.Str("maskChar", defaultMaskChar)
	at := strings.LastIndex(value, "@")
	if at <= 0 {
		// 非邮箱格式，降级为部分保留
		return partialAlgo{}.Mask(value, Params{"keepFirst": 1, "maskChar": maskChar})
	}
	local, domain := value[:at], value[at:]
	runes := []rune(local)
	if len(runes) == 1 {
		return string(runes[0]) + strings.Repeat(maskChar, 3) + domain
	}
	return string(runes[0]) + strings.Repeat(maskChar, len(runes)-1) + domain
}

// maskValue 对任意类型值执行脱敏，非字符串类型转为字符串处理后返回字符串
func maskValue(alg Algorithm, params Params, v any) any {
	switch t := v.(type) {
	case nil:
		return nil
	case string:
		if t == "" {
			return t
		}
		return alg.Mask(t, params)
	case []byte:
		if len(t) == 0 {
			return t
		}
		return alg.Mask(string(t), params)
	default:
		return alg.Mask(fmt.Sprintf("%v", t), params)
	}
}
