import { store } from '@/store/index.ts';
import { judementSameArr } from '@/common/utils/arrayOperation.ts';

// 单个权限验证
export function auth(value: string) {
    return store.state.userInfos.userInfos.permissions.some((v: any) => v === value);
}

// 多个权限验证，满足一个则为 true
export function auths(value: Array<string>) {
    let flag = false;
    store.state.userInfos.userInfos.permissions.map((val: any) => {
        value.map((v: any) => {
            if (val === v) flag = true;
        });
    });
    return flag;
}

// 多个权限验证，全部满足则为 true
export function authAll(value: Array<string>) {
    return judementSameArr(value, store.state.userInfos.userInfos.permissions);
}
