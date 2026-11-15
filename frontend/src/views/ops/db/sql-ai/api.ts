/**
 * SQL 草稿接口。
 *
 * 单独一份而不并进 db/api.ts：草稿是「文本产物」而非库操作，接口口径与审计口径都不同，
 * 混进主 api 桶会让「这个模块会不会动数据」变得难以判断。
 */
import Api from '@/common/Api';
import type { AiTerm, SqlDraftRequest, SqlDraftResult } from './types';

export const sqlDraftApi = {
    /** 生成草稿：只读结构 + 一次模型调用，不执行任何语句 */
    draft: Api.newPost<SqlDraftResult, SqlDraftRequest & { dbId: number }>('/dbs/{dbId}/sql-draft'),
};

/**
 * 业务词典（语义层）接口。
 *
 * 与草稿同属一个目录而不是放 db/api.ts：它是「喂给文生 SQL 的材料」，
 * 只有 sqldraft 消费，放进库资源主 api 会让它看起来像通用配置。
 */
export const aiTermApi = {
    /** 某库资源的全部条目（不按库过滤：跨库口径冲突要在界面上看得见） */
    list: Api.newGet<AiTerm[]>('/dbs/ai-terms'),
    save: Api.newPost<void, AiTerm>('/dbs/ai-terms'),
    update: Api.newPut<void, AiTerm>('/dbs/ai-terms'),
    remove: Api.newDelete<void, { id: number }>('/dbs/ai-terms/{id}'),
};
