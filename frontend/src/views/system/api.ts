import Api from '@/common/Api';
import type { PageParam, PageResult } from '@/types/common';
import type { Account, AccountListParam, SysConfig, SysLog, SysResource, SysRole } from './types';

export const resourceApi = {
    list: Api.newGet<SysResource[]>('/sys/resources'),
    detail: Api.newGet<SysResource>('/sys/resources/{id}'),
    save: Api.newPost<void>('/sys/resources'),
    update: Api.newPut<void>('/sys/resources/{id}'),
    del: Api.newDelete<void>('/sys/resources/{id}'),
    changeStatus: Api.newPut<void>('/sys/resources/{id}/{status}'),
    sort: Api.newPost<void>('/sys/resources/sort'),
    roles: Api.newGet<SysRole[]>('/sys/resources/{id}/roles'),
};

export const roleApi = {
    list: Api.newGet<PageResult<SysRole>, PageParam>('/sys/roles'),
    save: Api.newPost<void>('/sys/roles'),
    update: Api.newPut<void>('/sys/roles/{id}'),
    del: Api.newDelete<void>('/sys/roles/{id}'),
    // 获取指定角色拥有的资源id
    roleResourceIds: Api.newGet<number[]>('/sys/roles/{id}/resourceIds'),
    roleResources: Api.newGet<SysResource[]>('/sys/roles/{id}/resources'),
    saveResources: Api.newPost<void>('/sys/roles/{id}/resources'),
    roleAccounts: Api.newGet<Account[]>('/sys/roles/{id}/accounts'),
};

export const accountApi = {
    list: Api.newGet<PageResult<Account>, AccountListParam>('/sys/accounts'),
    querySimple: Api.newGet<PageResult<Account>, AccountListParam>('/sys/accounts/simple'),
    getAccountDetail: Api.newGet<Account>('/sys/accounts/detail'),
    save: Api.newPost<void>('/sys/accounts'),
    update: Api.newPut<void>('/sys/accounts/{id}'),
    del: Api.newDelete<void>('/sys/accounts/{id}'),
    changeStatus: Api.newPut<void>('/sys/accounts/change-status/{id}/{status}'),
    resetOtpSecret: Api.newPut<void>('/sys/accounts/{id}/reset-otp'),
    roles: Api.newGet<SysRole[]>('/sys/accounts/{id}/roles'),
    resources: Api.newGet<SysResource[]>('/sys/accounts/{id}/resources'),
    saveRole: Api.newPost<void>('/sys/accounts/roles'),
};

export const configApi = {
    list: Api.newGet<PageResult<SysConfig>, PageParam>('/sys/configs'),
    save: Api.newPost<void>('/sys/configs'),
    getValue: Api.newGet<string>('/sys/configs/value'),
};

export const logApi = {
    list: Api.newGet<PageResult<SysLog>, PageParam>('/syslogs'),
    detail: Api.newGet<SysLog>('/syslogs/{id}'),
};

export const authApi = {
    info: Api.newGet<Record<string, unknown>>('/sys/auth'),
    saveOAuth2: Api.newPut<void>('/sys/auth/oauth2'),
};
