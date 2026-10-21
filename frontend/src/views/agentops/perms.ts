/**
 * AgentOps 权限码集中声明。
 *
 * 三方同源：本文件 ↔ 后端 `internal/agentops/api/*.go` 的 `RequiredPermissionCode` ↔
 * 迁移 `agentopsResources`/`agentopsExtendResources` 的资源 code。
 *
 * 为什么集中：这些字符串曾散落在 5 个页面的 `perms` 字面量里，后端调整权限码时前端要逐页搜；
 * 集中一处后，三方对齐只需比对本文件与后端两张清单，避免漏改导致「按钮可见但接口 403」。
 */
export const AgentopsPerms = {
    /** 保存/编辑任务（含查看任务详情） */
    taskSave: 'agentops:task:save',
    taskDel: 'agentops:task:del',
    taskRun: 'agentops:task:run',

    /** 取消在途运行（人工取消） */
    runCancel: 'agentops:run:cancel',
    /** 运行反馈（点赞/点踩回流记忆） */
    runFeedback: 'agentops:run:feedback',
    /** 离线评测回放 */
    runEval: 'agentops:run:eval',
    /** 查看全部人的执行历史（不勾只能看到自己作为运行身份的记录） */
    runQueryAll: 'agentops:run:queryAll',
    /** 发起多 Agent 编排 */
    orchestrationRun: 'agentops:orchestration:run',

    /** 保存/编辑触发器；查看触发器详情（含 webhook 推送令牌）也用它 */
    triggerSave: 'agentops:trigger:save',
    triggerDel: 'agentops:trigger:del',

    proposalDel: 'agentops:proposal:del',

    /** 急停开关与一键清场（全局性危险动作） */
    governanceKillSwitch: 'agentops:governance:killswitch',
    /** 以他人账号身份派发运行（高危：运行身份即工具权限边界） */
    ownerImpersonate: 'agentops:owner:impersonate',
} as const;
