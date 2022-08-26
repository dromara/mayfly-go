import openApi from './openApi';

// 登录是否使用验证码配置key
const UseLoginCaptchaConfigKey = "UseLoginCaptcha"
const UseWartermarkConfigKey = "UseWartermark"

/**
 * 获取系统配置值
 * 
 * @param key 配置key
 * @returns 配置值
 */
export async function getConfigValue(key: string) : Promise<string> {
    return await openApi.getConfigValue({key}) as string
}

/**
 * 获取bool类型系统配置值
 * 
 * @param key 配置key
 * @param defaultValue 默认值
 * @returns 是否为ture，1: true；其他: false
 */
export async function getBoolConfigValue(key :string, defaultValue :boolean) : Promise<boolean> {
    const value = await getConfigValue(key)
    if (!value) {
        return defaultValue;
    }
    return value == "1";
}

/**
 * 是否使用登录验证码
 * 
 * @returns 
 */
export async function useLoginCaptcha() : Promise<boolean> {
    return await getBoolConfigValue(UseLoginCaptchaConfigKey, true)
}

/**
 * 是否启用水印
 * 
 * @returns 
 */
 export async function useWartermark() : Promise<boolean> {
    return await getBoolConfigValue(UseWartermarkConfigKey, true)
}