package application

import (
	"context"
	"errors"
	"testing"

	"mayfly-go/internal/ai/imsg"
	"mayfly-go/internal/ai/pkg/headless"
	"mayfly-go/pkg/i18n"

	"github.com/stretchr/testify/require"
)

func TestRunHeadless_NilRequest(t *testing.T) {
	_, err := RunHeadless(context.Background(), nil)
	require.Error(t, err)
}

func TestRunHeadless_FailClosedWithoutOwner(t *testing.T) {
	// 无服务身份 → fail-closed 拒绝（在构建 Agent 前即返回，无需基础设施）
	_, err := RunHeadless(context.Background(), &headless.RunReq{
		ProfileId: ProfileDiagnostician,
		Input:     "check db",
		// OwnerAccountId 缺省
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), i18n.T(imsg.ErrHeadlessOwnerRequired))
}

func TestRunHeadless_UnknownProfile(t *testing.T) {
	// owner 合法但 profile 未知 → 在运行时装配前返回错误
	_, err := RunHeadless(context.Background(), &headless.RunReq{
		ProfileId:      "__no_such_profile__",
		Input:          "hi",
		OwnerAccountId: 1,
		OwnerUsername:  "admin",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), i18n.T(imsg.ErrUnknownAgentProfile, "profileId", "__no_such_profile__"))
}

func TestRunHeadless_RegisteredThroughContract(t *testing.T) {
	// 契约包注册后，消费方经 headless.Run 调用适配器；fail-closed 路径无需基础设施
	headless.Register(RunHeadless)
	require.True(t, headless.Registered())

	_, err := headless.Run(context.Background(), &headless.RunReq{Input: "x"})
	require.Error(t, err)
	require.Contains(t, err.Error(), i18n.T(imsg.ErrHeadlessOwnerRequired))
}

func TestResolveRunStatus(t *testing.T) {
	ctx := context.Background()
	require.Equal(t, headless.RunStatusSucceeded, resolveRunStatus(ctx, nil))
	require.Equal(t, headless.RunStatusTimeout, resolveRunStatus(ctx, context.DeadlineExceeded))
	require.Equal(t, headless.RunStatusCanceled, resolveRunStatus(ctx, context.Canceled))
	require.Equal(t, headless.RunStatusFailed, resolveRunStatus(ctx, errors.New("boom")))

	// ctx 已超时（错误未直接携带 DeadlineExceeded）也应细分为 timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, -1)
	defer cancel()
	require.Equal(t, headless.RunStatusTimeout, resolveRunStatus(timeoutCtx, errors.New("wrapped")))
}
