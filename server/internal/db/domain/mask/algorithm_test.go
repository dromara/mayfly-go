package mask

import (
	"errors"
	"testing"

	"mayfly-go/pkg/errorx"
)

func TestFullAlgo(t *testing.T) {
	a := fullAlgo{}
	if got := a.Mask("abc", Params{}); got != "***" {
		t.Fatalf("expect ***, got %s", got)
	}
	// 多字节字符按rune处理
	if got := a.Mask("张三丰", Params{}); got != "***" {
		t.Fatalf("expect ***, got %s", got)
	}
	if got := a.Mask("abc", Params{"maskChar": "#"}); got != "###" {
		t.Fatalf("expect ###, got %s", got)
	}
}

func TestPartialAlgo(t *testing.T) {
	a := partialAlgo{}
	if got := a.Mask("13800001234", Params{"keepFirst": 3, "keepLast": 4}); got != "138****1234" {
		t.Fatalf("expect 138****1234, got %s", got)
	}
	// 空串
	if got := a.Mask("", Params{"keepFirst": 3, "keepLast": 4}); got != "" {
		t.Fatalf("expect empty, got %s", got)
	}
	// 超短串：保留位覆盖全部时不脱敏
	if got := a.Mask("abc", Params{"keepFirst": 2, "keepLast": 2}); got != "abc" {
		t.Fatalf("expect abc, got %s", got)
	}
	// 多字节
	if got := a.Mask("张三丰是个人名", Params{"keepFirst": 1, "keepLast": 1}); got != "张*****名" {
		t.Fatalf("expect 张*****名, got %s", got)
	}
}

func TestHashAlgo(t *testing.T) {
	a := hashAlgo{}
	got := a.Mask("abc123", Params{"keepLen": 8})
	if len(got) != 8 {
		t.Fatalf("expect 8 chars, got %s", got)
	}
	// 相同输入结果稳定
	if got != a.Mask("abc123", Params{"keepLen": 8}) {
		t.Fatal("hash should be deterministic")
	}
	// sha256
	got256 := a.Mask("abc123", Params{"algo": "sha256", "keepLen": 16})
	if len(got256) != 16 {
		t.Fatalf("expect 16 chars, got %s", got256)
	}
}

func TestPhoneEmailIdcardBankCard(t *testing.T) {
	if got := (phoneAlgo{}).Mask("13800001234", Params{}); got != "138****1234" {
		t.Fatalf("expect 138****1234, got %s", got)
	}
	if got := (idcardAlgo{}).Mask("110101199001011234", Params{}); got != "110***********1234" {
		t.Fatalf("expect 110***********1234, got %s", got)
	}
	if got := (bankCardAlgo{}).Mask("6222020000112345", Params{}); got != "6222********2345" {
		t.Fatalf("expect 6222********2345, got %s", got)
	}
	if got := (emailAlgo{}).Mask("test@qq.com", Params{}); got != "t***@qq.com" {
		t.Fatalf("expect t***@qq.com, got %s", got)
	}
	// 单字符本地部分
	if got := (emailAlgo{}).Mask("a@163.com", Params{}); got != "a***@163.com" {
		t.Fatalf("expect a***@163.com, got %s", got)
	}
	// 非邮箱格式降级
	if got := (emailAlgo{}).Mask("not-an-email", Params{}); got != "n***********" {
		t.Fatalf("expect n***********, got %s", got)
	}
}

func TestRegexReplaceAlgo(t *testing.T) {
	a := regexReplaceAlgo{}
	// 保留分组
	got := a.Mask("13800001234", Params{"regex": `(\d{3})\d{4}(\d{4})`, "replacement": "$1****$2"})
	if got != "138****1234" {
		t.Fatalf("expect 138****1234, got %s", got)
	}
	// 非法正则降级全掩码
	if got := a.Mask("abc", Params{"regex": "("}); got == "abc" {
		t.Fatal("invalid regex should mask value")
	}
	// 空正则原样返回
	if got := a.Mask("abc", Params{}); got != "abc" {
		t.Fatalf("expect abc, got %s", got)
	}
}

func TestRegistry(t *testing.T) {
	for _, name := range []string{AlgoFull, AlgoPartial, AlgoHash, AlgoRegexReplace, AlgoPhone, AlgoEmail, AlgoIdcard, AlgoBankCard} {
		if !Exists(name) {
			t.Fatalf("builtin algorithm %s should be registered", name)
		}
		if _, err := Get(name); err != nil {
			t.Fatalf("get algorithm %s failed: %s", name, err.Error())
		}
	}
	if _, err := Get("not-exist"); err == nil {
		t.Fatal("expect not found error")
	} else {
		var bizErr *errorx.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expect biz error, got %v", err)
		}
	}
	if len(Names()) < 8 {
		t.Fatalf("expect at least 8 algorithms, got %v", Names())
	}
}

func TestMaskValue(t *testing.T) {
	alg, _ := Get(AlgoPhone)
	// nil跳过
	if got := maskValue(alg, Params{}, nil); got != nil {
		t.Fatalf("expect nil, got %v", got)
	}
	if got := maskValue(alg, Params{}, "13800001234"); got != "138****1234" {
		t.Fatalf("expect 138****1234, got %v", got)
	}
	// 非字符串转字符串处理
	if got := maskValue(alg, Params{}, int64(13800001234)); got != "138****1234" {
		t.Fatalf("expect 138****1234, got %v", got)
	}
	// 空串保持
	if got := maskValue(alg, Params{}, ""); got != "" {
		t.Fatalf("expect empty, got %v", got)
	}
}

// TestPartialAlgoNegativeParams 负数参数不应panic且等价于0
func TestPartialAlgoNegativeParams(t *testing.T) {
	alg, _ := Get(AlgoPartial)
	got := alg.Mask("13800001234", Params{"keepFirst": -3, "keepLast": -1})
	if got != "***********" {
		t.Fatalf("expect all masked, got %q", got)
	}
	// 负maskChar参数也不影响
	got2 := alg.Mask("abc", Params{"keepFirst": 1, "keepLast": -5})
	if got2 != "a**" {
		t.Fatalf("expect a**, got %q", got2)
	}
}

// TestRegexReplaceAlgoInvalidRegex 非法正则降级为全掩码且结果稳定（编译结果被缓存）
func TestRegexReplaceAlgoInvalidRegex(t *testing.T) {
	alg, _ := Get(AlgoRegexReplace)
	invalid := Params{"regex": "("}
	first := alg.Mask("secret", invalid)
	if first != "******" {
		t.Fatalf("expect full mask fallback, got %q", first)
	}
	second := alg.Mask("secret2", invalid)
	if second != "*******" {
		t.Fatalf("expect stable cached fallback, got %q", second)
	}
	// 合法正则正常替换且分组引用可用
	valid := Params{"regex": `(\d{3})\d{4}(\d{4})`, "replacement": "$1****$2"}
	if got := alg.Mask("13800001234", valid); got != "138****1234" {
		t.Fatalf("expect 138****1234, got %q", got)
	}
}
