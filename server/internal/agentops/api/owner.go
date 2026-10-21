package api

import (
	"mayfly-go/internal/agentops/imsg"
	"mayfly-go/internal/pkg/consts"
	sysapp "mayfly-go/internal/sys/application"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/req"
)

// PermissionRunAsOwner 允许「以他人账号身份」派发 Agent 运行的权限点。
// 属于高危授权：定时/事件触发没有人工在场，运行是以该账号权限读资源、并登记变更提案的。
const PermissionRunAsOwner = "agentops:owner:impersonate"

// RunOwner 运行服务身份（owner）：账号 id 与账号名快照
type RunOwner struct {
	AccountId uint64
	Username  string
}

// resolveRunOwner 运行身份解析单点（任务 / 触发器 / 编排三处共用）：
//   - 未指定或指定自己 → 以当前登录账号为运行身份（谁建谁负责）
//   - 指定他人 → 仅当调用者是超级管理员，或持有 PermissionRunAsOwner
//
// 为什么必须在 api 层且只此一处：无头运行会把 owner 注入成运行上下文的登录态，工具权限与数据
// 可见面全部按该账号收敛（见 ai/application.RunHeadless 的 fail-closed 注释）。owner 若是请求方
// 可任意填写的字段，「能保存一条定时任务」就等于「能以管理员身份读取任意监控数据并登记变更提案」。
// application 层拿不到调用者语义（入队请求里只有 owner 本身），判不了这件事，因此不收在那里。
func resolveRunOwner(rc *req.Ctx, accountApp sysapp.Account, requestedAccountId uint64) (*RunOwner, error) {
	caller := rc.GetLoginAccount()
	if caller == nil || caller.Id == 0 {
		return nil, errorx.NewBizI(rc.MetaCtx, imsg.ErrLoginAccountRequired)
	}
	if requestedAccountId == 0 || requestedAccountId == caller.Id {
		return &RunOwner{AccountId: caller.Id, Username: caller.Username}, nil
	}
	if !mayRunAsOtherAccount(caller.Id) {
		return nil, errorx.NewBizI(rc.MetaCtx, imsg.ErrNotPermittedImpersonate)
	}
	owner, err := accountApp.GetById(requestedAccountId)
	if err != nil {
		return nil, errorx.NewBizI(rc.MetaCtx, imsg.ErrOwnerAccountNotFound)
	}
	return &RunOwner{AccountId: owner.Id, Username: owner.Username}, nil
}

// mayRunAsOtherAccount 「可代他人身份运行」的授权判定：超级管理员或持有专门权限点
func mayRunAsOtherAccount(accountId uint64) bool {
	if accountId == consts.AdminId {
		return true
	}
	registry := req.GetPermissionCodeRegistery()
	return registry != nil && registry.HasCode(accountId, PermissionRunAsOwner)
}

// assertResourceOwned 归属断言：非「可代他人身份运行」者只能操作自己 owner 的资源。
//
// 为什么复用 PermissionRunAsOwner 而不是新增一个 queryAll 权限点：能以任何账号身份
// 派发运行的人，逻辑上就已能操作该账号的任务/触发器；再加一个权限点只会造成
// 「两个码要同时配」的运维陷阱。run 侧早先已按同思路实现（canQueryAllRuns），那里用独立码是
// 因为当时还没有 impersonate 概念，两边现在按这里收敛。
//
// 越权统一报「不存在」而不是「无权限」：否则可以用错误码差异探测他人资源 id 是否存在。
func assertResourceOwned(rc *req.Ctx, ownerAccountId uint64) error {
	caller := runCallerId(rc)
	if mayRunAsOtherAccount(caller) || ownerAccountId == caller {
		return nil
	}
	return errorx.NewBizI(rc.MetaCtx, imsg.ErrResourceNotFound)
}

// scopeOwnerToCaller 列表收窄：非全局可见者只查自己 owner 的资源。
// 用回调而不是泛型接口：OwnerAccountId 是各 entity 自己的字段，为了一个查询辅助去给
// domain 实体加方法属于反向污染。
func scopeOwnerToCaller(rc *req.Ctx, setOwner func(accountId uint64)) {
	caller := runCallerId(rc)
	if mayRunAsOtherAccount(caller) {
		return
	}
	setOwner(caller)
}
