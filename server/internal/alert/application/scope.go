package application

import (
	"context"
	"encoding/json"
	"mayfly-go/internal/alert/domain/entity"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"strconv"
)

// expandResourceScope 将规则的关联范围展开为具体资源ID列表。
//
// 告警规则与静默规则共用同一套范围语义，避免两处实现随时间漂移：
//   - AlertScopeResource: ScopeValue 为单个资源ID（字符串）
//   - AlertScopeTagPath:  通过 ruleId 从标签关联表获取标签路径，再展开为资源 ID
//   - AlertScopeAll:      不展开，返回 nil（由调用方解释为"该资源类型下全部资源"）
//
// queryResourceIds: 通过资源 code 列表查询资源 ID 的函数（由评估器提供）
func expandResourceScope(ctx context.Context, tagTreeRelateApp tagapp.TagTreeRelate, tagTreeApp tagapp.TagTree, resourceType int8, ruleId uint64, scopeType entity.AlertScopeType, scopeValue string, queryResourceIds func(codes []string) []uint64) []uint64 {
	switch scopeType {
	case entity.AlertScopeResource:
		id, err := strconv.ParseUint(scopeValue, 10, 64)
		if err != nil {
			logx.Warnf("[alert] parse scope value[%s] error: %s", scopeValue, err.Error())
			return nil
		}
		return []uint64{id}
	case entity.AlertScopeTagPath:
		codes := expandByTagPath(ctx, tagTreeRelateApp, tagTreeApp, resourceType, ruleId, scopeValue)
		if len(codes) == 0 {
			return nil
		}
		if queryResourceIds == nil {
			logx.Errorf("[alert] queryResourceIds function is nil")
			return nil
		}
		return queryResourceIds(codes)
	}
	return nil
}

// expandByTagPath 通过标签关联展开资源标签 code 列表（参照计划任务实现）。
//
// 流程：
//  1. GetTagPathsByRelate: 获取规则关联的标签路径（ruleId > 0 时使用）
//  2. 或从 scopeValue 解析标签路径（ruleId = 0 时使用，用于静默规则）
//  3. ListByQuery: 查找这些标签路径下的所有资源标签
//  4. 返回资源标签的 code 列表（调用方需通过 code 查询资源 ID）
func expandByTagPath(ctx context.Context, tagTreeRelateApp tagapp.TagTreeRelate, tagTreeApp tagapp.TagTree, resourceType int8, ruleId uint64, scopeValue string) []string {
	// 1. 获取标签路径
	var relateCodePaths []string
	if ruleId > 0 {
		// 告警规则：从标签关联表获取
		relateCodePaths = tagTreeRelateApp.GetTagPathsByRelate(tagentity.TagRelateTypeAlertRule, ruleId)
	} else {
		// 静默规则：从 scopeValue 解析
		if err := json.Unmarshal([]byte(scopeValue), &relateCodePaths); err != nil {
			logx.Errorf("[alert] parse scope value[%s] error: %s", scopeValue, err.Error())
			return nil
		}
	}
	if len(relateCodePaths) == 0 {
		logx.Warnf("[alert] expandByTagPath: no tag paths found (ruleId=%d, scopeValue=%s)", ruleId, scopeValue)
		return nil
	}
	logx.Debugf("[alert] expandByTagPath: found %d tag paths", len(relateCodePaths))

	// 2. 查找这些标签路径下的所有资源标签
	var resourceTags []tagentity.TagTree
	resourceTagType := tagentity.TagType(resourceType)
	if err := tagTreeApp.ListByQuery(&tagentity.TagTreeQuery{
		CodePathLikes: relateCodePaths,
		Types:         []tagentity.TagType{resourceTagType},
	}, &resourceTags); err != nil {
		logx.Errorf("[alert] expandByTagPath: list resource tags error: %s", err.Error())
		return nil
	}
	if len(resourceTags) == 0 {
		logx.Warnf("[alert] expandByTagPath: no resource tags found under paths %v", relateCodePaths)
		return nil
	}
	logx.Debugf("[alert] expandByTagPath: found %d resource tags", len(resourceTags))

	// 3. 返回资源标签的 code 列表
	codes := collx.ArrayMap(resourceTags, func(tag tagentity.TagTree) string {
		return tag.Code
	})
	logx.Debugf("[alert] expandByTagPath: returning %d resource codes", len(codes))
	return collx.ArrayDeduplicate(codes)
}

// containsResourceId 判断资源ID是否在给定列表中
func containsResourceId(ids []uint64, id uint64) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}
