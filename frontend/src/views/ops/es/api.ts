import Api from '@/common/Api';
import MonacoEditorBox from '@/components/monaco/MonacoEditorBox';
import { i18n } from '@/i18n';

export const esApi = {
    instances: Api.newGet('/es/instance'),
    deleteInstance: Api.newDelete('/es/instance/{id}'),
    saveInstance: Api.newPost('/es/instance'),
    testConn: Api.newPost('/es/instance/test-conn'),

    // proxyGet: Api.newGet('/es/instance/proxy/{id}/{path}'),
    // proxyPost: Api.newPost('/es/instance/proxy/{id}/{path}'),
    // proxyPut: Api.newPut('/es/instance/proxy/{id}/{path}'),
    // proxyDelete: Api.newDelete('/es/instance/proxy/{id}/{path}'),

    proxyReq: async function (method: string, id: any, path: string, param?: any) {
        if (path.startsWith('/')) {
            path = path.substring(1);
        }
        let res = {} as any;
        const t = i18n.global.t;
        switch (method) {
            case 'get':
                res = await Api.newGet(`/es/instance/proxy/${id}/${path}`).request(param, { esProxyReq: true });
                break;
            case 'post':
                res = await Api.newPost(`/es/instance/proxy/${id}/${path}`).request(param, { esProxyReq: true });
                break;
            case 'put':
                res = await Api.newPut(`/es/instance/proxy/${id}/${path}`).request(param, { esProxyReq: true });
                break;
            case 'delete':
                res = await Api.newDelete(`/es/instance/proxy/${id}/${path}`).request(param, { esProxyReq: true });
                break;
        }
        let error = res.error || (res.failures && res.failures.length > 0 && res.failures[0]) || res.msg;
        if (error) {
            return await esApi.alertError(error, t('es.execError'));
        }
        return res;
    },

    alertError: async (errData: any, title: string) => {
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
