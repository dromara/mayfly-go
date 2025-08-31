import { useUserInfo } from '@/store/userInfo';

/**
 * 判断当前用户是否拥有指定权限
 * @param code 权限code
 * @returns
 */
export function hasPerm(code: string): boolean {
    if (!code) {
        return true;
    }
    return useUserInfo().userInfo.permissions.some((v: any) => v === code);
}

/**
 * 判断用户是否拥有权限对象里对应的code
 * @returns {"xxx:save": true}  key->permission code
 * @param permCodes
 */
export function hasPerms(permCodes: any[]): Record<string, boolean> {
    const res = {} as { [key: string]: boolean };
    for (let permCode of permCodes) {
        if (hasPerm(permCode)) {
            res[permCode] = true;
        }
    }
    return res;
}
