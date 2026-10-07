package keyvalue

import (
	"context"
	"strings"
	"testing"

	"mayfly-go/internal/redis/domain/entity"

	"github.com/redis/go-redis/v9"
)

// 该测试直接连本地 Redis（与项目约定的开发实例一致），连不上则跳过，避免无 redis 的环境变红
const (
	testRedisAddr = "127.0.0.1:6332"
	testRedisDb   = 15
)

func connect(t *testing.T) *redis.Client {
	t.Helper()

	cli := redis.NewClient(&redis.Options{Addr: testRedisAddr, DB: testRedisDb})
	if err := cli.Ping(context.Background()).Err(); err != nil {
		t.Skipf("local redis [%s] is unreachable, skip: %s", testRedisAddr, err.Error())
	}
	t.Cleanup(func() {
		_ = cli.FlushDB(context.Background()).Err()
		_ = cli.Close()
	})
	return cli
}

// TestRegistryContract 注册中心的结构自检：所有漂移都会在新增处理器时被这里抓到
func TestRegistryContract(t *testing.T) {
	descs := AllDescriptors()
	if len(descs) == 0 {
		t.Fatal("no view handler registered")
	}

	defaults := map[entity.KeyType]int{}
	views := map[string]bool{}
	for _, desc := range descs {
		if views[desc.View] {
			t.Fatalf("duplicated view: %s", desc.View)
		}
		views[desc.View] = true

		if desc.Label == "" || len(desc.Types) == 0 {
			t.Fatalf("view [%s] must declare label and types", desc.View)
		}
		// 展示文案必须是 i18n key，后端不产出裸文本
		if !strings.HasPrefix(desc.Label, "redis.") {
			t.Fatalf("view [%s] label must be an i18n key, got: %s", desc.View, desc.Label)
		}
		if desc.Default {
			for _, keyType := range desc.Types {
				defaults[keyType]++
			}
		}
		if !desc.Caps.Create && !desc.Caps.Update && !desc.Caps.Delete && !desc.Caps.Ops {
			t.Fatalf("view [%s] declares no capability", desc.View)
		}
		if desc.Form == nil && (desc.Caps.Create || desc.Caps.Update) {
			t.Fatalf("view [%s] supports writing members but declares no form", desc.View)
		}
		for _, col := range desc.Columns {
			if !strings.HasPrefix(col.Label, "redis.") {
				t.Fatalf("view [%s] column label must be an i18n key, got: %s", desc.View, col.Label)
			}
		}
		if len(desc.Ops) > 0 && !desc.Caps.Ops {
			t.Fatalf("view [%s] declares ops without the ops capability", desc.View)
		}
		checkOpPlan(t, desc)
	}

	for _, keyType := range []entity.KeyType{entity.KeyTypeString, entity.KeyTypeHash, entity.KeyTypeList, entity.KeyTypeSet, entity.KeyTypeZset, entity.KeyTypeStream} {
		if defaults[keyType] != 1 {
			t.Fatalf("type [%s] must have exactly one default view, got %d", keyType, defaults[keyType])
		}
	}
}

// checkOpPlan 描述符里声明的每个操作都必须被 PlanOp 认识：防止「前端有按钮，后端无人处理」
func checkOpPlan(t *testing.T, desc *entity.ViewDescriptor) {
	t.Helper()

	handler, ok := Get(desc.View).(OpPlanner)
	if len(desc.Ops) == 0 {
		if _, exists := Get(desc.View).(OpPlanner); exists {
			t.Fatalf("view [%s] implements OpPlanner without declaring ops", desc.View)
		}
		return
	}
	if !ok {
		t.Fatalf("view [%s] declares ops but does not implement OpPlanner", desc.View)
	}

	for _, spec := range desc.Ops {
		if !strings.HasPrefix(spec.Label, "redis.") {
			t.Fatalf("view [%s] op [%s] label must be an i18n key", desc.View, spec.Name)
		}
		_, err := handler.PlanOp(context.Background(), nil, &entity.OpRequest{Key: "k", View: desc.View, Op: spec.Name, Args: opArgs(spec.Name)})
		if err != nil && strings.Contains(err.Error(), " op: ") {
			t.Fatalf("view [%s] declares op [%s] which PlanOp does not handle", desc.View, spec.Name)
		}
	}
}

