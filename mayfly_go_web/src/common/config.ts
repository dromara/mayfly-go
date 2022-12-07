const config = {
    baseApiUrl: `${(window as any).globalConfig.BaseApiUrl}/tsu63gHksdMSbsunMxSueSVRwnRqZVsu/api/`,
    baseWsUrl: `${(window as any).globalConfig.BaseWsUrl || `${location.protocol == 'https:' ? 'wss:' : 'ws:'}//${location.host}`}/tsu63gHksdMSbsunMxSueSVRwnRqZVsu/api`,

    // 系统版本
    version: 'v1.3.1'
}

export default config