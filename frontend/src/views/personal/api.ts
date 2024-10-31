import Api from '@/common/Api';

export const personApi = {
    accountInfo: Api.newGet('/sys/accounts/self'),
    updateAccount: Api.newPut('/sys/accounts/self'),
    authStatus: Api.newGet('/auth/oauth2/status'),
    getMsgs: Api.newGet('/msgs/self'),
    unbindOauth2: Api.newGet('/auth/oauth2/unbind'),
};
