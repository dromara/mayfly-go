import type { App } from 'vue';
import { store } from '@/store/index.ts';
import { judementSameArr } from '@/common/utils/arrayOperation.ts';

// 用户权限指令
export function authDirective(app: App) {
    // 单个权限验证（v-auth="xxx"）
    app.directive('auth', {
        mounted(el, binding) {
            if (!store.state.userInfos.userInfos.permissions.some((v: any) => v === binding.value)) el.parentNode.removeChild(el);
        },
    });
    // 多个权限验证，满足一个则显示（v-auths="[xxx,xxx]"）
    app.directive('auths', {
        mounted(el, binding) {
            let flag = false;
            store.state.userInfos.userInfos.permissions.map((val: any) => {
                binding.value.map((v: any) => {
                    if (val === v) flag = true;
                });
            });
            if (!flag) el.parentNode.removeChild(el);
        },
    });
    // 多个权限验证，全部满足则显示（v-auth-all="[xxx,xxx]"）
    app.directive('auth-all', {
        mounted(el, binding) {
            const flag = judementSameArr(binding.value, store.state.userInfos.userInfos.permissions);
            if (!flag) el.parentNode.removeChild(el);
        },
    });
}
