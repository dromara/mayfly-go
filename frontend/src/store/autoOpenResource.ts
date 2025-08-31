import { defineStore } from 'pinia';

/**
 * 自动打开资源
 */
export const useAutoOpenResource = defineStore('autoOpenResource', {
    state: () => ({
        autoOpenResource: {
            codePath: '',
        },
    }),
    actions: {
        setCodePath(codePath: string) {
            this.autoOpenResource.codePath = codePath;
        },
    },
});
