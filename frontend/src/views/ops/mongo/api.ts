import Api from '@/common/Api';
import type { PageResult } from '@/types/common';
import type {
    AggParam,
    BatchDeleteParam,
    BatchUpdateParam,
    ConfirmNameParam,
    DeleteDocsParam,
    ExportParam,
    InsertDocsParam,
    Mongo,
    MongoCollection,
    MongoCollectionMeta,
    MongoCommandSpec,
    MongoDatabase,
    MongoDocPage,
    MongoExportResult,
    MongoListParam,
    MongoWriteResult,
    QueryDocsParam,
    RunCommandParam,
    UpdateDocParam,
} from './types';

/**
 * Mongo 模块 API。
 *
 * 数据面入口与后端一一对应：文档读写走 /query、/docs、/doc、/docs/delete，
 * 结构变更走集合与库的专用端点（不再借 run-command 执行 drop），命令控制台走 /run-command + /commands。
 */
export const mongoApi = {
    // ---- 实例元数据 ----
    mongoList: Api.newGet<PageResult<Mongo>, MongoListParam>('/mongos'),
    testConn: Api.newPost<void>('/mongos/test-conn'),
    saveMongo: Api.newPost<number>('/mongos'),
    deleteMongo: Api.newDelete<void>('/mongos/{id}'),

    // ---- 拓扑 ----
    databases: Api.newGet<MongoDatabase[]>('/mongos/{id}/databases'),
    collections: Api.newGet<MongoCollection[], { id: number; database: string }>('/mongos/{id}/databases/{database}/collections'),
    createCollection: Api.newPost<void, { id: number; database: string; collection: string }>('/mongos/{id}/databases/{database}/collections'),
    dropCollection: Api.newDelete<void, ConfirmNameParam>('/mongos/{id}/databases/{database}/collections/{collection}'),
    dropDatabase: Api.newDelete<void, { id: number; database: string; confirmName: string }>('/mongos/{id}/databases/{database}'),
    collectionMeta: Api.newGet<MongoCollectionMeta, { id: number; database: string; collection: string }>(
        '/mongos/{id}/databases/{database}/collections/{collection}/meta'
    ),
    createIndexes: Api.newPost<void, { id: number; database: string; collection: string; indexes: unknown }>(
        '/mongos/{id}/databases/{database}/collections/{collection}/indexes'
    ),
    dropIndex: Api.newDelete<void, ConfirmNameParam & { index: string }>('/mongos/{id}/databases/{database}/collections/{collection}/indexes/{index}'),

    // ---- 文档数据面 ----
    queryDocs: Api.newPost<MongoDocPage, QueryDocsParam>('/mongos/{id}/query'),
    insertDocs: Api.newPost<MongoWriteResult, InsertDocsParam>('/mongos/{id}/docs'),
    updateDoc: Api.newPut<MongoWriteResult, UpdateDocParam>('/mongos/{id}/doc'),
    deleteDocs: Api.newPost<MongoWriteResult, DeleteDocsParam>('/mongos/{id}/docs/delete'),
    aggregate: Api.newPost<MongoDocPage, AggParam>('/mongos/{id}/aggregate'),
    updateByFilter: Api.newPost<MongoWriteResult, BatchUpdateParam>('/mongos/{id}/docs/update'),
    deleteByFilter: Api.newPost<MongoWriteResult, BatchDeleteParam>('/mongos/{id}/docs/delete-by-filter'),
    exportDocs: Api.newPost<MongoExportResult, ExportParam>('/mongos/{id}/export'),

    // ---- 命令控制台 ----
    commands: Api.newGet<MongoCommandSpec[], { id: number }>('/mongos/{id}/commands'),
    runCommand: Api.newPost<Record<string, unknown>, RunCommandParam>('/mongos/{id}/run-command'),
};
