import request from './request';

export default {
    login: (param: any) => request.post('/auth/accounts/login', param),
    refreshToken: (param: any) => request.get('/auth/accounts/refreshToken', param),
    otpVerify: (param: any) => request.post('/auth/accounts/otp-verify', param),
    getPublicKey: () => request.get('/common/public-key'),
    getConfigValue: (params: any) => request.get('/sys/configs/value', params),
    oauth2LoginConfig: () => request.get('/auth/oauth2-config'),
    changePwd: (param: any) => request.post('/sys/accounts/change-pwd', param),
    captcha: () => request.get('/sys/captcha'),
    logout: () => request.post('/auth/accounts/logout'),
    getPermissions: () => request.get('/sys/accounts/permissions'),
    oauth2Callback: (params: any) => request.get('/auth/oauth2/callback', params),
    getLdapEnabled: () => request.get('/auth/ldap/enabled'),
    ldapLogin: (param: any) => request.post('/auth/ldap/login', param),
};