// checkUnsupportedMemberOps 处理器自己的 op 分派必须与描述符能力位一致：
// 未声明能力的 op 必须在读取任何数据之前就被拒绝（因此这里传 nil 命令面：
// 一旦实现顺序不对、先碰了 redis，本用例会直接 panic 暴露），
// 否则应用层按 caps 把关就形同虚设
func checkUnsupportedMemberOps(t *testing.T, desc *entity.ViewDescriptor) {
	t.Helper()

	for _, item := range []struct {
		op      entity.MemberOp
		allowed bool
	}{
		{entity.MemberOpCreate, desc.Caps.Create},
		{entity.MemberOpUpdate, desc.Caps.Update},
		{entity.MemberOpDelete, desc.Caps.Delete},
	} {
		if item.allowed {
			continue
		}
		if _, err := Get(desc.View).BuildWrite(context.Background(), nil, &entity.MemberWrite{Key: "k", View: desc.View, Op: item.op}); err == nil {
			t.Fatalf("view [%s] must reject the unsupported op [%s]", desc.View, item.op)
		}
	}

	if _, err := Get(desc.View).BuildWrite(context.Background(), nil, &entity.MemberWrite{Key: "k", View: desc.View, Op: "unknown"}); err == nil {
		t.Fatalf("view [%s] must reject an unknown member op", desc.View)
	}
}

// checkDeclaredMemberOps 声明了能力的 op 不能被自己的实现拒掉（用真实连接调用，缺参数报错可以接受）
func checkDeclaredMemberOps(t *testing.T, cmd *redis.Client, desc *entity.ViewDescriptor) {
	t.Helper()

	key := "mayfly:capcheck:" + desc.View
	defer func() { _ = cmd.Do(context.Background(), "DEL", key).Err() }()

	for _, item := range []struct {
		op      entity.MemberOp
		allowed bool
	}{
		{entity.MemberOpCreate, desc.Caps.Create},
		{entity.MemberOpUpdate, desc.Caps.Update},
		{entity.MemberOpDelete, desc.Caps.Delete},
	} {
		if !item.allowed {
			continue
		}
		_, err := Get(desc.View).BuildWrite(context.Background(), cmd, &entity.MemberWrite{Key: key, View: desc.View, Op: item.op})
		if err != nil && strings.Contains(err.Error(), "does not support the member op") {
			t.Fatalf("view [%s] declares the op [%s] but BuildWrite rejects it", desc.View, item.op)
		}
	}
}

// opArgs 为各视角操作提供能通过参数校验的最小入参，只用于检查 op 是否被处理
func opArgs(op string) map[string]string {
	return map[string]string{
		argValue: "v", argField: "f", argMember: "m1", argMemberTo: "m2", argIds: "1-1",
		argGroup: "g", argCount: "1", argIncrement: "1", argMaxLen: "10", argBit: "1",
		argStart: "0", argEnd: "-1", argScore: "1", argIndex: "0", argId: "*",
		argValues: "v1", argLongitude: "13.36", argLatitude: "38.11", argRadius: "100",
		argKind: kindUnion, argDestination: "dst", argSourceKeys: "src", argFields: "f",
	}
}

// TestMemberOpCaps 能力位与实现一致性检查（结构性断言，不需要连接）
func TestMemberOpCaps(t *testing.T) {
	for _, desc := range AllDescriptors() {
		t.Run(desc.View, func(t *testing.T) {
			checkUnsupportedMemberOps(t, desc)
		})
	}
}

// TestDeclaredMemberOps 声明了能力的 op 必须真的被实现接受（需要本地 redis）
func TestDeclaredMemberOps(t *testing.T) {
	cmd := connect(t)
	for _, desc := range AllDescriptors() {
		t.Run(desc.View, func(t *testing.T) {
			checkDeclaredMemberOps(t, cmd, desc)
		})
	}
}

