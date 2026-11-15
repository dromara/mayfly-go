package sqldraft

// 列取值采样：把「枚举语义列实际有哪些取值」带进上下文，并据此核对生成结果里的字面量。
//
// 为什么必须做：真机实测过一类只有结构校验抓不到的错——模型写了
// `... AND m.metric = 'query_time'`，列全都存在、语句能通过全部结构校验，但库里这块
// 指标的取值只有 lock_wait_count / ping_fail / pitr_ready / session_max_elapsed /
// sessions_active，语句跑得出 0 行。结构对了不等于语义对了，取值域是另一半。
//
// 三条安全边界（宁可少采，不可越界）：
//  1. 只采「枚举语义列」：列名命中 status/state/type/kind/metric/... 且类型是字符型；
//  2. 列名命中敏感词（password/token/secret/phone/email/...）一律不采，连值都不读；
//  3. 采样走既有 ColumnValues 通道：它自身已过表级 + 列级权限断言，且限行由方言分页改写。
//
// 采样结果只用于：① 上下文提示；② 生成后核对字面量是否落在采样集内（结论为 unverified，
// 因为采样是有限行，不在采样集不等于不存在）。

import (
	"context"
	"sort"
	"strings"

	"mayfly-go/pkg/logx"
)

const (
	// maxSampleColumnsPerTable 每张表最多采样几个枚举列
	maxSampleColumnsPerTable = 3
	// maxSampleTables 最多采样几张表（控制一次生成里的额外查询数）
	maxSampleTables = 4
	// sampleValueLimit 每列采样多少个去重值
	sampleValueLimit = 12
)

// enumColumnTokens 枚举语义列名（按分词命中，避免 "status_desc" 这类描述列被采）
var enumColumnTokens = []string{
	"status", "state", "type", "kind", "metric", "category", "level", "flag",
	"mode", "result", "source", "channel", "role", "gender", "stage", "priority",
}

// sensitiveColumnTokens 敏感列名：命中即不采样（连值都不读）
var sensitiveColumnTokens = []string{
	"password", "passwd", "pwd", "secret", "token", "key", "salt", "credential",
	"phone", "mobile", "email", "mail", "id_card", "idcard", "passport", "bank",
	"card_no", "address", "addr", "name", "remark", "comment", "description", "desc",
	"content", "payload", "title", "url", "path",
}

// sampledValue 一列的采样结果
type sampledValue struct {
	Table  string
	Column string
	Values []string
}

// isEnumLikeColumn 该列是否值得采样（枚举语义 + 字符类型）
func isEnumLikeColumn(column columnInfo) bool {
	if !isStringishType(column.Type) {
		return false
	}
	name := strings.ToLower(column.Name)
	if hasAnyToken(name, sensitiveColumnTokens) {
		return false
	}
	return hasAnyToken(name, enumColumnTokens)
}

// isStringishType 字符类型判定（只按类型名，不做方言特化：各方言的字符类型都含这些词）
func isStringishType(columnType string) bool {
	lower := strings.ToLower(columnType)
	for _, token := range []string{"char", "text", "enum", "set", "string", "clob"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

// hasAnyToken 按标识符分词命中（`db_status_code` 命中 status；`statusdesc` 不命中）
func hasAnyToken(name string, tokens []string) bool {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-' || r == '.' || r == ' '
	})
	for _, part := range parts {
		for _, token := range tokens {
			if part == token {
				return true
			}
		}
	}
	return false
}

// sampleTarget 一条采样目标
type sampleTarget struct {
	Table  string
	Column string
}

// collectSampleTargets 从选中表里挑出要采样的列（受每表/总量上限约束，保持稳定顺序）
func collectSampleTargets(tables []*tableInfo) []sampleTarget {
	targets := make([]sampleTarget, 0, maxSampleTables*maxSampleColumnsPerTable)
	for _, table := range tables {
		if len(targets) >= maxSampleTables*maxSampleColumnsPerTable {
			break
		}
		perTable := 0
		for _, column := range table.Columns {
			if perTable >= maxSampleColumnsPerTable {
				break
			}
			if !isEnumLikeColumn(column) {
				continue
			}
			targets = append(targets, sampleTarget{Table: table.Name, Column: column.Name})
			perTable++
		}
	}
	return targets
}

// sampleColumns 采样选中列（单列失败不影响其余；失败只记日志，不阻断生成）
func (r *draftRun) sampleColumns(ctx context.Context, targets []sampleTarget) []sampledValue {
	if r.sampler == nil || len(targets) == 0 {
		return nil
	}
	sampled := make([]sampledValue, 0, len(targets))
	for _, target := range targets {
		key := strings.ToLower(target.Table) + "." + strings.ToLower(target.Column)
		if cached, ok := r.sampleCache[key]; ok {
			if len(cached) > 0 {
				sampled = append(sampled, sampledValue{Table: target.Table, Column: target.Column, Values: cached})
			}
			continue
		}
		values, err := r.sampler.ColumnValues(ctx, r.req.DbId, r.req.Db, target.Table, target.Column, sampleValueLimit)
		if err != nil {
			// 权限不足 / 方言无分页改写能力 / 表不可达都会走到这里：取值只是增强项，
			// 不应让整次生成失败（缺失由上下文不出现该列取值体现，不静默伪装成"没有取值"）
			logx.WarnfContext(ctx, "[sqldraft] sample column %s.%s failed: %s", target.Table, target.Column, err.Error())
			r.sampleCache[key] = nil
			continue
		}
		values = trimAndSortValues(values, sampleValueLimit)
		r.sampleCache[key] = values
		if len(values) > 0 {
			sampled = append(sampled, sampledValue{Table: target.Table, Column: target.Column, Values: values})
		}
	}
	return sampled
}

// trimAndSortValues 去空、去重、截断并排序（排序是为了让上下文稳定，便于缓存与回归比对）
func trimAndSortValues(values []string, limit int) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if len([]rune(trimmed)) > 64 {
			// 长文本列即便名字命中枚举词也不适合放进上下文
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// sampleLabel 上下文里展示采样值的形态
func sampleLabel(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, "'"+value+"'")
	}
	return strings.Join(quoted, ", ")
}
