import Api from '@/common/Api';
import MonacoEditorBox from '@/components/monaco/MonacoEditorBox';
import { i18n } from '@/i18n';
import type { EsInstance, EsExportProgress, EsProxyRes, EsInstanceListParam } from './types';
import type { PageResult } from '@/types/common';

/** 惰性初始化 API 实例，避免模块加载时的循环依赖问题 */
let _instances: ReturnType<typeof Api.newGet<PageResult<EsInstance>, EsInstanceListParam>> | null = null;
let _deleteInstance: ReturnType<typeof Api.newDelete<void>> | null = null;
let _saveInstance: ReturnType<typeof Api.newPost<number>> | null = null;
let _testConn: ReturnType<typeof Api.newPost<{ version?: { number?: string } }>> | null = null;
let _exportData: ReturnType<typeof Api.newPost<string>> | null = null;
let _exportProgress: ReturnType<typeof Api.newGet<EsExportProgress>> | null = null;

function getInstances() {
    if (!_instances) _instances = Api.newGet<PageResult<EsInstance>, EsInstanceListParam>('/es/instance');
    return _instances;
}
function getDeleteInstance() {
    if (!_deleteInstance) _deleteInstance = Api.newDelete<void>('/es/instance/{id}');
    return _deleteInstance;
}
function getSaveInstance() {
    if (!_saveInstance) _saveInstance = Api.newPost<number>('/es/instance');
    return _saveInstance;
}
function getTestConn() {
    if (!_testConn) _testConn = Api.newPost<{ version?: { number?: string } }>('/es/instance/test-conn');
    return _testConn;
}
function getExportData() {
    if (!_exportData) _exportData = Api.newPost<string>('/es/instance/export/{instanceId}');
    return _exportData;
}
function getExportProgress() {
    if (!_exportProgress) _exportProgress = Api.newGet<EsExportProgress>('/es/instance/export/progress/{exportId}');
    return _exportProgress;
}

export const esApi = {
    get instances() { return getInstances(); },
    get deleteInstance() { return getDeleteInstance(); },
    get saveInstance() { return getSaveInstance(); },
    get testConn() { return getTestConn(); },
    get exportData() { return getExportData(); },
    get exportProgress() { return getExportProgress(); },

    // proxyGet: Api.newGet('/es/instance/proxy/{id}/{path}'),
    // proxyPost: Api.newPost('/es/instance/proxy/{id}/{path}'),
    // proxyPut: Api.newPut('/es/instance/proxy/{id}/{path}'),
    // proxyDelete: Api.newDelete('/es/instance/proxy/{id}/{path}'),

    proxyReq: async function <T = EsProxyRes>(method: string, id: number, path: string, param?: Record<string, unknown>): Promise<T> {
        if (path.startsWith('/')) {
            path = path.substring(1);
        }
        let res: EsProxyRes = {};
        const t = i18n.global.t;
        switch (method) {
            case 'get':
                res = await Api.newGet<EsProxyRes>(`/es/instance/proxy/${id}/${path}`).request(param, { esProxyReq: true }) as EsProxyRes;
                break;
            case 'post':
                res = await Api.newPost<EsProxyRes>(`/es/instance/proxy/${id}/${path}`).request(param, { esProxyReq: true }) as EsProxyRes;
                break;
            case 'put':
                res = await Api.newPut<EsProxyRes>(`/es/instance/proxy/${id}/${path}`).request(param, { esProxyReq: true }) as EsProxyRes;
                break;
            case 'delete':
                res = await Api.newDelete<EsProxyRes>(`/es/instance/proxy/${id}/${path}`).request(param, { esProxyReq: true }) as EsProxyRes;
                break;
        }
        let error = res.error || (Array.isArray(res.failures) && res.failures.length > 0 && res.failures[0]) || res.msg;
        if (error) {
            return await esApi.alertError(error, t('es.execError'));
        }
        return res as T;
    },

    alertError: async (errData: unknown, title: string) => {
        MonacoEditorBox({
            content: JSON.stringify(errData, null, 2),
            title,
            language: 'json',
            width: '600px',
            canChangeLang: false,
            options: { wordWrap: 'on', tabSize: 2, readOnly: true }, // 自动换行
        });

        return await Promise.reject(errData);
    },
};
