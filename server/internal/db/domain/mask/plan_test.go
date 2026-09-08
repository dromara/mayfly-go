package mask

import (
	"testing"
)

func mustAlg(t *testing.T, name string) Algorithm {
	t.Helper()
	alg, err := Get(name)
	if err != nil {
		t.Fatalf("get algorithm %s failed: %s", name, err.Error())
	}
	return alg
}

func testPlan(t *testing.T) *Plan {
	t.Helper()
	rules := []*Rule{
		{Name: "手机号", MatchType: MatchTypeRegex, Pattern: `(?i)(phone|mobile|tel)`, Algorithm: mustAlg(t, AlgoPhone)},
		{Name: "邮箱", MatchType: MatchTypeRegex, Pattern: `(?i)email`, Algorithm: mustAlg(t, AlgoEmail)},
		{Name: "精确姓名", MatchType: MatchTypeExact, Pattern: `user_name`, Algorithm: mustAlg(t, AlgoFull)},
		{Name: "前缀证件", MatchType: MatchTypePrefix, Pattern: `cert_`, Algorithm: mustAlg(t, AlgoIdcard)},
	}
	tags := []*ColumnTag{
		// 特异性最高的绑定标签：t_user.phone 用hash而非phone
		{DbName: "app", TableName: "t_user", ColumnName: "phone", Action: TagActionBind, Algorithm: mustAlg(t, AlgoHash)},
		// 全表豁免
		{TableName: "t_log", Action: TagActionExempt},
		// 精确豁免，覆盖正则误命中
		{TableName: "t_user", ColumnName: "telephone_ext", Action: TagActionExempt},
	}
	p, err := NewPlan(rules, tags)
	if err != nil {
		t.Fatalf("new plan failed: %s", err.Error())
	}
	return p
}

func TestPlanRuleMatching(t *testing.T) {
	p := testPlan(t)

	// 正则命中
	if alg, _ := p.Resolve("app", "t_user", "mobile"); alg == nil || alg.Name() != AlgoPhone {
		t.Fatalf("expect phone algo, got %v", alg)
	}
	// 精确规则优先于正则
	if alg, _ := p.Resolve("app", "t_user", "user_name"); alg.Name() != AlgoFull {
		t.Fatalf("expect full algo, got %s", alg.Name())
	}
	// 前缀命中
	if alg, _ := p.Resolve("app", "t_user", "cert_no"); alg.Name() != AlgoIdcard {
		t.Fatalf("expect idcard algo, got %s", alg.Name())
	}
	// 未命中
	if alg, _ := p.Resolve("app", "t_user", "address"); alg != nil {
		t.Fatalf("expect no match, got %s", alg.Name())
	}
	// 列名大小写不敏感（前缀）
	if alg, _ := p.Resolve("app", "t_user", "CERT_CODE"); alg == nil {
		t.Fatal("expect prefix match case-insensitive")
	}
}

func TestPlanTagPriority(t *testing.T) {
	p := testPlan(t)

	// 高特异性绑定标签覆盖全局规则
	if alg, _ := p.Resolve("app", "t_user", "phone"); alg.Name() != AlgoHash {
		t.Fatalf("expect hash algo by bind tag, got %s", alg.Name())
	}
	// 精确豁免标签覆盖正则规则
	if alg, _ := p.Resolve("app", "t_user", "telephone_ext"); alg != nil {
		t.Fatalf("expect exempt, got %s", alg.Name())
	}
	// 整表豁免覆盖一切规则
	if alg, _ := p.Resolve("app", "t_log", "mobile"); alg != nil {
		t.Fatalf("expect table exempt, got %s", alg.Name())
	}
	// 标签库名不匹配时不生效
	if alg, _ := p.Resolve("other_db", "t_user", "phone"); alg.Name() != AlgoPhone {
		t.Fatalf("expect phone algo for other db, got %s", alg.Name())
	}
}

