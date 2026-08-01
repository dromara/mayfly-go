import request from './request';
import type { LoginResult, PermissionInfo, OAuth2Config } from '@/views/system/types';
import type { FileRecord } from '@/views/ops/file/types';

interface LoginParam {
    username: string;
    password: string;
    captcha?: string;
    captchaId?: string;
}

interface RefreshTokenParam {
    refresh_token: string;
}

interface RefreshTokenResult {
    token: string;
    refresh_token: string;
}

interface OtpVerifyParam {
    otpToken: string;
    code: string;
}

interface ConfigValueParam {
    key: string;
}

export interface ChangePwdParam {
    oldPassword: string;
    newPassword: string;
}

interface Oauth2CallbackParam {
    code: string;
    state: string;
}

interface LdapLoginParam {
    username: string;
    password: string;
}

interface ServerConf {
    version: string;
    [key: string]: unknown;
}

interface OAuth2LoginConfigRes {
    enable: boolean;
    name: string;
}

interface CaptchaRes {
    captchaId: string;
    captchaImage: string;
}

export default {
    login: (param: LoginParam) => request.post<LoginResult>('/auth/accounts/login', param),
    refreshToken: (param: RefreshTokenParam) => request.get<RefreshTokenResult>('/auth/accounts/refreshToken', param),
    otpVerify: (param: OtpVerifyParam) => request.post<RefreshTokenResult>('/auth/accounts/otp-verify', param),
    getPublicKey: () => request.get<string>('/common/public-key'),
    getConfigValue: (params: ConfigValueParam) => request.get<string>('/sys/configs/value', params),
    getServerConf: () => request.get<ServerConf>('/sys/configs/server'),
    oauth2LoginConfig: () => request.get<OAuth2LoginConfigRes>('/auth/oauth2/config'),
    changePwd: (param: ChangePwdParam) => request.post<void>('/sys/accounts/change-pwd', param),
    captcha: () => request.get<CaptchaRes>('/sys/captcha'),
    logout: () => request.post<void>('/auth/accounts/logout'),
    getPermissions: () => request.get<PermissionInfo>('/sys/accounts/permissions'),
    oauth2Callback: (params: Oauth2CallbackParam) => request.get<LoginResult>('/auth/oauth2/callback', params),
    getLdapEnabled: () => request.get<{ enabled: boolean }>('/auth/ldap/enabled'),
    ldapLogin: (param: LdapLoginParam) => request.post<LoginResult>('/auth/ldap/login', param),
    getFileDetail: (keys: string[]) => request.get<FileRecord[]>(`/sys/files/detail/${keys.join(',')}`),
};
