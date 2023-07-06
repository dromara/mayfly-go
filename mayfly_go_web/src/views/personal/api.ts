import Api from '@/common/Api';

export const personApi = {
    accountInfo: Api.newGet('/sys/accounts/self'),
    updateAccount: Api.newPut('/sys/accounts/self'),
    getMsgs: Api.newGet('/msgs/self'),
};