func TestPlanRegexCompileError(t *testing.T) {
	// 正则非法且算法已设置时应返回编译错误
	if _, err := NewPlan([]*Rule{{MatchType: MatchTypeRegex, Pattern: "(", Algorithm: mustAlg(t, AlgoFull)}}, nil); err == nil {
		t.Fatal("expect invalid pattern error")
	}
	// 未设置算法的规则会被跳过，非法正则也不会被编译
	if _, err := NewPlan([]*Rule{{MatchType: MatchTypeRegex, Pattern: "("}}, nil); err != nil {
		t.Fatalf("expect nil algorithm rule skipped, got %s", err.Error())
	}
}

// TestPlanTagCrossPriority 验证标签合并排序：特异性高者优先，同分豁免优先
func TestPlanTagCrossPriority(t *testing.T) {
	hash := mustAlg(t, AlgoHash)
	// 低特异性绑定（仅列名，分4） vs 高特异性豁免（库+表+列，分7）：豁免应胜出
	p, err := NewPlan(nil, []*ColumnTag{
		{ColumnName: "phone", Action: TagActionBind, Algorithm: hash},
		{DbName: "app", TableName: "t_user", ColumnName: "phone", Action: TagActionExempt},
	})
	if err != nil {
		t.Fatal(err)
	}
	if alg, _ := p.Resolve("app", "t_user", "phone"); alg != nil {
		t.Fatal("expect exempt by higher specificity")
	}

	// 低特异性豁免（仅列名，分4） vs 高特异性绑定（库+表+列，分7）：绑定应胜出
	p2, err := NewPlan(nil, []*ColumnTag{
		{ColumnName: "phone", Action: TagActionExempt},
		{DbName: "app", TableName: "t_user", ColumnName: "phone", Action: TagActionBind, Algorithm: hash},
	})
	if err != nil {
		t.Fatal(err)
	}
	if alg, _ := p2.Resolve("app", "t_user", "phone"); alg == nil || alg.Name() != AlgoHash {
		t.Fatalf("expect hash by higher specificity bind, got %v", alg)
	}

	// 同特异性时豁免优先
	p3, err := NewPlan(nil, []*ColumnTag{
		{TableName: "t_user", ColumnName: "phone", Action: TagActionBind, Algorithm: hash},
		{TableName: "t_user", ColumnName: "phone", Action: TagActionExempt},
	})
	if err != nil {
		t.Fatal(err)
	}
	if alg, _ := p3.Resolve("app", "t_user", "phone"); alg != nil {
		t.Fatal("expect exempt wins on equal specificity")
	}
}

