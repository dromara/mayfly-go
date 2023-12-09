import request from './request';
import { randomUuid } from './utils/string';

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

    static abortControllers: Map<string, AbortController> = new Map();

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
     * 请求对应的该api
     * @param {Object} param 请求该api的参数
     */
    request(param: any = null, options: any = {}): Promise<any> {
        if (this.beforeHandler) {
            this.beforeHandler(param);
        }
        return request.request(this.method, this.url, param, options);
    }

    /**
     * 允许取消的请求, 使用Api.cancelReq(key) 取消请求
     * @param key 用于取消该key关联的请求
     * @param {Object} param 请求该api的参数
     */
    allowCancelReq(key: string, param: any = null, options: RequestInit = {}): Promise<any> {
        let controller = Api.abortControllers.get(key);
        if (!controller) {
            controller = new AbortController();
            Api.abortControllers.set(key, controller);
        }
        options.signal = controller.signal;

        return this.request(param, options);
    }

    /**    静态方法     **/

    /**
     * 取消请求
     * @param key 请求key
     */
    static cancelReq(key: string) {
        let controller = Api.abortControllers.get(key);
        if (controller) {
            controller.abort();
            Api.removeAbortKey(key);
        }
    }

    static removeAbortKey(key: string) {
        if (key) {
            console.log('remove abort key: ', key);
            Api.abortControllers.delete(key);
        }
    }

    /**
     * 根据旧key生成新的abort key，可能旧key未取消，造成多余无用对象
     * @param oldKey 旧key
     * @returns key
     */
    static genAbortKey(oldKey: string) {
        if (!oldKey) {
            return randomUuid();
        }
        if (Api.abortControllers.get(oldKey)) {
            return oldKey;
        }
        return randomUuid();
    }

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
