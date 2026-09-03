import Api from '@/common/Api';
import type { Conversation, RunningTurnVO, TurnGroupVO } from './protocol/types';
import type { SkillItem } from './input/types';

export const aiApi = {
    // 会话管理
    listConversations: Api.newGet<Conversation[]>('/ai/chat/conversations'),
    createConversation: Api.newPost<Conversation, { title?: string }>('/ai/chat/conversations'),
    deleteConversation: Api.newDelete('/ai/chat/conversations/{id}'),
    renameConversation: Api.newPost('/ai/chat/conversations/rename'),
    // TurnItem 查询（分页加载历史 turn 分组；会话入口一次性加载，按 turn 查询单独入口未启用）
    getTurnItems: Api.newGet<TurnGroupVO[], { id: number; beforeTurnId?: string; limit?: number }>('/ai/chat/conversations/{id}/items'),
    // 技能引用（'/' 触发菜单数据源；资源引用由 @ 触发的资源树面板走 ops 资源树接口懒加载）
    listSkills: Api.newGet<SkillItem[]>('/ai/chat/skills'),
    // 运行中 turn 列表（会话列表执行中指示器数据源，进入页面时校正）
    listRunningTurns: Api.newGet<RunningTurnVO[]>('/ai/chat/running-turns'),
};
