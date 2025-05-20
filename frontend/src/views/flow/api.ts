import Api from '@/common/Api';

export const procdefApi = {
    list: Api.newGet('/flow/procdefs'),
    detail: Api.newGet('/flow/procdefs/detail/{id}'),
    flowDef: Api.newGet('/flow/procdefs/flowdef/{id}'),
    getByResource: Api.newGet('/flow/procdefs/{resourceType}/{resourceCode}'),
    save: Api.newPost('/flow/procdefs'),
    saveFlowDef: Api.newPost('/flow/procdefs/flowdef'),
    del: Api.newDelete('/flow/procdefs/{id}'),
};

export const procinstApi = {
    list: Api.newGet('/flow/procinsts'),
    start: Api.newPost('/flow/procinsts/start'),
    detail: Api.newGet('/flow/procinsts/{id}'),
    cancel: Api.newPost('/flow/procinsts/{id}/cancel'),
    hisOp: Api.newGet('/flow/his-procinsts-op/{id}'),
};

export const procinstTaskApi = {
    tasks: Api.newGet('/flow/procinsts/tasks'),
    passTask: Api.newPost('/flow/procinsts/tasks/pass'),
    backTask: Api.newPost('/flow/procinsts/tasks/back'),
    rejectTask: Api.newPost('/flow/procinsts/tasks/reject'),
    save: Api.newPost('/flow/procdefs'),
    del: Api.newDelete('/flow/procdefs/{id}'),
};
