/**
 * 获取应用配置。
 * 需要后端将index.html文件中的<app-config />标签替换为script标签，并将配置项挂载到全局变量window.__APP_CONFIG__ 上
 * @returns 应用配置
 */
declare global {
    interface Window {
        __APP_CONFIG__?: Record<string, string>;
        globalConfig?: Record<string, string>;
    }
}

export function getAppConfig() {
    return window.__APP_CONFIG__;
}

export function getBaseApiUrl() {
    const config = getAppConfig();

    if (config) {
        if (!config.CTX_PATH) {
            return window.location.host;
        }
        return window.location.host + config.CTX_PATH;
    }

    let path = window.location.pathname;
    if (path == '/') {
        return window.location.host;
    }

    if (path.endsWith('/')) {
        // 去除最后一个/
        return window.location.host + path.replace(/\/$/, '');
    }
    return window.location.host + path;
}

const config = {
    baseApiUrl: `${window.globalConfig?.BaseApiUrl || location.protocol + '//' + getBaseApiUrl()}/api`,
    baseWsUrl: `${window.globalConfig?.BaseWsUrl || `${location.protocol == 'https:' ? 'wss:' : 'ws:'}//${getBaseApiUrl()}`}/api`,
};

export default config;
