import openApi from './openApi';

// 登录是否使用验证码配置key
const AccountLoginSecurity = 'AccountLoginSecurity';
const UseLoginCaptchaConfigKey = 'UseLoginCaptcha';
const UseWatermarkConfigKey = 'UseWatermark';

/**
 * 获取系统配置值
 *
 * @param key 配置key
 * @returns 配置值
 */
export async function getConfigValue(key: string): Promise<string> {
    return (await openApi.getConfigValue({ key })) as string;
}

/**
 * 获取bool类型系统配置值
 *
 * @param key 配置key
 * @param defaultValue 默认值
 * @returns 是否为ture，1: true；其他: false
 */
export async function getBoolConfigValue(key: string, defaultValue: boolean): Promise<boolean> {
    const value = await getConfigValue(key);
    return convertBool(value, defaultValue);
}

/**
 * 获取账号登录安全配置
 *
 * @returns
 */
export async function getAccountLoginSecurity(): Promise<any> {
    const value = await getConfigValue(AccountLoginSecurity);
    if (!value) {
        return null;
    }
    const jsonValue = JSON.parse(value);
    jsonValue.useCaptcha = convertBool(jsonValue.useCaptcha, true);
    jsonValue.useOtp = convertBool(jsonValue.useOtp, true);
    return jsonValue;
}

/**
 * 是否使用登录验证码
 *
 * @returns
 */
export async function useLoginCaptcha(): Promise<boolean> {
    return await getBoolConfigValue(UseLoginCaptchaConfigKey, true);
}

/**
 * 是否启用水印信息配置
 *
 * @returns
 */
export async function useWatermark(): Promise<any> {
    const value = await getConfigValue(UseWatermarkConfigKey);
    if (!value) {
        return {
            isUse: true,
        };
    }
    const jsonValue = JSON.parse(value);
    // 将字符串转为bool
    jsonValue.isUse = convertBool(jsonValue.isUse, true);
    return jsonValue;
}

function convertBool(value: string, defaultValue: boolean) {
    if (!value) {
        return defaultValue;
    }
    return value == '1' || value == 'true';
}

/**
 * 获取LDAP登录配置
 *
 * @returns
 */
export async function getLdapEnabled(): Promise<any> {
    const value = await openApi.getLdapEnabled();
    return convertBool(value, false);
}
