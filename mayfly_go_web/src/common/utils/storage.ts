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
    window.sessionStorage.clear();
}
