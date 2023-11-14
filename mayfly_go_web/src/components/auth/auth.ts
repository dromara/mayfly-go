import { useUserInfo } from '@/store/userInfo';

/**
 * 判断当前用户是否拥有指定权限
 * @param code 权限code
 * @returns
 */
export function hasPerm(code: string) {
    if (!code) {
        return true;
    }
    return useUserInfo().userInfo.permissions.some((v: any) => v === code);
}

/**
 * 判断用户是否拥有权限对象里对应的code
 * @param perms { save: "xxx:save"}
 * @returns {"xxx:save": true}  key->permission code
 */
export function hasPerms(permCodes: any[]) {
    const res = {};
    for (let permCode of permCodes) {
        if (hasPerm(permCode)) {
            res[permCode] = true;
        }
    }
    return res;
}
