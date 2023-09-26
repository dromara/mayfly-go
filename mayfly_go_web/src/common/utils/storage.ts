const TokenKey = 'token';
const UserKey = 'user';
const TagViewsKey = 'tagViews';

// 获取请求token
export function getToken(): string {
    return getLocal(TokenKey);
}

// 保存用户访问token
export function saveToken(token: string) {
    setLocal(TokenKey, token);
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

// 获取是否开启水印
export function getUseWatermark() {
    return getLocal('useWatermark');
}

export function saveUseWatermark(useWatermark: boolean) {
    setLocal('useWatermark', useWatermark);
}

// 清除用户相关的用户信息
export function clearUser() {
    removeLocal(TokenKey);
    removeLocal(UserKey);
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

// 1. localStorage
// 设置永久缓存
export function setLocal(key: string, val: any) {
    window.localStorage.setItem(key, JSON.stringify(val));
}

// 获取永久缓存
export function getLocal(key: string) {
    let json: any = window.localStorage.getItem(key);
    return JSON.parse(json);
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
    window.sessionStorage.setItem(key, JSON.stringify(val));
}

// 获取临时缓存
export function getSession(key: string) {
    let json: any = window.sessionStorage.getItem(key);
    return JSON.parse(json);
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
