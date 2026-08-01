import Api from '@/common/Api';
import MonacoEditorBox from '@/components/monaco/MonacoEditorBox';
import { i18n } from '@/i18n';
import type { EsInstance, EsExportProgress, EsProxyRes, EsInstanceListParam } from './types';
import type { PageResult } from '@/types/common';

export const esApi = {
    instances: Api.newGet<PageResult<EsInstance>, EsInstanceListParam>('/es/instance'),
    deleteInstance: Api.newDelete<void>('/es/instance/{id}'),
    saveInstance: Api.newPost<number>('/es/instance'),
    testConn: Api.newPost<{ version?: { number?: string } }>('/es/instance/test-conn'),
    exportData: Api.newPost<string>('/es/instance/export/{instanceId}'),
    exportProgress: Api.newGet<EsExportProgress>('/es/instance/export/progress/{exportId}'),

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
