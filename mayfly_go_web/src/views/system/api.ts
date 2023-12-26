import Api from '@/common/Api';

export const resourceApi = {
    list: Api.newGet('/sys/resources'),
    detail: Api.newGet('/sys/resources/{id}'),
    save: Api.newPost('/sys/resources'),
    update: Api.newPut('/sys/resources/{id}'),
    del: Api.newDelete('/sys/resources/{id}'),
    changeStatus: Api.newPut('/sys/resources/{id}/{status}'),
    sort: Api.newPost('/sys/resources/sort'),
};

export const roleApi = {
    list: Api.newGet('/sys/roles'),
    save: Api.newPost('/sys/roles'),
    update: Api.newPut('/sys/roles/{id}'),
    del: Api.newDelete('/sys/roles/{id}'),
    // 获取指定角色拥有的资源id
    roleResourceIds: Api.newGet('/sys/roles/{id}/resourceIds'),
    roleResources: Api.newGet('/sys/roles/{id}/resources'),
    saveResources: Api.newPost('/sys/roles/{id}/resources'),
    roleAccounts: Api.newGet('/sys/roles/{id}/accounts'),
};

export const accountApi = {
    list: Api.newGet('/sys/accounts'),
    save: Api.newPost('/sys/accounts'),
    update: Api.newPut('/sys/accounts/{id}'),
    del: Api.newDelete('/sys/accounts/{id}'),
    changeStatus: Api.newPut('/sys/accounts/change-status/{id}/{status}'),
    resetOtpSecret: Api.newPut('/sys/accounts/{id}/reset-otp'),
    roles: Api.newGet('/sys/accounts/{id}/roles'),
    resources: Api.newGet('/sys/accounts/{id}/resources'),
    saveRole: Api.newPost('/sys/accounts/roles'),
};

export const configApi = {
    list: Api.newGet('/sys/configs'),
    save: Api.newPost('/sys/configs'),
    getValue: Api.newGet('/sys/configs/value'),
};

export const logApi = {
    list: Api.newGet('/syslogs'),
};

export const authApi = {
    info: Api.newGet('/sys/auth'),
    saveOAuth2: Api.newPut('/sys/auth/oauth2'),
};
