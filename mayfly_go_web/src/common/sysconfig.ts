import openApi from './openApi';

// 登录是否使用验证码配置key
const AccountLoginSecurityKey = 'AccountLoginSecurity';
const MachineConfigKey = 'MachineConfig';
const SysStyleConfigKey = 'SysStyleConfig';

/**
 * 获取账号登录安全配置
 *
 * @returns
 */
export async function getAccountLoginSecurity(): Promise<any> {
    const value = await getConfigValue(AccountLoginSecurityKey);
    if (!value) {
        return null;
    }
    const jsonValue = JSON.parse(value);
    jsonValue.useCaptcha = convertBool(jsonValue.useCaptcha, true);
    jsonValue.useOtp = convertBool(jsonValue.useOtp, true);
    return jsonValue;
}

/**
 * 获取全局系统样式配置（logo、title等）
 *
 * @returns
 */
export async function getSysStyleConfig(): Promise<any> {
    const value = await getConfigValue(SysStyleConfigKey);
    const defaultValue = {
        useWatermark: true,
    };
    if (!value) {
        return defaultValue;
    }

    const jsonValue = JSON.parse(value);
    // 将字符串转为bool
    jsonValue.useWatermark = convertBool(jsonValue.useWatermark, true);
    return jsonValue;
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

/**
 * 获取机器配置
 *
 * @returns
 */
export async function getMachineConfig(): Promise<any> {
    const value = await getConfigValue(MachineConfigKey);
    const defaultValue = {
        // 默认1gb
        uploadMaxFileSize: '1GB',
    };
    if (!value) {
        return defaultValue;
    }
    try {
        const jsonValue = JSON.parse(value);
        return jsonValue;
    } catch (e) {
        return defaultValue;
    }
}

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

function convertBool(value: string, defaultValue: boolean) {
    if (!value) {
        return defaultValue;
    }
    return value == '1' || value == 'true';
}
