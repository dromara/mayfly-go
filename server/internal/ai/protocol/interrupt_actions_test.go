package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestResumeActionRoundTrip 动作 ↔ resume 决策类型必须双向一致。
//
// 缺陷类（真实发生过）：同一组取值在三处手写（映射函数、审批 switch、参数补全的
// paramActionComplete）。新增一种决策时只改一处不会报错——审批侧会把它归入「无效动作」
// 并返回 OPERATION_CANCELLED，补全侧则不认为已完成，两者都只在真机点一次才暴露
func TestResumeActionRoundTrip(t *testing.T) {
	cases := []struct{ action, resumeType string }{
		{ActionApprove, ResumeTypeApproved},
		{ActionReject, ResumeTypeRejected},
		{ActionComplete, ResumeTypeParamsCompleted},
	}
	for _, c := range cases {
		require.Equal(t, c.resumeType, ResumeTypeOfAction(c.action), "动作转决策类型不一致：%s", c.action)
		require.Equal(t, c.action, ResumeActionOfType(c.resumeType), "决策类型转动作为不一致：%s", c.resumeType)
	}

	// 未知取值原样透传：不猜语义（与「未声明形态不猜」同一口径）
	require.Equal(t, "custom_action", ResumeTypeOfAction("custom_action"))
	require.Equal(t, "custom_resume", ResumeActionOfType("custom_resume"))
}