// TestExactRuleCaseInsensitive 验证精确规则大小写不敏感（Oracle/PG结果列大写场景）
func TestExactRuleCaseInsensitive(t *testing.T) {
	full := mustAlg(t, AlgoFull)
	p, err := NewPlan([]*Rule{
		{MatchType: MatchTypeExact, Pattern: "user_name", Algorithm: full},
		{MatchType: MatchTypeExact, Pattern: "USER_NAME", Algorithm: mustAlg(t, AlgoHash)},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if alg, _ := p.Resolve("", "", "USER_NAME"); alg.Name() != AlgoFull {
		t.Fatalf("expect case-insensitive exact match first-declared, got %s", alg.Name())
	}
	if alg, _ := p.Resolve("", "", "User_Name"); alg.Name() != AlgoFull {
		t.Fatalf("expect case-insensitive exact match, got %s", alg.Name())
	}
}

func TestRowMasker(t *testing.T) {
	p := testPlan(t)
	marked := make(map[string]bool)
	m := NewRowMasker(p, "app", []string{"t_user"}, nil, []string{"phone", "mobile", "address", "telephone_ext"},
		map[string]string{"p": "phone"}, func(col string) { marked[col] = true })
	if m == nil {
		t.Fatal("expect non-nil masker")
	}
	// 别名列也可通过映射脱敏
	m2 := NewRowMasker(p, "app", []string{"t_user"}, nil, []string{"p"}, map[string]string{"p": "phone"}, nil)

	row := map[string]any{"phone": "13800001234", "mobile": "13900002345", "address": "北京市", "telephone_ext": "0101"}
	m.MaskRow(row)
	// phone 命中特异性更高的绑定标签，使用hash算法
	if row["phone"] == "13800001234" || len(row["phone"].(string)) != 8 {
		t.Fatalf("expect hashed phone, got %v", row["phone"])
	}
	if row["mobile"] != "139****2345" {
		t.Fatalf("expect 139****2345, got %v", row["mobile"])
	}
	if row["address"] != "北京市" {
		t.Fatalf("expect unchanged, got %v", row["address"])
	}
	if row["telephone_ext"] != "0101" {
		t.Fatalf("expect exempt unchanged, got %v", row["telephone_ext"])
	}
	if !marked["phone"] || !marked["mobile"] || marked["address"] {
		t.Fatalf("unexpected masked marks: %v", marked)
	}

	row2 := map[string]any{"p": "13800001234"}
	m2.MaskRow(row2)
	// 别名列应被脱敏为hash而非原文
	if row2["p"] == "13800001234" {
		t.Fatalf("expect masked alias column, got %v", row2["p"])
	}
}

func TestRowMaskerNilPlan(t *testing.T) {
	if m := NewRowMasker(nil, "app", nil, nil, []string{"phone"}, nil, nil); m != nil {
		t.Fatal("nil plan should return nil masker")
	}
	// 空计划返回nil
	if m := NewRowMasker(&Plan{}, "app", nil, nil, []string{"phone"}, nil, nil); m != nil {
		t.Fatal("empty plan should return nil masker")
	}
}

// TestRowMaskerTableOf 验证限定名逐表精确归属：tableOf键存在即锁定归属，
// 指定表未命中标签时不回退到其他表；值为空串表示确定无来源表，仅全局规则解析
func TestRowMaskerTableOf(t *testing.T) {
	hash := mustAlg(t, AlgoHash)
	phone := mustAlg(t, AlgoPhone)
	// t_user.phone绑定hash标签；t_order无标签；全局正则规则phone→phone算法
	p, err := NewPlan([]*Rule{{MatchType: MatchTypeRegex, Pattern: "(?i)phone", Algorithm: phone}}, []*ColumnTag{
		{TableName: "t_user", ColumnName: "phone", Action: TagActionBind, Algorithm: hash},
	})
	if err != nil {
		t.Fatal(err)
	}
	marked := make(map[string]bool)
	// op显式归属t_order：不命中t_user的hash标签，也不应误用t_user标签 → 走全局规则phone算法；
	// ep锁定为无来源表（空串）：仅全局规则，不逐表尝试命中t_user hash；
	// up未指定归属（键不存在），按tables顺序命中t_user标签
	m := NewRowMasker(p, "app", []string{"t_user", "t_order"}, map[string]string{"op": "t_order", "ep": ""},
		[]string{"up", "op", "ep"}, map[string]string{"up": "phone", "op": "phone", "ep": "phone"}, func(col string) { marked[col] = true })
	row := map[string]any{"up": "13800001234", "op": "13900002345", "ep": "13700003456"}
	m.MaskRow(row)
	if len(row["up"].(string)) != 8 {
		t.Fatalf("expect up hashed by t_user tag, got %v", row["up"])
	}
	if row["op"] != "139****2345" {
		t.Fatalf("expect op phone-algo via global rule (t_order has no tag, no fallback to t_user), got %v", row["op"])
	}
	if row["ep"] != "137****3456" {
		t.Fatalf("expect ep locked to empty-table context (global rule only, no per-table fallback), got %v", row["ep"])
	}
	if !marked["up"] || !marked["op"] || !marked["ep"] {
		t.Fatalf("unexpected masked marks: %v", marked)
	}
}

func TestRowMaskerNilAndNonString(t *testing.T) {
	p, err := NewPlan([]*Rule{{MatchType: MatchTypeExact, Pattern: "phone", Algorithm: mustAlg(t, AlgoPhone)}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m := NewRowMasker(p, "", nil, nil, []string{"phone"}, nil, nil)

	// nil值保持nil
	row := map[string]any{"phone": nil}
	m.MaskRow(row)
	if row["phone"] != nil {
		t.Fatalf("expect nil, got %v", row["phone"])
	}
	// 非字符串转字符串处理
	row3 := map[string]any{"phone": int64(13800001234)}
	m.MaskRow(row3)
	if row3["phone"] != "138****1234" {
		t.Fatalf("expect 138****1234, got %v", row3["phone"])
	}
	// map中不存在的列key不会panic
	m.MaskRow(map[string]any{})
}
