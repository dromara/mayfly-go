import Api from '@/common/Api';

export const personApi = {
    accountInfo: Api.create("/sys/accounts/self", 'get'),
    updateAccount: Api.create("/sys/accounts/self", 'put'),
    getMsgs: Api.create("/sys/accounts/msgs", 'get'),
}

