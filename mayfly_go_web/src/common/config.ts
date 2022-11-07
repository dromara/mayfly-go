const config = {
    baseApiUrl: `${(window as any).globalConfig.BaseApiUrl}/api`,
    baseWsUrl: `${(window as any).globalConfig.BaseWsUrl || `${location.protocol == 'https:' ? 'wss:' : 'ws:'}//${location.host}`}/api`,

    // 系统版本
    version: 'v1.3.0'
}

export default config