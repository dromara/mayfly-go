import { randomUuid } from './string';

const TokenKey = 'm-token';
const RefreshTokenKey = 'm-refresh-token';
const UserKey = 'm-user';
const TagViewsKey = 'm-tagViews';
const ClientIdKey = 'm-clientId';

// 获取请求token
export function getToken(): string {
    return getLocal(TokenKey);
}

// 保存用户访问token
export function saveToken(token: string) {
    setLocal(TokenKey, token);
}

export function getRefreshToken(): string {
    return getLocal(RefreshTokenKey);
}

export function saveRefreshToken(refreshToken: string) {
    return setLocal(RefreshTokenKey, refreshToken);
}

// 获取登录用户基础信息
export function getUser() {
    return getLocal(UserKey);
}

// 保存用户信息
export function saveUser(userinfo: any) {
    setLocal(UserKey, userinfo);
}

export function saveThemeConfig(themeConfig: any) {
    setLocal('themeConfig', themeConfig);
}

export function getThemeConfig() {
    return getLocal('themeConfig');
}

/**
 * 清除当前登录用户相关信息
 */
export function clearUser() {
    removeLocal(TokenKey);
    removeLocal(UserKey);
    removeLocal(RefreshTokenKey);
}

export function getTagViews() {
    return getSession(TagViewsKey);
}

export function setTagViews(tagViews: Array<object>) {
    setSession(TagViewsKey, tagViews);
}

export function removeTagViews() {
    removeSession(TagViewsKey);
}

// 获取客户端UUID
export function getClientId(): string {
    let uuid = getSession(ClientIdKey);
    if (uuid == null) {
        uuid = randomUuid();
        setSession(ClientIdKey, uuid);
    }
    return uuid;
}

// 1. localStorage
// 设置永久缓存
export function setLocal(key: string, val: any) {
    if (typeof val == 'object') {
        val = JSON.stringify(val);
    }
    window.localStorage.setItem(key, val);
}

// 获取永久缓存
export function getLocal(key: string) {
    let val: any = window.localStorage.getItem(key);
    try {
        return JSON.parse(val);
    } catch (e) {
        return val;
    }
}

// 移除永久缓存
export function removeLocal(key: string) {
    window.localStorage.removeItem(key);
}

// 移除全部永久缓存
export function clearLocal() {
    window.localStorage.clear();
}

// 2. sessionStorage
// 设置临时缓存
export function setSession(key: string, val: any) {
    if (typeof val == 'object') {
        val = JSON.stringify(val);
    }
    window.sessionStorage.setItem(key, val);
}

// 获取临时缓存
export function getSession(key: string) {
    let val: any = window.sessionStorage.getItem(key);
    try {
        return JSON.parse(val);
    } catch (e) {
        return val;
    }
}

// 移除临时缓存
export function removeSession(key: string) {
    window.sessionStorage.removeItem(key);
}

// 移除全部临时缓存
export function clearSession() {
    clearUser();
    window.sessionStorage.clear();
}
