package application

import (
	"context"
	"sort"
	"strings"

	"mayfly-go/internal/redis/application/keyvalue"
	"mayfly-go/internal/redis/domain/entity"
	"mayfly-go/internal/redis/imsg"
	"mayfly-go/internal/redis/rdm"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
)

// KeyValue key 数据的类型化读写服务。
//
// 与 RunCmd 的分工：命令的构造知识全部在 keyvalue 注册中心的各视角处理器里，
// 本服务只负责「取命令面 → 解析生效视角 → 编排执行 → 审批校验」，
// 因此新增数据类型只需要在 keyvalue 包加一个处理器文件，这里零改动
type KeyValue interface {
	// Views 返回全部数据视角描述符，并按该实例实际支持的命令裁剪（低版本实例不会收到高版本才有的入口）
	Views(ctx context.Context, conn *rdm.RedisConn) ([]*entity.ViewDescriptor, error)

	// Commands 返回实例自身的命令目录，供命令控制台做命令名提示、键参数位置判断与执行前确认
	Commands(ctx context.Context, conn *rdm.RedisConn) ([]*entity.CommandSpec, error)

	// Meta key 元信息（类型、编码、TTL、内存、成员总数）与当前生效视角
	Meta(ctx context.Context, conn *rdm.RedisConn, key, view string) (*entity.KeyMeta, error)

	// Page 分页读取 key 的成员数据
	Page(ctx context.Context, conn *rdm.RedisConn, q *entity.MemberQuery, ack WarnAck) (*entity.MemberPage, error)

	// WriteMember 新增/修改/删除成员，key 不存在时按视角创建该类型的 key
	WriteMember(ctx context.Context, conn *rdm.RedisConn, w *entity.MemberWrite, ack WarnAck) (any, error)

	// RunViewOp 执行视角扩展操作（集合运算、位统计、GEO 检索等）
	RunViewOp(ctx context.Context, conn *rdm.RedisConn, o *entity.OpRequest, ack WarnAck) (any, error)

	// SummarizeKeys 批量获取 key 的类型与剩余过期时间，供 key 列表展示角标与按类型筛选
	SummarizeKeys(ctx context.Context, conn *rdm.RedisConn, keys []string) ([]*entity.KeySummary, error)

	// SetKeyTtl 设置 key 过期时间，ttl <= 0 表示持久化
	SetKeyTtl(ctx context.Context, conn *rdm.RedisConn, key string, ttl int64, ack WarnAck) error

	// RenameKey 重命名 key
	RenameKey(ctx context.Context, conn *rdm.RedisConn, key, newKey string, ack WarnAck) error

	// DeleteKeys 删除若干 key，返回删除数量
	DeleteKeys(ctx context.Context, conn *rdm.RedisConn, keys []string, ack WarnAck) (int64, error)

	// CopyKey 复制 key 到指定库
	CopyKey(ctx context.Context, conn *rdm.RedisConn, key, newKey string, db int, replace bool, ack WarnAck) error
}

var _ KeyValue = (*keyValueAppImpl)(nil)

type keyValueAppImpl struct {
	redisApp Redis `inject:"T"`
}

// keyView 一次操作解析出的上下文：生效处理器、key 原生类型与实际视角
type keyView struct {
	handler keyvalue.Handler
	keyType entity.KeyType
	view    string
}

func (k *keyValueAppImpl) Views(ctx context.Context, conn *rdm.RedisConn) ([]*entity.ViewDescriptor, error) {
	cmd, err := connCmdable(conn)
	if err != nil {
		return nil, err
	}
	return keyvalue.WithSupport(ctx, cmd, keyvalue.AllDescriptors()), nil
}

