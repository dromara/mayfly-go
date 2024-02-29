import Api from '@/common/Api';

export const procdefApi = {
    list: Api.newGet('/flow/procdefs'),
    getByKey: Api.newGet('/flow/procdefs/{key}'),
    save: Api.newPost('/flow/procdefs'),
    del: Api.newDelete('/flow/procdefs/{id}'),
};

export const procinstApi = {
    list: Api.newGet('/flow/procinsts'),
    detail: Api.newGet('/flow/procinsts/{id}'),
    cancel: Api.newPost('/flow/procinsts/{id}/cancel'),
    tasks: Api.newGet('/flow/procinsts/tasks'),
    completeTask: Api.newPost('/flow/procinsts/tasks/complete'),
    backTask: Api.newPost('/flow/procinsts/tasks/back'),
    rejectTask: Api.newPost('/flow/procinsts/tasks/reject'),
    save: Api.newPost('/flow/procdefs'),
    del: Api.newDelete('/flow/procdefs/{id}'),
};