// TestViewRoundTrip 逐个视角跑通「写成员 → 读列表 → 改成员 → 删成员」，等价于 UI 上的完整操作闭环
func TestViewRoundTrip(t *testing.T) {
	cmd := connect(t)
	ctx := context.Background()

	cases := []struct {
		view       string
		create     map[string]string
		update     map[string]string
		row        *entity.Member
		del        []*entity.Member
		skipDel    bool
		wantReject string
		extra      map[string]string
	}{
		{view: ViewString, create: map[string]string{argValue: "hello"}, update: map[string]string{argValue: "world"}, skipDel: true},
		{view: ViewHash, create: map[string]string{argField: "f1", argValue: "v1"}, update: map[string]string{argField: "f2", argValue: "v2"}, row: &entity.Member{Field: "f1"}},
		{view: ViewList, create: map[string]string{argValues: "a\nb", argPosition: positionTail}, update: map[string]string{argIndex: "0", argValue: "z"}, row: &entity.Member{Index: 1}, del: []*entity.Member{{Index: 0}}},
		{view: ViewSet, create: map[string]string{argValues: "m1\nm2"}, update: map[string]string{argValue: "m1x"}, row: &entity.Member{Value: "m1"}, del: []*entity.Member{{Value: "m2"}}},
		{view: ViewZset, create: map[string]string{argScore: "1", argValue: "a"}, update: map[string]string{argScore: "2", argValue: "b"}, row: &entity.Member{Value: "a"}, del: []*entity.Member{{Value: "b"}}},
		{view: ViewStream, create: map[string]string{argId: "1-1", argValue: "k=v"}, wantReject: entity.MemberOpUpdate, extra: map[string]string{argId: "1-2", argValue: "k=v2"}, row: &entity.Member{Id: "1-1"}},
		{view: ViewBitmap, create: map[string]string{argIndex: "10", argValue: "1"}, update: map[string]string{argIndex: "11", argValue: "0"}, skipDel: true},
		{view: ViewHyperLogLog, create: map[string]string{argValues: "x\ny"}, skipDel: true},
		{view: ViewGeo, create: map[string]string{argField: "pal", argLongitude: "13.361389", argLatitude: "38.115556"}, update: map[string]string{argField: "pal2", argLongitude: "15.087269", argLatitude: "37.502669"}, row: &entity.Member{Field: "pal"}},
	}

	for _, tc := range cases {
		t.Run(tc.view, func(t *testing.T) {
			key := "mayfly:kvtest:" + tc.view
			if err := cmd.Do(ctx, "DEL", key).Err(); err != nil {
				t.Fatalf("clean scratch key failed: %s", err.Error())
			}

			write := func(op string, args map[string]string, row *entity.Member) {
				t.Helper()
				cmds, err := Get(tc.view).BuildWrite(ctx, cmd, &entity.MemberWrite{Key: key, View: tc.view, Op: op, Args: args, Member: row})
				if err != nil {
					t.Fatalf("build %s cmd failed: %s", op, err.Error())
				}
				for _, built := range cmds {
					if err := cmd.Do(ctx, built...).Err(); err != nil {
						t.Fatalf("exec %v failed: %s", built, err.Error())
					}
				}
			}

			write(entity.MemberOpCreate, tc.create, nil)
			if tc.extra != nil {
				// 再写一条，才能验证「删一行」不会把整个 key 清空
				write(entity.MemberOpCreate, tc.extra, nil)
			}
			if tc.update != nil {
				write(entity.MemberOpUpdate, tc.update, tc.row)
			}
			if tc.wantReject != "" {
				if _, err := Get(tc.view).BuildWrite(ctx, cmd, &entity.MemberWrite{Key: key, View: tc.view, Op: tc.wantReject}); err == nil {
					t.Fatalf("view [%s] should reject the unsupported op [%s]", tc.view, tc.wantReject)
				}
			}

			keyType := entity.GetKeyType(cmd.Type(ctx, key).Val())
			handler, err := Resolve(ctx, cmd, keyType, tc.view, key)
			if err != nil {
				t.Fatalf("resolve view failed: %s", err.Error())
			}
			page, err := handler.Load(ctx, cmd, &entity.MemberQuery{Key: key, View: tc.view, Size: 50})
			if err != nil {
				t.Fatalf("load members failed: %s", err.Error())
			}
			if len(page.Members) == 0 {
				t.Fatalf("view [%s] loaded no member after write", tc.view)
			}
			if size, err := handler.Size(ctx, cmd, key); err != nil || size <= 0 {
				t.Fatalf("view [%s] size should be positive, got %d err %v", tc.view, size, err)
			}

			if tc.skipDel {
				return
			}
			write(entity.MemberOpDelete, nil, tc.row)
			if tc.del != nil {
				cmds, err := Get(tc.view).BuildWrite(ctx, cmd, &entity.MemberWrite{Key: key, View: tc.view, Op: entity.MemberOpDelete, Members: tc.del})
				if err != nil {
					t.Fatalf("build batch delete cmd failed: %s", err.Error())
				}
				for _, built := range cmds {
					if err := cmd.Do(ctx, built...).Err(); err != nil {
						t.Fatalf("exec batch delete %v failed: %s", built, err.Error())
					}
				}
			}
			left, err := handler.Size(ctx, cmd, key)
			if err != nil {
				t.Fatalf("size after delete failed: %s", err.Error())
			}
			if tc.del == nil && left <= 0 {
				t.Fatalf("view [%s] lost all members after deleting one", tc.view)
			}
		})
	}
}

