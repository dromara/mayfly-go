import { templateResolve } from '@/common/utils/string';
import { RequestOptions, useApiFetch } from '@/hooks/useRequest';
import config from './config';
import request, { joinClientParams } from './request';

/**
 * 文件上传选项
 */
export interface UploadOptions {
    /** 成功回调 */
    onSuccess?: () => void;
    /** 错误回调 */
    onError?: (error: Error) => void;
}

/**
 * 可用于各模块定义各自api请求
 * T 请求返回的数据类型
 * P 请求参数类型
 */
class Api<T = unknown, P = unknown> {
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
    beforeHandler?: (param: P) => Promise<P> | P;

    constructor(url: string, method: string) {
        this.url = url;
        this.method = method;
    }

    /**
     * 设置请求前处理回调函数
     * @param func 请求前处理器
     * @returns this
     */
    withBeforeHandler(func: (param: P) => Promise<P> | P) {
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
     * @param param 请求参数
     * @param reqOptions 其他可选值
     * @returns
     */
    useApi(param?: P, reqOptions?: RequestOptions) {
        return useApiFetch<T, P>(this, param, reqOptions);
    }

    /**
     * fetch 请求对应的该api
     * @param {Object} param 请求该api的参数
     * @param options options
     */
    async request(param?: P, options: RequestOptions = {}): Promise<T> {
        const { execute, data } = this.useApi(param, options);
        const res = await execute();
        return (data.value as T) || (res as T);
    }

    /**
     * 文件上传请求
     * @param formData FormData 对象（调用方自行构建，包含文件和其他参数）
     * @param options 上传选项
     * @returns { abort: () => void } 返回中止方法
     */
    upload(formData: FormData, options: UploadOptions = {}): { abort: () => void } {
        const { onSuccess, onError } = options;

        const url = `${config.baseApiUrl}${this.url}?${joinClientParams()}`;

        // 创建 AbortController 用于取消请求
        const abortController = new AbortController();

        // 发起 fetch 请求
        fetch(url, {
            method: 'POST',
            body: formData,
            signal: abortController.signal,
        })
            .then((response) => {
                if (!response.ok) {
                    throw new Error(`HTTP ${response.status}`);
                }
                return response;
            })
            .then(() => {
                onSuccess?.();
            })
            .catch((error) => {
                // 如果是主动取消，不触发错误回调
                if (error.name === 'AbortError') {
                    return;
                }
                onError?.(new Error(`upload failed: ${error.message}`));
            });

        // 返回中止方法
        return {
            abort: () => {
                abortController.abort();
            },
        };
    }

    /**
     * 原始文件流上传请求（直接使用文件流作为 body，参数通过 URL query 传递）
     * @param file 文件对象
     * @param queryParams URL 查询参数对象（可选）
     * @param options 上传选项（可包含自定义 headers）
     * @returns { abort: () => void } 返回中止方法
     */
    uploadRaw(file: File, queryParams?: Record<string, string>, options: UploadOptions & { headers?: Record<string, string> } = {}): { abort: () => void } {
        const { onSuccess, onError, headers = {} } = options;

        // 构建 URL，兼容没有 queryParams 的情况
        let url = `${config.baseApiUrl}${this.url}`;
        // 简单判断该url是否是restful风格
        if (url.indexOf('{') != -1 && queryParams) {
            url = templateResolve(url, queryParams);
        }

        const searchParams = new URLSearchParams();

        // 添加业务参数
        if (queryParams) {
            Object.entries(queryParams).forEach(([key, value]) => {
                searchParams.append(key, value);
            });
        }

        // 添加客户端参数
        const clientParams = joinClientParams();
        if (clientParams) {
            // 将 joinClientParams 返回的字符串追加到 searchParams
            const clientParamsObj = new URLSearchParams(clientParams);
            clientParamsObj.forEach((value, key) => {
                searchParams.append(key, value);
            });
        }

        // 拼接完整的 query string
        const queryString = searchParams.toString();
        if (queryString) {
            url += `?${queryString}`;
        }

        // 创建 AbortController 用于取消请求
        const abortController = new AbortController();

        // 构建请求头
        const requestHeaders: Record<string, string> = {
            ...headers,
        };

        // 发起 fetch 请求，直接使用文件流作为 body
        fetch(url, {
            method: 'POST',
            body: file,
            signal: abortController.signal,
            headers: requestHeaders,
        })
            .then((response) => {
                if (!response.ok) {
                    throw new Error(`HTTP ${response.status}`);
                }
                return response;
            })
            .then(() => {
                onSuccess?.();
            })
            .catch((error) => {
                if (error.name === 'AbortError') {
                    return;
                }
                onError?.(new Error(`upload failed: ${error.message}`));
            });

        // 返回中止方法
        return {
            abort: () => {
                abortController.abort();
            },
        };
    }

    /**    静态方法     **/

    /**
     * 静态工厂，返回Api对象，并设置url与method属性
     * @param url url
     * @param method 请求方法(get,post,put,delete...)
     */
    static create<T = unknown, P = unknown>(url: string, method: string): Api<T, P> {
        return new Api<T, P>(url, method);
    }

    /**
     * 创建get api
     * @param url url
     */
    static newGet<T = unknown, P = unknown>(url: string): Api<T, P> {
        return Api.create<T, P>(url, 'get');
    }

    /**
     * new post api
     * @param url url
     */
    static newPost<T = unknown, P = unknown>(url: string): Api<T, P> {
        return Api.create<T, P>(url, 'post');
    }

    /**
     * new put api
     * @param url url
     */
    static newPut<T = unknown, P = unknown>(url: string): Api<T, P> {
        return Api.create<T, P>(url, 'put');
    }

    /**
     * new delete api
     * @param url url
     */
    static newDelete<T = unknown, P = unknown>(url: string): Api<T, P> {
        return Api.create<T, P>(url, 'delete');
    }

    /**
     * 创建文件上传 api
     * @param url url
     */
    static newUpload<T = unknown, P = unknown>(url: string): Api<T, P> {
        return Api.create<T, P>(url, 'upload');
    }
}

export default Api;

export class PageRes<T = unknown> {
    list: T[] = [];
    total: number = 0;
}
