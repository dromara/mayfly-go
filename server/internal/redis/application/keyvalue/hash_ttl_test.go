package keyvalue

import (
	"context"
	"testing"

	"mayfly-go/internal/redis/domain/entity"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
)

// hashTtlFixture 把「只构造不执行」的契约跑成真实读写，等价于 UI 上的操作闭环
type hashTtlFixture struct {
	t     *testing.T
	ctx   context.Context
	cmd   *redis.Client
	key   string
	write Handler
}

func newHashTtlFixture(t *testing.T) *hashTtlFixture {
	t.Helper()

	return &hashTtlFixture{t: t, ctx: context.Background(), cmd: connect(t), key: "mayfly:hashttl", write: Get(ViewHash)}
}

func (f *hashTtlFixture) build(op entity.MemberOp, args map[string]string, row *entity.Member) [][]any {
	f.t.Helper()

	cmds, err := f.write.BuildWrite(f.ctx, f.cmd, &entity.MemberWrite{Key: f.key, View: ViewHash, Op: op, Args: args, Member: row})
	if err != nil {
		f.t.Fatalf("build %s failed: %s", op, err.Error())
	}
	return cmds
}

func (f *hashTtlFixture) run(cmds [][]any) {
	f.t.Helper()

	for _, built := range cmds {
		if err := f.cmd.Do(f.ctx, built...).Err(); err != nil {
			f.t.Fatalf("exec %v failed: %s", built, err.Error())
		}
	}
}

// ttlColumn 从成员列表里取某字段的过期读数，验证「派生列随成员一起返回」
func (f *hashTtlFixture) ttlColumn(field string) int64 {
	f.t.Helper()

	page, err := f.write.Load(f.ctx, f.cmd, &entity.MemberQuery{Key: f.key, View: ViewHash, Size: 50})
	if err != nil {
		f.t.Fatalf("load members failed: %s", err.Error())
	}
	for _, member := range page.Members {
		if member.Field != field {
			continue
		}
		raw, ok := member.Extra[extraTtl]
		if !ok {
			f.t.Fatalf("field [%s] carries no ttl column", field)
		}
		return cast.ToInt64(raw)
	}
	f.t.Fatalf("field [%s] not found in members", field)
	return 0
}

// TestHashFieldTtlWriteRead 新增带过期 → 读到剩余秒数 → 改值不丢过期 → 显式清除
func TestHashFieldTtlWriteRead(t *testing.T) {
	f := newHashTtlFixture(t)

	f.run(f.build(entity.MemberOpCreate, map[string]string{argField: "a", argValue: "1", extraTtl: "300"}, nil))
	if left := f.ttlColumn("a"); left < 290 || left > 300 {
		t.Fatalf("new field should carry the requested ttl, got %d", left)
	}

	// 改值时不填过期：按读取到的剩余秒续回（HSET 本身会清掉字段过期）
	f.run(f.build(entity.MemberOpUpdate, map[string]string{argField: "a", argValue: "2"},
		&entity.Member{Field: "a", Extra: map[string]string{extraTtl: "280"}}))
	if left := f.ttlColumn("a"); left < 270 || left > 280 {
		t.Fatalf("update must keep the field ttl, got %d", left)
	}

	// 填 0 表示清除：必须走 HPERSIST，HEXPIRE 传 0 的语义是立即过期，会把字段删掉
	f.run(f.build(entity.MemberOpUpdate, map[string]string{argField: "a", argValue: "3", extraTtl: "0"},
		&entity.Member{Field: "a", Extra: map[string]string{extraTtl: "270"}}))
	if left := f.ttlColumn("a"); left != -1 {
		t.Fatalf("cleared field should report -1 (no expiry), got %d", left)
	}
	if _, err := f.cmd.HGet(f.ctx, f.key, "a").Result(); err != nil {
		t.Fatalf("field must survive the ttl clear: %s", err.Error())
	}
}

// TestHashRenameFieldKeepsTtl 改字段名时先写新再删旧，过期要落在新字段上
func TestHashRenameFieldKeepsTtl(t *testing.T) {
	f := newHashTtlFixture(t)

	f.run(f.build(entity.MemberOpCreate, map[string]string{argField: "old", argValue: "v", extraTtl: "600"}, nil))
	f.run(f.build(entity.MemberOpUpdate, map[string]string{argField: "new", argValue: "v"},
		&entity.Member{Field: "old", Extra: map[string]string{extraTtl: "590"}}))

	if left := f.ttlColumn("new"); left < 580 || left > 590 {
		t.Fatalf("renamed field should inherit the ttl, got %d", left)
	}
	if _, err := f.cmd.HGet(f.ctx, f.key, "old").Result(); err != redis.Nil {
		t.Fatalf("the old field must be deleted, got err %v", err)
	}
}

