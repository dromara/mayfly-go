package api

import (
	"strconv"
	"strings"

	"mayfly-go/pkg/biz"
)

// parseCommaIds 解析路径参数中以逗号分隔的 ID 列表。
// 此前实现会静默跳过无法解析的 ID，导致"只删除了部分 ID"却返回成功
func parseCommaIds(idsStr string) []uint64 {
	parts := strings.Split(idsStr, ",")
	ids := make([]uint64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		id, err := strconv.ParseUint(part, 10, 64)
		biz.IsTrue(err == nil && id > 0, "invalid id: [%s]", part)
		ids = append(ids, id)
	}
	return ids
}
