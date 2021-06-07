import { Module } from 'vuex';
import { getSession } from '@/common/utils/storage.ts';
// 此处加上 `.ts` 后缀报错，具体原因不详
import { UserInfosState, RootStateTypes } from '@/store/interface/index';

const userInfosModule: Module<UserInfosState, RootStateTypes> = {
    namespaced: true,
    state: {
        userInfos: {},
    },
    mutations: {
        // 设置用户信息
        getUserInfos(state: any, data: object) {
            state.userInfos = data;
        },
    },
    actions: {
        // 设置用户信息
        async setUserInfos({ commit }, data: object) {
            if (data) {
                commit('getUserInfos', data);
            } else {
                if (getSession('userInfo')) commit('getUserInfos', getSession('userInfo'));
            }
        },
    },
};

export default userInfosModule;
