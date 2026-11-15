//go:build golden

package sqldraft

// 真实准确率评测（不进默认门禁）：真实模型 + 真实库 + 真的执行。
//
// 与 golden_test.go 的分工：那边用录制回放锁「判定不漂移」，这边回答「准确率是多少」——
// 模型现算、SQL 真跑、结果行数与形状比对。两者缺一不可：只跑回放会与真实模型脱节，
// 只跑真机则无法在 CI 回归。
//
// 运行方式（三个环境变量都指向本机起着的实例即可）：
//
//	export MAYFLY_GOLDEN_API=http://127.0.0.1:18888
//	export MAYFLY_GOLDEN_USER=admin MAYFLY_GOLDEN_PASS=admin123
//	export MAYFLY_GOLDEN_DB_ID=1 MAYFLY_GOLDEN_DB=mayfly_go
//	export MAYFLY_GOLDEN_DSN='root:111049@tcp(127.0.0.1:3306)/mayfly_go?parseTime=true'
//	go test -tags golden -count=1 -run TestGoldenLive -v ./internal/db/application/sqldraft/
//
// 判定口径（三项分别统计，不做加权）：
//   - verifiedRate：结构与权限校验通过的比例；
//   - execRate：能真实执行且不报错的比例（写语句不执行，记为 n/a）；
//   - nonEmptyRate：只读语句返回至少一行数据的比例（0 行往往意味着取值/过滤语义错了）。
//
// 模型有概率性，单次失败不等于回归；要看趋势就把同一份用例多跑几轮取比例。

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// liveCase 真实评测用例（testdata/golden_live_cases.json）
type liveCase struct {
	ID     string `json:"id"`
	Intent string `json:"intent"`
	Prompt string `json:"prompt"`
	Anchor string `json:"anchor"`
	Expect struct {
		// Verification 期望的校验结论（空表示不约束）
		Verification string `json:"verification"`
		// MinRows 只读语句执行后至少要返回多少行（0 表示不约束）
		MinRows int `json:"minRows"`
		// RequireSubstrings / ForbidSubstrings 对生成 SQL 的确定性断言（不区分大小写）。
		// 用于「改写类」用例：结果行数依赖数据，但「函数是否被展开成范围条件」与数据无关。
		RequireSubstrings []string `json:"requireSubstrings"`
		ForbidSubstrings  []string `json:"forbidSubstrings"`
	} `json:"expect"`
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// loginToken 真实登录拿 token：登录接口要求 RSA-OAEP(SHA-256) 加密口令，
// 与前端同一套公钥流程，避免手工粘贴 token 过期后整套评测失效。
func loginToken(t *testing.T, api, username, password string) string {
	t.Helper()
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(api + "/api/common/public-key")
	if err != nil {
		t.Fatalf("get public key: %v", err)
	}
	defer resp.Body.Close()
	var keyResp struct {
		Data string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&keyResp); err != nil {
		t.Fatalf("decode public key: %v", err)
	}
	block, _ := pem.Decode([]byte(keyResp.Data))
	if block == nil {
		t.Fatalf("public key is not PEM: %q", keyResp.Data)
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatalf("parse public key: %v", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		t.Fatalf("public key is not RSA")
	}
	encrypted, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPub, []byte(password), nil)
	if err != nil {
		t.Fatalf("encrypt password: %v", err)
	}
	body, _ := json.Marshal(map[string]string{
		"username": username,
		"password": base64.StdEncoding.EncodeToString(encrypted),
	})
	loginResp, err := client.Post(api+"/api/auth/accounts/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer loginResp.Body.Close()
	raw, _ := io.ReadAll(loginResp.Body)
	var parsed struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("decode login: %v (%s)", err, string(raw))
	}
	if parsed.Code != 200 {
		t.Fatalf("login failed: code=%d msg=%s", parsed.Code, parsed.Msg)
	}
	// 不同版本 data 可能是对象（含 token）或字符串
	var asObject struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(parsed.Data, &asObject); err == nil && asObject.Token != "" {
		return asObject.Token
	}
	var asString string
	if err := json.Unmarshal(parsed.Data, &asString); err == nil && asString != "" {
		return asString
	}
	t.Fatalf("login response carries no token: %s", string(parsed.Data))
	return ""
}

// draftViaAPI 调真实接口生成一份草稿
func draftViaAPI(t *testing.T, api, token string, dbId uint64, dbName string, one liveCase) *Draft {
	t.Helper()
	payload, _ := json.Marshal(map[string]any{
		"db":              dbName,
		"intent":          one.Intent,
		"prompt":          one.Prompt,
		"anchorStatement": one.Anchor,
	})
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/dbs/%d/sql-draft", api, dbId), bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	req.Header.Set("ClientId", "golden-live")
	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("draft request: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var parsed struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data *Draft `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("decode draft: %v (%s)", err, string(raw))
	}
	if parsed.Code != 200 || parsed.Data == nil {
		t.Fatalf("draft failed: code=%d msg=%s", parsed.Code, parsed.Msg)
	}
	return parsed.Data
}

// isReadStatement 只读语句判定（写语句不执行，评测里记 n/a）
func isReadStatement(sqlText string) bool {
	lower := strings.ToLower(strings.TrimSpace(sqlText))
	return strings.HasPrefix(lower, "select ") || strings.HasPrefix(lower, "with ") || strings.HasPrefix(lower, "show ")
}

// TestGoldenLive 真实模型 + 真实库的执行等价评测
func TestGoldenLive(t *testing.T) {
	api := envOr("MAYFLY_GOLDEN_API", "http://127.0.0.1:18888")
	username := envOr("MAYFLY_GOLDEN_USER", "admin")
	password := envOr("MAYFLY_GOLDEN_PASS", "admin123")
	dbName := envOr("MAYFLY_GOLDEN_DB", "mayfly_go")
	dsn := envOr("MAYFLY_GOLDEN_DSN", "root:111049@tcp(127.0.0.1:3306)/mayfly_go?parseTime=true")
	var dbId uint64 = 1
	if value := os.Getenv("MAYFLY_GOLDEN_DB_ID"); value != "" {
		if _, err := fmt.Sscan(value, &dbId); err != nil {
			t.Fatalf("parse MAYFLY_GOLDEN_DB_ID: %v", err)
		}
	}

	raw, err := os.ReadFile(filepath.Join("testdata", "golden_live_cases.json"))
	if err != nil {
		t.Fatalf("read live cases: %v", err)
	}
	var cases []liveCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("parse live cases: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("live cases are empty")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open dsn: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Fatalf("ping dsn: %v", err)
	}

	token := loginToken(t, api, username, password)

	var verified, execOK, nonEmpty, readCases, expectNonEmpty int
	for _, one := range cases {
		one := one
		t.Run(one.ID, func(t *testing.T) {
			started := time.Now()
			draft := draftViaAPI(t, api, token, dbId, dbName, one)
			elapsed := time.Since(started).Truncate(time.Millisecond)

			if one.Expect.Verification != "" && draft.Verification != one.Expect.Verification {
				t.Errorf("verification: want %q got %q (checks=%+v)", one.Expect.Verification, draft.Verification, draft.Checks)
			}
			if draft.Verification == "verified" {
				verified++
			}

			if one.Intent == "explain" {
				if strings.TrimSpace(draft.Explanation) == "" {
					t.Errorf("explain returned empty prose")
				}
				t.Logf("explain ok elapsed=%s tokens=%v", elapsed, draft.Usage)
				return
			}

			for _, statement := range draft.Statements {
				lowerSQL := normalizeSQL(statement.SQL)
				for _, want := range one.Expect.RequireSubstrings {
					if !strings.Contains(lowerSQL, strings.ToLower(want)) {
						t.Errorf("generated SQL must contain %q\nSQL: %s", want, statement.SQL)
					}
				}
				for _, forbid := range one.Expect.ForbidSubstrings {
					if strings.Contains(lowerSQL, strings.ToLower(forbid)) {
						t.Errorf("generated SQL must not contain %q\nSQL: %s", forbid, statement.SQL)
					}
				}
				if !isReadStatement(statement.SQL) {
					t.Logf("skip execution of write statement: %s", shorten(statement.SQL, 80))
					continue
				}
				readCases++
				rows, queryErr := db.QueryContext(context.Background(), statement.SQL)
				if queryErr != nil {
					t.Errorf("generated SQL failed to execute: %v\nSQL: %s", queryErr, statement.SQL)
					continue
				}
				count := 0
				for rows.Next() {
					count++
				}
				rows.Close()
				execOK++
				if count > 0 {
					nonEmpty++
				}
				if one.Expect.MinRows > 0 {
					expectNonEmpty++
					if count < one.Expect.MinRows {
						t.Errorf("rows: want >= %d got %d\nSQL: %s", one.Expect.MinRows, count, statement.SQL)
					}
				}
				t.Logf("executed rows=%d elapsed=%s tokens=%v sql=%s", count, elapsed, draft.Usage, shorten(statement.SQL, 100))
			}
		})
	}

	total := len(cases)
	t.Logf("=== 准确率小结：用例 %d，校验通过 %d/%d，可执行 %d/%d，非空结果 %d/%d（按用例声明的最小行数统计）===",
		total, verified, total, execOK, readCases, nonEmpty, expectNonEmpty)
	if readCases > 0 && execOK < readCases {
		t.Fatalf("有生成结果无法执行：execOk=%d readCases=%d", execOK, readCases)
	}
	// 「0 行」只对显式声明了 MinRows 的用例算失败：过滤条件严格的查询（如按当天过滤）
	// 返回 0 行是数据决定的正常结果，用统一阈值会把正常情况判成缺陷
	if expectNonEmpty > 0 && nonEmpty < expectNonEmpty {
		t.Fatalf("有只读语句返回的行数低于用例声明的最小值：nonEmpty=%d expectNonEmpty=%d", nonEmpty, expectNonEmpty)
	}
}

// normalizeSQL 归一化后再做断言：去掉标识符引用符、压缩空白、统一小写。
//
// 断言必须针对**语义**而不是书写风格：实测同一句范围条件会被写成
// “ `i`.`create_time` >= CURDATE() “，若直接匹配 "create_time >=" 就会漏掉反引号
// 而误报失败——这类假失败会让人去修没问题的实现。
func normalizeSQL(sqlText string) string {
	trimmed := strings.NewReplacer("`", "", "\"", "", "\n", " ", "\t", " ").Replace(sqlText)
	return strings.ToLower(strings.Join(strings.Fields(trimmed), " "))
}

func shorten(text string, limit int) string {
	runes := []rune(strings.ReplaceAll(text, "\n", " "))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "…"
}
