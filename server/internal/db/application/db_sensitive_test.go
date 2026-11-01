package application

// 敏感列分类的判据：既要认出常见敏感命名，也不能把订单号、金额这类业务列误标成敏感。
// 误标一次，运维就会把整份扫描结论当噪音，功能等于没有。

import "testing"

func TestClassifySensitiveColumn(t *testing.T) {
	cases := []struct {
		column     string
		columnType string
		category   string
		level      string
	}{
		{"phone", "varchar(64)", "phone", "high"},
		{"user_phone", "varchar(64)", "phone", "high"},
		{"mobile", "varchar(64)", "phone", "high"},
		{"contact_tel", "varchar(64)", "phone", "high"},
		{"id_card", "varchar(64)", "idcard", "high"},
		{"identity_no", "varchar(64)", "idcard", "high"},
		{"password", "varchar(64)", "credential", "high"},
		{"api_secret", "varchar(64)", "credential", "high"},
		{"access_key", "varchar(64)", "credential", "high"},
		{"bank_card", "varchar(64)", "bankcard", "high"},
		{"card_no", "varchar(64)", "bankcard", "high"},
		{"email", "varchar(64)", "email", "medium"},
		{"real_name", "varchar(64)", "name", "medium"},
		{"username", "varchar(64)", "name", "medium"},
		{"home_address", "varchar(64)", "address", "medium"},
		{"zip_code", "varchar(64)", "address", "medium"},
		{"birthday", "varchar(64)", "other", "medium"},
		// 业务列不得误标
		{"order_id", "varchar(64)", "", ""},
		{"amount", "varchar(64)", "", ""},
		{"status", "varchar(64)", "", ""},
		{"create_time", "varchar(64)", "", ""},
		{"card_id", "varchar(64)", "", ""},
	}
	for _, tc := range cases {
		category, level := classifySensitiveColumn(tc.column, tc.columnType)
		if category != tc.category || level != tc.level {
			t.Fatalf("classifySensitiveColumn(%q, %q) = (%q, %q), want (%q, %q)", tc.column, tc.columnType, category, level, tc.category, tc.level)
		}
	}

	// 数值列收紧：末段就是敏感词才算（真实误报来自 t_aiops_run.token_budget bigint）
	numeric := []struct {
		column     string
		columnType string
		category   string
	}{
		{"token_budget", "bigint", ""},
		{"secret_count", "bigint", ""},
		{"phone", "bigint", "phone"},
		{"user_phone", "decimal(20,0)", "phone"},
		{"card_no", "bigint", "bankcard"},
		{"amount", "bigint", ""},
	}
	for _, tc := range numeric {
		category, _ := classifySensitiveColumn(tc.column, tc.columnType)
		if category != tc.category {
			t.Fatalf("numeric %q(%s) => %q, want %q", tc.column, tc.columnType, category, tc.category)
		}
	}
}
