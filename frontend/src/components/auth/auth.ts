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
    // permissions 可能尚未写入（如本地缓存的用户信息缺该字段），按「无权限」判定而非抛错：
    // 懒加载弹层在 setup 里调用本函数时抛错，会让整块内容直接空白
    return useUserInfo().userInfo.permissions?.some((v: string) => v === code) ?? false;
}

/**
 * 判断用户是否拥有权限对象里对应的code
 * @returns {"xxx:save": true}  key->permission code
 * @param permCodes
 */
export function hasPerms(permCodes: string[]): Record<string, boolean> {
    const res: Record<string, boolean> = {};
    for (const permCode of permCodes) {
        if (hasPerm(permCode)) {
            res[permCode] = true;
        }
    }
    return res;
}
