import { defineStore } from 'pinia';

/**
 * 自动打开资源
 */
export const useAutoOpenResource = defineStore('autoOpenResource', {
    state: () => ({
        autoOpenResource: {
            machineCodePath: '',
            dbCodePath: '',
            redisCodePath: '',
            mongoCodePath: '',
        },
    }),
    actions: {
        setMachineCodePath(codePath: string) {
            this.autoOpenResource.machineCodePath = codePath;
        },
        setDbCodePath(codePath: string) {
            this.autoOpenResource.dbCodePath = codePath;
        },
        setRedisCodePath(codePath: string) {
            this.autoOpenResource.redisCodePath = codePath;
        },
        setMongoCodePath(codePath: string) {
            this.autoOpenResource.mongoCodePath = codePath;
        },
    },
});
