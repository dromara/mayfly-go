package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"mayfly-go/internal/mongo/application/mongodoc"
	"mayfly-go/internal/mongo/imsg"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
)

// fakePermRegistry 只实现 HasCode：requireCmdPerm 唯一的依赖。
// 不连 Redis 也不建库，让「透传入口的分级鉴权」这条安全边界可被单测覆盖。
type fakePermRegistry struct {
	codes []string
}

func (f fakePermRegistry) SaveCodes(uint64, []string) {}

func (f fakePermRegistry) HasCode(_ uint64, code string) bool {
	for _, item := range f.codes {
		if item == code {
			return true
		}
	}
	return false
}

func (f fakePermRegistry) Remove(uint64) {}

func newPermCtx(codes ...string) *req.Ctx {
	req.SetPermissionCodeRegistery(fakePermRegistry{codes: codes})

	ctx := context.Background()
	ctx = contextx.WithLoginAccount(ctx, &model.LoginAccount{Id: 7, Username: "tester"})
	ctx = i18n.NewCtxWithLang(ctx, i18n.Zh_CN)
	return &req.Ctx{MetaCtx: ctx}
}

// requirePanic 捕获 requireCmdPerm 抛出的业务错误；无异常则返回 nil。
func requirePanic(t *testing.T, fn func()) error {
	t.Helper()

	var thrown error
	func() {
		defer func() {
			if r := recover(); r != nil {
				thrown = asError(r)
			}
		}()
		fn()
	}()
	return thrown
}

func asError(v any) error {
	if err, ok := v.(error); ok {
		return err
	}
	return errors.New(fmt.Sprint(v))
}

// TestRequireCmdPermIsPerLevel 透传命令入口的鉴权必须按命令语义逐级别生效。
//
// 这是本次修复的最高优先级项：旧入口零鉴权，只读账号即可 dropDatabase / createUser。
func TestRequireCmdPermIsPerLevel(t *testing.T) {
	m := &Mongo{}

	levels := []struct {
		level mongodoc.Level
		code  string
	}{
		{mongodoc.LevelRead, ""},
		{mongodoc.LevelDataSave, mongodoc.PermDataSave},
		{mongodoc.LevelDataDel, mongodoc.PermDataDel},
		{mongodoc.LevelStructSave, mongodoc.PermDDLSave},
		{mongodoc.LevelStructDel, mongodoc.PermDDLDel},
		{mongodoc.LevelAdmin, mongodoc.PermCmdAdmin},
	}

	// 只授予只读：除只读命令外全部必须被拒
	rc := newPermCtx()
	for _, item := range levels {
		err := requirePanic(t, func() { m.requireCmdPerm(rc, "someCmd", item.level) })
		if item.code == "" {
			if err != nil {
				t.Fatalf("read-level command must pass without any permission code, got %v", err)
			}
			continue
		}
		if err == nil {
			t.Fatalf("level %s must be denied without %s", item.level, item.code)
		}
		// 提示必须说清「是哪条命令、什么级别、缺哪个权限码」，裸权限码对操作者没有信息量
		message := err.Error()
		for _, want := range []string{"someCmd", item.code, i18n.TC(rc.MetaCtx, levelNames[item.level])} {
			if !strings.Contains(message, want) {
				t.Fatalf("denied message %q must contain %q", message, want)
			}
		}
	}

	// 逐级授权后各自放行，且不越级生效：只给写权限不能执行删库
	for _, item := range levels {
		if item.code == "" {
			continue
		}
		granted := newPermCtx(item.code)
		if err := requirePanic(t, func() { m.requireCmdPerm(granted, "someCmd", item.level) }); err != nil {
			t.Fatalf("level %s should pass with %s, got %v", item.level, item.code, err)
		}

		other := levels[len(levels)-1]
		if other.code != "" && other.code != item.code {
			cross := newPermCtx(item.code)
			if err := requirePanic(t, func() { m.requireCmdPerm(cross, "otherCmd", other.level) }); err == nil {
				t.Fatalf("permission %s must not satisfy level %s", item.code, other.level)
			}
		}
	}
}

// TestRequireCmdPermDeniesReadOnlyAccountFromDestructiveCommands 用真实命令名复现被修掉的越权路径。
func TestRequireCmdPermDeniesReadOnlyAccountFromDestructiveCommands(t *testing.T) {
	m := &Mongo{}
	rc := newPermCtx()

	for _, command := range []string{
		`{"dropDatabase":1}`,
		`{"drop":"users"}`,
		`{"createUser":"hacker","pwd":"p","roles":["root"]}`,
		`{"someNotRegisteredCommand":1}`,
	} {
		decoded, err := mongodoc.Decode([]byte(command))
		if err != nil {
			t.Fatalf("decode %s: %v", command, err)
		}
		name, level, err := mongodoc.Classify(decoded)
		if err != nil {
			t.Fatalf("classify %s: %v", command, err)
		}
		if err := requirePanic(t, func() { m.requireCmdPerm(rc, name, level) }); err == nil {
			t.Fatalf("read-only account must not be able to run %s (level %s)", command, level)
		}
	}

	// 只读命令仍然可用，否则「能看不能查」变成新的可用性问题
	decoded, err := mongodoc.Decode([]byte(`{"dbStats":1}`))
	if err != nil {
		t.Fatalf("decode dbStats: %v", err)
	}
	name, level, err := mongodoc.Classify(decoded)
	if err != nil {
		t.Fatalf("classify dbStats: %v", err)
	}
	if err := requirePanic(t, func() { m.requireCmdPerm(rc, name, level) }); err != nil {
		t.Fatalf("dbStats must stay available for a read-only account, got %v", err)
	}
}

// TestImsgMessagesRegisteredForPermissionDenial 权限拒绝提示依赖的消息 id 必须双语齐全。
func TestImsgMessagesRegisteredForPermissionDenial(t *testing.T) {
	for _, msgId := range []i18n.MsgId{
		imsg.ErrMongoCmdPermDenied,
		imsg.LevelReadName,
		imsg.LevelDataSaveName,
		imsg.LevelDataDelName,
		imsg.LevelStructSaveName,
		imsg.LevelStructDelName,
		imsg.LevelAdminName,
	} {
		for _, lang := range []string{i18n.Zh_CN, i18n.En} {
			if msg := i18n.TL(lang, msgId); msg == "" {
				t.Fatalf("msg id %d has no text in %s", msgId, lang)
			}
		}
	}

	// 未登记的级别必须落到 admin 文案，不能因为漏配而拼出空提示
	if _, ok := levelNames[mongodoc.Level(99)]; ok {
		t.Fatal("levelNames should only contain defined levels; unknown levels fall back to the admin text")
	}
}