// TestHashExpireOpCommands 批量设置/清除字段过期的命令构造（不碰连接）
func TestHashExpireOpCommands(t *testing.T) {
	planner, ok := Get(ViewHash).(OpPlanner)
	if !ok {
		t.Fatal("hash handler must implement OpPlanner")
	}
	ctx := context.Background()

	cmds, err := planner.PlanOp(ctx, nil, &entity.OpRequest{Key: "k", View: ViewHash, Op: opHExpire, Args: map[string]string{argFields: "a\nb", extraTtl: "60"}})
	if err != nil {
		t.Fatalf("plan hexpire failed: %s", err.Error())
	}
	want := []any{"HEXPIRE", "k", int64(60), "FIELDS", 2, "a", "b"}
	if len(cmds) != 1 || len(cmds[0]) != len(want) {
		t.Fatalf("unexpected command shape: %v", cmds)
	}
	for i, arg := range want {
		if cmds[0][i] != arg {
			t.Fatalf("command arg [%d] should be %v, got %v (%v)", i, arg, cmds[0][i], cmds[0])
		}
	}

	// 留空或 0 一律按「清除过期」处理
	for _, raw := range []string{"", "0", "-5"} {
		cmds, err := planner.PlanOp(ctx, nil, &entity.OpRequest{Key: "k", View: ViewHash, Op: opHExpire, Args: map[string]string{argFields: "a", extraTtl: raw}})
		if err != nil {
			t.Fatalf("hexpire with ttl %q failed: %s", raw, err.Error())
		}
		if cast.ToString(cmds[0][0]) != "HPERSIST" {
			t.Fatalf("hexpire with ttl %q must clear via HPERSIST, got %v", raw, cmds[0])
		}
	}

	if _, err := planner.PlanOp(ctx, nil, &entity.OpRequest{Key: "k", View: ViewHash, Op: opHExpire, Args: map[string]string{extraTtl: "60"}}); err == nil {
		t.Fatal("hexpire without any field must be rejected")
	}
}

// TestParseFieldTtl 入参解析：留空是「不改动」，0/负数是「清除」，非整数必须报错而不是被当成 0
func TestParseFieldTtl(t *testing.T) {
	cases := []struct {
		raw  string
		want int64
	}{
		{raw: "", want: -1},
		{raw: "  ", want: -1},
		{raw: "300", want: 300},
		{raw: "0", want: 0},
		{raw: "-10", want: 0},
	}
	for _, item := range cases {
		got, err := parseFieldTtl(map[string]string{extraTtl: item.raw})
		if err != nil {
			t.Fatalf("parse %q failed: %s", item.raw, err.Error())
		}
		if got != item.want {
			t.Fatalf("parse %q should give %d, got %d", item.raw, item.want, got)
		}
	}

	if _, err := parseFieldTtl(map[string]string{extraTtl: "abc"}); err == nil {
		t.Fatal("a non numeric ttl must be rejected instead of being treated as 0")
	}
}

// TestTrimHelpersKeepSharedDescriptor 裁剪描述符必须返回副本：一次探测不能永久影响所有实例
func TestTrimHelpersKeepSharedDescriptor(t *testing.T) {
	columns, fields, ops := len(hashDesc.Columns), len(hashDesc.Form.Fields), len(hashDesc.Ops)

	if got := len(withoutColumn(hashDesc.Columns, extraTtl)); got != columns-1 {
		t.Fatalf("withoutColumn should drop exactly the ttl column, got %d", got)
	}
	if got := len(withoutFormField(hashDesc.Form, extraTtl).Fields); got != fields-1 {
		t.Fatalf("withoutFormField should drop exactly the ttl field, got %d", got)
	}
	if got := len(withoutOp(hashDesc.Ops, opHExpire)); got != ops-1 {
		t.Fatalf("withoutOp should drop exactly the hexpire op, got %d", got)
	}

	if len(hashDesc.Columns) != columns || len(hashDesc.Form.Fields) != fields || len(hashDesc.Ops) != ops {
		t.Fatal("the process-wide descriptor must not be mutated by trimming")
	}
}

// TestSupportsFieldTtlProbe 探测必须是只读的：不能在目标库留下任何 key
func TestSupportsFieldTtlProbe(t *testing.T) {
	cmd := connect(t)
	ctx := context.Background()

	if !supportsFieldTtl(ctx, cmd) {
		t.Skip("local redis has no hash field ttl support")
	}
	if exists, err := cmd.Exists(ctx, ttlProbeKey).Result(); err != nil || exists != 0 {
		t.Fatalf("the capability probe must not create any key, exists=%d err=%v", exists, err)
	}
}
