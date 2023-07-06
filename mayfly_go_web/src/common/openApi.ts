import request from './request';

export default {
    login: (param: any) => request.post('/sys/accounts/login', param),
    otpVerify: (param: any) => request.post('/sys/accounts/otp-verify', param),
    changePwd: (param: any) => request.post('/sys/accounts/change-pwd', param),
    getPublicKey: () => request.get('/common/public-key'),
    getConfigValue: (params: any) => request.get('/sys/configs/value', params),
    captcha: () => request.get('/sys/captcha'),
    logout: () => request.post('/sys/accounts/logout/{token}'),
    getPermissions: () => request.get('/sys/accounts/permissions'),
};
