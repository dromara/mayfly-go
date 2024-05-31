import router from '../router';
import config from './config';
import { getClientId, getToken } from './utils/storage';
import { templateResolve } from './utils/string';
import { ElMessage } from 'element-plus';
import axios from 'axios';
import { useApiFetch } from '../hooks/useRequest';
import Api from './Api';

export default {
    request,
    xhrReq,
    get,
    post,
    put,
    del,
    getApiUrl,
};

export interface Result {
    /**
     * 响应码
     */
    code: number;
    /**
     * 响应消息
     */
    msg: string;
    /**
     * 数据
     */
    data?: any;
}

export enum ResultEnum {
    SUCCESS = 200,
    ERROR = 400,
    PARAM_ERROR = 405,
    SERVER_ERROR = 500,
    NO_PERMISSION = 501,
    ACCESS_TOKEN_INVALID = 502, // accessToken失效
}

export const baseUrl: string = config.baseApiUrl;
// const baseUrl: string = 'http://localhost:18888/api';
// const baseWsUrl: string = config.baseWsUrl;

/**
 * 通知错误消息
 * @param msg 错误消息
 */
function notifyErrorMsg(msg: string) {
    // 危险通知
    ElMessage.error(msg);
}

// create an axios instance
const axiosInst = axios.create({
    baseURL: baseUrl, // url = base url + request url
    timeout: 60000, // request timeout
});

// request interceptor
axiosInst.interceptors.request.use(
    (config: any) => {
        // do something before request is sent
        const token = getToken();
        if (token) {
            // 设置token
            config.headers['Authorization'] = token;
            config.headers['ClientId'] = getClientId();
        }
        return config;
    },
    (error) => {
        return Promise.reject(error);
    }
);

// response interceptor
axiosInst.interceptors.response.use(
    (response) => response,
    (e: any) => {
        const rejectPromise = Promise.reject(e);

        if (axios.isCancel(e)) {
            console.log('请求已取消');
            return rejectPromise;
        }

        const statusCode = e.response?.status;
        if (statusCode == 500) {
            notifyErrorMsg('服务器未知异常');
            return rejectPromise;
        }

        if (statusCode == 404) {
            notifyErrorMsg('请求接口未找到');
            return rejectPromise;
        }

        if (e.message) {
            // 对响应错误做点什么
            if (e.message.indexOf('timeout') != -1) {
                notifyErrorMsg('网络请求超时');
                return rejectPromise;
            }

            if (e.message == 'Network Error') {
                notifyErrorMsg('网络连接错误');
                return rejectPromise;
            }
        }

        notifyErrorMsg('网络请求错误');
        return rejectPromise;
    }
);

/**
 * xhr请求url
 *
 * @param method 请求方法
 * @param url url
 * @param params 参数
 * @param options 可选
 * @returns
 */
export function xhrReq(method: string, url: string, params: any = null, options: any = {}) {
    if (!url) {
        throw new Error('请求url不能为空');
    }

    // 简单判断该url是否是restful风格
    if (url.indexOf('{') != -1) {
        url = templateResolve(url, params);
    }

    const req: any = {
        method,
        url,
        ...options,
    };

    // post和put使用json格式传参
    if (method === 'post' || method === 'put') {
        req.data = params;
    } else {
        req.params = params;
    }

    return axiosInst
        .request(req)
        .then((response) => {
            // 获取请求返回结果
            const result: Result = response.data;
            return parseResult(result);
        })
        .catch((e) => {
            return Promise.reject(e);
        });
}

/**
 * fetch请求url
 *
 * 该方法已处理请求结果中code != 200的message提示,如需其他错误处理(取消加载状态,重置对象状态等等),可catch继续处理
 *
 * @param {Object} method 请求方法(GET,POST,PUT,DELTE等)
 * @param {Object} uri    uri
 * @param {Object} params 参数
 */
async function request(method: string, url: string, params: any = null, options: any = {}): Promise<any> {
    const { execute, data } = useApiFetch(Api.create(url, method), params, options);
    await execute();
    return data.value;
}

/**
 * get请求uri
 * 该方法已处理请求结果中code != 200的message提示,如需其他错误处理(取消加载状态,重置对象状态等等),可catch继续处理
 *
 * @param {Object} url   uri
 * @param {Object} params 参数
 */
function get(url: string, params: any = null, options: any = {}): Promise<any> {
    return request('get', url, params, options);
}

function post(url: string, params: any = null, options: any = {}): Promise<any> {
    return request('post', url, params, options);
}

function put(url: string, params: any = null, options: any = {}): Promise<any> {
    return request('put', url, params, options);
}

function del(url: string, params: any = null, options: any = {}): Promise<any> {
    return request('delete', url, params, options);
}

function getApiUrl(url: string) {
    // 只是返回api地址而不做请求，用在上传组件之类的
    return baseUrl + url + '?' + joinClientParams();
}

// 组装客户端参数，包括 token 和 clientId
export function joinClientParams(): string {
    return `token=${getToken()}&clientId=${getClientId()}`;
}

function parseResult(result: Result) {
    if (result.code === ResultEnum.SUCCESS) {
        return result.data;
    }

    // 如果提示没有权限，则移除token，使其重新登录
    if (result.code === ResultEnum.NO_PERMISSION) {
        router.push({
            path: '/401',
        });
    }

    // 如果返回的code不为成功，则会返回对应的错误msg，则直接统一通知即可。忽略登录超时或没有权限的提示（直接跳转至401页面）
    if (result.msg && result?.code != ResultEnum.NO_PERMISSION) {
        notifyErrorMsg(result.msg);
    }

    return Promise.reject(result);
}
