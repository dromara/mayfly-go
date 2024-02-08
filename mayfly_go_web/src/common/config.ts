function getBaseApiUrl() {
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
    baseApiUrl: `${(window as any).globalConfig.BaseApiUrl || location.protocol + '//' + getBaseApiUrl()}/api`,
    baseWsUrl: `${(window as any).globalConfig.BaseWsUrl || `${location.protocol == 'https:' ? 'wss:' : 'ws:'}//${getBaseApiUrl()}`}/api`,

    // 系统版本
    version: 'v1.7.3',
};

export default config;
