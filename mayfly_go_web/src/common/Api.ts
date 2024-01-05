import request from './request';
import { useApiFetch } from '@/hooks/useRequest';

/**
 * 可用于各模块定义各自api请求
 */
class Api {
    /**
     * 请求url
     */
    url: string;

    /**
     * 请求方法
     */
    method: string;

    /**
     * 请求前处理函数
     * param1: param请求参数
     */
    beforeHandler: Function;

    constructor(url: string, method: string) {
        this.url = url;
        this.method = method;
    }

    /**
     * 设置请求前处理回调函数
     * @param func 请求前处理器
     * @returns this
     */
    withBeforeHandler(func: Function) {
        this.beforeHandler = func;
        return this;
    }

    /**
     * 获取权限的完整url
     */
    getUrl() {
        return request.getApiUrl(this.url);
    }

    /**
     * 响应式使用该api
     * @param params 响应式params
     * @param reqOptions 其他可选值
     * @returns
     */
    useApi<T>(params: any = null, reqOptions: RequestInit = {}) {
        return useApiFetch<T>(this, params, reqOptions);
    }

    /**
     * fetch 请求对应的该api
     * @param {Object} param 请求该api的参数
     */
    async request(param: any = null, options: any = {}): Promise<any> {
        const { execute, data } = this.useApi(param, options);
        await execute();
        return data.value;
    }

    /**
     * xhr 请求对应的该api
     * @param {Object} param 请求该api的参数
     */
    async xhrReq(param: any = null, options: any = {}): Promise<any> {
        if (this.beforeHandler) {
            this.beforeHandler(param);
        }
        return request.xhrReq(this.method, this.url, param, options);
    }

    /**    静态方法     **/

    /**
     * 静态工厂，返回Api对象，并设置url与method属性
     * @param url url
     * @param method 请求方法(get,post,put,delete...)
     */
    static create(url: string, method: string): Api {
        return new Api(url, method);
    }

    /**
     * 创建get api
     * @param url url
     */
    static newGet(url: string): Api {
        return Api.create(url, 'get');
    }

    /**
     * new post api
     * @param url url
     */
    static newPost(url: string): Api {
        return Api.create(url, 'post');
    }

    /**
     * new put api
     * @param url url
     */
    static newPut(url: string): Api {
        return Api.create(url, 'put');
    }

    /**
     * new delete api
     * @param url url
     */
    static newDelete(url: string): Api {
        return Api.create(url, 'delete');
    }
}

export default Api;

export class PageRes {
    list: any[] = [];
    total: number = 0;
}
