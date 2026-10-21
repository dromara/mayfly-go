import Api from '@/common/Api';
import type { PageParam, PageResult } from '@/types/common';
import type { AgentProfileInfo, AgentRun, AgentTask, AgentTrigger, AgentProposal, ChatResource, GovernanceStats } from './types';

/**
 * Agent 装配档目录：档位集合与「是否需运行许可」的唯一真源。
 *
 * 后端 RegisterProfile 是开放扩展点，前端不得再抄一份静态档位清单：新增一档在此自动可选可筛。
 */
export const agentProfileApi = {
    list: Api.newGet<AgentProfileInfo[]>('/agentops/profiles'),
};

/**
 * 事件触发源目录：已注册来源的类型标识清单。
 *
 * 后端 trigger.Registry 是开放扩展点，前端不得再拄一份静态来源清单：新增一种来源在此自动可选可筛
 * （展示文案仍由前端字典持有，未配译文时回退原始类型键）
 */
export const agentTriggerSourceApi = {
    list: Api.newGet<string[]>('/agentops/trigger-sources'),
};

/**
 * 可引用资源列表（与聊天 `@` 触发菜单同源，全类型一次返回）：
 * 触发器匹配条件的「资源白名单」可视化选择器用它按类型取资源，让配置者选名字而非手抄 id
 */
export const chatResourceApi = {
    list: Api.newGet<ChatResource[]>('/ai/chat/resources'),
};

/** 定时/计划任务：CRUD + 手动触发（触发后由后端有界工作池异步产出运行记录） */
export const agentTaskApi = {
    list: Api.newGet<PageResult<AgentTask>, PageParam>('/agentops/tasks'),
    detail: Api.newGet<AgentTask>('/agentops/tasks/{id}'),
    // 保存入参为 AgentTaskForm；此处不绑定参数泛型（P=unknown），以兼容 AutoFormDrawer 的 confirm-api 契约（与 cronJobApi.save 一致）
    save: Api.newPost<number>('/agentops/tasks'),
    del: Api.newDelete<void>('/agentops/tasks/{ids}'),
    run: Api.newPost<{ runId: number; runNo: string }>('/agentops/tasks/run/{id}'),
};

/** 运行记录（执行历史）：查询 + 详情 + 取消（急停）+ 人类反馈 + 离线评测回放 */
export const agentRunApi = {
    list: Api.newGet<PageResult<AgentRun>, PageParam>('/agentops/runs'),
    detail: Api.newGet<AgentRun>('/agentops/runs/{id}'),
    cancel: Api.newPost<void>('/agentops/runs/{id}/cancel'),
    // 人类反馈（点赞/点踩 + 备注）：落库并沉淀为操作员经验记忆（学习闭环）；id 为路径参数
    feedback: Api.newPost<void, { id: number; feedback: string; note?: string }>('/agentops/runs/{id}/feedback'),
    // 离线评测回放：诊断档重跑原始输入 + 审查档对比打分（异步执行，返回评测运行）
    eval: Api.newPost<{ runId: number; runNo: string }>('/agentops/runs/{id}/eval'),
};

/** 多 Agent 编排（Supervisor 模式）：Planner 拆解 → 步骤执行（写步骤转提案审批）→ Reviewer 审查 */
export const agentOrchestrationApi = {
    // 发起编排（异步执行，返回父运行；编排轨迹在执行历史中以 triggerRef=orch:* 子运行呈现）
    orchestrate: Api.newPost<
        { runId: number; runNo: string },
        {
            goal: string;
            instruction?: string;
            ownerAccountId?: number;
            timeoutSeconds?: number;
            maxStep?: number;
            tokenBudget?: number;
        }
    >('/agentops/orchestrations'),
};

/** 事件驱动触发器：CRUD（告警联动经 EventBus 自动派发，无需前端入口；webhook 推送地址由令牌拼接） */
export const agentTriggerApi = {
    list: Api.newGet<PageResult<AgentTrigger>, PageParam>('/agentops/triggers'),
    detail: Api.newGet<AgentTrigger>('/agentops/triggers/{id}'),
    // 保存入参为 AgentTriggerForm；不绑定参数泛型（P=unknown）以兼容 AutoFormDrawer 的 confirm-api 契约（与 agentTaskApi.save 一致）
    save: Api.newPost<{ id: number; webhookToken: string }>('/agentops/triggers'),
    del: Api.newDelete<void>('/agentops/triggers/{ids}'),
};

/** 变更提案：查询 + 详情 + 删除（由诊断档经 propose_remediation 工具登记，非用户手建；审批在 flow 工单页完成） */
export const agentProposalApi = {
    list: Api.newGet<PageResult<AgentProposal>, PageParam>('/agentops/proposals'),
    detail: Api.newGet<AgentProposal>('/agentops/proposals/{id}'),
    del: Api.newDelete<void>('/agentops/proposals/{ids}'),
};

/** 治理：看板统计（days 窗口）+ 急停开关 + 一键清场（全局性动作，权限点 agentops:governance:killswitch） */
export const agentGovernanceApi = {
    stats: Api.newGet<GovernanceStats, { days?: number }>('/agentops/governance/stats'),
    setKillSwitch: Api.newPost<void>('/agentops/governance/kill-switch'),
    cancelAll: Api.newPost<{ canceled: number }>('/agentops/governance/cancel-all'),
};
