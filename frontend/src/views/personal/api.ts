import Api from '@/common/Api';
import type { Msg, PageResult } from '@/types/common';
import type { SysRole } from '../system/types';

/** 个人账号信息 (对应 vo.AccountPersonVO) */
export interface AccountPersonVO {
    roles: SysRole[];
}

/** OAuth2 绑定状态 (对应 vo.Oauth2Status) */
export interface Oauth2Status {
    enable: boolean;
    bind: boolean;
}

export const personApi = {
    accountInfo: Api.newGet<AccountPersonVO>('/sys/accounts/self'),
    updateAccount: Api.newPut('/sys/accounts/self'),
    authStatus: Api.newGet<Oauth2Status>('/auth/oauth2/status'),
    getMsgs: Api.newGet<PageResult<Msg>>('/msgs/self'),
    getUnreadMsgCount: Api.newGet<number>('/msgs/self/unread/count'),
    readMsg: Api.newGet('/msgs/self/read'),
    unbindOauth2: Api.newGet('/auth/oauth2/unbind'),
};
