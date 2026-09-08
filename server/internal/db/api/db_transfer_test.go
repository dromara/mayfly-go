package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// FileDel此前make(len(ids))预填零值后append，多传文件id时前面多出len(ids)个0，
// 导致误删id为0的记录。parseFileIds必须只返回真实有效的id
func TestParseFileIds(t *testing.T) {
	// 正常多id解析
	assert.Equal(t, []uint64{1, 2, 3}, parseFileIds("1,2,3"))

	// 单id
	assert.Equal(t, []uint64{42}, parseFileIds("42"))

	// 非法片段过滤，不影响有效id（原实现会让非法值转为0混入结果）
	assert.Equal(t, []uint64{1, 2}, parseFileIds("1,abc,2"))

	// 零值id过滤，杜绝误删id为0的记录
	assert.Equal(t, []uint64{1}, parseFileIds("0,1"))
	assert.Empty(t, parseFileIds("0,0"))

	// 空串与纯分隔符
	assert.Empty(t, parseFileIds(""))
	assert.Empty(t, parseFileIds(","))

	// 片段前后空格可容忍（cast转换前TrimSpace）
	assert.Equal(t, []uint64{5, 6}, parseFileIds(" 5 , 6 "))

	// 负数无效
	assert.Empty(t, parseFileIds("-1,-2"))
}
