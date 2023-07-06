import { defineStore } from 'pinia';
import { getSession } from '@/common/utils/storage';

export const useUserInfo = defineStore('userInfo', {
    state: (): UserInfoState => ({
        userInfo: {},
    }),
    actions: {
        // 设置用户信息
        async setUserInfo(data: object) {
            const ui = getSession('userInfo');
            if (ui) {
                this.userInfo = ui;
            } else {
                this.userInfo = data;
            }
        },
    },
});
