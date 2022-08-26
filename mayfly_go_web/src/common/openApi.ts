import request from './request'

export default {
    login: (param: any) => request.request('POST', '/sys/accounts/login', param),
    changePwd: (param: any) => request.request('POST', '/sys/accounts/change-pwd', param),
    getPublicKey: () => request.request('GET', '/common/public-key'),
    getConfigValue: (param: any) => request.request('GET', '/sys/configs/value', param),
    captcha: () => request.request('GET', '/sys/captcha'),
    logout: (param: any) => request.request('POST', '/sys/accounts/logout/{token}', param),
    getMenuRoute: (param: any) => request.request('Get', '/sys/resources/account', param)
}