package keyvalue

import (
	"context"

	"mayfly-go/internal/redis/imsg"
	"mayfly-go/pkg/errorx"
)

// 视角处理器的统一错误构造：结构性失败走模块 i18n（用户可见），参数校验失败保留英文细节（排障可见）

func ErrUnsupportedType(ctx context.Context, keyType string) error {
	return errorx.NewBizI(ctx, imsg.ErrRedisKeyTypeUnsupported, "type", keyType)
}

func ErrUnsupportedView(ctx context.Context, view, keyType string) error {
	return errorx.NewBizI(ctx, imsg.ErrRedisKeyViewUnsupported, "view", view, "type", keyType)
}

func ErrKeyNotFound(ctx context.Context) error {
	return errorx.NewBizI(ctx, imsg.ErrRedisKeyNotFound)
}

// ErrUnsupportedMemberOp 该视角未声明此成员操作的能力位
func ErrUnsupportedMemberOp(view, op string) error {
	return errorx.NewBizf("the view [%s] does not support the member operation [%s]", view, op)
}

// ErrUnknownMemberOp 请求里的成员操作不属于契约定义的操作集合
func ErrUnknownMemberOp(op string) error {
	return errorx.NewBizf("unknown redis member operation: %s", op)
}

func errInvalidArg(arg string) error {
	return errorx.NewBizf("invalid redis key member argument: %s", arg)
}
