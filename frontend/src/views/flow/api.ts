import Api from '@/common/Api';
import type { PageParam, PageResult } from '@/types/common';
import type { Procdef, ProcdefResourceParam, Procinst, ProcinstTask, HisProcinstOp } from './types';

export const procdefApi = {
    list: Api.newGet<PageResult<Procdef>, PageParam>('/flow/procdefs'),
    detail: Api.newGet<Procdef>('/flow/procdefs/detail/{id}'),
    flowDef: Api.newGet<Record<string, unknown>>('/flow/procdefs/flowdef/{id}'),
    getByResource: Api.newGet<Procdef, ProcdefResourceParam>('/flow/procdefs/{resourceType}/{resourceCode}'),
    save: Api.newPost<void>('/flow/procdefs'),
    saveFlowDef: Api.newPost<void>('/flow/procdefs/flowdef'),
    del: Api.newDelete<void>('/flow/procdefs/{id}'),
};

export const procinstApi = {
    list: Api.newGet<PageResult<Procinst>, PageParam>('/flow/procinsts'),
    start: Api.newPost<void>('/flow/procinsts/start'),
    detail: Api.newGet<Procinst>('/flow/procinsts/{id}'),
    cancel: Api.newPost<void>('/flow/procinsts/{id}/cancel'),
    hisOp: Api.newGet<HisProcinstOp[]>('/flow/his-procinsts-op/{id}'),
};

export const procinstTaskApi = {
    tasks: Api.newGet<PageResult<ProcinstTask>, PageParam>('/flow/procinsts/tasks'),
    passTask: Api.newPost<void>('/flow/procinsts/tasks/pass'),
    backTask: Api.newPost<void>('/flow/procinsts/tasks/back'),
    rejectTask: Api.newPost<void>('/flow/procinsts/tasks/reject'),
    save: Api.newPost<void>('/flow/procdefs'),
    del: Api.newDelete<void>('/flow/procdefs/{id}'),
};
