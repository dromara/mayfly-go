const config = {
    baseApiUrl: `${(window as any).globalConfig.BaseApiUrl}/api`,
    baseWsUrl: `${(window as any).globalConfig.BaseWsUrl || `${location.protocol == 'https:' ? 'wss:' : 'ws:'}//${location.host}`}/api`
}

export default config