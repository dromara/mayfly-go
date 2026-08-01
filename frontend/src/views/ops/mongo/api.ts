import Api from '@/common/Api';
import type { PageParam, PageResult } from '@/types/common';
import type { Mongo, MongoDatabasesResult, MongoUpdateResult, MongoDeleteResult, MongoInsertResult } from './types';

export const mongoApi = {
    mongoList: Api.newGet<PageResult<Mongo>, PageParam>('/mongos'),
    mongoTags: Api.newGet<Mongo[]>('/mongos/tags'),
    testConn: Api.newPost<void>('/mongos/test-conn'),
    saveMongo: Api.newPost<void>('/mongos'),
    deleteMongo: Api.newDelete<void>('/mongos/{id}'),
    databases: Api.newGet<MongoDatabasesResult>('/mongos/{id}/databases'),
    collections: Api.newGet<string[]>('/mongos/{id}/collections'),
    runCommand: Api.newPost<Record<string, unknown>>('/mongos/{id}/run-command'),
    findCommand: Api.newPost<Record<string, unknown>[]>('/mongos/{id}/command/find'),
    updateByIdCommand: Api.newPost<MongoUpdateResult>('/mongos/{id}/command/update-by-id'),
    deleteByIdCommand: Api.newPost<MongoDeleteResult>('/mongos/{id}/command/delete-by-id'),
    insertCommand: Api.newPost<MongoInsertResult>('/mongos/{id}/command/insert'),
};
