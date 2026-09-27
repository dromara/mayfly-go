package api

import (
	"mayfly-go/internal/pkg/event"
	"mayfly-go/internal/redis/api/form"
	"mayfly-go/internal/redis/application/keyvalue"
	"mayfly-go/internal/redis/domain/entity"
	"mayfly-go/internal/redis/imsg"
	"mayfly-go/internal/redis/rdm"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

// redis 数据面权限码，与前端按钮权限、角色资源配置同源，避免「只在前端隐藏按钮」的假管控
const (
	permDataSave = "redis:data:save"
	permDataDel  = "redis:data:del"
)

// 本文件的错误一律直接透出：视角处理器与应用服务给出的已是可读的业务文案（含 i18n 与 Redis 原始提示），
// 再叠一层英文前缀会让提示变成「rename key error: 目标 key 已存在…」这种中英混排

// Views 返回全部内置数据视角描述符（已按该实例能力裁剪），前端据此渲染类型选择器与表格结构，
// 新增类型前端零改动
func (r *Redis) Views(rc *req.Ctx) {
	ri := r.getRedisConn(rc)

	descs, err := r.keyValueApp.Views(rc.MetaCtx, ri)
	biz.ErrIsNil(err)
	rc.ResData = descs
}

// Commands 返回实例自身的命令目录（命令名、参数个数与标志、键参数位置、是否需确认）
func (r *Redis) Commands(rc *req.Ctx) {
	ri := r.getRedisConn(rc)

	specs, err := r.keyValueApp.Commands(rc.MetaCtx, ri)
	biz.ErrIsNil(err)
	rc.ResData = specs
}

// KeyMeta key 元信息：类型、编码、TTL、内存、成员总数与可切换视角
func (r *Redis) KeyMeta(rc *req.Ctx) {
	ri, key := r.checkKeyAndGetRedisConn(rc)

	meta, err := r.keyValueApp.Meta(rc.MetaCtx, ri, key, rc.Query("view"))
	biz.ErrIsNil(err)
	rc.ResData = meta
}

// KeyValues 分页读取 key 的成员数据
func (r *Redis) KeyValues(rc *req.Ctx) {
	ri := r.getRedisConn(rc)
	pageForm := rc.BindJson[form.KeyMemberPageForm]()

	page, err := r.keyValueApp.Page(rc.MetaCtx, ri, &entity.MemberQuery{
		Key:     pageForm.Key,
		View:    pageForm.View,
		Cursor:  pageForm.Cursor,
		Offset:  pageForm.Offset,
		Size:    pageForm.Size,
		Keyword: pageForm.Keyword,
	})
	biz.ErrIsNil(err)
	rc.ResData = page
}

// PutKeyValue 成员写操作（新增 / 修改 / 删除，op 决定语义），key 不存在时按视角创建对应类型的 key。
//
// 三种操作共用一个入口：请求里本就带 op 与成员行数据，再按操作拆成不同 method 只会让
// 「delete 无法携带 body」这类传输差异变成额外约束
func (r *Redis) PutKeyValue(rc *req.Ctx) {
	ri := r.getRedisConn(rc)
	writeForm := rc.BindJson[form.KeyMemberWriteForm]()
	biz.IsTrue(memberOps[writeForm.Op], "unsupported member op: %s", writeForm.Op)
	if writeForm.Op == entity.MemberOpDelete {
		biz.IsTrue(len(writeForm.Members) > 0 || writeForm.Member != nil, "key member cannot be empty")
	}

	rc.ReqParam = collx.Kvs("redis", ri.Info.Name, "key", writeForm.Key, "view", writeForm.View, "op", writeForm.Op)
	// 删除成员要求删除权限，新增/修改要求保存权限
	if writeForm.Op == entity.MemberOpDelete {
		r.requirePerm(rc, permDataDel)
	} else {
		r.requirePerm(rc, permDataSave)
	}
	r.publishResourceOpEvent(rc, ri)

	res, err := r.keyValueApp.WriteMember(rc.MetaCtx, ri, &entity.MemberWrite{
		Key:     writeForm.Key,
		View:    writeForm.View,
		Op:      writeForm.Op,
		Member:  writeForm.Member,
		Members: writeForm.Members,
		Args:    writeForm.Args,
		Ttl:     writeForm.Ttl,
	})
	biz.ErrIsNil(err)
	rc.ResData = res
}

// memberOps 成员写入口接受的操作集合，与前端契约中的 op 取值一致
var memberOps = map[string]bool{
	entity.MemberOpCreate: true,
	entity.MemberOpUpdate: true,
	entity.MemberOpDelete: true,
}

// RunKeyOp 执行视角扩展操作，写操作受保存权限约束
func (r *Redis) RunKeyOp(rc *req.Ctx) {
	ri := r.getRedisConn(rc)
	opForm := rc.BindJson[form.KeyOpForm]()

	rc.ReqParam = collx.Kvs("redis", ri.Info.Name, "key", opForm.Key, "view", opForm.View, "op", opForm.Op)
	if isWriteViewOp(opForm.View, opForm.Op) {
		r.requirePerm(rc, permDataSave)
		r.publishResourceOpEvent(rc, ri)
	}

	res, err := r.keyValueApp.RunViewOp(rc.MetaCtx, ri, &entity.OpRequest{
		Key:  opForm.Key,
		View: opForm.View,
		Op:   opForm.Op,
		Args: opForm.Args,
	})
	biz.ErrIsNil(err)
	rc.ResData = res
}

// KeySummary 批量获取 key 的类型与剩余过期时间，供 key 树展示角标与按类型筛选
func (r *Redis) KeySummary(rc *req.Ctx) {
	ri := r.getRedisConn(rc)
	keysForm := rc.BindJson[form.KeysForm]()

	summaries, err := r.keyValueApp.SummarizeKeys(rc.MetaCtx, ri, keysForm.Keys)
	biz.ErrIsNil(err)
	rc.ResData = summaries
}

// SetKeyTtl 设置 key 过期时间，ttl <= 0 表示持久化
func (r *Redis) SetKeyTtl(rc *req.Ctx) {
	ri := r.getRedisConn(rc)
	ttlForm := rc.BindJson[form.KeyTtlForm]()

	rc.ReqParam = collx.Kvs("redis", ri.Info.Name, "key", ttlForm.Key, "ttl", ttlForm.Ttl)
	r.requirePerm(rc, permDataSave)
	r.publishResourceOpEvent(rc, ri)

	biz.ErrIsNil(r.keyValueApp.SetKeyTtl(rc.MetaCtx, ri, ttlForm.Key, ttlForm.Ttl))
}

// RenameKey 重命名 key
func (r *Redis) RenameKey(rc *req.Ctx) {
	ri := r.getRedisConn(rc)
	renameForm := rc.BindJson[form.KeyRenameForm]()

	rc.ReqParam = collx.Kvs("redis", ri.Info.Name, "key", renameForm.Key, "newKey", renameForm.NewKey)
	r.requirePerm(rc, permDataSave)
	r.publishResourceOpEvent(rc, ri)

	biz.ErrIsNil(r.keyValueApp.RenameKey(rc.MetaCtx, ri, renameForm.Key, renameForm.NewKey))
}

// CopyKey 复制 key，可跨库
func (r *Redis) CopyKey(rc *req.Ctx) {
	ri := r.getRedisConn(rc)
	copyForm := rc.BindJson[form.KeyCopyForm]()
	// 集群模式没有多库概念，跨库复制直接拒绝（而不是让服务端报模糊错误）
	biz.IsTrue(ri.Info.Mode != rdm.ClusterMode || copyForm.TargetDb == nil || *copyForm.TargetDb == ri.Info.Db, "redis cluster does not support copying key to another db")

	db := ri.Info.Db
	if copyForm.TargetDb != nil {
		db = *copyForm.TargetDb
	}
	rc.ReqParam = collx.Kvs("redis", ri.Info.Name, "key", copyForm.Key, "newKey", copyForm.NewKey, "targetDb", db)
	r.requirePerm(rc, permDataSave)
	r.publishResourceOpEvent(rc, ri)

	biz.ErrIsNil(r.keyValueApp.CopyKey(rc.MetaCtx, ri, copyForm.Key, copyForm.NewKey, db, copyForm.Replace))
}

// DeleteKeys 批量删除 key（POST 而非 DELETE：key 本身可能含逗号，只能按 JSON 数组传输）
func (r *Redis) DeleteKeys(rc *req.Ctx) {
	ri := r.getRedisConn(rc)
	keysForm := rc.BindJson[form.KeysForm]()

	rc.ReqParam = collx.Kvs("redis", ri.Info.Name, "keys", keysForm.Keys)
	r.requirePerm(rc, permDataDel)
	r.publishResourceOpEvent(rc, ri)

	delNum, err := r.keyValueApp.DeleteKeys(rc.MetaCtx, ri, keysForm.Keys)
	biz.ErrIsNil(err)
	rc.ResData = collx.Kvs("delNum", delNum)
}

// requirePerm 校验当前账号是否拥有指定权限码
func (r *Redis) requirePerm(rc *req.Ctx, code string) {
	biz.IsTrue(req.GetPermissionCodeRegistery().HasCode(rc.GetLoginAccount().Id, code), "no permission: %s", code)
}

// requireDangerPerm 高危命令（FLUSHDB/CONFIG/KEYS 等）要求独立的数据删除权限。
// 这里不复用 requirePerm：裸权限码对操作者没有信息量，需要说清「是这条命令太危险」
func (r *Redis) requireDangerPerm(rc *req.Ctx) {
	if req.GetPermissionCodeRegistery().HasCode(rc.GetLoginAccount().Id, permDataDel) {
		return
	}
	biz.ErrIsNil(errorx.NewBizI(rc.MetaCtx, imsg.ErrRedisDangerousCmd))
}

// publishResourceOpEvent 通知资源正在被操作，供资源树的在线状态与角标刷新
func (r *Redis) publishResourceOpEvent(rc *req.Ctx, ri *rdm.RedisConn) {
	if len(ri.Info.CodePath) == 0 {
		return
	}
	global.EventBus.Publish(rc.MetaCtx, event.EventTopicResourceOp, ri.Info.CodePath[0])
}

// isWriteViewOp 判断视角操作是否为写操作，以描述符的声明为唯一真源
func isWriteViewOp(view, op string) bool {
	handler := keyvalue.Get(view)
	if handler == nil {
		return true
	}
	for _, spec := range handler.Descriptor().Ops {
		if spec.Name == op {
			return spec.Write
		}
	}
	// 未声明的操作一律按写操作处理，权限不足的宁可多要求一次，不能反过来放行
	return true
}
