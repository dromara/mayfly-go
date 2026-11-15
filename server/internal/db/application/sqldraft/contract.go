package sqldraft

// 输出契约解析。
//
// 为什么不用 response_format/json_mode：本项目模型走 OpenAI 兼容网关，未知参数
// 在部分网关会直接 400（仓库里已有 enable_thinking 的同款踩坑记录）。因此约束写在
// 提示词里，解析侧做宽容处理：剥代码块 → 取首个 JSON 对象 → 失败再退化为「整段就是
// 一条 SQL」。解析失败不是致命错误，会被当作一次校验问题回喂模型修复一轮。

import (
	"encoding/json"
	"strings"
)

type contractStatement struct {
	SQL         string   `json:"sql"`
	Notes       string   `json:"notes"`
	Assumptions []string `json:"assumptions"`
}

type contractPayload struct {
	Statements  []contractStatement `json:"statements"`
	Explanation string              `json:"explanation"`
}

// parseContract 解析模型输出为语句列表；ok=false 表示没解析出任何语句
func parseContract(text string) ([]Statement, bool) {
	body := stripCodeFence(text)
	if payload, ok := decodePayload(body); ok {
		statements := make([]Statement, 0, len(payload.Statements))
		for _, item := range payload.Statements {
			sql := strings.TrimSpace(item.SQL)
			if sql == "" {
				continue
			}
			statements = append(statements, Statement{
				SQL:         sql,
				Notes:       strings.TrimSpace(item.Notes),
				Assumptions: item.Assumptions,
			})
		}
		if len(statements) > 0 {
			return statements, true
		}
	}
	// 兜底：模型直接吐了 SQL 正文（没有 JSON 外壳）
	if looksLikeSQL(body) {
		return []Statement{{SQL: strings.TrimSpace(body)}}, true
	}
	return nil, false
}

// parseExplanation 解析散文契约（explain 意图）：优先取 JSON 里的 explanation，
// 取不到就把整段文本当作解释正文（模型直接输出散文是常见形态，不该因此判失败）
func parseExplanation(text string) (string, bool) {
	body := stripCodeFence(text)
	if payload, ok := decodePayload(body); ok {
		if strings.TrimSpace(payload.Explanation) != "" {
			return strings.TrimSpace(payload.Explanation), true
		}
	}
	if body == "" {
		return "", false
	}
	return body, true
}

// decodePayload 取文本里第一个完整的 JSON 对象并解码
func decodePayload(body string) (*contractPayload, bool) {
	start := strings.Index(body, "{")
	end := strings.LastIndex(body, "}")
	if start < 0 || end <= start {
		return nil, false
	}
	payload := new(contractPayload)
	if err := json.Unmarshal([]byte(body[start:end+1]), payload); err != nil {
		return nil, false
	}
	return payload, true
}

// parseStringArray 解析 {"tables":[...]} 或裸数组，返回其中的字符串
func parseStringArray(text string) []string {
	body := stripCodeFence(text)
	start, end := strings.Index(body, "["), strings.LastIndex(body, "]")
	if start < 0 || end <= start {
		return nil
	}
	var names []string
	if err := json.Unmarshal([]byte(body[start:end+1]), &names); err != nil {
		return nil
	}
	return names
}

// stripCodeFence 剥掉 ```sql / ```json 包裹
func stripCodeFence(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}
	if index := strings.Index(trimmed, "\n"); index >= 0 {
		trimmed = trimmed[index+1:]
	}
	if index := strings.LastIndex(trimmed, "```"); index >= 0 {
		trimmed = trimmed[:index]
	}
	return strings.TrimSpace(trimmed)
}

// sqlLeadingKeywords 判「这段文本是不是一条 SQL」（兜底解析用）
var sqlLeadingKeywords = map[string]struct{}{
	"select": {}, "with": {}, "insert": {}, "update": {}, "delete": {},
	"create": {}, "alter": {}, "drop": {}, "truncate": {}, "replace": {},
	"show": {}, "explain": {}, "merge": {}, "call": {},
}

func looksLikeSQL(text string) bool {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return false
	}
	_, ok := sqlLeadingKeywords[strings.ToLower(fields[0])]
	return ok
}