// assertRejected 视角声明了写能力时不该有静默成功的路径：不支持的 op 必须返回错误
func assertRejected(t *testing.T, view, op string) {
	t.Helper()

	if _, err := Get(view).BuildWrite(context.Background(), nil, &entity.MemberWrite{Key: "k", View: view, Op: op}); err == nil {
		t.Fatalf("view [%s] should reject the unsupported op [%s]", view, op)
	}
}

// TestUnsupportedType 未知类型（module 类型）必须显式报错，而不是静默返回空表格
func TestUnsupportedType(t *testing.T) {
	if _, err := Resolve(context.Background(), nil, "ReJSON-RL", "", "k"); err == nil {
		t.Fatal("module type must be rejected with an error")
	}
}

// TestScanMatchEscape 关键字必须被当作普通文本匹配，不能让用户输入的 * 变成通配
func TestScanMatchEscape(t *testing.T) {
	if got := scanMatch(""); got != "*" {
		t.Fatalf("empty keyword should match all, got %s", got)
	}
	// 关键字里的 glob 元字符必须逐个被 \ 转义，否则用户搜 a*b 会命中根本不含 * 的 key
	for _, item := range []struct{ keyword, want string }{
		{"ab*c", `*ab\*c*`},
		{"a?b", `*a\?b*`},
		{"k[1]", `*k\[1\]*`},
		{"{a,b}", `*\{a,b\}*`},
		{`back\slash`, `*back\\slash*`},
		{"plain", "*plain*"},
	} {
		if got := scanMatch(item.keyword); got != item.want {
			t.Fatalf("scanMatch(%q) = %s, want %s", item.keyword, got, item.want)
		}
	}
	if got := scanMatch("a[b"); !strings.Contains(got, `\[`) {
		t.Fatalf("unclosed bracket must still be escaped, got %s", got)
	}
}

// TestEveryViewDeclaresReadCmdInItsHints 面板读内容的治理口径必须由视角自己声明，且与命令台模板同源。
//
// ReadCmd 用于「点开 key 看内容」这路的触发策略判定，同时是前端「申请查看」拼命令的依据；
// 它若与 ConsoleHints 脱节（或漏声明），就会出现「命令台拦得住、面板放过」或判定用了一个不存在的命令名
func TestEveryViewDeclaresReadCmdInItsHints(t *testing.T) {
	for _, desc := range AllDescriptors() {
		if desc.ReadCmd == "" {
			t.Errorf("view %s must declare ReadCmd for policy checks on content reading", desc.View)
			continue
		}
		found := false
		for _, hint := range desc.ConsoleHints {
			if strings.EqualFold(strings.Fields(hint)[0], desc.ReadCmd) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("view %s ReadCmd %q is not one of its console command hints %v", desc.View, desc.ReadCmd, desc.ConsoleHints)
		}
	}
}
