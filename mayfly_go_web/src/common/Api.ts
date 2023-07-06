import request from './request';

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

    constructor(url: string, method: string) {
        this.url = url;
        this.method = method;
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
    request(param: any = null, options: any = null, headers: any = null): Promise<any> {
        return request.request(this.method, this.url, param, headers, options);
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
