import Api from '@/common/Api';
import type { Msg, PageResult } from '@/types/common';

/** 账号角色信息 (对应 vo.AccountRoleVO：后端字段为 roleName，不是角色实体里的 name) */
export interface AccountRoleVO {
    roleId: number;
    roleName: string;
    code: string;
    status: number;
    /** 分配时间 */
    createTime?: string;
    creator?: string;
}

/** 个人账号信息 (对应 vo.AccountPersonVO) */
export interface AccountPersonVO {
    roles: AccountRoleVO[];
}

/** OAuth2 绑定状态 (对应 vo.Oauth2Status) */
export interface Oauth2Status {
    enable: boolean;
    bind: boolean;
}

export const personApi = {
    accountInfo: Api.newGet<AccountPersonVO>('/sys/accounts/self'),
    updateAccount: Api.newPut<void>('/sys/accounts/self'),
    authStatus: Api.newGet<Oauth2Status>('/auth/oauth2/status'),
    getMsgs: Api.newGet<PageResult<Msg>>('/msgs/self'),
    getUnreadMsgCount: Api.newGet<number>('/msgs/self/unread/count'),
    readMsg: Api.newGet<void>('/msgs/self/read'),
    unbindOauth2: Api.newGet<void>('/auth/oauth2/unbind'),
};
