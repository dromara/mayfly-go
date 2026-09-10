package tokenizer

import (
	"unicode/utf8"
)

// LeadingKeyword 返回 SQL 的首个关键字（小写整词），跳过前导空白与注释区域。
//
// 语句切割保留注释原文后，「取前若干字符判断语句类型」的兜底逻辑会失真
// （如 "-- 说明\nDELETE FROM t" 的前缀命中注释里的单词），故需要方言感知的取词能力：
//   - 行注释/块注释（含 PG 嵌套注释）整体跳过；
//   - mysql 的 /*!40101 ... */ 是待执行内容，跳过前缀后继续在其内部找关键字；
//   - 起始即为非词字符（如 '('、引用标识符）时返回空串，由调用方决定回落策略
func LeadingKeyword(sql string, cfg DialectConfig) string {
	for i := 0; i < len(sql); {
		r, size := utf8.DecodeRuneInString(sql[i:])
		if cfg.IsLineCommentStart(sql, i) {
			i = cfg.SkipLineComment(sql, i)
			continue
		}
		if cfg.IsBlockCommentStart(sql, i) {
			if cfg.IsExecutableComment(sql, i) {
				i += 3 // 跳过 "/*!"，其后的版本号为门控标记，均不属于关键字
				for i < len(sql) && sql[i] >= '0' && sql[i] <= '9' {
					i++
				}
				continue
			}
			end, closed := cfg.SkipBlockComment(sql, i)
			if !closed {
				return ""
			}
			i = end
			continue
		}
		if isScannerSpace(r) {
			i += size
			continue
		}
		if !IsWordStart(i, sql) {
			return ""
		}
		word, _ := readWordLower(sql, i)
		return word
	}
	return ""
}
