import { defineStore } from 'pinia';
import { getUser } from '@/common/utils/storage';

export interface UserInfo {
    username: string;
    name: string;
    lastLoginIp?: string;
    lastLoginTime?: string;
    photo?: string; // 头像
    permissions?: string[]; // 权限
}

export const useUserInfo = defineStore('userInfo', {
    state: (): UserInfoState => ({
        userInfo: {} as UserInfo,
    }),
    actions: {
        // 设置用户信息
        async setUserInfo(data: UserInfo) {
            const ui = getUser();
            if (ui) {
                this.userInfo = ui;
            } else {
                this.userInfo = data;
            }
        },
    },
});
