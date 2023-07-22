import Api from '@/common/Api';

export const personApi = {
    accountInfo: Api.newGet('/sys/accounts/self'),
    updateAccount: Api.newPut('/sys/accounts/self'),
    authStatus: Api.newGet('/sys/auth/status'),
    getMsgs: Api.newGet('/msgs/self'),
};
