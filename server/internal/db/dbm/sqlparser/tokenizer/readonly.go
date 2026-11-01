package tokenizer

import "strings"

// readOnlyLeadingKeywords 未结构化建模语句的只读白名单。
//
// 这些动词开头的语句不会写业务数据（导航/会话/事务/维护类）。采用白名单而非黑名单：
// 漏登记一个写动词的后果是「写语句被当只读执行」（绕过审批与审计），而多登记一个无害动词
// 只会让它多走一次写通道，方向安全。
//
// 刻意不登记（按可能写处理）：CALL/EXEC/EXECUTE（过程体可写）、DO（可执行表达式）、
// PREPARE（后续可执行写）、LOAD DATA / COPY FROM（批量写入）、FLUSH/KILL/LOCK（管理写）。
var readOnlyLeadingKeywords = map[string]bool{
	"SHOW": true, "EXPLAIN": true, "DESC": true, "DESCRIBE": true,
	"SET": true, "USE": true, "RESET": true, "DISCARD": true,
	"BEGIN": true, "START": true, "COMMIT": true, "ROLLBACK": true,
	"SAVEPOINT": true, "RELEASE": true, "DEALLOCATE": true,
	// 维护类：只改统计信息/回收空间，不改业务数据
	"VACUUM": true, "ANALYZE": true, "PRAGMA": true, "CHECKPOINT": true,
}

// writeLeadingKeywords 明确会写数据的首词。与只读白名单互补：用于把「已知写」与
// 「完全未知」区分开，便于分类器给出更准确的类型（两者都走写通道，fail-closed 一致）。
var writeLeadingKeywords = map[string]bool{
	"CALL": true, "EXEC": true, "EXECUTE": true, "DO": true, "PREPARE": true,
	"LOAD": true, "COPY": true, "FLUSH": true, "KILL": true, "LOCK": true, "UNLOCK": true,
}

// IsReadOnlyLeadingKeyword 判定首词是否属于只读白名单。
//
// 大小写归一放在函数内部：tokenizer.LeadingKeyword 返回小写、ACL 侧取到的是大写，
// 调用点各转一次迟早漏一处（本轮 DESC 就是这么被判成"可能写"的）。
func IsReadOnlyLeadingKeyword(keyword string) bool {
	return readOnlyLeadingKeywords[strings.ToUpper(strings.TrimSpace(keyword))]
}

// IsWriteLeadingKeyword 判定首词是否属于已明确会写数据的动词（同样大小写不敏感）。
func IsWriteLeadingKeyword(keyword string) bool {
	return writeLeadingKeywords[strings.ToUpper(strings.TrimSpace(keyword))]
}
