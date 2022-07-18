import request from './request'

export default {
    login: (param: any) => request.request('POST', '/sys/accounts/login', param, null),
    changePwd: (param: any) => request.request('POST', '/sys/accounts/change-pwd', param, null),
    getPublicKey: () => request.request('GET', '/common/public-key', null, null),
    captcha: () => request.request('GET', '/sys/captcha', null, null),
    logout: (param: any) => request.request('POST', '/sys/accounts/logout/{token}', param, null),
    getMenuRoute: (param: any) => request.request('Get', '/sys/resources/account', param, null)
}