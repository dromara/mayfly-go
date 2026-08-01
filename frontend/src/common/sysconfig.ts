import openApi from './openApi';

// 登录是否使用验证码配置key
const AccountLoginSecurityKey = 'AccountLoginSecurity';
const MachineConfigKey = 'MachineConfig';
const SysStyleConfigKey = 'SysStyleConfig';

/**
 * 账号登录安全配置 (对应后端 config.AccountLoginSecurity)
 */
export interface AccountLoginSecurity {
    useCaptcha: boolean;
    useOtp: boolean;
    loginFailCount: number;
    loginFailMin: number;
}

/**
 * 获取账号登录安全配置
 *
 * @returns
 */
export async function getAccountLoginSecurity(): Promise<AccountLoginSecurity | null> {
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
export async function getSysStyleConfig(): Promise<{ title?: string; viceTitle?: string; logoIcon?: string; useWatermark?: boolean; watermarkContent?: string }> {
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
export async function getLdapEnabled(): Promise<boolean> {
    const res = await openApi.getLdapEnabled();
    return res.enabled;
}

/**
 * 机器配置 (对应后端机器系统配置)
 */
export interface MachineConfig {
    uploadMaxFileSize: string;
    [key: string]: unknown;
}

/**
 * 获取机器配置
 *
 * @returns
 */
export async function getMachineConfig(): Promise<MachineConfig> {
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
 * 获取系统服务启动配置
 *
 * @returns 配置信息
 */
export async function getServerConf(): Promise<{ i18n: string; version: string }> {
    return openApi.getServerConf() as Promise<{ i18n: string; version: string }>;
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
