package api

import (
	"fmt"
	"strings"
	"testing"

	"mayfly-go/internal/mongo/mgm"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// TestMongoErrorUnauthorizedWording 认证类失败必须说成「补凭证/换账号」，并保留服务端原文。
//
// 这条判据的价值在于可行动性：未认证的实例上 ping 也通、listDatabases 才被拒，
// 如果只回一句「执行失败」，用户会去反复改命令文本，而真正要改的是连接串。
func TestMongoErrorUnauthorizedWording(t *testing.T) {
	rc := newPermCtx()
	ctx := rc.MetaCtx

	unauthorized := mongo.CommandError{Code: mgm.CodeUnauthorized, Message: "Command listDatabases requires authentication"}

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"服务端 13 码", unauthorized, true},
		{"被包装的 13 码", fmt.Errorf("run command: %w", unauthorized), true},
		{"探测出的凭证缺失", fmt.Errorf("%w: %s", mgm.ErrNoCredentials, unauthorized.Error()), true},
		{"其他命令错误", mongo.CommandError{Code: 26, Message: "namespace not found"}, false},
		{"普通错误", fmt.Errorf("dial tcp: connect refused"), false},
	}

	for _, c := range cases {
		err := mongoError(ctx, c.err)
		if err == nil {
			t.Fatalf("%s: mongoError must not swallow the error", c.name)
		}
		message := err.Error()
		// 新文案独有的两处：认证措辞与修法示例
		got := strings.Contains(message, "authSource") && strings.Contains(message, "认证")
		if got != c.want {
			t.Fatalf("%s: wording auth-hint = %v, want %v; message = %q", c.name, got, c.want, message)
		}
		// 无论走哪条文案，服务端原文都要留下，否则远程排障无从对齐
		if strings.Contains(c.err.Error(), "listDatabases") && !strings.Contains(message, "listDatabases") {
			t.Fatalf("%s: server detail must be kept, got %q", c.name, message)
		}
	}
}

// TestMongoErrorNilIsNil 无错误时必须原样返回 nil，调用方才能直接 biz.ErrIsNil。
func TestMongoErrorNilIsNil(t *testing.T) {
	if err := mongoError(newPermCtx().MetaCtx, nil); err != nil {
		t.Fatalf("mongoError(nil) = %v, want nil", err)
	}
}
