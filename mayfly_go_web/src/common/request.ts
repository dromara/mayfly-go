import router from '../router';
import Axios from 'axios';
import config from './config';
import { getSession } from './utils/storage';
import { templateResolve } from './utils/string';
import { ElMessage } from 'element-plus';

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

enum ResultEnum {
    SUCCESS = 200,
    ERROR = 400,
    PARAM_ERROR = 405,
    SERVER_ERROR = 500,
    NO_PERMISSION = 501,
}

const baseUrl: string = config.baseApiUrl;
const baseWsUrl: string = config.baseWsUrl;

/**
 * 通知错误消息
 * @param msg 错误消息
 */
function notifyErrorMsg(msg: string) {
    // 危险通知
    ElMessage.error(msg);
}

// create an axios instance
const service = Axios.create({
    baseURL: baseUrl, // url = base url + request url
    timeout: 20000, // request timeout
});

// request interceptor
service.interceptors.request.use(
    (config: any) => {
        // do something before request is sent
        const token = getSession('token');
        if (token) {
            // 设置token
            config.headers['Authorization'] = token;
        }
        return config;
    },
    (error) => {
        return Promise.reject(error);
    }
);

// response interceptor
service.interceptors.response.use(
    (response) => {
        // 获取请求返回结果
        const data: Result = response.data;
        if (data.code === ResultEnum.SUCCESS) {
            return data.data;
        }
        // 如果提示没有权限，则移除token，使其重新登录
        if (data.code === ResultEnum.NO_PERMISSION) {
            router.push({
                path: '/401',
            });
        }
        return Promise.reject(data);
    },
    (e: any) => {
        const rejectPromise = Promise.reject(e);

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
 * 请求uri
 * 该方法已处理请求结果中code != 200的message提示,如需其他错误处理(取消加载状态,重置对象状态等等),可catch继续处理
 *
 * @param {Object} method 请求方法(GET,POST,PUT,DELTE等)
 * @param {Object} uri    uri
 * @param {Object} params 参数
 */
function request(method: string, url: string, params: any = null, headers: any = null, options: any = null): Promise<any> {
    if (!url) throw new Error('请求url不能为空');
    // 简单判断该url是否是restful风格
    if (url.indexOf('{') != -1) {
        url = templateResolve(url, params);
    }
    const query: any = {
        method,
        url: url,
        ...options,
    };
    if (headers) {
        query.headers = headers;
    }

    // post和put使用json格式传参
    if (method === 'post' || method === 'put') {
        query.data = params;
    } else {
        query.params = params;
    }
    return service
        .request(query)
        .then((res) => res)
        .catch((e) => {
            // 如果返回的code不为成功，则会返回对应的错误msg，则直接统一通知即可
            if (e.msg) {
                notifyErrorMsg(e.msg);
            }
            return Promise.reject(e);
        });
}

/**
 * get请求uri
 * 该方法已处理请求结果中code != 200的message提示,如需其他错误处理(取消加载状态,重置对象状态等等),可catch继续处理
 *
 * @param {Object} url   uri
 * @param {Object} params 参数
 */
function get(url: string, params: any = null, headers: any = null, options: any = null): Promise<any> {
    return request('get', url, params, headers, options);
}

function post(url: string, params: any = null, headers: any = null, options: any = null): Promise<any> {
    return request('post', url, params, headers, options);
}

function put(url: string, params: any = null, headers: any = null, options: any = null): Promise<any> {
    return request('put', url, params, headers, options);
}

function del(url: string, params: any = null, headers: any = null, options: any = null): Promise<any> {
    return request('delete', url, params, headers, options);
}

function getApiUrl(url: string) {
    // 只是返回api地址而不做请求，用在上传组件之类的
    return baseUrl + url + '?token=' + getSession('token');
}

export default {
    request,
    get,
    post,
    put,
    del,
    getApiUrl,
};
