import Api from '@/common/Api';

export const resourceApi = {
    list: Api.create("/sys/resources", 'get'),
    detail: Api.create("/sys/resources/{id}", 'get'),
    save: Api.create("/sys/resources", 'post'),
    update: Api.create("/sys/resources/{id}", 'put'),
    del: Api.create("/sys/resources/{id}", 'delete'),
    changeStatus: Api.create("/sys/resources/{id}/{status}", 'put')
}

export const roleApi = {
    list: Api.create("/sys/roles", 'get'),
    save: Api.create("/sys/roles", 'post'),
    update: Api.create("/sys/roles/{id}", 'put'),
    del: Api.create("/sys/roles/{id}", 'delete'),
    // 获取指定角色拥有的资源id
    roleResourceIds: Api.create("/sys/roles/{id}/resourceIds", 'get'),
    roleResources: Api.create("/sys/roles/{id}/resources", 'get'),
    saveResources: Api.create("/sys/roles/{id}/resources", 'post')
}

export const accountApi = {
    list: Api.create("/sys/accounts", 'get'),
    save: Api.create("/sys/accounts", 'post'),
    update: Api.create("/sys/accounts/{id}", 'put'),
    del: Api.create("/sys/accounts/{id}", 'delete'),
    changeStatus: Api.create("/sys/accounts/change-status/{id}/{status}", 'put'),
    roleIds: Api.create("/sys/accounts/{id}/roleIds", 'get'),
    roles: Api.create("/sys/accounts/{id}/roles", 'get'),
    resources: Api.create("/sys/accounts/{id}/resources", 'get'),
    saveRoles: Api.create("/sys/accounts/roles", 'post')
}

export const configApi = {
    list: Api.create("/sys/configs", 'get'),
    save: Api.create("/sys/configs", 'post'),
    getValue: Api.create("/sys/configs/value", 'get'),
}

export const logApi = {
    list: Api.create("/syslogs", "get")
}