func (k *keyValueAppImpl) Commands(ctx context.Context, conn *rdm.RedisConn) ([]*entity.CommandSpec, error) {
	cmd, err := connCmdable(conn)
	if err != nil {
		return nil, err
	}
	// COMMAND 不带参数即返回实例全部命令的信息（go-redis 已解析成结构化结果）
	infos, err := cmd.Command(ctx).Result()
	if err != nil {
		// 命令目录只服务输入提示：老版本或受限实例拿不到就返回空列表，
		// 控制台失去提示但仍能执行命令，不能因此把整个 tab 变成报错
		logx.Debugf("redis command info error: %s", err.Error())
		return []*entity.CommandSpec{}, nil
	}

	specs := make([]*entity.CommandSpec, 0, len(infos))
	for _, info := range infos {
		if info == nil {
			continue
		}
		name := strings.ToUpper(info.Name)
		specs = append(specs, &entity.CommandSpec{
			Name:        name,
			Arity:       int(info.Arity),
			Flags:       info.Flags,
			FirstKey:    int(info.FirstKeyPos),
			LastKey:     int(info.LastKeyPos),
			Step:        int(info.StepCount),
			NeedConfirm: needConfirm(name, info),
		})
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].Name < specs[j].Name })
	return specs, nil
}

// needConfirm 执行前是否需要二次确认：平台登记的高危命令，或 Redis 自己标了 dangerous/admin 的命令。
//
// 与执行侧的权限判定同一口径，前端只消费这个布尔，不再自己维护一份命令名单
func needConfirm(name string, info *redis.CommandInfo) bool {
	if rdm.IsDangerousCmd(name) {
		return true
	}
	for _, flags := range [][]string{info.Flags, info.ACLFlags} {
		for _, flag := range flags {
			switch strings.ToLower(strings.TrimPrefix(flag, "@")) {
			case "dangerous", "admin":
				return true
			}
		}
	}
	return false
}

