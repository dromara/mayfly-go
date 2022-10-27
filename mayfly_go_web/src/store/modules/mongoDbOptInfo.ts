import { Module } from 'vuex';
// 此处加上 `.ts` 后缀报错，具体原因不详
import {DbOptInfoState, RootStateTypes} from '@/store/interface';

const mongoDbOptInfoModule: Module<DbOptInfoState, RootStateTypes> = {
    namespaced: true,
    state: {
        dbOptInfo: {
            tagPath: '',
            dbId: 0,
            db: '0',
        },
    },
    mutations: {
        // 设置用户信息
        getMongoDbOptInfo(state: any, data: object) {
            state.dbOptInfo = data;
        },
    },
    actions: {
        // 设置用户信息
        async setMongoDbOptInfo({ commit }, data: object) {
            if (data) {
                commit('getMongoDbOptInfo', data);
            }
        },
    },
};

export default mongoDbOptInfoModule;
