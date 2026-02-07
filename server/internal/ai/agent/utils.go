package agent

import (
	"bytes"
	"errors"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"regexp"
	"strings"
)

// ParseLLMJSON 尝试从大模型输出中解析 JSON
func ParseLLMJSON[T any](raw string) (*T, error) {
	candidates := extractJSONCandidates(raw)

	var lastErr error
	for _, c := range candidates {
		if v, err := jsonx.To[T](c); err == nil {
			return v, nil
		} else {
			lastErr = err
		}
	}

	if lastErr == nil {
		lastErr = errors.New("no json candidate found")
	}
	return nil, lastErr
}

// ParseLLMJSON2Map 解析 LLM 返回的JSON为map
func ParseLLMJSON2Map(raw string) (collx.M, error) {
	if res, err := ParseLLMJSON[collx.M](raw); err != nil {
		return nil, err
	} else {
		return *res, nil
	}
}


func extractJSONCandidates(raw string) []string {
	var results []string
	text := strings.TrimSpace(raw)

	// 1. 优先提取 code block 中的 JSON（对象 or 数组）
	codeBlockRe := regexp.MustCompile(
		"(?s)```(?:json)?\\s*([\\[{].*?[\\]}])\\s*```",
	)
	matches := codeBlockRe.FindAllStringSubmatch(text, -1)
	for _, m := range matches {
		results = append(results, strings.TrimSpace(m[1]))
	}

	// 2. 如果没找到 code block，尝试从全文裁剪 JSON
	if len(results) == 0 {
		if clipped := clipJSONValue(text); clipped != "" {
			results = append(results, clipped)
		}
	}

	return results
}

func clipJSONValue(s string) string {
	objIdx := strings.Index(s, "{")
	arrIdx := strings.Index(s, "[")

	start := -1
	var open, close byte

	switch {
	case objIdx != -1 && (arrIdx == -1 || objIdx < arrIdx):
		start = objIdx
		open, close = '{', '}'
	case arrIdx != -1:
		start = arrIdx
		open, close = '[', ']'
	default:
		return ""
	}

	var buf bytes.Buffer
	depth := 0

	for i := start; i < len(s); i++ {
		ch := s[i]
		buf.WriteByte(ch)

		switch ch {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return buf.String()
			}
		}
	}

	return ""
}