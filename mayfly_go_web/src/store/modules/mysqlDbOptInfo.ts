import { Module } from 'vuex';
// 此处加上 `.ts` 后缀报错，具体原因不详
import { DbOptInfoState, RootStateTypes } from '@/store/interface';

const sqlExecInfoModule: Module<DbOptInfoState, RootStateTypes> = {
    namespaced: true,
    state: {
        dbOptInfo: {
            projectId: 0,
            envId: 0,
            dbId: 0,
            db: '0',
        }
    },
    mutations: {
        // 设置用户信息
        getSqlExecInfo(state: any, data: object) {
            state.dbOptInfo = data;
        },
    },
    actions: {
        // 设置用户信息
        async setSqlExecInfo({ commit }, data: object) {
            if (data) {
                commit('getSqlExecInfo', data);
            }
        },
    },
};

export default sqlExecInfoModule;