func (k *keyValueAppImpl) Meta(ctx context.Context, conn *rdm.RedisConn, key, view string) (*entity.KeyMeta, error) {
	cmd, err := connCmdable(conn)
	if err != nil {
		return nil, err
	}
	keyType, err := cmd.Type(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	meta := &entity.KeyMeta{
		Key:    key,
		Type:   entity.GetKeyType(keyType),
		Exists: entity.GetKeyType(keyType) != entity.KeyTypeNone,
		TTL:    ttlSeconds(cmd.TTL(ctx, key).Val()),
	}
	meta.Views = viewOptions(meta.Type)
	if !meta.Exists {
		return meta, nil
	}

	meta.MemUse = cmd.MemoryUsage(ctx, key).Val()
	meta.Encoding = cmd.ObjectEncoding(ctx, key).Val()

	hv, err := k.resolve(ctx, cmd, meta.Type, view, key)
	if err != nil {
		// 无内置视角的类型（module 类型等）仍返回元信息，前端据此提示改用命令控制台
		return meta, nil
	}
	meta.View = hv.view
	meta.Caps = hv.handler.Descriptor().Caps
	if size, err := hv.handler.Size(ctx, cmd, key); err == nil {
		meta.Size = size
	}
	return meta, nil
}

// Page 读取成员分页。
//
// key 不存在（刚被删除或过期）时返回空页而不是报错：读接口对不存在的键给空结果是 Redis 语义，
// 前端已按元信息的 exists 展示「key 已不存在」，此处再抛错只会在切换 key 的竞态里弹出无意义的提示。
// 写接口不适用该宽松处理，见 WriteMember
func (k *keyValueAppImpl) Page(ctx context.Context, conn *rdm.RedisConn, q *entity.MemberQuery, ack WarnAck) (*entity.MemberPage, error) {
	cmd, err := connCmdable(conn)
	if err != nil {
		return nil, err
	}
	if keyTypeOf(ctx, cmd, q.Key) == entity.KeyTypeNone {
		return &entity.MemberPage{}, nil
	}
	hv, err := k.resolveByKey(ctx, cmd, q.Key, q.View)
	if err != nil {
		return nil, err
	}
	q.View = hv.view
	// 面板读内容等价于该视角声明的读命令（hash→HGETALL 等），必须与命令台敲同一条命令走同一份名单判定：
	// 此前这里没有任何接缝，实测出现「命令台 GET 被拦、点开面板明文可见同一个值」的旁路
	if err := k.checkReadTrigger(ctx, conn, hv.handler.Descriptor(), q.Key, ack); err != nil {
		return nil, err
	}
	return hv.handler.Load(ctx, cmd, q)
}

// checkReadTrigger 按视角声明的读命令判定本次内容读取能否进行。
//
// 判定用 ReadCmd 而不是底层实际命令（分页时 hash 走 HSCAN）：治理口径必须是管理员能看懂、
// 且在命令台敲同一句会被同样命中的那个命令名，否则配了规则也拦不住真实通路
func (k *keyValueAppImpl) checkReadTrigger(ctx context.Context, conn *rdm.RedisConn, desc *entity.ViewDescriptor, key string, ack WarnAck) error {
	if desc == nil || desc.ReadCmd == "" {
		// 视角没声明读命令就不判定：宁可少判，也不能拿猜出来的命令名去命中规则（那会误拦合法读取）
		return nil
	}
	return k.redisApp.CheckCmdFlow(ctx, conn, []any{desc.ReadCmd, key}, ack)
}

func (k *keyValueAppImpl) WriteMember(ctx context.Context, conn *rdm.RedisConn, w *entity.MemberWrite, ack WarnAck) (any, error) {
	cmd, err := connCmdable(conn)
	if err != nil {
		return nil, err
	}
	keyType := keyTypeOf(ctx, cmd, w.Key)

	// key 尚不存在时只有新增能创建它，视角即目标类型；其余情况按 key 实际类型解析视角
	created := keyType == entity.KeyTypeNone
	var handler keyvalue.Handler
	if created {
		if w.Op != entity.MemberOpCreate {
			return nil, keyvalue.ErrKeyNotFound(ctx)
		}
		if handler = keyvalue.Get(w.View); handler == nil {
			return nil, keyvalue.ErrUnsupportedType(ctx, w.View)
		}
	} else {
		hv, resolveErr := k.resolve(ctx, cmd, keyType, w.View, w.Key)
		if resolveErr != nil {
			return nil, resolveErr
		}
		handler, w.View = hv.handler, hv.view
	}

	if err := checkCaps(handler.Descriptor().Caps, w); err != nil {
		return nil, err
	}

	cmds, err := handler.BuildWrite(ctx, cmd, w)
	if err != nil {
		return nil, err
	}
	res, err := k.runCmds(ctx, conn, cmds, ack)
	if err != nil {
		return nil, err
	}

	// 只有本次真的把 key 从「不存在」写成「存在」才落 TTL；
	// 否则往已有 key 上加一个成员就会顺手改掉它的过期策略
	if w.Ttl > 0 && created {
		if _, err := k.runCmds(ctx, conn, [][]any{{"EXPIRE", w.Key, w.Ttl}}, ack); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (k *keyValueAppImpl) RunViewOp(ctx context.Context, conn *rdm.RedisConn, o *entity.OpRequest, ack WarnAck) (any, error) {
	cmd, err := connCmdable(conn)
	if err != nil {
		return nil, err
	}
	hv, err := k.resolveByKey(ctx, cmd, o.Key, o.View)
	if err != nil {
		return nil, err
	}

	planner, ok := hv.handler.(keyvalue.OpPlanner)
	if !ok {
		return nil, keyvalue.ErrUnsupportedView(ctx, o.View, hv.keyType)
	}
	o.View = hv.view
	cmds, err := planner.PlanOp(ctx, cmd, o)
	if err != nil {
		return nil, err
	}
	return k.runCmds(ctx, conn, cmds, ack)
}

func (k *keyValueAppImpl) SummarizeKeys(ctx context.Context, conn *rdm.RedisConn, keys []string) ([]*entity.KeySummary, error) {
	if len(keys) == 0 {
		return []*entity.KeySummary{}, nil
	}

	typeCmds := make([]*redis.StatusCmd, len(keys))
	ttlCmds := make([]*redis.DurationCmd, len(keys))
	// 单条命令的失败（如 key 已被并发删除）不阻断整批，按 cmd 对象各自取值
	if err := conn.Pipelined(ctx, func(pipe redis.Pipeliner) {
		for i, key := range keys {
			typeCmds[i] = pipe.Type(ctx, key)
			ttlCmds[i] = pipe.TTL(ctx, key)
		}
	}); err != nil {
		logx.Debugf("redis key summary pipeline exec error: %s", err.Error())
	}

	summaries := make([]*entity.KeySummary, 0, len(keys))
	for i, key := range keys {
		summaries = append(summaries, &entity.KeySummary{
			Key:  key,
			Type: entity.GetKeyType(typeCmds[i].Val()),
			TTL:  ttlSeconds(ttlCmds[i].Val()),
		})
	}
	return summaries, nil
}

func (k *keyValueAppImpl) SetKeyTtl(ctx context.Context, conn *rdm.RedisConn, key string, ttl int64, ack WarnAck) error {
	cmds := [][]any{{"PERSIST", key}}
	if ttl > 0 {
		cmds = [][]any{{"EXPIRE", key, ttl}}
	}
	_, err := k.runCmds(ctx, conn, cmds, ack)
	return err
}

func (k *keyValueAppImpl) RenameKey(ctx context.Context, conn *rdm.RedisConn, key, newKey string, ack WarnAck) error {
	if key == newKey {
		// 改成同名：Redis 的 RENAME 会直接报 same object，这里按「无需操作」收敛
		return nil
	}
	cmd, err := connCmdable(conn)
	if err != nil {
		return err
	}
	// RENAME 会静默覆盖目标 key：用户只是想改个名却把另一个 key 连带数据删掉是数据事故，
	// 因此目标已存在时先拒绝；需要覆盖请显式用「复制 key」的覆盖开关
	if cmd.Exists(ctx, newKey).Val() > 0 {
		return errorx.NewBizI(ctx, imsg.ErrRedisKeyAlreadyExist, "key", newKey)
	}
	_, err = k.runCmds(ctx, conn, [][]any{{"RENAME", key, newKey}}, ack)
	return err
}

func (k *keyValueAppImpl) DeleteKeys(ctx context.Context, conn *rdm.RedisConn, keys []string, ack WarnAck) (int64, error) {
	if len(keys) == 0 {
		return 0, errorx.NewBiz("redis key cannot be empty")
	}
	args := make([]any, 0, len(keys)+1)
	args = append(args, "DEL")
	for _, key := range keys {
		args = append(args, key)
	}
	res, err := k.runCmds(ctx, conn, [][]any{args}, ack)
	if err != nil {
		return 0, err
	}
	return cast.ToInt64(res), nil
}

func (k *keyValueAppImpl) CopyKey(ctx context.Context, conn *rdm.RedisConn, key, newKey string, db int, replace bool, ack WarnAck) error {
	args := []any{"COPY", key, newKey}
	if db >= 0 {
		args = append(args, "DB", db)
	}
	if replace {
		args = append(args, "REPLACE")
	}
	res, err := k.runCmds(ctx, conn, [][]any{args}, ack)
	if err != nil {
		return err
	}
	// COPY 的返回值就是「有没有真的复制」：目标已存在又没勾选覆盖时它返回 0 但不算命令失败，
	// 不检查会让界面在什么都没发生的情况下提示成功
	if !cast.ToBool(res) {
		return errorx.NewBizI(ctx, imsg.ErrRedisCopySkipped, "key", newKey)
	}
	return nil
}

// checkCaps 按视角声明的能力位把关成员写操作，能力之外的操作直接拒绝，
// 不需要各处理器再各写一套判断
func checkCaps(caps entity.Capabilities, w *entity.MemberWrite) error {
	allowed := map[entity.MemberOp]bool{
		entity.MemberOpCreate: caps.Create,
		entity.MemberOpUpdate: caps.Update,
		entity.MemberOpDelete: caps.Delete,
	}
	ok, known := allowed[w.Op]
	if !known {
		return keyvalue.ErrUnknownMemberOp(w.Op)
	}
	if !ok {
		return keyvalue.ErrUnsupportedMemberOp(w.View, w.Op)
	}
	if len(w.Members) > 1 && !caps.BatchDelete {
		return keyvalue.ErrUnsupportedMemberOp(w.View, "batch "+w.Op)
	}
	return nil
}

// runCmds 按序执行命令：先过触发策略校验，再统一通过连接执行，保证所有写入口只有一条通路
// runCmds 按序执行命令：先过触发策略校验，再统一通过连接执行，保证所有写入口只有一条通路。
// ack 由发起这次操作的入口带上：key 面板的类型化操作同样是用户点出来的，和命令控制台一样参与
// 「仅提醒」确认，而不是悄悄执行完
func (k *keyValueAppImpl) runCmds(ctx context.Context, conn *rdm.RedisConn, cmds [][]any, ack WarnAck) (any, error) {
	var res any
	// 同一连接上生效的流程定义相同：解析一次逐条复用，别把批量操作变成每条命令两趟库查询
	procdef := k.redisApp.TriggerProcdefOf(ctx, conn)
	for _, args := range cmds {
		if err := k.redisApp.CheckCmdTrigger(ctx, procdef, args, ack); err != nil {
			return nil, err
		}

		result, err := conn.RunCmd(ctx, args...)
		// 读取不存在的键不算失败（并发删除场景），返回空结果继续后续命令
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return nil, err
		}
		res = result
	}
	return res, nil
}

// resolve 按 key 原生类型解析生效视角处理器
func (k *keyValueAppImpl) resolve(ctx context.Context, cmd redis.Cmdable, keyType entity.KeyType, view, key string) (*keyView, error) {
	handler, err := keyvalue.Resolve(ctx, cmd, keyType, view, key)
	if err != nil {
		return nil, err
	}
	if view == "" {
		view = handler.Descriptor().View
	}
	return &keyView{handler: handler, keyType: keyType, view: view}, nil
}

// resolveByKey 先读 key 的实际类型再解析视角
func (k *keyValueAppImpl) resolveByKey(ctx context.Context, cmd redis.Cmdable, key, view string) (*keyView, error) {
	keyType := keyTypeOf(ctx, cmd, key)
	if keyType == entity.KeyTypeNone {
		return nil, keyvalue.ErrKeyNotFound(ctx)
	}
	return k.resolve(ctx, cmd, keyType, view, key)
}

func keyTypeOf(ctx context.Context, cmd redis.Cmdable, key string) entity.KeyType {
	return entity.GetKeyType(cmd.Type(ctx, key).Val())
}

// connCmdable 取连接的命令执行面；模式不对时 GetCmdable 会返回 nil 接口，
// 直接交给视角处理器会在调用第一个命令时 panic，因此在这里统一转成业务错误
func connCmdable(conn *rdm.RedisConn) (redis.Cmdable, error) {
	if conn == nil {
		return nil, errorx.NewBiz("redis connection not exist")
	}
	cmd := conn.GetCmdable()
	if cmd == nil {
		return nil, errorx.NewBizf("unsupported redis mode: %s", conn.Info.Mode)
	}
	return cmd, nil
}

// ttlSeconds 剩余过期秒数；任何负值（Redis 的 -1 无过期 / -2 key 不存在）都归一成 -1「永久」，
// key 是否存在由 EXISTS 单独判定，不靠这里的返回值区分
func ttlSeconds(ttl time.Duration) int64 {
	if ttl < 0 {
		return -1
	}
	return int64(ttl.Seconds())
}

// viewOptions 该原生类型下可切换的视角清单
func viewOptions(keyType entity.KeyType) []*entity.ViewOption {
	descs := keyvalue.Descriptors(keyType)
	options := make([]*entity.ViewOption, 0, len(descs))
	for _, desc := range descs {
		options = append(options, &entity.ViewOption{View: desc.View, Label: desc.Label})
	}
	return options
}
